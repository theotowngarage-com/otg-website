package main

import (
	"log"

	"github.com/BurntSushi/toml"
)

var config = LoadConfig()

var defaultConfig = Config{
	Backend: BackendConf{
		Host:             "localhost",
		Protocol:         "http",
		CookiePrivateKey: "cookieMonster",
	},
	Stripe: stripeConf{
		Key:            "",
		EndpointSecret: "",
	},
	Email: emailConf{
		User:     "info@theotowngarage.com",
		Host:     "smtppro.zoho.com",
		Port:     465,
		Password: "",
	},
}

type stripeConf struct {
	// sk_xxx...xxx
	Key string
	// whsec_xxx...xxx
	EndpointSecret string
}

type emailConf struct {
	User     string
	Host     string
	Port     int
	Password string
}

type BackendConf struct {
	// localhost, 10.11.12.13, theotowngarage.com
	Host string
	// http, https
	Protocol string
	Port     int
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	CookiePrivateKey string
}

type Config struct {
	Backend BackendConf
	Stripe  stripeConf
	Email   emailConf
}

func LoadConfig() Config {
	var parseconf Config
	_, err := toml.DecodeFile("../config/_default/config.toml", &parseconf)
	if err != nil {
		log.Print("Failed to load confg - ", err)
		parseconf = defaultConfig
	}
	return parseconf
}
