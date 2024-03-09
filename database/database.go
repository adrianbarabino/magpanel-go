package database

import (
	"database/sql"
	"fmt"
	"log"
)

type DatabaseStruct struct {
	connection *sql.DB
}

func NewDatabase(dbUser, dbPass, dbName, dbHost string) (*DatabaseStruct, error) {

	// Construir la cadena de conexión
	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, dbHost, dbName)

	db, err := sql.Open("mysql", connectionString)

	// Cambia los detalles de conexión según tu configuración de MySQL
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()

	return &DatabaseStruct{connection: db}, err
}

func (db *DatabaseStruct) Close() {
	db.connection.Close()
}

func (db *DatabaseStruct) Insert(prepare bool, query string, args ...interface{}) (int64, error) {
	if prepare {
		stmt, err := db.connection.Prepare(query)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()
	}
	result, err := db.connection.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (db *DatabaseStruct) Update(prepare bool, query string, args ...interface{}) (int64, error) {
	if prepare {
		stmt, err := db.connection.Prepare(query)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()
	}
	result, err := db.connection.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}

func (db *DatabaseStruct) Delete(prepare bool, query string, args ...interface{}) (int64, error) {
	if prepare {
		stmt, err := db.connection.Prepare(query)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()
	}
	result, err := db.connection.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}

// el retorno rows requiere un defer rows.Close()
func (db *DatabaseStruct) Select(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.connection.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (db *DatabaseStruct) SelectRow(query string, args ...interface{}) (*sql.Row, error) {
	row := db.connection.QueryRow(query, args...)
	return row, nil
}
