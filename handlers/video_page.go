package handlers

import (
	"database/sql"
	"net/http"
	"regexp"
	"yt_rss2/database"
	"yt_rss2/pkg/transcript"

	"github.com/gorilla/mux"
	"yt_rss2/templates"
)

func VideoPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]
	user := r.Context().Value("user").(templates.User)

	youtubeVideoRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)
	if !youtubeVideoRegex.MatchString(videoID) {
		http.Error(w, "Invalid video ID", http.StatusBadRequest)
		return
	}

	api := transcript.NewYouTubeTranscriptApi()
	trans, err := api.Fetch(videoID, []string{"en"}, false)
	if err != nil {
		// Transcript fetch failed, but continue with page load
	} else {
		formatter := &transcript.TextFormatter{}
		formatter.FormatTranscript(trans, nil)
		// Transcript is formatted for the summary endpoint
	}

	var progress int
	err = database.DB.QueryRow("SELECT progress_seconds FROM video_progress WHERE user_id = ? AND video_id = ?", user.ID, videoID).Scan(&progress)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Failed to get progress", http.StatusInternalServerError)
		return
	}

	templates.Layout(user, templates.VideoPage(videoID, progress)).Render(r.Context(), w)
}
