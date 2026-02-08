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

var Goodbye Template = Template{
	path:    "email_templates/goodbye.html",
	subject: "See you again soon!",
}

var resetMail Template = Template{
	path:    "email_templates/reset.html",
	subject: "Reset your password",
}

var Unsubscription Template = Template{
	path:    "email_templates/unsubscription.html",
	subject: "Canceled subscription",
}

var NewMember Template = Template{
	path:    "email_templates/notify_new_member.html",
	subject: "New member!",
}

func sendMail(to string, user User, email_template Template, link string, data any) error {
	// Parse the template
	tmpl, err := template.ParseFiles(email_template.path)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return err
	}

	// Execute the template with dynamic data
	var body bytes.Buffer
	err = tmpl.Execute(&body, struct {
		Name  string
		Link  string
		Email string
		Data  any
	}{Name: user.Name, Link: link, Email: user.Email, Data: data})

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
