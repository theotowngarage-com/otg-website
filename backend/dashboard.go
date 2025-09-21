package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/subscription"
)

var ErrNoSubs = errors.New("No subscriptions available")

func get_subscriptions(customer_id string) *subscription.Iter {
	params := &stripe.SubscriptionListParams{}
	params.Customer = stripe.String(customer_id)
	// Filter by active status (could also be `all`, `active`, `ended`, `canceled`)
	params.Status = stripe.String(`active`)
	// Get at most the last 1 subscription(s)
	params.Limit = stripe.Int64(1)
	return subscription.List(params)
}
func render_subscriptions(user User) (bytes.Buffer, error) {
	log.Print("Getting subs for customer id - ", user.Customer_id)
	subscriptions := get_subscriptions(user.Customer_id)
	// we only expect at most 1 subscription. If there isn't even one, we can expect there are none
	user.Active = subscriptions.Next()
	if user.Active {
		s := subscriptions.Subscription()
		log.Printf("Subscription ID: %s\n", s.ID)
		log.Printf("Status: %s\n", s.Status)
		log.Printf("Current Period Start: %v\n", s.LatestInvoice.PeriodStart)
		log.Printf("Current Period End: %v\n", s.LatestInvoice.PeriodEnd)
		log.Println("---")
	}
	templ := template.Must(template.ParseFiles("htmx_templates/subscriptions.html"))

	// Execute the template with dynamic data
	var subscriptionsHTML bytes.Buffer
	err := templ.Execute(&subscriptionsHTML, struct {
		User User
	}{User: user})

	if err != nil {
		fmt.Println("error executing template: ", err)
		return bytes.Buffer{}, err
	}
	return subscriptionsHTML, err
}

func serve_subscriptions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		session, _ := store.Get(request, "theotowngarage.com")

		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		subs, err := render_subscriptions(toUser(*session))
		if err != nil && err != ErrNoSubs {
			log.Print(err)
			fmt.Fprintln(w, "<p>Internal server error -", err, "</p>")
		}
		fmt.Fprint(w, subs.String())

		// Print secret message
	}
}
