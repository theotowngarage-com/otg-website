// sessions.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"

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
func login(w http.ResponseWriter, request *http.Request) {
	session, _ := store.Get(request, "cookie-name")

	// Authentication goes here

	log.Print("Sign-in request")
	if request.ParseForm() != nil || !validateSignInInput(request.Form) {
		log.Fatal("malformed request") // highlight - potential attack
		// do not give reason for a failure (on purpose)
		http.Redirect(w, request, "http://"+host_addr+"/login/?reason=misc", http.StatusSeeOther)
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
	if err != nil {
		if err == sql.ErrNoRows {
			log.Print("User not found")
			// TODO handle this
			http.Redirect(crw, request, "http://"+host_addr+"/login/?reason=combo_fail", http.StatusSeeOther)
			return
		}
		log.Fatal("User not found: ", err)
		// TODO handle this ?
		return
	}

	err = bcrypt.CompareHashAndPassword(password, []byte(request.Form.Get("password")))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		log.Print("Psw mismatch")
		http.Redirect(w, request, "http://"+host_addr+"/login/?reason=combo_fail", http.StatusFound)
		return
	} else if err != nil {
		fmt.Println("Encryption failed:", err)
		http.Redirect(w, request, "http://"+host_addr+"/login/?reason=failed_crypt", http.StatusFound)
		return
	}

	// Set user as authenticated
	session.Values["authenticated"] = true
	session.Save(request, w)
	log.Print("Auth successful", err)
	http.Redirect(w, request, "http://"+host_addr+"/secret", http.StatusSeeOther)
}

func logout(w http.ResponseWriter, request *http.Request) {
	session, _ := store.Get(request, "cookie-name")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(request, w)
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
