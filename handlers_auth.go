package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"magpanel/models"
	"net/http"
	"time"
)

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

	var u models.User
	var salt string

	rows, err := dataBase.SelectRow("SELECT id, username, password_hash, salt FROM users WHERE username = ?", loginData.Username)
	if err != nil {
		http.Error(w, "No se encontró el usuario", http.StatusInternalServerError)
		return
	}
	rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &salt)

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

func requestPasswordRecovery(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Buscar usuario por email
	var u models.User
	rows, err := dataBase.SelectRow("SELECT id, email FROM users WHERE email = ?", requestData.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Email no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&u.ID, &u.Email)

	// Generar recovery token y almacenar en la base de datos con la fecha y hora actual
	recoveryToken := fmt.Sprintf("%x", md5.Sum([]byte(time.Now().String()+u.Email)))

	// Asumiendo que ahora tienes una columna recovery_hash_time
	_, err = dataBase.Update(true, "UPDATE users SET recovery_hash = ?, recovery_hash_time = NOW() WHERE id = ?", recoveryToken, u.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enviar recovery token por correo electrónico al usuario (implementa esta parte según tu lógica de envío de correos)

	// send recovery with mailgun

	err = sendRecoveryEmail(u.Email, recoveryToken)
	if err != nil {
		fmt.Println("Error al enviar el correo electrónico de recuperación:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Instrucciones de recuperación enviadas."})
}

func changePassword(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var userID int
	var recoveryHashTimeBytes []byte

	// Recuperar la fecha y hora del token además del ID del usuario
	rows, err := dataBase.SelectRow("SELECT id, recovery_hash_time FROM users WHERE recovery_hash = ?", requestData.Token)
	if err != nil {
		fmt.Println("Error al obtener el token de recuperación:", err)
		http.Error(w, "Error al obtener el token de recuperación", http.StatusInternalServerError)
		return
	}
	rows.Scan(&userID, &recoveryHashTimeBytes)

	// Convertir recoveryHashTimeBytes a una cadena y luego parsearla como una fecha y hora
	recoveryHashTimeString := string(recoveryHashTimeBytes)
	recoveryHashTime, err := time.Parse("2006-01-02 15:04:05", recoveryHashTimeString) // Ajusta el formato según sea necesario
	if err != nil {
		fmt.Printf("Error al parsear 'recoveryHashTime': %v\n", err)
		http.Error(w, "Error al parsear 'recoveryHashTime'", http.StatusInternalServerError)
		return
	}
	// Verificar si han pasado 24 horas desde que se generó el token
	if time.Since(recoveryHashTime).Hours() > 24 {
		http.Error(w, "El token de recuperación ha expirado", http.StatusBadRequest)
		return
	}

	// Generar nueva sal y hashear la nueva contraseña
	newSalt := generateSalt()
	newHashedPassword := hashPassword(requestData.NewPassword, newSalt)

	_, err = dataBase.Update(true, "UPDATE users SET password_hash = ?, salt = ?, recovery_hash = NULL WHERE id = ?", newHashedPassword, newSalt, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Contraseña actualizada con éxito."})
}
