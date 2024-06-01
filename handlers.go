package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"html/template"
	"log"
	"net/http"
	"time"
)

// LoginHandler handles the login logic
func (*App) LoginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	pass := r.FormValue("password")

	if username == app.config.defaultAdminUser && pass == app.config.defaultAdminPassword {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    pass,
			Path:     "/",
			HttpOnly: true,
			Secure:   !app.config.isDevelopment,
			Expires:  time.Now().Add(1 * time.Hour),
		})
		http.Redirect(w, r, "/shortlinks", http.StatusSeeOther)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// LogoutHandler handles the logout logic
func (*App) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !app.config.isDevelopment,
	})
	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (*App) FollowShortURL(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "shortURL")
	// Get the long URL from the database
	longURL, err := getLinkByShortLink(shortURL)
	if err != nil {
		log.Printf("Failed to get Link by short link \"%s\". Error: %s", shortURL, err)
		if errors.As(err, &sql.ErrNoRows) {
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	log.Println("Redirecting user to:", longURL)
	http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)

}

// ShortenURL handles the URL shortening logic
func (*App) ShortenURL(w http.ResponseWriter, r *http.Request) {
	longURL := r.FormValue("url")
	shortURL := r.FormValue("shortURL")
	_, err := insertLink(Link{ShortCode: shortURL, LongLink: longURL})
	if err != nil {
		log.Println("Failed to insert Link:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/shortlinks", http.StatusSeeOther)
}

func (*App) ShortenURLApi(w http.ResponseWriter, r *http.Request) {
	type requestType struct {
		URL      string `json:"url"`
		ShortUrl string `json:"short_code"`
	}
	var req requestType
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Failed to decode body: ", err)
		http.Error(w, "Cannot parse body", http.StatusBadRequest)
		return
	}
	i, err := insertLink(Link{ShortCode: req.ShortUrl, LongLink: req.URL})
	if err != nil {
		log.Println("Failed to insert Link: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	link, err := getLinkByID(i)
	if err != nil {
		log.Println("Failed to retrieve full Link: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(link)
	if err != nil {
		log.Println("Failed to marshal link body: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(data)
}

func (*App) DeleteShortLink(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "shortURL")
	err := deleteLink(shortURL)
	if err != nil {
		log.Println("Failed to delete Link:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/shortlinks", http.StatusSeeOther)
}

func (*App) ShortLinksHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl := template.Must(template.ParseFiles("views/shortlinks.html"))
	links, err := getAllLinks()
	if err != nil {
		log.Println("Failed to get all links:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	type ViewModel struct {
		Links   []Link
		SiteUrl string
	}

	var siteUlr string

	// use the port if in development mode
	if app.config.isDevelopment {
		siteUlr = fmt.Sprintf("%s:%s", app.config.baseUrl, app.config.port)
	} else {
		siteUlr = app.config.baseUrl
	}

	tmpl.Execute(w, ViewModel{
		Links:   links,
		SiteUrl: siteUlr,
	},
	)
}
