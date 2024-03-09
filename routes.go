package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func initRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(SecurityHeaders)
	r.Use(middleware.Logger)
	r.Use(CORSMiddleware) // Agrega el middleware de CORS aquí
	// Aplica el middleware de tasa de límite al directorio root
	r.Group(func(r chi.Router) {
		r.Use(RateLimit) // Aplica la tasa de límite a todas las rutas dentro de este grupo
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			// Tu código para manejar la raíz, como devolver un estado de API o una página principal
			// devolver un acceso restrigido:
			http.Error(w, "Acceso restringido", http.StatusUnauthorized)

			//w.Write([]byte("Bienvenido a la API"))
		})
	})

	r.Get("/version", getVersion) // GET /version - Devuelve la versión de la API

	// Aplica el middleware de tasa de límite solo al endpoint de login
	r.Group(func(r chi.Router) {
		r.Use(RateLimit) // Este middleware se aplicará solo a las rutas dentro de este grupo
		r.Post("/login", loginUser)
	})

	r.Post("/request-recovery", requestPasswordRecovery) // POST /request-recovery - Solicitar recuperación de contraseña
	r.Post("/change-password", changePassword)           // POST /change-password - Cambio de contraseña para un usuario

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		// Definir las rutas para usuarios
		r.Route("/users", func(r chi.Router) {
			r.Get("/", getUsers)          // GET /users - Obtener todos los usuarios
			r.Get("/{id}", getUserByID)   // GET /users/{id} - Obtener un usuario por su ID
			r.Post("/", createUser)       // POST /users - Crear un nuevo usuario
			r.Put("/{id}", updateUser)    // PUT /users/{id} - Actualizar un usuario existente
			r.Delete("/{id}", deleteUser) // DELETE /users/{id} - Eliminar un usuario

			// Rutas adicionales para operaciones específicas de usuarios
		})
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
		r.Get("/logs", getLogs)

		// Rutas para "settings"
		r.Route("/settings", func(r chi.Router) {
			r.Get("/", getSettings)
			r.Post("/", createSetting)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", updateSetting)
				r.Delete("/", deleteSetting)
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

	return r
}
