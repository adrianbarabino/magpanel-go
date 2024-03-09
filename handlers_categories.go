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

func getCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := dataBase.Select("SELECT id, type, name, fields FROM categories")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
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
	var c models.Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO categories (type, name, fields) VALUES (?, ?, ?)", c.Type, c.Name, c.Fields)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// convert lastInsertID to int

	c.ID = int(lastInsertID)

	newValueBytes, err := json.Marshal(c)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nueva categoria: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_category", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de categoria: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}
func updateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id")

	var c models.Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.Category
	rows, err := dataBase.SelectRow("SELECT id, type, name, fields FROM categories WHERE id = ?", categoryID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoria no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Type, &old.Name, &old.Fields)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antigua categoria: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Update(true, "UPDATE categories SET type = ?, name = ?, fields = ? WHERE id = ?", c.Type, c.Name, c.Fields, categoryID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(c)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nueva categoria: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_category", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de categoria: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Categoría con ID %s actualizada correctamente", categoryID))
}

func deleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id")

	var old models.Category
	rows, err := dataBase.SelectRow("SELECT id, type, name, fields FROM categories WHERE id = ?", categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoria no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Type, &old.Name, &old.Fields)

	_, err = dataBase.Delete(true, "DELETE FROM categories WHERE id = ?", categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antigua categoria: %v", err)
	}
	oldValue := string(oldValueBytes)

	// Registro del evento de eliminación
	if err := insertLog("delete_category", oldValue, "", r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de eliminación de categoría: %v", err)
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}

func getCategoryByID(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id") // Obtiene el ID de la categoría de la URL

	var c models.Category
	rows, err := dataBase.SelectRow("SELECT id, type, name, fields FROM categories WHERE id = ?", categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoría no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&c.ID, &c.Type, &c.Name, &c.Fields)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}
