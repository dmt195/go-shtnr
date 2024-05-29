package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gobuffalo/envy"
)

var app App

func main() {

	app = App{
		config{
			envy.Get("BASE_URL", "http://localhost"),
			envy.Get("PORT", "3001"),
			envy.Get("IS_DEV", "false") == "true",
			envy.Get("ADMIN_USERNAME", "admin"),
			envy.Get("ADMIN_PASSWORD", "password"),
			envy.Get("API_KEY", "123456"),
		},
		nil,
	}

	setupDatabase()

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "views/login.html")
	})

	// Public route
	r.Post("/login", app.LoginHandler)
	r.Get("/logout", app.LogoutHandler)
	r.Get("/favicon.ico", http.NotFound) // just to stop errors

	r.With(ApiKeyAuthMiddleware).Group(func(r chi.Router) {
		r.Post("/api/shorten", app.ShortenURLApi)
	})

	// Protected routes
	r.With(AuthMiddleware).Group(func(r chi.Router) {
		r.Get("/shortlinks", app.ShortLinksHandler)
		r.Post("/shorten", app.ShortenURL)
		r.Post("/{shortURL}/delete", app.DeleteShortLink)
		//r.Get("/{shortURL}/info", app.GetShortLinkInfo)
	},
	)

	// last route to catch all
	r.Get("/{shortURL}", app.FollowShortURL)

	http.Handle("/", r)
	fmt.Println("Server started at :", app.config.port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", app.config.port), nil)
	if err != nil {
		panic(err)
	}
}

// AuthMiddleware is the middleware for authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value != "authenticated" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ApiKeyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-KEY")
		if apiKey != app.config.apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
