package handlers

import (
	"database/sql"
	"net/http"
	"regexp"
	"yt_rss2/database"

	"github.com/gorilla/mux"
	"yt_rss2/templates"
)

var youtubeVideoRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

func VideoPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]
	user := r.Context().Value("user").(templates.User)

	if !youtubeVideoRegex.MatchString(videoID) {
		http.Error(w, "Invalid video ID", http.StatusBadRequest)
		return
	}

	var progress int
	err := database.DB.QueryRow("SELECT progress_seconds FROM video_progress WHERE user_id = ? AND video_id = ?", user.ID, videoID).Scan(&progress)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Failed to get progress", http.StatusInternalServerError)
		return
	}

	templates.Layout(user, templates.VideoPage(videoID, progress)).Render(r.Context(), w)
}
