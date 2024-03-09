package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"magpanel/models"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// 31a1243d85cd16ed13476c944890a556-8c90f339-4737e546
func getLocations(w http.ResponseWriter, r *http.Request) {

	rows, err := dataBase.Select("SELECT id, name, lat, lng, state, city, country FROM locations")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var locations []models.Location
	for rows.Next() {
		var l models.Location
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
	var l models.Location
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO locations(name, lat, lng, state, city, country) VALUES(?, ?, ?, ?, ?, ?)", l.Name, l.Lat, l.Lng, l.State, l.City, l.Country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// convert lastInsertID to int

	l.ID = int(lastInsertID)

	newValueBytes, err := json.Marshal(l)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nueva ubicación: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_location", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de ubicación: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(l)
}
func updateLocation(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	var l models.Location
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.Location

	rows, err := dataBase.SelectRow("SELECT id, name, lat, lng, state, city, country FROM locations WHERE id = ?", locationID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Ubicación no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Name, &old.Lat, &old.Lng, &old.State, &old.City, &old.Country)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antigua ubicación: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Update(true, "UPDATE locations SET name = ?, lat = ?, lng = ?, state = ?, city = ?, country = ? WHERE id = ?", l.Name, l.Lat, l.Lng, l.State, l.City, l.Country, locationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(l)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nueva ubicación: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_location", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de ubicación: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Ubicación con ID %s actualizada correctamente", locationID))
}
func deleteLocation(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	var old models.Location

	rows, err := dataBase.SelectRow("SELECT id, name, lat, lng, state, city, country FROM locations WHERE id = ?", locationID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Ubicación no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Name, &old.Lat, &old.Lng, &old.State, &old.City, &old.Country)

	_, err = dataBase.Delete(true, "DELETE FROM locations WHERE id = ?", locationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antigua ubicación: %v", err)
	}
	oldValue := string(oldValueBytes)

	// Registro del evento de eliminación
	if err := insertLog("delete_location", oldValue, "", r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de eliminación de ubicación: %v", err)
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}
func getLocationByID(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	var l models.Location

	rows, err := dataBase.SelectRow("SELECT id, name, lat, lng, state, city, country FROM locations WHERE id = ?", locationID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Ubicación no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&l.ID, &l.Name, &l.Lat, &l.Lng, &l.State, &l.City, &l.Country)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l)
}
