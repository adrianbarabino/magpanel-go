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

func getProviders(w http.ResponseWriter, r *http.Request) {

	rows, err := dataBase.Select("SELECT id, code, name, address, phone, email, web, city, category_id, company FROM providers")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var providers []models.Provider
	for rows.Next() {
		var c models.Provider
		err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		providers = append(providers, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

func createProvider(w http.ResponseWriter, r *http.Request) {
	var c models.Provider
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Preparar la consulta SQL.

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO providers(name, code, address, phone, email, web, city, category_id, company) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)", c.Name, c.Code, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// convert lastInsertID to int

	c.ID = int(lastInsertID)

	newValueBytes, err := json.Marshal(c)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo providere: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_provider", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de providere: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func getProviderByID(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "id")

	var c models.Provider

	rows, err := dataBase.SelectRow("SELECT id, code, name, address, phone, email, web, city, category_id, company FROM providers WHERE id = ?", providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Providere no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&c.ID, &c.Code, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func updateProvider(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "id")

	var c models.Provider
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.Provider
	// No es necesario verificar el código del providere, ya que no se puede modificar
	rows, err := dataBase.SelectRow("SELECT id, name, address, phone, email, web, city, category_id, company FROM providers WHERE id = ?", providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Providere no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Name, &old.Address, &old.Phone, &old.Email, &old.Web, &old.City, &old.CategoryID, &old.Company)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo providere: %v", err)
	}
	oldValue := string(oldValueBytes)

	// No es necesario verificar el código del providere, ya que no se puede modificar
	_, err = dataBase.Update(true, "UPDATE providers SET name = ?, address = ?, phone = ?, email = ?, web = ?, city = ?, category_id = ?, company = ? WHERE id = ?", c.Name, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company, providerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(c)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo providere: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_provider", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de providere: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(fmt.Sprintf("Providere con ID %s actualizado correctamente", providerID))
}

func deleteProvider(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "id")

	_, err := dataBase.Delete(true, "DELETE FROM providers WHERE id = ?", providerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var old models.Provider

	rows, err := dataBase.SelectRow("SELECT id, code, name, address, phone, email, web, city, category_id, company FROM providers WHERE id = ?", providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Providere no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Code, &old.Name, &old.Address, &old.Phone, &old.Email, &old.Web, &old.City, &old.CategoryID, &old.Company)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo providere: %v", err)
	}
	oldValue := string(oldValueBytes)

	// Registro del evento de eliminación
	if err := insertLog("delete_provider", oldValue, "", r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de eliminación de providere: %v", err)
	}
	w.WriteHeader(http.StatusNoContent) // 204 No Content se suele utilizar para respuestas exitosas sin cuerpo
}
