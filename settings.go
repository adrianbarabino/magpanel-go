package main

import (
	"fmt"
	"net/http"
	"time"
)

func getMailgunConfig() (string, string, error) {
	var domain, apiKey string

	// Obtener el dominio de Mailgun
	rows, err := dataBase.SelectRow("SELECT `value` FROM settings WHERE `key` = 'mailgun_domain'")
	if err != nil {
		return "", "", err
	}
	rows.Scan(&domain)

	// Obtener la API key de Mailgun
	rows, err = dataBase.SelectRow("SELECT `value` FROM settings WHERE `key` = 'mailgun_api_key'")
	if err != nil {
		return "", "", err
	}
	rows.Scan(&apiKey)

	return domain, apiKey, nil
}

func getVersion(w http.ResponseWriter, r *http.Request) {
	since := time.Since(uptime)
	returnString := "API v1.1.7 - Uptime: " + since.String() + " - Total requests: " + fmt.Sprintf("%d", totalRequests)
	w.Write([]byte(returnString))
}
