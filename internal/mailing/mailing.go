package mailing

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"

	"github.com/vladwithcode/juzgados/internal/db"
	"gopkg.in/gomail.v2"
)

const SEND_AS = "no-reply@certx-mx.org"

func SendVerificationMail(recipient string, otl *db.OTLink) error {
	pw := os.Getenv("GOOGLE_MAIL_APP_PASS")
	emailAddress := os.Getenv("GOOGLE_MAIL_ADDRESS")
	siteHostname := os.Getenv("TSJ_SITE_HOSTNAME")

	if pw == "" || emailAddress == "" {
		fmt.Printf("[Mailing] Env is not set-up correctly. pw:%v email:%v", pw, emailAddress)
		return errors.New("Env Missing")
	}

	msg := gomail.NewMessage()
	msg.SetAddressHeader("From", SEND_AS, "TSJ Search")
	msg.SetHeader("To", recipient)
	msg.SetHeader("Subject", "Confirma tu registro")
	//msg.SetBody("text/html", "<a href=\"http://localhost:8080\">Verifica tu correo aquí</a>")

	templ, err := template.ParseFiles("web/templates/emails/layout.html", "web/templates/emails/confirm-signup.html")

	if err != nil {
		return nil
	}

	var b bytes.Buffer
	href := fmt.Sprintf(
		"http://%v/api/users/verification?code=%v&userId=%v",
		siteHostname,
		otl.Code.String(),
		otl.UserId.String(),
	)

	err = templ.Execute(&b, map[string]string{
		"VerificationLink": href,
	})

	if err != nil {
		return err
	}

	msg.SetBody("text/html", b.String())

	d := gomail.NewDialer("smtp.gmail.com", 587, emailAddress, pw)
	if err := d.DialAndSend(msg); err != nil {
		fmt.Printf("Send err: %v\n", err)
		return err
	}

	return nil
}

func Test() error {
	var pw = os.Getenv("GOOGLE_MAIL_APP_PASS")
	var email_address = os.Getenv("GOOGLE_MAIL_ADDRESS")

	if pw == "" || email_address == "" {
		fmt.Printf("[Mailing] Env is not set-up correctly. pw:%v email:%v", pw, email_address)
		return errors.New("Env Missing")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "no-reply@certx-mx.org")
	m.SetHeader("To", "vladwithcode@gmail.com")
	m.SetHeader("Subject", "Hello!")
	m.SetBody("text/html", "<h1>Tsj Search</h1><p>Hello world!</p>")

	d := gomail.NewDialer("smtp.gmail.com", 587, email_address, pw)

	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("Send err: %v\n", err)
		return err
	}

	fmt.Printf("Testing mail: %t\n", true)
	return nil
}
