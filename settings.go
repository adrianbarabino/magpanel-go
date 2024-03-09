package main

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
