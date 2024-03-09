package models

import (
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt"
)

type User struct {
	ID           int            `json:"id"`
	Username     string         `json:"username"`
	Rank         int            `json:"rank"`
	Email        string         `json:"email"`
	Name         string         `json:"name,omitempty"`          // `omitempty` para que los valores nulos no aparezcan en el JSON
	PasswordHash string         `json:"password_hash,omitempty"` // No se incluirá en las respuestas JSON
	RecoveryHash sql.NullString `json:"recovery_hash,omitempty"` // Maneja NULL
	CreatedAt    time.Time      `json:"created_at,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at,omitempty"`
}

type Client struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Code       string `json:"code,omitempty"` // El campo omitempty indica que el campo puede ser omitido si está vacío
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

type Contact struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Position  string `json:"position,omitempty"` // Omite si está vacío
	Phone     string `json:"phone,omitempty"`    // Omite si está vacío
	Email     string `json:"email"`
	ClientIDs []int  `json:"client_ids,omitempty"` // Omite si está vacío
	CreatedAt string `json:"created_at,omitempty"` // Omite si está vacío, manejado por la DB
	UpdatedAt string `json:"updated_at,omitempty"` // Omite si está vacío, manejado por la DB
}

// ContactWithClients es una estructura extendida de Contact para incluir los ClientIDs asociados
type ContactWithClients struct {
	Contact         // Incorporación anónima de la estructura Contact
	ClientIDs []int `json:"client_ids"` // Slice de IDs de clientes
}

// ClientContact representa la relación muchos a muchos entre Contactos y Clientes
type ClientContact struct {
	ClientID  int `json:"client_id"`
	ContactID int `json:"contact_id"`
}

type Provider struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Code       string `json:"code,omitempty"` // El campo omitempty indica que el campo puede ser omitido si está vacío
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

// ContactWithProviders es una estructura extendida de Contact para incluir los ProviderIDs asociados
type ContactWithProviders struct {
	Contact           // Incorporación anónima de la estructura Contact
	ProviderIDs []int `json:"provider_ids"` // Slice de IDs de provideres
}

// ProviderContact representa la relación muchos a muchos entre Contactos y Provideres
type ProviderContact struct {
	ProviderID int `json:"provider_id"`
	ContactID  int `json:"contact_id"`
}

type Log struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	OldValue  string `json:"old_value"`
	NewValue  string `json:"new_value"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
type Category struct {
	ID     int            `json:"id"`
	Type   string         `json:"type"`
	Code   string         `json:"code,omitempty"` // El campo omitempty indica que el campo puede ser omitido si está vacío
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
	ID           int    `json:"id"`
	StatusName   string `json:"status_name"`
	CategoryID   int    `json:"category_id,omitempty"`
	CategoryName string `json:"category_name,omitempty"`
	Order        int    `json:"order"`
}

type Project struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code,omitempty"`
	Description  string `json:"description,omitempty"`
	CategoryID   int    `json:"category_id,omitempty"`
	StatusID     int    `json:"status_id"`
	LocationID   int    `json:"location_id,omitempty"`
	AuthorID     int    `json:"author_id"`
	ClientID     int    `json:"client_id"`
	ClientName   string `json:"client_name,omitempty"`
	CategoryName string `json:"category_name,omitempty"`
	StatusName   string `json:"status_name,omitempty"`
	LocationName string `json:"location_name,omitempty"`
	AuthorName   string `json:"author_name,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
	UpdatedAt    string `json:"updated_at,omitempty"` // Asume que este campo es manejado automáticamente por la base de datos
}
type Setting struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type SettingVal struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// Estructura para almacenar las reclamaciones (claims) del token JWT
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}
