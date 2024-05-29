package main

import "database/sql"

type App struct {
	config config
	db     *sql.DB
}

type config struct {
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
