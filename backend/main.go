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
	"strconv"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
	_ "github.com/tursodatabase/go-libsql"
	"golang.org/x/crypto/bcrypt"
)

var host_addr string = config.Backend.Host + ":" + strconv.Itoa(config.Backend.Port)
var host_url string = config.Backend.Protocol + "://" + host_addr

func main() {
	// You can find your test secret API key at https://dashboard.stripe.com/test/apikeys.
	stripe.Key = config.Stripe.Key

	db, err := openDB( /* isTest */ true)
	if err != nil {
		log.Fatal("Failed to open database - ", err)
		return
	}
	err = initDatabase(db)

	// Serve the static website built with Hugo
	http.Handle("/", http.FileServer(http.Dir("../public")))
	http.HandleFunc("/webhook", handleWebhook) // handle stripe webhooks
	http.HandleFunc("POST /checkout/", createCheckoutSession(db))
	http.HandleFunc("/re-checkout", createCheckoutSession(db))

	http.HandleFunc("/subscriptions", serve_subscriptions(db))           // dashboard.go
	http.HandleFunc("/cancel-subscription", CancelSubscriptionHandler()) // dashboard.go

	http.HandleFunc("/logout", logout) // sessions.go
	// override request in order to serve /dashboard if user already logged in
	http.HandleFunc("GET /login/", request_login) // sessions.go
	http.HandleFunc("POST /login/", login(db))    // sessions.go

	http.HandleFunc("POST /request-reset", requestPasswordResetHandler(db)) // sessions.go
	http.HandleFunc("POST /reset-password/", resetPasswordHandler(db))      // sessions.go
	http.HandleFunc("POST /reset-password", resetPasswordHandler(db))       // sessions.go

	log.Printf("Listening on %s", host_url)
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
func createCheckoutSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		session, _ := store.Get(request, "theotowngarage.com")

		log.Print("check if user is logged in")
		// Check if user is authenticated
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			log.Print("New member signup request")
		} else if session.Values["active"] != nil && session.Values["active"].(bool) {
			log.Print(session.Values["email"].(string), " wants to sign up, but they already have an active membership")
			http.Redirect(w, request, host_url+"/dashboard?error=already_active", http.StatusSeeOther)
			return
		} else {
			log.Print(session.Values["email"].(string), " wants to sign up again?")
			user := toUser(*session)
			ServeStripeCheckoutSession(w, request, user)
			return
		}

		// Serve checkout session for a brand new Customer

		if request.ParseForm() != nil || !validateInput(request.Form) {
			log.Fatal("malformed request") // highlight - potential attack
			// do not give reason for a failure (on purpose)
			http.Redirect(w, request, host_url+"/checkout/", http.StatusSeeOther)
			return
		}
		exists, err := emailExists(request, true)
		if err != nil {
			fmt.Println("Failed to check if email exists??:", err)
			return
		}
		if exists {
			http.Redirect(w, request, host_url+"/checkout/?reason=email_exists", http.StatusSeeOther)
			return
		}
		// Only hashing the password at this stage to make sure it doesn't error out after the payment is done
		// We do not use the result of the hash to avoid sending the final hashed pw through the internet pipes
		_, err = bcrypt.GenerateFromPassword([]byte(request.Form.Get("pass")), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println("Encryption failed:", err)
			http.Redirect(w, request, host_url+"/checkout/?reason=failed_crypt", http.StatusSeeOther)
			return
		}
		for key, value := range request.Form {
			log.Print(key, " : ", value)
		}
		user := FormToUser(request)
		ServeStripeCheckoutSession(w, request, user)
	}
}

func ServeStripeCheckoutSession(w http.ResponseWriter, request *http.Request, user User) {
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(host_url + "/checkout/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(host_url + "/checkout/?reason=stripe_cancel"),
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
	}
	// "You may only specify one of these parameters: customer, customer_email.
	if len(user.CustomerID) > 0 {
		params.Customer = stripe.String(user.CustomerID)
	} else {
		params.CustomerEmail = stripe.String(user.Email)
	}

	// Metadata is forwarded to the successful webhook, so we can register the new user in the db
	user.AddToStripeMetadata(params)
	stripeSession, err := session.New(params)
	if err != nil {
		log.Printf("session.New: %v", err)
		http.Redirect(w, request, host_url+"/checkout/?reason=failed_session", http.StatusSeeOther)
		return
	}
	http.Redirect(w, request, stripeSession.URL, http.StatusSeeOther)
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

func initDatabase(db *sql.DB) error {

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user(
		id INTEGER PRIMARY KEY,
		email TEXT UNIQUE,
		name TEXT,
		phone TEXT,
		password BLOB,
		active INT,
		customer_id TEXT);`)
	if err != nil {
		return err
	}
	// password_reset_tokens table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS password_reset_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL,
		token TEXT NOT NULL,
		expires_at DATETIME NOT NULL
	);`)
	if err != nil {
		log.Print("Databases creation failed")
		return err
	}
	log.Print("Databases created")
	return nil
}

