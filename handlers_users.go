package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"magpanel/models"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func getUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User

	rows, err := dataBase.Select("SELECT id, username, rank, email, name, recovery_hash, created_at, updated_at FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var u models.User
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
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.User

	rows, err := dataBase.SelectRow("SELECT id, username, rank, email, name, password_hash FROM users WHERE id = ?", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}
	rows.Scan(&old.ID, &old.Username, &old.Rank, &old.Email, &old.Name, &old.PasswordHash)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo usuario: %v", err)
	}
	oldValue := string(oldValueBytes)

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
		fmt.Println("Nueva contraseña:", u.PasswordHash)
		fmt.Println("Nueva contraseña hasheada:", newHashedPassword)
		fmt.Println("Nueva sal:", newSalt)

		query = "UPDATE users SET `username` = ?, `rank` = ?, `email` = ?, `name` = ?, `password_hash` = ?, `salt` = ? WHERE `id` = ?"
		args = append(args, u.Username, u.Rank, u.Email, u.Name, newHashedPassword, newSalt, userID)
	} else {
		query = "UPDATE users SET `username` = ?, `rank` = ?, `email` = ?, `name` = ? WHERE `id` = ?"
		args = append(args, u.Username, u.Rank, u.Email, u.Name, userID)
	}

	_, err = dataBase.Update(true, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(u)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo usuario: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_user", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de usuario: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Usuario con ID %s actualizado correctamente", userID))
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
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

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO users (`username`, `rank`, `email`, `name`, `password_hash`, `salt`) VALUES (?, ?, ?, ?, ?, ?)", u.Username, u.Rank, u.Email, u.Name, hashedPassword, salt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u.ID = int(lastInsertID)

	newValueBytes, err := json.Marshal(u)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo usuario: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_user", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de usuario: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	u.PasswordHash = "" // No devolver la contraseña hasheada
	json.NewEncoder(w).Encode(u)
}

func getUserByID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var u models.User
	var createdAt, updatedAt []byte // Usar []byte para leer los valores de fecha y hora
	var recoveryHash sql.NullString // Usar sql.NullString para manejar valores NULL

	rows, err := dataBase.SelectRow("SELECT `id`, `username`, `rank`, `email`, `name`, `password_hash`, `recovery_hash`, `created_at`, `updated_at` FROM users WHERE id = ?", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&u.ID, &u.Username, &u.Rank, &u.Email, &u.Name, &u.PasswordHash, &recoveryHash, &createdAt, &updatedAt)

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

	var old models.User
	rows, err := dataBase.SelectRow("SELECT `id`, `username`, `rank`, `email`, `name`, `password_hash` FROM users WHERE id = ?", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Username, &old.Rank, &old.Email, &old.Name, &old.PasswordHash)

	_, err = dataBase.Delete(true, "DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo usuario: %v", err)
	}
	oldValue := string(oldValueBytes)

	// Registro del evento de eliminación
	if err := insertLog("delete_user", oldValue, "", r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de eliminación de usuario: %v", err)
	}
	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}
