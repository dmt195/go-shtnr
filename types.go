package main

import (
	"database/sql"
	"github.com/gorilla/securecookie"
)

type App struct {
	config       *Config
	db           *sql.DB
	hashPassword string
	sc           *securecookie.SecureCookie
}

type Config struct {
	baseUrl              string
	port                 string
	isDevelopment        bool
	defaultAdminUser     string
	defaultAdminPassword string
	apiKey               string
}

type user struct {
	id             int
	username       string
	password       string
	passwordHashed string
	salt           string
}

type Link struct {
	ID            int    `json:"id"`
	ShortCode     string `json:"short_code"`
	LongLink      string `json:"long_link"`
	TimesAccessed int    `json:"times_accessed"`
}
