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

func createContact(w http.ResponseWriter, r *http.Request) {
	var contactData models.ContactWithClientsAndProviders // Asume esta estructura tiene los ClientIDs

	if err := json.NewDecoder(r.Body).Decode(&contactData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO contacts(name, position, phone, email) VALUES(?, ?, ?, ?)", contactData.Name, contactData.Position, contactData.Phone, contactData.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	contactData.ID = int(lastInsertID)

	// Insertar relaciones en la tabla intermedia client_contact
	for _, clientID := range contactData.ClientIDs {
		_, err := dataBase.Insert(false, "INSERT INTO client_contact(client_id, contact_id) VALUES(?, ?)", clientID, contactData.ID)
		if err != nil {
			// Considerar rollback o manejo de errores adecuado aquí
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// Insertar relaciones en la tabla intermedia client_contact
	for _, providerID := range contactData.ProviderIDs {
		_, err := dataBase.Insert(false, "INSERT INTO provider_contact(provider_id, contact_id) VALUES(?, ?)", providerID, contactData.ID)
		if err != nil {
			// Considerar rollback o manejo de errores adecuado aquí
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(contactData)
}

func updateContact(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")
	var contactData models.ContactWithClientsAndProviders // Asume esta estructura tiene los ClientIDs

	if err := json.NewDecoder(r.Body).Decode(&contactData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := dataBase.Update(true, "UPDATE contacts SET name = ?, position = ?, phone = ?, email = ? WHERE id = ?", contactData.Name, contactData.Position, contactData.Phone, contactData.Email, contactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Primero, eliminar todas las asociaciones existentes para este contacto
	_, err = dataBase.Delete(false, "DELETE FROM client_contact WHERE contact_id = ?", contactID)
	if err != nil {
		// Considerar rollback o manejo de errores adecuado aquí
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = dataBase.Delete(false, "DELETE FROM provider_contact WHERE contact_id = ?", contactID)
	if err != nil {
		// Considerar rollback o manejo de errores adecuado aquí
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Luego, insertar nuevas relaciones en la tabla intermedia client_contact
	for _, clientID := range contactData.ClientIDs {
		_, err := dataBase.Insert(false, "INSERT INTO client_contact(client_id, contact_id) VALUES(?, ?)", clientID, contactID)
		if err != nil {
			// Considerar rollback o manejo de errores adecuado aquí
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Luego, insertar nuevas relaciones en la tabla intermedia provider_contact
	for _, providerID := range contactData.ProviderIDs {
		_, err := dataBase.Insert(false, "INSERT INTO provider_contact(provider_id, contact_id) VALUES(?, ?)", providerID, contactID)
		if err != nil {
			// Considerar rollback o manejo de errores adecuado aquí
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(fmt.Sprintf("Contacto con ID %s actualizado correctamente", contactID))
}

func getContacts(w http.ResponseWriter, r *http.Request) {
	var contacts []models.Contact
	rowsC, err := dataBase.Select("SELECT id, name, position, phone, email FROM contacts")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rowsC.Close()

	for rowsC.Next() {
		var c models.Contact
		var con models.Connections
		if err := rowsC.Scan(&c.ID, &c.Name, &c.Position, &c.Phone, &c.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Obtener los ClientIDs asociados a este contacto
		contactID := c.ID

		var clientIDs []int
		rows, err := dataBase.Select("SELECT client_id FROM client_contact WHERE contact_id = ?", contactID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var clientID int
			if err := rows.Scan(&clientID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			clientIDs = append(clientIDs, clientID)
		}
		//clientIDs = append(clientIDs, 1)

		con.ClientIDs = clientIDs

		var providerIDs []int
		rows, err = dataBase.Select("SELECT provider_id FROM provider_contact WHERE contact_id = ?", contactID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var providerID int
			if err := rows.Scan(&providerID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			providerIDs = append(providerIDs, providerID)
		}
		con.ProviderIDs = providerIDs
		c.Connections = con
		contacts = append(contacts, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}

func getContactByID(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")

	var contact models.Contact
	rowsC, err := dataBase.SelectRow("SELECT id, name, position, phone, email, created_at, updated_at FROM contacts WHERE id = ?", contactID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Contacto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rowsC.Scan(&contact.ID, &contact.Name, &contact.Position, &contact.Phone, &contact.Email, &contact.CreatedAt, &contact.UpdatedAt)

	// parse to time and add 3 hours
	parsedCreatedAt, err := time.Parse("2006-01-02 15:04:05", contact.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parsedUpdatedAt, err := time.Parse("2006-01-02 15:04:05", contact.UpdatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// convert to string and save
	parsedCreatedAt = parsedCreatedAt.Add(-3 * time.Hour)
	parsedUpdatedAt = parsedUpdatedAt.Add(-3 * time.Hour)

	// convert parsedCreatedAt (time) to String

	contact.CreatedAt = parsedCreatedAt.Format("2006-01-02 15:04:05")
	contact.UpdatedAt = parsedUpdatedAt.Format("2006-01-02 15:04:05")

	// Obtener los ClientIDs asociados a este contacto
	var clientIDs []int
	rows, err := dataBase.Select("SELECT client_id FROM client_contact WHERE contact_id = ?", contactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var clientID int
		if err := rows.Scan(&clientID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clientIDs = append(clientIDs, clientID)
	}

	// Providers

	// Obtener los ClientIDs asociados a este contacto
	var providerIDs []int
	rows, err = dataBase.Select("SELECT provider_id FROM provider_contact WHERE contact_id = ?", contactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var providerID int
		if err := rows.Scan(&providerID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		providerIDs = append(providerIDs, providerID)
	}

	// Envío del contacto y sus ClientIDs como respuesta
	response := struct {
		models.Contact
		ClientIDs   []int `json:"client_ids"`
		ProviderIDs []int `json:"provider_ids"`
	}{
		Contact:     contact,
		ClientIDs:   clientIDs,
		ProviderIDs: providerIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteContact(w http.ResponseWriter, r *http.Request) {
	contactID := chi.URLParam(r, "id")

	_, err := dataBase.Delete(true, "DELETE FROM contacts WHERE id = ?", contactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var old models.Contact

	rowsC, err := dataBase.SelectRow("SELECT id, name, position, phone, email FROM contacts WHERE id = ?", contactID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Contacto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rowsC.Scan(&old.ID, &old.Name, &old.Position, &old.Phone, &old.Email)

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
