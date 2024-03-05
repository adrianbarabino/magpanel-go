package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/argon2"

	"github.com/mailgun/mailgun-go"
	"gopkg.in/ini.v1"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID           int            `json:"id"`
	Username     string         `json:"username"`
	Rank         int            `json:"rank"`
	Email        string         `json:"email"`
	Name         string         `json:"name,omitempty"`          // `omitempty` para que los valores nulos no aparezcan en el JSON
	PasswordHash string         `json:"password_hash,omitempty"` // No se incluirá en las respuestas JSON
	RecoveryHash sql.NullString `json:"recovery_hash,omitempty"` // Maneja NULL
	CreatedAt    time.Time      `json:"created_at,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at,omitempty"`
}

type Client struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Phone      string `json:"phone,omitempty"` // El campo omitempty indica que el campo puede ser omitido si está vacío
	Email      string `json:"email"`
	Web        string `json:"web,omitempty"`
	City       string `json:"city"`
	CategoryID int    `json:"category_id,omitempty"`
	Company    string `json:"company,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
	UpdatedAt  string `json:"updated_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
}
type Category struct {
	ID     int            `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Fields sql.NullString `json:"fields,omitempty"` // Cambiado a sql.NullString para manejar valores NULL
}
type Location struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Lat     float64 `json:"lat,omitempty"` // Usa float64 para coordenadas
	Lng     float64 `json:"lng,omitempty"` // Usa float64 para coordenadas
	State   string  `json:"state,omitempty"`
	City    string  `json:"city"`
	Country string  `json:"country"`
}
type ProjectStatus struct {
	ID         int    `json:"id"`
	StatusName string `json:"status_name"`
	Order      int    `json:"order"`
}

type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CategoryID  int    `json:"category_id,omitempty"`
	StatusID    int    `json:"status_id"`
	LocationID  int    `json:"location_id,omitempty"`
	AuthorID    int    `json:"author_id"`
	CreatedAt   string `json:"created_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
	UpdatedAt   string `json:"updated_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
}
type Setting struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

var db *sql.DB
var jwtKey []byte

func initDB() {
	var err error

	// Cargar el archivo de configuración
	cfg, err := ini.Load("data.conf")
	if err != nil {
		log.Fatal("Error al cargar el archivo de configuración: ", err)
	}

	// Leer las propiedades de la sección "database"
	dataSection := cfg.Section("keys")
	jwtKey = []byte(dataSection.Key("JWT_KEY").String())
	dbSection := cfg.Section("database")
	username := dbSection.Key("DB_USER").String()
	password := dbSection.Key("DB_PASS").String()
	host := dbSection.Key("DB_HOST").String()
	database := dbSection.Key("DB_NAME").String()

	// Construir la cadena de conexión
	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, host, database)

	db, err = sql.Open("mysql", connectionString)

	// Cambia los detalles de conexión según tu configuración de MySQL
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	var port string
	flag.StringVar(&port, "port", "3001", "Define el puerto en el que el servidor debería escuchar")
	flag.Parse()
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(CORSMiddleware) // Agrega el middleware de CORS aquí
	r.Post("/login", loginUser)

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		// Definir las rutas para usuarios
		r.Route("/users", func(r chi.Router) {
			r.Get("/", getUsers)          // GET /users - Obtener todos los usuarios
			r.Get("/{id}", getUserByID)   // GET /users/{id} - Obtener un usuario por su ID
			r.Post("/", createUser)       // POST /users - Crear un nuevo usuario
			r.Put("/{id}", updateUser)    // PUT /users/{id} - Actualizar un usuario existente
			r.Delete("/{id}", deleteUser) // DELETE /users/{id} - Eliminar un usuario

			// Rutas adicionales para operaciones específicas de usuarios
			r.Post("/change-password", changePassword)           // POST /users/change-password - Cambio de contraseña para un usuario
			r.Post("/request-recovery", requestPasswordRecovery) // POST /users/request-recovery - Solicitar recuperación de contraseña
		})
		// Rutas para "clients"
		r.Route("/clients", func(r chi.Router) {
			r.Get("/", getClients)
			r.Post("/", createClient)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getClientByID)
				r.Put("/", updateClient)
				r.Delete("/", deleteClient)
			})
		})

		// Rutas para "projects"
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", getProjects)
			r.Post("/", createProject)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getProjectByID)
				r.Put("/", updateProject)
				r.Delete("/", deleteProject)
			})
		})

		// Rutas para "project-statuses"
		r.Route("/project-statuses", func(r chi.Router) {
			r.Get("/", getProjectStatuses)
			r.Post("/", createProjectStatus)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getProjectStatusByID)
				r.Put("/", updateProjectStatus)
				r.Delete("/", deleteProjectStatus)
			})
		})

		// Rutas para "locations"
		r.Route("/locations", func(r chi.Router) {
			r.Get("/", getLocations)
			r.Post("/", createLocation)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getLocationByID)
				r.Put("/", updateLocation)
				r.Delete("/", deleteLocation)
			})
		})
		// Rutas para "settings"
		r.Route("/settings", func(r chi.Router) {
			r.Get("/", getSettings)
			r.Post("/", createSetting)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", updateSetting)
				r.Delete("/", deleteSetting)
			})
		})

		r.Route("/categories", func(r chi.Router) {
			r.Get("/", getCategories)   // Obtener todas las categorías
			r.Post("/", createCategory) // Crear una nueva categoría
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getCategoryByID)
				r.Put("/", updateCategory)
				r.Delete("/", deleteCategory)
			})
		})
	})

	// Inicia el servidor en el puerto especificado
	log.Printf("Servidor corriendo en el puerto %s\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obtenemos el token de autorización del encabezado
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No autorizado. Token no proporcionado.", http.StatusUnauthorized)
			return
		}

		// Verificamos si el token es "token-secreto" y lo salteamos
		if authHeader == "token-secreto" {
			next.ServeHTTP(w, r)
			return
		}

		// El token debe estar en el formato "Bearer {token}"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(w, "No autorizado. Formato de token inválido.", http.StatusUnauthorized)
			return
		}

		// Parseamos y validamos el token
		tokenString := tokenParts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				http.Error(w, "No autorizado. Token de autenticación inválido.", http.StatusUnauthorized)
				return
			}
			http.Error(w, "No autorizado. Token de autenticación inválido.", http.StatusUnauthorized)
			return
		}
		if !token.Valid {
			http.Error(w, "No autorizado. Token de autenticación inválido.", http.StatusUnauthorized)
			return
		}

		// Si el token es válido, pasamos al siguiente middleware o controlador
		next.ServeHTTP(w, r)
	})
}

// Estructura para almacenar las reclamaciones (claims) del token JWT
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

func generateSalt() string {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		// Manejar error adecuadamente
	}
	return hex.EncodeToString(salt)
}
func getSettings(w http.ResponseWriter, r *http.Request) {
	var settings []Setting

	rows, err := db.Query("SELECT id, `key`, `value`, `description` FROM settings")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.Description); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		settings = append(settings, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func createSetting(w http.ResponseWriter, r *http.Request) {
	var s Setting
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO settings (`key`, value, description) VALUES (?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.Key, s.Value, s.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func updateSetting(w http.ResponseWriter, r *http.Request) {
	settingID := chi.URLParam(r, "id")
	var s Setting
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE settings SET `key` = ?, `value` = ?, description = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.Key, s.Value, s.Description, settingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Ajuste con ID %s actualizado correctamente", settingID))
}

func deleteSetting(w http.ResponseWriter, r *http.Request) {
	settingID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM settings WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(settingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// 31a1243d85cd16ed13476c944890a556-8c90f339-4737e546
func getLocations(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, lat, lng, state, city, country FROM locations")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Lat, &l.Lng, &l.State, &l.City, &l.Country); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		locations = append(locations, l)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locations)
}

func createLocation(w http.ResponseWriter, r *http.Request) {
	var l Location
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO locations (name, lat, lng, state, city, country) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(l.Name, l.Lat, l.Lng, l.State, l.City, l.Country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(l)
}
func updateLocation(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	var l Location
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE locations SET name = ?, lat = ?, lng = ?, state = ?, city = ?, country = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(l.Name, l.Lat, l.Lng, l.State, l.City, l.Country, locationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Ubicación con ID %s actualizada correctamente", locationID))
}
func deleteLocation(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM locations WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(locationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}
func getLocationByID(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	var l Location
	err := db.QueryRow("SELECT id, name, lat, lng, state, city, country FROM locations WHERE id = ?", locationID).Scan(&l.ID, &l.Name, &l.Lat, &l.Lng, &l.State, &l.City, &l.Country)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Ubicación no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l)
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

func loginUser(w http.ResponseWriter, r *http.Request) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Intento de login con usuario: %s, contraseña: %s", loginData.Username, loginData.Password)

	var u User
	var salt string
	err := db.QueryRow("SELECT id, username, password_hash, salt FROM users WHERE username = ?", loginData.Username).Scan(&u.ID, &u.Username, &u.PasswordHash, &salt)
	if err != nil {
		http.Error(w, "No se encontró el usuario", http.StatusInternalServerError)

		return
	}
	log.Printf("Intento de login al usuario: %s, contraseña: %s y salt: %s", u.Username, u.PasswordHash, salt)

	// Verificar la contraseña utilizando la función comparePasswords
	if !comparePasswords(u.PasswordHash, loginData.Password, salt) {
		// log.Println("Contraseña incorrecta para el usuario:", u.Username)
		// log.Println("Contraseña proporcionada:", loginData.Password)
		// log.Println("Hash almacenado:", u.PasswordHash)
		// log.Println("Sal utilizada:", salt)
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	accessToken, err := generateAccessToken(u.ID)
	if err != nil {
		http.Error(w, "Error al generar el Access Token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
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
func requestPasswordRecovery(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Buscar usuario por email
	var u User
	err := db.QueryRow("SELECT id, email FROM users WHERE email = ?", requestData.Email).Scan(&u.ID, &u.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Email no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Generar recovery token (este es un ejemplo simplificado, considera usar algo más seguro)
	recoveryToken := fmt.Sprintf("%x", md5.Sum([]byte(time.Now().String()+u.Email)))

	// Almacenar recovery token en la base de datos (asumiendo que has añadido un campo para ello)
	_, err = db.Exec("UPDATE users SET recovery_hash = ? WHERE id = ?", recoveryToken, u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enviar recovery token por correo electrónico al usuario (implementa esta parte según tu lógica de envío de correos)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Instrucciones de recuperación enviadas."})
}
func getUsers(w http.ResponseWriter, r *http.Request) {
	var users []User

	rows, err := db.Query("SELECT `id`, `username`, `rank`, `email`, `name`, `recovery_hash`, `created_at`, `updated_at` FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var u User
		var createdAt, updatedAt []byte // Usar []byte para leer los valores de fecha y hora
		var recoveryHash sql.NullString // Usar sql.NullString para manejar valores NULL

		if err := rows.Scan(&u.ID, &u.Username, &u.Rank, &u.Email, &u.Name, &recoveryHash, &createdAt, &updatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convertir createdAt y updatedAt a time.Time
		u.CreatedAt, err = time.Parse("2006-01-02 15:04:05", string(createdAt))
		if err != nil {
			fmt.Printf("Error al parsear 'createdAt': %v\n", err)
		}
		u.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", string(updatedAt))
		if err != nil {
			fmt.Printf("Error al parsear 'updatedAt': %v\n", err)
		}

		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
func updateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verificar si el nombre de usuario ya existe para otro ID
	// if usernameExistsForOtherID(u.Username, userID) {
	//     http.Error(w, "El nombre de usuario ya existe para otro usuario", http.StatusBadRequest)
	//     return
	// }

	// Preparar la consulta SQL, incluyendo la contraseña y la sal solo si la contraseña ha sido proporcionada
	var query string
	var args []interface{}

	if u.PasswordHash != "" {
		// Generar nueva sal y hashear la nueva contraseña
		newSalt := generateSalt()
		newHashedPassword := hashPassword(u.PasswordHash, newSalt)

		query = "UPDATE users SET username = ?, rank = ?, email = ?, name = ?, password_hash = ?, salt = ? WHERE id = ?"
		args = append(args, u.Username, u.Rank, u.Email, u.Name, newHashedPassword, newSalt, userID)
	} else {
		query = "UPDATE users SET username = ?, rank = ?, email = ?, name = ? WHERE id = ?"
		args = append(args, u.Username, u.Rank, u.Email, u.Name, userID)
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Usuario con ID %s actualizado correctamente", userID))
}

// Verificar si el nombre de usuario ya existe en la base de datos
func usernameExists(username string) bool {
	var id int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			// No se encontró el nombre de usuario, por lo que no existe
			return false
		}
		// Manejar otros posibles errores
		log.Printf("Error al verificar el nombre de usuario: %v\n", err)
	}
	// Si la consulta no devolvió ErrNoRows, significa que se encontró un registro
	return true
}
func createUser(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verificar si el nombre de usuario ya existe
	if usernameExists(u.Username) {
		http.Error(w, "El nombre de usuario ya existe", http.StatusBadRequest)
		return
	}

	// Generar sal y hashear la contraseña
	salt := generateSalt()
	hashedPassword := hashPassword(u.PasswordHash, salt)
	// log.Println("Hash almacenado:", hashedPassword)
	// log.Println("Salt almacenado:", salt)
	// log.Println("Contraseña cruda almacenado:", u.PasswordHash)

	stmt, err := db.Prepare("INSERT INTO users (username, rank, email, name, password_hash, salt) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(u.Username, u.Rank, u.Email, u.Name, hashedPassword, salt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	u.PasswordHash = "" // No devolver la contraseña hasheada
	json.NewEncoder(w).Encode(u)
}

func changePassword(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Email       string `json:"email"`
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = ? AND recovery_hash = ?", requestData.Email, requestData.Token).Scan(&userID)
	if err != nil {
		http.Error(w, "Error al generar el token de acceso", http.StatusInternalServerError)

		return
	}

	// Generar nueva sal y hashear la nueva contraseña
	newSalt := generateSalt()
	newHashedPassword := hashPassword(requestData.NewPassword, newSalt)

	_, err = db.Exec("UPDATE users SET password_hash = ?, salt = ?, recovery_hash = NULL WHERE id = ?", newHashedPassword, newSalt, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Contraseña actualizada con éxito."})
}

func getMailgunConfig() (string, string, error) {
	var domain, apiKey string

	// Obtener el dominio de Mailgun
	err := db.QueryRow("SELECT `value` FROM settings WHERE `key` = 'mailgun_domain'").Scan(&domain)
	if err != nil {
		return "", "", err
	}

	// Obtener la API key de Mailgun
	err = db.QueryRow("SELECT `value` FROM settings WHERE `key` = 'mailgun_api_key'").Scan(&apiKey)
	if err != nil {
		return "", "", err
	}

	return domain, apiKey, nil
}

func sendRecoveryEmail(email, token string) error {
	// Obtener la configuración de Mailgun desde la base de datos
	domain, apiKey, err := getMailgunConfig()
	if err != nil {
		return err
	}

	// Configuración de Mailgun
	mg := mailgun.NewMailgun(domain, apiKey)

	// Construir el mensaje de correo electrónico
	sender := "no-reply@mag-servicios.com" // Considera también almacenar esto en la tabla de configuraciones
	subject := "Recuperación de contraseña"
	body := fmt.Sprintf("Tu token de recuperación es: %s introdúcelo en la página web para poder recuperar tu contraseña: https://gestion.mag-servicios.com/password-recovery/%s/ ", token, token)
	recipient := email

	message := mg.NewMessage(sender, subject, body, recipient)

	// Enviar el correo electrónico
	_, _, err = mg.Send(message)
	return err
}
func getUserByID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var u User
	var createdAt, updatedAt []byte // Usar []byte para leer los valores de fecha y hora
	var recoveryHash sql.NullString // Usar sql.NullString para manejar valores NULL

	err := db.QueryRow("SELECT `id`, `username`, `rank`, `email`, `name`, `password_hash`, `recovery_hash`, `created_at`, `updated_at` FROM users WHERE id = ?", userID).Scan(&u.ID, &u.Username, &u.Rank, &u.Email, &u.Name, &u.PasswordHash, &recoveryHash, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Convertir createdAt y updatedAt a time.Time
	u.CreatedAt, err = time.Parse("2006-01-02 15:04:05", string(createdAt))
	if err != nil {
		fmt.Printf("Error al parsear 'createdAt': %v\n", err)
	}
	u.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", string(updatedAt))
	if err != nil {
		fmt.Printf("Error al parsear 'updatedAt': %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM users WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}

func getCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, type, name, fields FROM categories")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Fields); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		categories = append(categories, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}
func createCategory(w http.ResponseWriter, r *http.Request) {
	var c Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO categories (type, name, fields) VALUES (?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.Type, c.Name, c.Fields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}
func updateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id")

	var c Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE categories SET type = ?, name = ?, fields = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.Type, c.Name, c.Fields, categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Categoría con ID %s actualizada correctamente", categoryID))
}

func deleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM categories WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}

func getClients(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, address, phone, email, web, city, category_id, company FROM clients")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clients = append(clients, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}
func getCategoryByID(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id") // Obtiene el ID de la categoría de la URL

	var c Category
	err := db.QueryRow("SELECT id, type, name, fields FROM categories WHERE id = ?", categoryID).Scan(&c.ID, &c.Type, &c.Name, &c.Fields)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoría no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func createClient(w http.ResponseWriter, r *http.Request) {
	var c Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Preparar la consulta SQL.
	stmt, err := db.Prepare("INSERT INTO clients(name, address, phone, email, web, city, category_id, company) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Ejecutar la consulta con los datos del cliente.
	result, err := stmt.Exec(c.Name, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener el ID del cliente recién creado.
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// convert lastInsertID to int

	c.ID = int(lastInsertID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func getClientByID(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var c Client
	err := db.QueryRow("SELECT id, name, address, phone, email, web, city, category_id, company FROM clients WHERE id = ?", clientID).Scan(&c.ID, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Cliente no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func updateClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var c Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE clients SET name = ?, address = ?, phone = ?, email = ?, web = ?, city = ?, category_id = ?, company = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.Name, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company, clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(fmt.Sprintf("Cliente con ID %s actualizado correctamente", clientID))
}

func deleteClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM clients WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content se suele utilizar para respuestas exitosas sin cuerpo
}

func getProjectStatuses(w http.ResponseWriter, r *http.Request) {
	var statuses []ProjectStatus

	rows, err := db.Query("SELECT id, status_name, `order` FROM project_statuses")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s ProjectStatus
		if err := rows.Scan(&s.ID, &s.StatusName, &s.Order); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		statuses = append(statuses, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func createProjectStatus(w http.ResponseWriter, r *http.Request) {
	var s ProjectStatus
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO project_statuses (status_name, `order`) VALUES (?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(s.StatusName, s.Order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.ID = int(lastInsertID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func getProjectStatusByID(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")

	var s ProjectStatus
	err := db.QueryRow("SELECT id, status_name, `order` FROM project_statuses WHERE id = ?", statusID).Scan(&s.ID, &s.StatusName, &s.Order)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Estado de proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func updateProjectStatus(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")
	var s ProjectStatus
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE project_statuses SET status_name = ?, `order` = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.StatusName, s.Order, statusID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Estado de proyecto con ID %s actualizado correctamente", statusID))
}

func deleteProjectStatus(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM project_statuses WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(statusID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getProjects(w http.ResponseWriter, r *http.Request) {
	var projects []Project

	rows, err := db.Query("SELECT id, name, description, category_id, status_id, location_id, author_id, created_at, updated_at FROM projects")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.StatusID, &p.LocationID, &p.AuthorID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		projects = append(projects, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func createProject(w http.ResponseWriter, r *http.Request) {
	var p Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO projects (name, description, category_id, status_id, location_id, author_id) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.ID = int(lastInsertID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func getProjectByID(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var p Project
	err := db.QueryRow("SELECT id, name, description, category_id, status_id, location_id, author_id, created_at, updated_at FROM projects WHERE id = ?", projectID).Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.StatusID, &p.LocationID, &p.AuthorID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func updateProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	var p Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE projects SET name = ?, description = ?, category_id = ?, status_id = ?, location_id = ?, author_id = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Proyecto con ID %s actualizado correctamente", projectID))
}

func deleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM projects WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Ajusta esto según tus necesidades
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return // Para preflight request
		}

		next.ServeHTTP(w, r)
	})
}
