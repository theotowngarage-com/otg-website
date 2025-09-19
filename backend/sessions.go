// sessions.go
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("<secret-cookie-jar-key>")
	store = sessions.NewCookieStore(key)
)

func secret(w http.ResponseWriter, request *http.Request) {
	session, _ := store.Get(request, "cookie-name")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Print secret message
	fmt.Fprintln(w, "The cake is a lie!")
}

// Only handles POST requests. GET requests wil serve the plain website
func login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		session, _ := store.Get(request, "cookie-name")

		// Authentication goes here

		log.Print("Sign-in request")
		if request.ParseForm() != nil || !validateSignInInput(request.Form) {
			log.Print("malformed request") // highlight - potential attack
			// do not give reason for a failure (on purpose)
			http.Redirect(w, request, host_url+"/login/?reason=misc", http.StatusSeeOther)
			return
		}
		// look up user

		db, err := openDB(true)
		if err != nil {
			return
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
		var password []byte
		err = db.QueryRow("SELECT password FROM user WHERE email = ?", request.Form.Get("email")).Scan(&password)
		if err == sql.ErrNoRows {
			log.Print("User not found")
			http.Redirect(w, request, host_url+"/login/?reason=combo_fail", http.StatusSeeOther)
			return
		} else if err != nil {
			log.Fatal("User not found: ", err)
			// TODO handle this ?
			return
		}

		err = bcrypt.CompareHashAndPassword(password, []byte(request.Form.Get("password")))
		if err == bcrypt.ErrMismatchedHashAndPassword {
			log.Print("Psw mismatch")
			http.Redirect(w, request, host_url+"/login/?reason=combo_fail", http.StatusFound)
			return
		} else if err != nil {
			fmt.Println("Encryption failed:", err)
			http.Redirect(w, request, host_url+"/login/?reason=failed_crypt", http.StatusFound)
			return
		}

		// Set user as authenticated
		session.Values["authenticated"] = true
		session.Save(request, w)
		log.Print("Auth successful", err)
		http.Redirect(w, request, host_url+"/secret", http.StatusSeeOther)
	}
}

func logout(w http.ResponseWriter, request *http.Request) {
	session, _ := store.Get(request, "cookie-name")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(request, w)
	http.Redirect(w, request, host_url, http.StatusSeeOther)
}

func validateSignInInput(form url.Values) bool {
	for _, id := range []string{"email", "password"} {
		if !form.Has(id) {
			return false
		}
	}
	return true
}

func hash_and_salt(password string) ([]byte, error) {
	// Only hashing the password at this stage to make sure it doesn't error out after the payment is done
	// We do not use the result of the hash to avoid sending the final hashed pw through the internet pipes
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Encryption failed:", err)
		return []byte{}, err
	}
	return hashedPassword, err
}

// Reset password
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func sendResetEmail(toEmail, token string) error {
	resetLink := fmt.Sprintf(host_url+"/reset-password?token=%s", token)
	return sendMail(toEmail, User{}, resetMail, resetLink)
}

func requestPasswordResetHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Print("Processing password reset query")
		email := r.FormValue("email")
		if len(email) == 0 {
			log.Print("Form submitted with empty or none email")
			http.Redirect(w, r, host_url+"/login/?reason=misc", http.StatusSeeOther)
			return
		}

		// look up user
		result := 0
		err := db.QueryRow("SELECT 1 FROM user WHERE email = ?;", email).Scan(&result)

		if err == sql.ErrNoRows {
			log.Print("result : ", result)
			log.Print("Email not found : ", email)
			http.Redirect(w, r, host_url+"/login/?reason=wrong_email", http.StatusSeeOther)
			return
		} else if err != nil {
			http.Redirect(w, r, host_url+"/login/?reason=misc", http.StatusSeeOther)
			log.Fatal("Db error: ", err)
			// TODO handle this ?
			return
		}

		token, err := generateToken()
		if err != nil {
			log.Print("Failed to generate token")
			http.Redirect(w, r, host_url+"/login/?reason=misc", http.StatusSeeOther)
			return
		}

		expiry := time.Now().Add(1 * time.Hour)

		_, err = db.Exec("INSERT INTO password_reset_tokens (email, token, expires_at) VALUES (?, ?, ?)", email, token, expiry)
		if err != nil {
			log.Print("db query error - ", err)
			http.Redirect(w, r, host_url+"/login/?reason=db_query", http.StatusSeeOther)
			return
		}

		err = sendResetEmail(email, token)
		if err != nil {
			http.Redirect(w, r, host_url+"/login/?reason=email", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, host_url+"/reset-sent", http.StatusSeeOther)
	}
}

func resetPasswordHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !r.URL.Query().Has("token") {
			log.Print("Token not in request - ", r.URL.Query())
			http.Redirect(w, r, host_url+"/login/?reason=token_invalid", http.StatusSeeOther)
			return
		}
		log.Print("Password reset request")
		token := r.URL.Query().Get("token")

		newPassword := r.FormValue("password")

		var email string
		var expiresAt time.Time
		err := db.QueryRow("SELECT email, expires_at FROM password_reset_tokens WHERE token = ?", token).Scan(&email, &expiresAt)
		if err == sql.ErrNoRows {
			log.Print("Token not found")
			http.Redirect(w, r, host_url+"/login/?reason=token_invalid", http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Print("db query error - ", err)
			http.Redirect(w, r, host_url+"/login/?reason=db_query", http.StatusSeeOther)
			return
		}

		if time.Now().After(expiresAt) {
			http.Error(w, "Token expired", http.StatusBadRequest)
			return
		}

		hashed_psw, err := hash_and_salt(newPassword)
		if err != nil {
			http.Redirect(w, r, host_url+"/login/?reason=failed_crypt", http.StatusSeeOther)
			return
		}
		_, err = db.Exec("UPDATE user SET password = ? WHERE email = ?", hashed_psw, email)
		if err != nil {
			http.Redirect(w, r, host_url+"/login/?reason=db_update", http.StatusSeeOther)
			return
		}

		db.Exec("DELETE FROM password_reset_tokens WHERE token = ?", token)
		log.Print("Password reset success")

		http.Redirect(w, r, host_url+"/reset-success", http.StatusSeeOther)
	}
}
