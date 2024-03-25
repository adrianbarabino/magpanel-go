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

func getSettings(w http.ResponseWriter, r *http.Request) {
	var settings []models.Setting

	rows, err := dataBase.Select("SELECT id, `key`, `value`, description FROM settings")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s models.Setting
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
	var s models.Setting
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO settings (`key`, `value`, description) VALUES (?, ?, ?)", s.Key, s.Value, s.Description)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener el ID del ajuste recién creado.
	s.ID = int(lastInsertID)
	newValueBytes, err := json.Marshal(s)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo ajuste: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_setting", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de ajuste: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func updateSetting(w http.ResponseWriter, r *http.Request) {
	settingID := chi.URLParam(r, "id")
	var s models.SettingVal
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.Setting
	rows, err := dataBase.SelectRow("SELECT `id`, `key`, `value` FROM settings WHERE id = ?", settingID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Estado de proyecto no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Key, &old.Value)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antigua Estado de proyecto: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Update(true, "UPDATE settings SET `value` = ? WHERE id = ?", s.Value, settingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// convert settingID type string to int

	s.ID = settingID

	newValueBytes, err := json.Marshal(s)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nueva configuracion: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_setting", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de configuracion: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Ajuste con ID %s actualizado correctamente", settingID))
}

func deleteSetting(w http.ResponseWriter, r *http.Request) {
	settingID := chi.URLParam(r, "id")

	_, err := dataBase.Delete(true, "DELETE FROM settings WHERE id = ?", settingID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getLogs(w http.ResponseWriter, r *http.Request) {
	var logs []models.Log

	query := "SELECT logs.id, logs.type, logs.old_value, logs.new_value, logs.user_id, logs.created_at, users.username FROM logs JOIN users ON logs.user_id = users.id "
	if order := r.URL.Query().Get("order"); order != "" {
		query += "ORDER BY " + order
	} else {
		query += "ORDER BY logs.created_at DESC "

	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		query += "LIMIT " + limit + " "
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		query += "OFFSET " + offset
	}

	rows, err := dataBase.Select(query)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var l models.Log
		if err := rows.Scan(&l.ID, &l.Type, &l.OldValue, &l.NewValue, &l.UserID, &l.CreatedAt, &l.Username); err != nil {

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// set the date -3 hours for fix gmt-3 of argentina
		createdAt, err := time.Parse("2006-01-02 15:04:05", l.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		createdAt = createdAt.Add(-3 * time.Hour)
		l.CreatedAt = createdAt.Format("2006-01-02 15:04:05")

		logs = append(logs, l)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
