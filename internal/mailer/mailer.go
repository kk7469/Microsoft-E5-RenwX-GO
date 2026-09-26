package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"e5renewx/internal/model"
)

func Send(settings model.Settings, to, subject, body string) error {
	if to == "" || settings.SMTPHost == "" || settings.SMTPUser == "" {
		return fmt.Errorf("smtp not configured")
	}
	from := settings.SMTPFrom
	if from == "" {
		from = settings.SMTPUser
	}
	port := settings.SMTPPort
	if port == 0 {
		port = 465
	}
	addr := fmt.Sprintf("%s:%d", settings.SMTPHost, port)
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPassword, settings.SMTPHost)
	if port == 465 {
		return sendTLS(addr, settings.SMTPHost, auth, from, []string{to}, []byte(msg))
	}
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func sendTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Quit()
	if err := c.Auth(auth); err != nil {
		return err
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}
