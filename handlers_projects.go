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
	// if get GET["limit"] and GET["offset"] values, use them in the query
	// if GET["limit"] is not provided, get all
	// if GET["offset"] is not provided, start from 0
	// if GET["order"] is provided, order by that column
	query := "SELECT p.id, p.code, p.name, p.description, p.category_id, p.client_id, cl.name, p.status_id, p.location_id, p.author_id, p.created_at, p.updated_at, c.name, ps.status_name, l.name, u.name FROM projects p JOIN categories c ON p.category_id = c.id JOIN project_statuses ps ON p.status_id = ps.id JOIN locations l ON p.location_id = l.id JOIN users u ON p.author_id = u.id JOIN clients cl ON p.client_id = cl.id "
	if order := r.URL.Query().Get("order"); order != "" {
		query += "ORDER BY " + order + " "
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		query += "LIMIT " + limit + " "
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		query += "OFFSET " + offset
	}

	// get also the categoryName, statusName, locationName and authorName with a JOIN, c.name and client_id and name
	rows, err := dataBase.Select(query)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.CategoryID, &p.ClientID, &p.ClientName, &p.StatusID, &p.LocationID, &p.AuthorID, &p.CreatedAt, &p.UpdatedAt, &p.CategoryName, &p.StatusName, &p.LocationName, &p.AuthorName); err != nil {
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

	// obtain the code from the client_id
	var clientCode string
	row, err := dataBase.SelectRow("SELECT code FROM clients WHERE id = ?", p.ClientID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Cliente no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	row.Scan(&clientCode)

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO projects (name, description, category_id, status_id, location_id, author_id, client_id) VALUES (?, ?, ?, ?, ?, ?, ?)", p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID, p.ClientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.ID = int(lastInsertID)

	var categoryCode string
	row, err = dataBase.SelectRow("SELECT code FROM categories WHERE id = ?", p.CategoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoría no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	row.Scan(&categoryCode)

	p.Code = categoryCode + "-" + clientCode + "-" + fmt.Sprintf("%04d", p.ID)

	// save the code
	_, err = dataBase.Update(true, "UPDATE projects SET code = ? WHERE id = ?", p.Code, p.ID)
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
	rows, err := dataBase.SelectRow("SELECT p.id, p.code, p.name, p.description, c.id, c.name, ps.id, ps.status_name, l.id, l.name, l.lat, l.lng, u.id, u.name, p.client_id, cl.name, p.created_at, p.updated_at FROM projects p JOIN categories c ON p.category_id = c.id JOIN project_statuses ps ON p.status_id = ps.id JOIN locations l ON p.location_id = l.id JOIN users u ON p.author_id = u.id JOIN clients cl ON p.client_id = cl.id WHERE p.id = ?", projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName, &p.StatusID, &p.StatusName, &p.LocationID, &p.LocationName, &p.LocationLat, &p.LocationLng, &p.AuthorID, &p.AuthorName, &p.ClientID, &p.ClientName, &p.CreatedAt, &p.UpdatedAt)

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
	rows, err := dataBase.SelectRow("SELECT id, name, description, category_id, status_id, location_id, author_id, client_id, created_at, updated_at FROM projects WHERE id = ?", projectID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Name, &old.Description, &old.CategoryID, &old.StatusID, &old.LocationID, &old.AuthorID, &old.ClientID, &old.CreatedAt, &old.UpdatedAt)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo Proyecto: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Update(true, "UPDATE projects SET name = ?, description = ?, category_id = ?, status_id = ?, location_id = ?, author_id = ?, client_id = ? WHERE id = ?", p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID, p.ClientID, projectID)
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
	rows, err := dataBase.SelectRow("SELECT id, code, name, description, category_id, status_id, location_id, author_id, client_id, created_at, updated_at FROM projects WHERE id = ?", projectID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Code, &old.Name, &old.Description, &old.CategoryID, &old.StatusID, &old.LocationID, &old.AuthorID, &old.ClientID, &old.CreatedAt, &old.UpdatedAt)

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
