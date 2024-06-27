package mailer

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

//go:embed "templates"
var templateFs embed.FS

type Mailer struct {
	dialer  *mail.Dialer
	sender  string
	retries int
}

func New(host string, port int, username, password, sender string, retries int) (Mailer, error) {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	if retries < 1 {
		return Mailer{}, errors.New("mailer: retries must be >= 1")
	}

	return Mailer{
		dialer:  dialer,
		sender:  sender,
		retries: retries,
	}, nil
}

func (m Mailer) Send(recipient string, templateFile string, data any) error {
	templ, err := template.New("email").ParseFS(templateFs, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := bytes.NewBuffer([]byte{})
	err = templ.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := bytes.NewBuffer([]byte{})
	err = templ.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := bytes.NewBuffer([]byte{})
	err = templ.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	for i := 0; i < m.retries; i++ {
		err = m.dialer.DialAndSend(msg)

		if nil == err {
			return nil
		}

		time.Sleep(time.Millisecond * 500)
	}

	return err
}
