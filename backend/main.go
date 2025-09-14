package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	gomail "gopkg.in/mail.v2"
)

func main() {
	// This test secret API key is a placeholder. Don't include personal details in requests with this key.
	// To see your test secret API key embedded in code samples, sign in to your Stripe account.
	// You can also find your test secret API key at https://dashboard.stripe.com/test/apikeys.
	stripe.Key = "sk_test_xxxx...xxxxxx"

	http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/create-checkout-session", createCheckoutSession)
	addr := "localhost:4242"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

/*
*

	Add an endpoint on your server that creates a Checkout Session.
	A Checkout Session controls what your customer sees on the payment page such as line items,
	the order amount and currency, and acceptable payment methods.
	Stripe enables cards and other common payment methods for you by default,
	and you can enable or disable payment methods directly in the Stripe Dashboard.
*/
func createCheckoutSession(w http.ResponseWriter, response *http.Request) {
	// domain := "http://localhost:4242"
	SuccessURL := "http://localhost:1313/checkout/success/?session_id={CHECKOUT_SESSION_ID}"
	CancelURL := "http://localhost:1313/checkout/cancel/"
	BillingString := "flexible"
	BillingMode := stripe.CheckoutSessionSubscriptionDataBillingModeParams{Type: &BillingString}

	params := &stripe.CheckoutSessionParams{
		SuccessURL: &SuccessURL,
		CancelURL:  &CancelURL,
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				Price: stripe.String("price_1S79v8EFGoOPzKA9JlFqQE34"),
				// For usage-based billing, don't pass quantity
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			BillingMode: &BillingMode,
		},
	}

	session, err := session.New(params)

	if err != nil {
		log.Printf("session.New: %v", err)
	}

	http.Redirect(w, response, session.URL, http.StatusSeeOther)
}

func create_subscription() {
	product_params := &stripe.ProductParams{
		Name:        stripe.String("Starter Subscription"),
		Description: stripe.String("$12/Month subscription"),
	}
	starter_product, _ := product.New(product_params)

	price_params := &stripe.PriceParams{
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		Product:  stripe.String(starter_product.ID),
		Recurring: &stripe.PriceRecurringParams{
			Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
		},
		UnitAmount: stripe.Int64(1200),
	}
	starter_price, _ := price.New(price_params)

	fmt.Println("Success! Here is your starter subscription product id: " + starter_product.ID)
	fmt.Println("Success! Here is your starter subscription price id: " + starter_price.ID)
}

func send_mail() {
	// Create a new message
	message := gomail.NewMessage()

	// Set email headers
	message.SetHeader("From", "youremail@email.com")
	message.SetHeader("To", "recipient1@email.com")
	message.SetHeader("Subject", "Hello from the Mailtrap team")

	// Set email body
	message.SetBody("text/plain", "This is the Test Body")

	// Set up the SMTP dialer
	dialer := gomail.NewDialer("live.smtp.mailtrap.io", 587, "api", "1a2b3c4d5e6f7g")

	// Send the email
	if err := dialer.DialAndSend(message); err != nil {
		fmt.Println("Error:", err)
		panic(err)
	} else {
		fmt.Println("Email sent successfully!")
	}
}
