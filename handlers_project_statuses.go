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

func getProjectStatuses(w http.ResponseWriter, r *http.Request) {
	var statuses []models.ProjectStatus

	rows, err := dataBase.Select("SELECT id, status_name, `order` FROM project_statuses")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s models.ProjectStatus
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
	var s models.ProjectStatus
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lastInsertID, err := dataBase.Insert(true, "INSERT INTO project_statuses (status_name, `order`) VALUES (?, ?)", s.StatusName, s.Order)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.ID = int(lastInsertID)

	newValueBytes, err := json.Marshal(s)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo estado de proyecto: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_project_status", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de estado de proyecto: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func getProjectStatusByID(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")

	var s models.ProjectStatus
	rows, err := dataBase.SelectRow("SELECT id, status_name, `order` FROM project_statuses WHERE id = ?", statusID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Estado de proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&s.ID, &s.StatusName, &s.Order)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func updateProjectStatus(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")
	var s models.ProjectStatus
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.ProjectStatus
	rows, err := dataBase.SelectRow("SELECT id, status_name, `order` FROM project_statuses WHERE id = ?", statusID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Estado de proyecto no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.StatusName, &old.Order)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antigua Estado de proyecto: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Update(true, "UPDATE project_statuses SET status_name = ?, `order` = ? WHERE id = ?", s.StatusName, s.Order, statusID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(s)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo estado de proyecto: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_location", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de estado de proyecto: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Estado de proyecto con ID %s actualizado correctamente", statusID))
}

func deleteProjectStatus(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")

	var old models.ProjectStatus
	rows, err := dataBase.SelectRow("SELECT id, status_name, `order` FROM project_statuses WHERE id = ?", statusID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Estado de proyecto no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.StatusName, &old.Order)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antigua Estado de proyecto: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Delete(true, "DELETE FROM project_statuses WHERE id = ?", statusID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Registro del evento de eliminación
	if err := insertLog("delete_project_status", oldValue, "", r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de eliminación de estado proyecto: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
