package handlers

import (
	"net/http"
	"regexp"

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

	templates.Layout(user, templates.VideoPage(videoID)).Render(r.Context(), w)
}