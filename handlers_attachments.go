package main

import (
	"fmt"
	"net/http"

	"github.com/minio/minio-go/v7"
)

func HandleRemove(minioClient *minio.Client, bucketName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verificar si se ha enviado un archivo
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		// Verificar si el cliente MinIO es nulo
		if minioClient == nil {
			http.Error(w, "Cliente MinIO nulo", http.StatusInternalServerError)
			return
		}
		// filePath es el archivo, recuerda que lo recibimos en json como file

		// filePath := r.FormValue("file
		filePath := r.FormValue("fileId")
		// remove the https://...../ to the first slash with split
		// Eliminar el archivo del bucket de DigitalOcean Spaces
		err := minioClient.RemoveObject(r.Context(), bucketName, filePath, minio.RemoveObjectOptions{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("Archivo eliminado: " + filePath + " del bucket: " + bucketName))
		// Respuesta exitosa
		w.WriteHeader(http.StatusOK)
	}
}

func HandleUpload(minioClient *minio.Client, bucketName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verificar si se ha enviado un archivo
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		// Parsear el archivo del formulario
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Verificar si el cliente MinIO es nulo
		if minioClient == nil {
			http.Error(w, "Cliente MinIO nulo", http.StatusInternalServerError)
			return
		}
		destinationFolder := "attachments/" + r.FormValue("folder")

		options := minio.PutObjectOptions{}

		options = minio.PutObjectOptions{
			UserMetadata: map[string]string{
				"x-amz-acl": "public-read", // Establece el archivo como público
			},
		}

		// Subir el archivo al bucket de DigitalOcean Spaces
		_, err = minioClient.PutObject(r.Context(), bucketName, destinationFolder+"/"+header.Filename, file, -1, options)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Respuesta exitosa
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "https://magservicios.sfo3.cdn.digitaloceanspaces.com/%s/%s", destinationFolder, header.Filename)
	}
}
