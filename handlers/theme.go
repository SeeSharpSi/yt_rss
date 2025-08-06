package handlers

import (
	"net/http"
	"yt_rss2/database"
	"yt_rss2/templates"
)

var themes = []string{"rose-pine", "nord", "gruvbox"}

func CycleThemeHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(templates.User)

	// Find the next theme in the cycle.
	currentIndex := -1
	for i, theme := range themes {
		if theme == user.Theme {
			currentIndex = i
			break
		}
	}
	nextIndex := (currentIndex + 1) % len(themes)
	nextTheme := themes[nextIndex]

	// Update the theme in the database.
	_, err := database.DB.Exec("UPDATE users SET theme = ? WHERE id = ?", nextTheme, user.ID)
	if err != nil {
		http.Error(w, "Failed to update theme", http.StatusInternalServerError)
		return
	}

	// Tell the browser to do a full page refresh.
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}
