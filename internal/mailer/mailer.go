package mailer

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
)

type Mailer struct {
	host string
	port int
	user string
	pass string
	from string
}

func New() *Mailer {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	return &Mailer{
		host: os.Getenv("SMTP_HOST"),
		port: port,
		user: os.Getenv("SMTP_USER"),
		pass: os.Getenv("SMTP_PASS"),
		from: os.Getenv("SMTP_FROM"),
	}
}

func (m *Mailer) SendMagicLink(to, magicURL string) error {
	auth := smtp.PlainAuth("", m.user, m.pass, m.host)

	body := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: Votre lien de connexion\r\n\r\n"+
			"Cliquez sur ce lien pour vous connecter (valable 10 minutes) :\r\n\r\n%s\r\n\r\n"+
			"Si vous n'avez pas demandé ce lien, ignorez cet email.",
		m.from, to, magicURL,
	)

	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(body))
}
