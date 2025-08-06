package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"yt_rss2/database"
	"yt_rss2/templates"
	_ "yt_rss2/config"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store *sessions.CookieStore

func init() {
	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		log.Fatal("SESSION_KEY environment variable not set. Please set it to a random 32-byte string.")
	}
	store = sessions.NewCookieStore([]byte(sessionKey))
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Pass a default user for the layout
		templates.Layout(templates.User{Theme: "rose-pine"}, templates.RegisterPage("")).Render(r.Context(), w)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	_, err = database.DB.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, string(hashedPassword))
	if err != nil {
		templates.Layout(templates.User{Theme: "rose-pine"}, templates.RegisterPage("Username already taken")).Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		templates.Layout(templates.User{Theme: "rose-pine"}, templates.LoginPage("")).Render(r.Context(), w)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	var user templates.User
	var storedHash string
	err := database.DB.QueryRow("SELECT id, username, theme, password_hash FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Theme, &storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			templates.Layout(templates.User{Theme: "rose-pine"}, templates.LoginPage("Invalid username or password")).Render(r.Context(), w)
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		templates.Layout(templates.User{Theme: "rose-pine"}, templates.LoginPage("Invalid username or password")).Render(r.Context(), w)
		return
	}

	session, _ := store.Get(r, "session-name")
	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["user_id"] = nil
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session-name")
		userID, ok := session.Values["user_id"].(int)

		if !ok || userID == 0 {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		
		var user templates.User
		err := database.DB.QueryRow("SELECT id, username, theme FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Theme)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
