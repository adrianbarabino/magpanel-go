package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"gopkg.in/ini.v1"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Phone      string `json:"phone,omitempty"` // El campo omitempty indica que el campo puede ser omitido si está vacío
	Email      string `json:"email"`
	Web        string `json:"web,omitempty"`
	City       string `json:"city"`
	CategoryID int    `json:"category_id,omitempty"`
	Company    string `json:"company,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
	UpdatedAt  string `json:"updated_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
}
type Category struct {
	ID     int            `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Fields sql.NullString `json:"fields,omitempty"` // Cambiado a sql.NullString para manejar valores NULL
}
type Location struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Lat     float64 `json:"lat,omitempty"` // Usa float64 para coordenadas
	Lng     float64 `json:"lng,omitempty"` // Usa float64 para coordenadas
	State   string  `json:"state,omitempty"`
	City    string  `json:"city"`
	Country string  `json:"country"`
}
type ProjectStatus struct {
	ID         int    `json:"id"`
	StatusName string `json:"status_name"`
	Order      int    `json:"order"`
}

type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CategoryID  int    `json:"category_id,omitempty"`
	StatusID    int    `json:"status_id"`
	LocationID  int    `json:"location_id,omitempty"`
	AuthorID    int    `json:"author_id"`
	CreatedAt   string `json:"created_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
	UpdatedAt   string `json:"updated_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
}

var db *sql.DB

func initDB() {
	var err error

	// Cargar el archivo de configuración
	cfg, err := ini.Load("data.conf")
	if err != nil {
		log.Fatal("Error al cargar el archivo de configuración: ", err)
	}

	// Leer las propiedades de la sección "database"
	dbSection := cfg.Section("database")
	username := dbSection.Key("DB_USER").String()
	password := dbSection.Key("DB_PASS").String()
	host := dbSection.Key("DB_HOST").String()
	database := dbSection.Key("DB_NAME").String()

	// Construir la cadena de conexión
	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, host, database)

	db, err = sql.Open("mysql", connectionString)

	// Cambia los detalles de conexión según tu configuración de MySQL
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	var port string
	flag.StringVar(&port, "port", "3001", "Define el puerto en el que el servidor debería escuchar")
	flag.Parse()
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(CORSMiddleware) // Agrega el middleware de CORS aquí

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		// Rutas para "clients"
		r.Route("/clients", func(r chi.Router) {
			r.Get("/", getClients)
			r.Post("/", createClient)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getClientByID)
				r.Put("/", updateClient)
				r.Delete("/", deleteClient)
			})
		})

		// Rutas para "projects"
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", getProjects)
			r.Post("/", createProject)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getProjectByID)
				r.Put("/", updateProject)
				r.Delete("/", deleteProject)
			})
		})

		// Rutas para "project-statuses"
		r.Route("/project-statuses", func(r chi.Router) {
			r.Get("/", getProjectStatuses)
			r.Post("/", createProjectStatus)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getProjectStatusByID)
				r.Put("/", updateProjectStatus)
				r.Delete("/", deleteProjectStatus)
			})
		})

		// Rutas para "locations"
		r.Route("/locations", func(r chi.Router) {
			r.Get("/", getLocations)
			r.Post("/", createLocation)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getLocationByID)
				r.Put("/", updateLocation)
				r.Delete("/", deleteLocation)
			})
		})

		r.Route("/categories", func(r chi.Router) {
			r.Get("/", getCategories)   // Obtener todas las categorías
			r.Post("/", createCategory) // Crear una nueva categoría
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", getCategoryByID)
				r.Put("/", updateCategory)
				r.Delete("/", deleteCategory)
			})
		})
	})

	// Inicia el servidor en el puerto especificado
	log.Printf("Servidor corriendo en el puerto %s\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulación simple de autenticación. En producción, implementa una verificación adecuada.
		token := r.Header.Get("Authorization")
		if token != "token-secreto" {
			http.Error(w, "No autorizado", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getLocations(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, lat, lng, state, city, country FROM locations")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		var l Location
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
	var l Location
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO locations (name, lat, lng, state, city, country) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(l.Name, l.Lat, l.Lng, l.State, l.City, l.Country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(l)
}
func updateLocation(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	var l Location
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE locations SET name = ?, lat = ?, lng = ?, state = ?, city = ?, country = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(l.Name, l.Lat, l.Lng, l.State, l.City, l.Country, locationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Ubicación con ID %s actualizada correctamente", locationID))
}
func deleteLocation(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM locations WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(locationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}
func getLocationByID(w http.ResponseWriter, r *http.Request) {
	locationID := chi.URLParam(r, "id")

	var l Location
	err := db.QueryRow("SELECT id, name, lat, lng, state, city, country FROM locations WHERE id = ?", locationID).Scan(&l.ID, &l.Name, &l.Lat, &l.Lng, &l.State, &l.City, &l.Country)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Ubicación no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l)
}

func getCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, type, name, fields FROM categories")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
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
	var c Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO categories (type, name, fields) VALUES (?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.Type, c.Name, c.Fields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}
func updateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id")

	var c Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE categories SET type = ?, name = ?, fields = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.Type, c.Name, c.Fields, categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Categoría con ID %s actualizada correctamente", categoryID))
}

func deleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM categories WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(categoryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}

func getClients(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, address, phone, email, web, city, category_id, company FROM clients")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clients = append(clients, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}
func getCategoryByID(w http.ResponseWriter, r *http.Request) {
	categoryID := chi.URLParam(r, "id") // Obtiene el ID de la categoría de la URL

	var c Category
	err := db.QueryRow("SELECT id, type, name, fields FROM categories WHERE id = ?", categoryID).Scan(&c.ID, &c.Type, &c.Name, &c.Fields)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Categoría no encontrada", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func createClient(w http.ResponseWriter, r *http.Request) {
	var c Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Preparar la consulta SQL.
	stmt, err := db.Prepare("INSERT INTO clients(name, address, phone, email, web, city, category_id, company) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Ejecutar la consulta con los datos del cliente.
	result, err := stmt.Exec(c.Name, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener el ID del cliente recién creado.
	lastInsertID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// convert lastInsertID to int

	c.ID = int(lastInsertID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func getClientByID(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var c Client
	err := db.QueryRow("SELECT id, name, address, phone, email, web, city, category_id, company FROM clients WHERE id = ?", clientID).Scan(&c.ID, &c.Name, &c.Address, &c.Phone, &c.Email, &c.Web, &c.City, &c.CategoryID, &c.Company)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Cliente no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func updateClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var c Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE clients SET name = ?, address = ?, phone = ?, email = ?, web = ?, city = ?, category_id = ?, company = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.Name, c.Address, c.Phone, c.Email, c.Web, c.City, c.CategoryID, c.Company, clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(fmt.Sprintf("Cliente con ID %s actualizado correctamente", clientID))
}

func deleteClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM clients WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content se suele utilizar para respuestas exitosas sin cuerpo
}

func getProjectStatuses(w http.ResponseWriter, r *http.Request) {
	var statuses []ProjectStatus

	rows, err := db.Query("SELECT id, status_name, `order` FROM project_statuses")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s ProjectStatus
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
	var s ProjectStatus
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO project_statuses (status_name, `order`) VALUES (?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(s.StatusName, s.Order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.ID = int(lastInsertID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func getProjectStatusByID(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")

	var s ProjectStatus
	err := db.QueryRow("SELECT id, status_name, `order` FROM project_statuses WHERE id = ?", statusID).Scan(&s.ID, &s.StatusName, &s.Order)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Estado de proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

func updateProjectStatus(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")
	var s ProjectStatus
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE project_statuses SET status_name = ?, `order` = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.StatusName, s.Order, statusID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Estado de proyecto con ID %s actualizado correctamente", statusID))
}

func deleteProjectStatus(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM project_statuses WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(statusID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getProjects(w http.ResponseWriter, r *http.Request) {
	var projects []Project

	rows, err := db.Query("SELECT id, name, description, category_id, status_id, location_id, author_id, created_at, updated_at FROM projects")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.StatusID, &p.LocationID, &p.AuthorID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		projects = append(projects, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func createProject(w http.ResponseWriter, r *http.Request) {
	var p Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO projects (name, description, category_id, status_id, location_id, author_id) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.ID = int(lastInsertID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func getProjectByID(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var p Project
	err := db.QueryRow("SELECT id, name, description, category_id, status_id, location_id, author_id, created_at, updated_at FROM projects WHERE id = ?", projectID).Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.StatusID, &p.LocationID, &p.AuthorID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Proyecto no encontrado", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func updateProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	var p Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE projects SET name = ?, description = ?, category_id = ?, status_id = ?, location_id = ?, author_id = ? WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(p.Name, p.Description, p.CategoryID, p.StatusID, p.LocationID, p.AuthorID, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fmt.Sprintf("Proyecto con ID %s actualizado correctamente", projectID))
}

func deleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	stmt, err := db.Prepare("DELETE FROM projects WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content como respuesta exitosa sin cuerpo
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Ajusta esto según tus necesidades
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return // Para preflight request
		}

		next.ServeHTTP(w, r)
	})
}