func emailExists(request *http.Request, isTest bool) (bool, error) {
	// exects a form t be parsed
	email := request.Form.Get("email")
	db, err := openDB(isTest)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeError := db.Close(); closeError != nil {
			fmt.Println("Error closing database", closeError)
			if err == nil {
				err = closeError
			}
		}
	}()

	var exists int
	err = db.QueryRow("SELECT COUNT(*) FROM user WHERE email = ?", email).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists > 0, nil
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
		user.Email, user.Name, user.Phone, user.Password, user.Active, user.CustomerID)
	// N.B. libsql does not wrap the error code into the returned error. Therefore we cannot know what went wrong except from parsing the returned message...
	// Too lazy to analyze if the email is already in use.
	if err != nil {
		// Alert the user??
		return err
	}
	return sendMail(user.Email, user, Welcome, "https://discord.gg/CGBgKNwT", struct{}{})
}

func getNumberOfUsers(isTest bool) (int, error) {

	db, err := openDB(isTest)
	if err != nil {
		return 0, err
	}
	defer func() {
		if closeError := db.Close(); closeError != nil {
			fmt.Println("Error closing database", closeError)
			if err == nil {
				err = closeError
			}
		}
	}()
	var number_active_users int
	err = db.QueryRow("SELECT COUNT(*) FROM user WHERE active = 1").Scan(&number_active_users)
	if err != nil {
		log.Printf("Error querying database: %v\n", err)
		return 0, err
	}
	return number_active_users, nil
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
	endpointSecret := config.Stripe.EndpointSecret
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
	} else if event.Type == stripe.EventTypeCustomerSubscriptionDeleted {
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = handleSubscriptionEnded(subscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating user status: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}
	w.WriteHeader(http.StatusOK)
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
	hashedPassword, err := hash_and_salt(meta["pass"])
	if err != nil {
		// TODO this should be an absolute failure, because we already tried to encrypt it beforehand
		log.Fatal("Encryption failed:", err)
		return err
	}

	user := User{
		Name:       meta["name"],
		Email:      meta["email"],
		Phone:      meta["phone"],
		Active:     true,
		Password:   hashedPassword,
		CustomerID: session.Customer.ID,
	}
	dbErr := addUser(user, true)
	if dbErr != nil {
		log.Fatal("Error while creating the user - ", dbErr)
		// maybe forward all the Fatal exceptions via email?
		return dbErr
	}
	// Send email to our ourselves
	nbUsers, err := getNumberOfUsers(true)
	if err != nil {
		log.Printf("Couldn't read db: %v\n", err)
		return err
	}
	err = sendMail(defaultConfig.Email.User, user, NewMember, "", struct{ Number int }{Number: nbUsers})

	// Redirect to the main site
	// TODO: Send confirmation mail to the user ?
	return err
}

// getUserByCustomerID retrieves a user from the database by their Stripe customer ID
func getUserByCustomerID(customerID string, isTest bool) (User, error) {
	db, err := openDB(isTest)
	if err != nil {
		return User{}, err
	}
	defer db.Close()

	var user User
	err = db.QueryRow("SELECT email, name, phone, password, active, customer_id FROM user WHERE customer_id = ?", customerID).
		Scan(&user.Email, &user.Name, &user.Phone, &user.Password, &user.Active, &user.CustomerID)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func handleSubscriptionEnded(subscription stripe.Subscription) error {
	var err error
	// Update user's active status in database
	db, err := openDB(true)
	if err != nil {
		log.Printf("Error opening database: %v\n", err)
		return err
	}
	defer db.Close()

	_, err = db.Exec("UPDATE user SET active = ? WHERE customer_id = ?", false, subscription.Customer.ID)
	user, err := getUserByCustomerID(subscription.Customer.ID, true)
	if err != nil {
		log.Printf("Error updating database: %v\n", err)
		return err
	}
	var number_active_users int
	err = db.QueryRow("SELECT COUNT(*) FROM user WHERE active = 1").Scan(&number_active_users)
	if err != nil {
		log.Printf("Error querying database: %v\n", err)
		return err
	}
	// Send email to our ourselves
	nbUsers, err := getNumberOfUsers(true)
	if err != nil {
		log.Printf("Couldn't read db: %v\n", err)
		return err
	}
	err = sendMail(defaultConfig.Email.User, user, Unsubscription, "", struct{ Number int }{Number: nbUsers})
	if err != nil {
		log.Printf("Couldn't send email: %v\n", err)
		return err
	}
	// Send email to the user
	err = sendMail(user.Email, user, Goodbye, "https://discord.gg/CGBgKNwT", struct{}{})
	if err != nil {
		log.Printf("Couldn't send email: %v\n", err)
		return err
	}
	log.Printf("Registered event: Subscription ended")
	return err
}
