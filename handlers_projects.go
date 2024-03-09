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

func getProjects(w http.ResponseWriter, r *http.Request) {
	var projects []models.Project

	// get also the categoryName, statusName, locationName and authorName with a JOIN
	rows, err := dataBase.Select("SELECT p.id, p.name, p.description, p.category_id, p.status_id, p.location_id, p.author_id, p.created_at, p.updated_at, c.name, ps.status_name, l.name, u.name FROM projects p JOIN categories c ON p.category_id = c.id JOIN project_statuses ps ON p.status_id = ps.id JOIN locations l ON p.location_id = l.id JOIN users u ON p.author_id = u.id")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.StatusID, &p.LocationID, &p.AuthorID, &p.CreatedAt, &p.UpdatedAt, &p.CategoryName, &p.StatusName, &p.LocationName, &p.AuthorName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		projects = append(projects, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func createProject(w http.ResponseWriter, r *http.Request) {
	var p models.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	currentUser, err := getCurrentUser(r)

	if err != nil {
		http.Error(w, "Error al obtener el usuario actual", http.StatusInternalServerError)
		return
	}

	p.AuthorID = currentUser.ID
	lastInsertID, err := dataBase.Insert(true, "INSERT INTO projects (name, description, category_id, status_id, location_id, author_id) VALUES (?, ?, ?, ?, ?, ?)", p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.ID = int(lastInsertID)

	newValueBytes, err := json.Marshal(p)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo proyecto: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_project", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de proyecto: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func getProjectByID(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var p models.Project

	// Change the query for the project to include the status, author, location and category names

	// also return the category, status, location, author NAME and ID, both of them
	rows, err := dataBase.SelectRow("SELECT p.id, p.name, p.description, c.id, c.name, ps.id, ps.status_name, l.id, l.name, u.id, u.name, p.created_at, p.updated_at FROM projects p JOIN categories c ON p.category_id = c.id JOIN project_statuses ps ON p.status_id = ps.id JOIN locations l ON p.location_id = l.id JOIN users u ON p.author_id = u.id WHERE p.id = ?", projectID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName, &p.StatusID, &p.StatusName, &p.LocationID, &p.LocationName, &p.AuthorID, &p.AuthorUsername, &p.CreatedAt, &p.UpdatedAt)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func updateProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	var p models.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.Project
	rows, err := dataBase.SelectRow("SELECT id, name, description, category_id, status_id, location_id, author_id, created_at, updated_at FROM projects WHERE id = ?", projectID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Name, &old.Description, &old.CategoryID, &old.StatusID, &old.LocationID, &old.AuthorID, &old.CreatedAt, &old.UpdatedAt)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo Proyecto: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Update(true, "UPDATE projects SET name = ?, description = ?, category_id = ?, status_id = ?, location_id = ?, author_id = ? WHERE id = ?", p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(p)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo proyecto: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_project", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de proyecto: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Proyecto con ID %s actualizado correctamente", projectID))
}

func deleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var old models.Project
	rows, err := dataBase.SelectRow("SELECT id, name, description, category_id, status_id, location_id, author_id, created_at, updated_at FROM projects WHERE id = ?", projectID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Name, &old.Description, &old.CategoryID, &old.StatusID, &old.LocationID, &old.AuthorID, &old.CreatedAt, &old.UpdatedAt)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo Proyecto: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Delete(true, "DELETE FROM projects WHERE id = ?", projectID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Registro del evento de eliminación
	if err := insertLog("delete_project", oldValue, "", r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de eliminación de proyecto: %v", err)
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}
