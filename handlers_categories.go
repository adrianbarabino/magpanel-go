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
	rows, err := dataBase.Select("SELECT id, type, name, code, fields FROM categories")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []models.Category

	for rows.Next() {
		var c models.Category
		var codeNullString sql.NullString

		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &codeNullString, &c.FieldsJSON); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Si el valor es válido, asigne a c.Code, de lo contrario, c.Code será una cadena vacía
		if codeNullString.Valid {
			c.Code = codeNullString.String
		} else {
			c.Code = ""
		}

		if c.FieldsJSON != "" {
			if err := json.Unmarshal([]byte(c.FieldsJSON), &c.Fields); err != nil {
				http.Error(w, "Error al deserializar los campos: "+err.Error(), http.StatusInternalServerError)
				return
			}
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

	// if is Type project and have c.Code, check if there any category with the same code COALESCE
	if c.Type == "projects" && c.Code != "" {
		// check if there any category with the same code
		var existing models.Category
		row, err := dataBase.SelectRow("SELECT id FROM categories WHERE code = ?", c.Code)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = row.Scan(&existing.ID)
		if err != sql.ErrNoRows {
			http.Error(w, "Ya existe una categoría con el mismo código", http.StatusConflict)
			return
		}
	}
	var fieldsDataString string
	if c.Fields == nil {
		fieldsDataString = "[]" // empty array

	} else {
		fieldsData, err := json.Marshal(c.Fields)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fieldsDataString = string(fieldsData)

	}
	lastInsertID, err := dataBase.Insert(true, "INSERT INTO categories (type, name, fields) VALUES (?, ?, ?)", c.Type, c.Name, fieldsDataString)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if c.Type == "projects" && c.Code != "" {
		// update the code
		_, err = dataBase.Update(true, "UPDATE categories SET code = ? WHERE id = ?", c.Code, lastInsertID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

	// Serializa los campos antes de actualizar en la base de datos
	fieldsData, err := json.Marshal(c.Fields)
	if err != nil {
		http.Error(w, "Error al serializar los campos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var old models.Category
	rows, err := dataBase.SelectRow("SELECT id, type, name, fields FROM categories WHERE id = ?", categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoría no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := rows.Scan(&old.ID, &old.Type, &old.Name, &old.FieldsJSON); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Deserializa el campo fields del formato JSON para old
	if err := json.Unmarshal([]byte(old.FieldsJSON), &old.Fields); err != nil {
		http.Error(w, "Error al deserializar los campos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		log.Printf("Error al serializar antigua categoría: %v", err)
	}
	oldValue := string(oldValueBytes)

	_, err = dataBase.Update(true, "UPDATE categories SET type = ?, name = ?, fields = ? WHERE id = ?", c.Type, c.Name, string(fieldsData), categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(c)
	if err != nil {
		log.Printf("Error al serializar nueva categoría: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_category", oldValue, newValue, r); err != nil {
		log.Printf("Error al insertar el registro de actualización de categoría: %v", err)
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
	var codeNullString sql.NullString

	rows, err := dataBase.SelectRow("SELECT id, code, type, name, fields FROM categories WHERE id = ?", categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoría no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Asumiendo que FieldsJSON es un campo en models.Category que se utiliza para escanear el JSON crudo
	if err := rows.Scan(&c.ID, &codeNullString, &c.Type, &c.Name, &c.FieldsJSON); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Si el valor es válido, asigne a c.Code, de lo contrario, c.Code será una cadena vacía
	if codeNullString.Valid {
		c.Code = codeNullString.String
	} else {
		c.Code = ""
	}

	if c.FieldsJSON != "" {
		if err := json.Unmarshal([]byte(c.FieldsJSON), &c.Fields); err != nil {
			http.Error(w, "Error al deserializar los campos: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}
