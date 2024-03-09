package main

import (
	"magpanel/models"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(1, 3) // Permite 1 solicitud por segundo con un burst de 3.

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obtenemos el token de autorización del encabezado
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No autorizado. Token no proporcionado.", http.StatusUnauthorized)
			return
		}

		// Verificamos si el token es "token-secreto" y lo salteamos
		if authHeader == "token-secreto" {
			next.ServeHTTP(w, r)
			return
		}

		// El token debe estar en el formato "Bearer {token}"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(w, "No autorizado. Formato de token inválido.", http.StatusUnauthorized)
			return
		}

		// Parseamos y validamos el token
		tokenString := tokenParts[1]
		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				http.Error(w, "No autorizado. Token de autenticación inválido.", http.StatusUnauthorized)
				return
			}
			http.Error(w, "No autorizado. Token de autenticación inválido.", http.StatusUnauthorized)
			return
		}
		if !token.Valid {
			http.Error(w, "No autorizado. Token de autenticación inválido.", http.StatusUnauthorized)
			return
		}

		// Si el token es válido, pasamos al siguiente middleware o controlador
		next.ServeHTTP(w, r)
	})
}

func SecurityHeaders(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
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

func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Demasiadas solicitudes, intenta de nuevo más tarde.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
