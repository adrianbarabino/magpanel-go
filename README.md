# MagPanel-Go

MagPanel-Go es el backend de la aplicación de gestión MagPanel, construido con Go y el framework de enrutamiento Chi. Proporciona una API RESTful para manejar clientes, proyectos, estados de proyectos, ubicaciones y categorías.

## Endpoints

La API está alojada en `api.mag-servicios.com` y expone los siguientes endpoints:

### Clients

- `GET /clients`: Obtiene todos los clientes.
- `POST /clients`: Crea un nuevo cliente.
- `GET /clients/{id}`: Obtiene un cliente por ID.
- `PUT /clients/{id}`: Actualiza un cliente por ID.
- `DELETE /clients/{id}`: Elimina un cliente por ID.

### Projects

- `GET /projects`: Obtiene todos los proyectos.
- `POST /projects`: Crea un nuevo proyecto.
- `GET /projects/{id}`: Obtiene un proyecto por ID.
- `PUT /projects/{id}`: Actualiza un proyecto por ID.
- `DELETE /projects/{id}`: Elimina un proyecto por ID.

### Project Statuses

- `GET /project-statuses`: Obtiene todos los estados de los proyectos.
- `POST /project-statuses`: Crea un nuevo estado de proyecto.
- `GET /project-statuses/{id}`: Obtiene un estado de proyecto por ID.
- `PUT /project-statuses/{id}`: Actualiza un estado de proyecto por ID.
- `DELETE /project-statuses/{id}`: Elimina un estado de proyecto por ID.

### Locations

- `GET /locations`: Obtiene todas las ubicaciones.
- `POST /locations`: Crea una nueva ubicación.
- `GET /locations/{id}`: Obtiene una ubicación por ID.
- `PUT /locations/{id}`: Actualiza una ubicación por ID.
- `DELETE /locations/{id}`: Elimina una ubicación por ID.

### Categories

- `GET /categories`: Obtiene todas las categorías.
- `POST /categories`: Crea una nueva categoría.
- `GET /categories/{id}`: Obtiene una categoría por ID.
- `PUT /categories/{id}`: Actualiza una categoría por ID.
- `DELETE /categories/{id}`: Elimina una categoría por ID.
