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

func getClients(w http.ResponseWriter, r *http.Request) {

	rows, err := dataBase.Select("SELECT id, code, name, address, phone, email, web, city, category_id, company FROM clients")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var c models.Client
		err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clients = append(clients, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

func createClient(w http.ResponseWriter, r *http.Request) {
	var c models.Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Preparar la consulta SQL.

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO clients(name, code, address, phone, email, web, city, category_id, company) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)", c.Name, c.Code, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// convert lastInsertID to int

	c.ID = int(lastInsertID)

	newValueBytes, err := json.Marshal(c)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo cliente: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de creación
	if err := insertLog("create_client", "", newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de creación de cliente: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func getClientByID(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var c models.Client

	rows, err := dataBase.SelectRow("SELECT id, code, name, address, phone, email, web, city, category_id, company FROM clients WHERE id = ?", clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Cliente no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&c.ID, &c.Code, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func updateClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var c models.Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var old models.Client
	// No es necesario verificar el código del cliente, ya que no se puede modificar
	rows, err := dataBase.SelectRow("SELECT id, name, address, phone, email, web, city, category_id, company FROM clients WHERE id = ?", clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Cliente no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Name, &old.Address, &old.Phone, &old.Email, &old.Web, &old.City, &old.CategoryID, &old.Company)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo cliente: %v", err)
	}
	oldValue := string(oldValueBytes)

	// No es necesario verificar el código del cliente, ya que no se puede modificar
	_, err = dataBase.Update(true, "UPDATE clients SET name = ?, address = ?, phone = ?, email = ?, web = ?, city = ?, category_id = ?, company = ? WHERE id = ?", c.Name, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company, clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newValueBytes, err := json.Marshal(c)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar nuevo cliente: %v", err)
	}
	newValue := string(newValueBytes)

	// Registro del evento de actualización
	if err := insertLog("update_client", oldValue, newValue, r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de actualización de cliente: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(fmt.Sprintf("Cliente con ID %s actualizado correctamente", clientID))
}

func deleteClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	_, err := dataBase.Delete(true, "DELETE FROM clients WHERE id = ?", clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var old models.Client

	rows, err := dataBase.SelectRow("SELECT id, code, name, address, phone, email, web, city, category_id, company FROM clients WHERE id = ?", clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Cliente no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows.Scan(&old.ID, &old.Code, &old.Name, &old.Address, &old.Phone, &old.Email, &old.Web, &old.City, &old.CategoryID, &old.Company)

	oldValueBytes, err := json.Marshal(old)
	if err != nil {
		// Manejar error de serialización
		log.Printf("Error al serializar antiguo cliente: %v", err)
	}
	oldValue := string(oldValueBytes)

	// Registro del evento de eliminación
	if err := insertLog("delete_client", oldValue, "", r); err != nil {
		// Manejar el error de inserción del log aquí
		log.Printf("Error al insertar el registro de eliminación de cliente: %v", err)
	}
	w.WriteHeader(http.StatusNoContent) // 204 No Content se suele utilizar para respuestas exitosas sin cuerpo
}
