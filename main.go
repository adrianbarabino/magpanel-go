package main

import (
	"flag"
	"fmt"
	"log"
	"magpanel/database"
	"net/http"
	"time"

	"gopkg.in/ini.v1"

	_ "github.com/go-sql-driver/mysql"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var totalRequests int
var uptime time.Time
var dataBase *database.DatabaseStruct
var minioClient *minio.Client
var jwtKey []byte
var bucketName string

func main() {
	initConfig()
	defer dataBase.Close()
	var port string
	flag.StringVar(&port, "port", "3001", "Define el puerto en el que el servidor debería escuchar")
	flag.Parse()

	r := initRoutes()
	uptime = time.Now()

	// Inicia el servidor en el puerto especificado
	log.Printf("Servidor corriendo en el puerto %s\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}

func initConfig() {
	var err error

	// Cargar el archivo de configuración
	cfg, err := ini.Load("data.conf")
	if err != nil {
		log.Fatal("Error al cargar el archivo de configuración: ", err)
	}

	// Leer las propiedades de la sección "database"
	dataSection := cfg.Section("keys")
	jwtKey = []byte(dataSection.Key("JWT_KEY").String())
	dbSection := cfg.Section("database")
	username := dbSection.Key("DB_USER").String()
	password := dbSection.Key("DB_PASS").String()
	host := dbSection.Key("DB_HOST").String()
	databaseName := dbSection.Key("DB_NAME").String()
	endpoint := dataSection.Key("ENDPOINT").String()
	accessKeyID := dataSection.Key("ACCESS_KEY_ID").String()
	secretAccessKey := dataSection.Key("SECRET_ACCESS_KEY").String()
	bucketName = dataSection.Key("BUCKET_NAME").String()

	// Inicializa un cliente de DigitalOcean Spaces
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}
	minioClient.SetAppInfo("magpanel", "1.0.0")
	if err != nil {
		log.Fatalln(err)
	}

	// Inicializar la base de datos
	dataBase, err = database.NewDatabase(username, password, databaseName, host)
	if err != nil {
		log.Fatal(err)
	}
}
