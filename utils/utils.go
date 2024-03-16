package utils

import (
	"fmt"

	"github.com/mailgun/mailgun-go"
)

type MailConfig struct {
	Domain string
	APIKey string
}

func SendRecoveryEmail(config MailConfig, email, token string) error {
	// Obtener la configuración de Mailgun desde la base de datos

	domain := config.Domain
	apiKey := config.APIKey

	// Configuración de Mailgun
	mg := mailgun.NewMailgun(domain, apiKey)

	// Construir el mensaje de correo electrónico en formato HTML
	sender := "no-reply@mag-servicios.com" // Considera también almacenar esto en la tabla de configuraciones
	subject := "Recuperación de contraseña"
	logoURL := "https://mag-servicios.com/wp-content/uploads/2022/12/01-4.png"
	body := fmt.Sprintf(`
	<html>
	<body>
		<div style="text-align: center;">
			<img src="%s" alt="Logo MAG Servicios" style="max-width: 200px; margin-bottom: 20px;">
			<p>Tu token de recuperación es: <strong>%s</strong></p>
			<p>Introdúcelo en la página web para poder recuperar tu contraseña:</p>
			<a href="https://gestion.mag-servicios.com/password-recovery/%s/" style="display: inline-block; background-color: #007BFF; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Recuperar Contraseña</a>
		</div>
	</body>
	</html>
	`, logoURL, token, token)

	recipient := email

	// Asegúrate de utilizar la función adecuada para enviar mensajes en formato HTML.
	// Si estás utilizando Mailgun, `NewMessage` debería ser reemplazado por `NewMessage` con el parámetro adecuado para indicar que el contenido es HTML
	message := mg.NewMessage(sender, subject, "", recipient) // El cuerpo vacío se reemplaza por el parámetro de HTML a continuación
	message.SetHtml(body)

	// Enviar el correo electrónico
	_, _, err := mg.Send(message)
	return err
}
