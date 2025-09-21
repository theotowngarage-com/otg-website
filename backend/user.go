package main

import "github.com/gorilla/sessions"

type User struct {
	Email       string
	Name        string
	Phone       string
	Password    []byte
	Active      bool
	Customer_id string
}

func toUser(session sessions.Session) User {
	return User{
		Email:       session.Values["email"].(string),
		Customer_id: session.Values["customer_id"].(string),
	}
}
