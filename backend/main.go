package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
	_ "github.com/tursodatabase/go-libsql"
	"golang.org/x/crypto/bcrypt"
)

const host_addr string = "localhost:4242"

type User struct {
	email       string
	name        string
	phone       string
	password    []byte
	active      bool
	customer_id string
}

func main() {
	// You can find your test secret API key at https://dashboard.stripe.com/test/apikeys.
	stripe.Key = "sk_xxx...xxx"

	// Serve the static website built with Hugo
	http.Handle("/", http.FileServer(http.Dir("../public")))
	http.HandleFunc("/webhook", handleWebhook) // handle stripe webhooks
	http.HandleFunc("/create-checkout-session", createCheckoutSession)

	http.HandleFunc("/secret", secret)    // sessions.go
	http.HandleFunc("/logout", logout)    // sessions.go
	http.HandleFunc("POST /login", login) // sessions.go
	// http.HandleFunc("/login", logout)  // sessions.go

	log.Printf("Listening on %s", host_addr)
	err := initDatabase(true)
	if err != nil {
		log.Fatal("Could not initialise DB ", err)
		os.Exit(1)
	}
	log.Fatal(http.ListenAndServe(host_addr, nil))
}

/*
Add an endpoint on your server that creates a Checkout Session.
A Checkout Session controls what your customer sees on the payment page such as line items,
the order amount and currency, and acceptable payment methods.
Stripe enables cards and other common payment methods for you by default,
and you can enable or disable payment methods directly in the Stripe Dashboard.
*/
func createCheckoutSession(w http.ResponseWriter, request *http.Request) {
	log.Print("New member signup request")
	if request.ParseForm() != nil || !validateInput(request.Form) {
		log.Fatal("malformed request") // highlight - potential attack
		// do not give reason for a failure (on purpose)
		http.Redirect(w, request, "http://"+host_addr+"/checkout/", http.StatusSeeOther)
		return
	}
	// Only hashing the password at this stage to make sure it doesn't error out after the payment is done
	// We do not use the result of the hash to avoid sending the final hashed pw through the internet pipes
	_, err := bcrypt.GenerateFromPassword([]byte(request.Form.Get("pass")), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Encryption failed:", err)
		http.Redirect(w, request, "http://"+host_addr+"/checkout/cancel/?reason=failed_crypt", http.StatusSeeOther)
		return
	}
	for key, value := range request.Form {
		log.Print("%s : %s", key, value)
	}

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String("http://" + host_addr + "/checkout/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("http://" + host_addr + "/checkout/cancel/"),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				Price: stripe.String("price_1S79v8EFGoOPzKA9JlFqQE34"),
				// For usage-based billing, don't pass quantity
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			BillingMode: &stripe.CheckoutSessionSubscriptionDataBillingModeParams{Type: stripe.String("flexible")},
		},
		CustomerEmail: stripe.String(request.Form.Get("email")),
	}

	// Metadata is forwarded to the successful webhook, so we can register the new user in the db
	for _, id := range []string{"email", "name", "phone", "pass"} {
		params.AddMetadata(id, request.Form.Get(id))
	}

	session, err := session.New(params)

	if err != nil {
		log.Printf("session.New: %v", err)
		http.Redirect(w, request, "http://"+host_addr+"/checkout/cancel/?reason=failed_session", http.StatusSeeOther)
		return
	}
	http.Redirect(w, request, session.URL, http.StatusSeeOther)
}

func validateInput(form url.Values) bool {
	for _, id := range []string{"email", "name", "pass"} {
		if !form.Has(id) {
			return false
		}
	}
	return true
}

func openDB(isTest bool) (*sql.DB, error) {
	var filename string
	if isTest {
		filename = "test.db"
	} else {
		filename = "production.db"
	}
	return sql.Open("libsql", "file:./"+filename)
}

func initDatabase(isTest bool) error {

	db, err := openDB(isTest)
	if err != nil {
		return err
	}
	defer func() {
		if closeError := db.Close(); closeError != nil {
			fmt.Println("Error closing database", closeError)
			if err == nil {
				err = closeError
			}
		}
	}()

	_, dbErr := db.Exec("CREATE TABLE user(id INTEGER PRIMARY KEY, email TEXT, name TEXT, phone TEXT, password BLOB, active INT, customer_id TEXT)")
	// ignore error if table already exists
	if dbErr == nil {
		log.Print("Database created")
	} else {
		log.Print("Using existing Database - ", dbErr)
	}
	return err
}

func addUser(user User, isTest bool) error {
	db, err := openDB(isTest)
	if err != nil {
		return err
	}
	defer func() {
		if closeError := db.Close(); closeError != nil {
			fmt.Println("Error closing database", closeError)
			if err == nil {
				err = closeError
			}
		}
	}()
	// no need to specify id, libsql will use an available id, usually an increment over the max
	_, err = db.Query("INSERT INTO user (email, name, phone , password , active, customer_id) VALUES (?, ?, ?, ?, ?, ?)",
		user.email, user.name, user.phone, user.password, user.active, user.customer_id)
	sendMail(user.email, user, Welcome)
	return err
}

func handleWebhook(w http.ResponseWriter, request *http.Request) {

	const MaxBodyBytes = int64(65536)
	request.Body = http.MaxBytesReader(w, request.Body, MaxBodyBytes)

	body, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Pass the request body and Stripe-Signature header to ConstructEvent, along with the webhook signing key
	// Use the secret provided by Stripe CLI for local testing
	// or your webhook endpoint's secret.
	endpointSecret := "whsec_xxx...xxx"
	EventOptions := webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true}
	event, err := webhook.ConstructEventWithOptions(body, request.Header.Get("Stripe-Signature"), endpointSecret, EventOptions)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
		return
	}

	if event.Type == stripe.EventTypeCheckoutSessionCompleted ||
		event.Type == stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded {
		var checkout_session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkout_session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = FulfillCheckout(checkout_session.ID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
			return
		}

	}
}

func FulfillCheckout(checkout_session string) error {
	log.Print("Successful subscription")
	// TODO: Create a user and push to the database
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("line_items")
	session, err := session.Get(checkout_session, params)
	if err != nil {
		// TODO this should never happen
		log.Fatal("Failed to retrieve metadata from payment checkout:", err)
		// At this point we could create a secondary user creation page, but not worth it
		return err
	}
	// Retrieve the metadata injected before the checkout
	meta := session.Metadata
	// Hashing the password again after the payment to avoid sending the stored hashed pw through the internet pipes
	// TODO : Salt the password for proper storage...
	// FIXME : How do we migrate the existing passwords? tag them with "insecure" and check without salt?
	hashedPassword, err := hash_and_salt(meta["pass"])
	if err != nil {
		// TODO this should be an absolute failure, because we already tried to encrypt it beforehand
		log.Fatal("Encryption failed:", err)
		return err
	}

	user := User{
		name:        meta["name"],
		email:       meta["email"],
		phone:       meta["phone"],
		active:      true,
		password:    hashedPassword,
		customer_id: session.Customer.ID,
	}

	if dbErr := addUser(user, true); dbErr != nil {
		log.Fatal("Error closing database", dbErr)
		// maybe forward all the Fatal exceptions via email?
		return dbErr
	}

	// Redirect to the main site
	// TODO: Send confirmation mail to the user ?
	return err
}
