package main

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"magpanel/models"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/mailgun/mailgun-go"
	"golang.org/x/crypto/argon2"
)

func sendRecoveryEmail(email, token string) error {
	// Obtener la configuración de Mailgun desde la base de datos
	domain, apiKey, err := getMailgunConfig()
	if err != nil {
		return err
	}

	// Configuración de Mailgun
	mg := mailgun.NewMailgun(domain, apiKey)

	// Construir el mensaje de correo electrónico en formato HTML
	sender := "no-reply@mag-servicios.com" // Considera también almacenar esto en la tabla de configuraciones
	subject := "Recuperación de contraseña"
	logoURL := "https://mag-servicios.com/wp-content/uploads/2022/12/01-4.png"
	body := fmt.Sprintf(`
	<html>
	<body>
		<div style="text-align: center;">
			<img src="%s" alt="Logo MAG Servicios" style="max-width: 200px; margin-bottom: 20px;">
			<p>Tu token de recuperación es: <strong>%s</strong></p>
			<p>Introdúcelo en la página web para poder recuperar tu contraseña:</p>
			<a href="https://gestion.mag-servicios.com/password-recovery/%s/" style="display: inline-block; background-color: #007BFF; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Recuperar Contraseña</a>
		</div>
	</body>
	</html>
	`, logoURL, token, token)

	recipient := email

	// Asegúrate de utilizar la función adecuada para enviar mensajes en formato HTML.
	// Si estás utilizando Mailgun, `NewMessage` debería ser reemplazado por `NewMessage` con el parámetro adecuado para indicar que el contenido es HTML
	message := mg.NewMessage(sender, subject, "", recipient) // El cuerpo vacío se reemplaza por el parámetro de HTML a continuación
	message.SetHtml(body)

	// Enviar el correo electrónico
	_, _, err = mg.Send(message)
	return err
}

// Verificar si el nombre de usuario ya existe en la base de datos
func usernameExists(username string) bool {
	var id int
	rows, err := dataBase.SelectRow("SELECT id FROM users WHERE username = ?", username)
	if err != nil {
		if err == sql.ErrNoRows {
			// No se encontró el nombre de usuario, por lo que no existe
			return false
		}
		// Manejar otros posibles errores
		log.Printf("Error al verificar el nombre de usuario: %v\n", err)
	}
	rows.Scan(&id)
	// Si la consulta no devolvió ErrNoRows, significa que se encontró un registro
	return true
}

func generateAccessToken(userID int) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // El token expira en 24 horas

	// Crear un nuevo token que será del tipo JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID, // Puedes añadir más datos del usuario según sea necesario
		"exp":     expirationTime.Unix(),
	})

	// Firmar el token con tu clave secreta
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func generateSalt() string {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		// Manejar error adecuadamente
	}
	return hex.EncodeToString(salt)
}

func getUserFromToken(tokenString string) (*models.User, error) {
	// Parsea el token
	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil // `jwtKey` es tu clave secreta para validar el token
	})

	if err != nil {
		return nil, err // Maneja el error de parseo
	}

	// Comprueba la validez del token
	if claims, ok := token.Claims.(*models.Claims); ok && token.Valid {
		// Aquí `Claims` es la estructura que usaste para guardar la información en el token
		userID := claims.UserID

		// Usar el `userID` para buscar al usuario en tu base de datos
		var user models.User
		rows, err := dataBase.SelectRow("SELECT id, username, email FROM users WHERE id = ?", userID)
		if err != nil {
			return nil, err // Maneja el error de la base de datos
		}
		rows.Scan(&user.ID, &user.Username, &user.Email)

		return &user, nil
	} else {
		return nil, fmt.Errorf("token inválido o expirado")
	}
}

func hashPassword(password, salt string) string {
	saltBytes, _ := hex.DecodeString(salt)
	hash := argon2.IDKey([]byte(password), saltBytes, 1, 64*1024, 4, 32)
	return hex.EncodeToString(hash)
}
func comparePasswords(hashedPassword, password, salt string) bool {
	// Decodificar la sal desde hexadecimal a bytes
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		log.Println("Error al decodificar la sal:", err)
		return false
	}

	// Decodificar el hash de la contraseña desde hexadecimal a bytes
	hashedPasswordBytes, err := hex.DecodeString(hashedPassword)
	if err != nil {
		log.Println("Error al decodificar el hash de la contraseña:", err)
		return false
	}

	// Calcular el hash de la contraseña proporcionada
	hash := argon2.IDKey([]byte(password), saltBytes, 1, 64*1024, 4, 32)

	log.Println("Contraseña proporcionada:", password)
	log.Println("Hash de la contraseña proporcionada:", hex.EncodeToString(hash))
	log.Println("Hash almacenado:", hashedPassword)
	log.Println("Sal utilizada:", salt)

	// Comparar los hashes
	return subtle.ConstantTimeCompare(hashedPasswordBytes, hash) == 1
}

func verifyPassword(password, hashedPassword, salt string) bool {
	// Generar el hash de la contraseña proporcionada con la sal almacenada
	newHash := hashPassword(password, salt)
	// Comparar los hashes
	return hashedPassword == newHash
}

func checkAccessToken(accessToken string) (int, error) {
	// Parsear el token
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return 0, err
	}

	// Verificar si el token es válido
	if !token.Valid {
		return 0, fmt.Errorf("Token inválido")
	}

	// Obtener las reclamaciones (claims) del token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("Error al obtener las reclamaciones del token")
	}

	// Obtener el ID de usuario desde las reclamaciones
	userID := int(claims["user_id"].(float64))

	return userID, nil
}

func getCurrentUser(r *http.Request) (*models.User, error) {
	authToken := r.Header.Get("Authorization")

	// Caso especial para "token-secreto"
	if authToken == "token-secreto" {
		// Asigna un usuario administrador predeterminado o realiza alguna otra acción específica
		// Este es solo un ejemplo, ajusta según tu lógica de negocio
		return &models.User{ID: 1, Username: "admin", Email: "admin@example.com"}, nil
	}

	// Para los casos que no son el "token-secreto", asumimos que es un JWT
	// Quitamos el prefijo "Bearer " si está presente
	tokenString := strings.TrimPrefix(authToken, "Bearer ")

	// Parsea y valida el JWT para obtener el usuario
	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil // `jwtKey` es tu clave secreta para validar el token
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("token inválido o expirado")
	}

	if claims, ok := token.Claims.(*models.Claims); ok {
		// Usar el `userID` para buscar al usuario en tu base de datos
		var user models.User
		rows, err := dataBase.SelectRow("SELECT id, username, email FROM users WHERE id = ?", claims.UserID)
		if err != nil {
			return nil, err // Maneja el error de la base de datos
		}
		rows.Scan(&user.ID, &user.Username, &user.Email)
		return &user, nil
	}

	return nil, fmt.Errorf("no se pudo procesar el token")
}
