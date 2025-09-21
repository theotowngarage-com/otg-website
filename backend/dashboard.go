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
	SubID := ""
	if user.Active {
		s := subscriptions.Subscription()
		SubID = s.ID
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
		User  User
		SubID string
	}{User: user, SubID: SubID})

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

// CancelSubscriptionHandler handles HTTP requests to cancel a Stripe subscription
func CancelSubscriptionHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "theotowngarage.com")

		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if !r.URL.Query().Has("sub_id") {
			log.Print("Token not in request - ", r.URL.Query())
			// http.Redirect(w, r, host_url+"/login/?reason=misc", http.StatusSeeOther)
			return
		}
		SubID := r.URL.Query().Get("sub_id")

		// Cancel the subscription immediately (or set CancelAtPeriodEnd = true to cancel later)
		params := &stripe.SubscriptionCancelParams{}
		_, err := subscription.Cancel(SubID, params)
		if err != nil {
			log.Printf("Failed to cancel subscription: %v", err)
			http.Error(w, "Failed to cancel subscription", http.StatusInternalServerError)
			return
		}
		_, err = db.Exec("UPDATE user SET active = ? WHERE email = ?", false, session.Values["email"])
		if err != nil {
			log.Printf("Failed to update user active status in DB: ", err)
			return
		}

		fmt.Fprint(w, "Subscription canceled successfully")
	}
}
