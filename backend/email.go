package main

import (
	"bytes"
	"fmt"
	"html/template"

	gomail "gopkg.in/mail.v2"
)

type Template struct {
	path    string
	subject string
}

var Welcome Template = Template{
	path:    "email_templates/welcome.html",
	subject: "Welcome to The O'Town Garage!",
}

var resetMail Template = Template{
	path:    "email_templates/reset.html",
	subject: "Reset your password",
}

func sendMail(to string, user User, email_template Template, link string) error {
	// Parse the template
	tmpl, err := template.ParseFiles(email_template.path)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return err
	}

	// Execute the template with dynamic data
	var body bytes.Buffer
	err = tmpl.Execute(&body, struct {
		Name string
		Link string
	}{Name: user.Name, Link: link})

	if err != nil {
		fmt.Println("error executing template:", err)
		return err
	}

	// Create a new message
	message := gomail.NewMessage()
	message.SetHeader("From", config.Email.User)
	message.SetHeader("To", to)
	message.SetHeader("Subject", email_template.subject)
	message.SetBody("text/html", body.String())

	dialer := gomail.NewDialer(config.Email.Host, config.Email.Port, config.Email.User, config.Email.Password)
	if err := dialer.DialAndSend(message); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Email sent successfully!")
	}
	return err
}
