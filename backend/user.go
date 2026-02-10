package main

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/stripe/stripe-go/v82"
)

type User struct {
	Email         string
	Name          string
	Phone         string
	Password      []byte
	PlainPassword string
	Active        bool
	CustomerID    string
}

func toUser(session sessions.Session) User {
	return User{
		Email:      session.Values["email"].(string),
		CustomerID: session.Values["customer_id"].(string),
	}
}

func FormToUser(request *http.Request) User {
	return User{
		Name:          request.Form.Get("name"),
		Email:         request.Form.Get("email"),
		PlainPassword: request.Form.Get("pass"),
		Phone:         request.Form.Get("phone"),
	}
}

func (user *User) AddToStripeMetadata(params *stripe.CheckoutSessionParams) {
	params.AddMetadata("email", user.Email)
	if len(user.Name) > 0 {
		params.AddMetadata("name", user.Name)
	}
	if len(user.Phone) > 0 {
		params.AddMetadata("phone", user.Name)
	}
	if len(user.PlainPassword) > 0 {
		params.AddMetadata("pass", user.Name)
	}
}
