package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"magpanel/models"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func getReports(w http.ResponseWriter, r *http.Request) {
	var reports []models.Report

	query := "SELECT r.id, r.project_id, r.category_id, r.fields, r.author_id, r.created_at, r.updated_at, c.name, u.name FROM reports r JOIN categories c ON r.category_id = c.id JOIN users u ON r.author_id = u.id "
	if order := r.URL.Query().Get("order"); order != "" {
		// order is like "created_at,desc", we need to check if it has a comma
		if len(order) > 0 {
			orderParts := strings.Split(order, ",")
			if len(orderParts) == 2 {
				// and we need to sanitize the input
				if orderParts[1] == "asc" || orderParts[1] == "desc" {
					query += "ORDER BY " + orderParts[0] + " " + orderParts[1] + " "
				} else {
					http.Error(w, "Invalid order direction", http.StatusBadRequest)
					return
				}
			}
		}

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
		var report models.Report
		if err := rows.Scan(&report.ID, &report.ProjectID, &report.CategoryID, &report.Fields, &report.AuthorID, &report.CreatedAt, &report.UpdatedAt, &report.CategoryName, &report.AuthorName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reports = append(reports, report)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func getReportsByProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var reports []models.Report
	query := `
	SELECT r.id, r.project_id, r.category_id, r.fields, r.author_id, r.created_at, r.updated_at, 
		   c.name, u.name 
	FROM reports r 
	JOIN categories c ON r.category_id = c.id 
	JOIN users u ON r.author_id = u.id 
	WHERE r.project_id = ? 
	`
	if order := r.URL.Query().Get("order"); order != "" {
		// order is like "created_at,desc", we need to check if it has a comma

		if len(order) > 0 {
			orderParts := strings.Split(order, ",")
			if len(orderParts) == 2 {
				// and we need to sanitize the input
				if orderParts[1] == "asc" || orderParts[1] == "desc" {
					query += "ORDER BY " + orderParts[0] + " " + orderParts[1] + " "
				} else {
					http.Error(w, "Invalid order direction", http.StatusBadRequest)
					return
				}
			}
		}

	}

	rows, err := dataBase.Select(query, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var report models.Report
		if err := rows.Scan(&report.ID, &report.ProjectID, &report.CategoryID, &report.Fields, &report.AuthorID, &report.CreatedAt, &report.UpdatedAt, &report.CategoryName, &report.AuthorName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reports = append(reports, report)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func createReport(w http.ResponseWriter, r *http.Request) {
	var report models.Report
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentUser, err := getCurrentUser(r)

	if err != nil {
		http.Error(w, "Error al obtener el usuario actual", http.StatusInternalServerError)
		return
	}

	report.AuthorID = currentUser.ID

	lastInsertID, err := dataBase.Insert(true, "INSERT INTO reports (project_id, category_id, fields, author_id) VALUES (?, ?, ?, ?)", report.ProjectID, report.CategoryID, report.Fields, report.AuthorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	report.ID = int(lastInsertID)

	if err := insertLog("create_report", "", string(report.Fields), r); err != nil {
		log.Printf("Error al insertar el registro de creación de reporte: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(report)
}
func getReportByID(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "id")

	var report models.Report

	query := `
        SELECT r.id, r.project_id, r.category_id, r.fields, r.author_id, r.created_at, r.updated_at,
		p.name AS project_name, p.code AS project_code, c.name AS category_name, u.name AS author_name
        FROM reports r
        LEFT JOIN projects p ON r.project_id = p.id
        LEFT JOIN categories c ON r.category_id = c.id
        LEFT JOIN users u ON r.author_id = u.id
        WHERE r.id = ?`

	row, err := dataBase.SelectRow(query, reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := row.Scan(&report.ID, &report.ProjectID, &report.CategoryID, &report.Fields, &report.AuthorID, &report.CreatedAt, &report.UpdatedAt, &report.ProjectName, &report.ProjectCode, &report.CategoryName, &report.AuthorName); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Reporte no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Opcional: Registro de la acción de lectura del reporte
	// if err := insertLog("read_report", fmt.Sprintf("Report ID %s accessed", reportID), "", r); err != nil {
	// 	log.Printf("Error al insertar el registro de lectura de reporte: %v", err)
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func updateReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "id")
	var report models.Report
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	oldReport, err := getReportByIDInternal(reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	oldValueBytes, err := json.Marshal(oldReport)
	if err != nil {
		log.Printf("Error al serializar reporte antiguo: %v", err)
	}

	_, err = dataBase.Update(true, "UPDATE reports SET project_id = ?, category_id = ?, fields = ?, author_id = ? WHERE id = ?", report.ProjectID, report.CategoryID, report.Fields, report.AuthorID, reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := insertLog("update_report", string(oldValueBytes), string(report.Fields), r); err != nil {
		log.Printf("Error al insertar el registro de actualización de reporte: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Reporte con ID %s actualizado correctamente", reportID))
}

func getReportByIDInternal(reportID string) (*models.Report, error) {
	var report models.Report
	row, err := dataBase.SelectRow("SELECT id, project_id, category_id, fields, author_id, created_at, updated_at FROM reports WHERE id = ?", reportID)
	if err != nil {
		return nil, err
	}

	if err := row.Scan(&report.ID, &report.ProjectID, &report.CategoryID, &report.Fields, &report.AuthorID, &report.CreatedAt, &report.UpdatedAt); err != nil {
		return nil, err
	}
	return &report, nil
}

func deleteReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "id")

	oldReport, err := getReportByIDInternal(reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	oldValueBytes, err := json.Marshal(oldReport)
	if err != nil {
		log.Printf("Error al serializar reporte antiguo: %v", err)
	}

	_, err = dataBase.Delete(true, "DELETE FROM reports WHERE id = ?", reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := insertLog("delete_report", string(oldValueBytes), "", r); err != nil {
		log.Printf("Error al insertar el registro de eliminación de reporte: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
