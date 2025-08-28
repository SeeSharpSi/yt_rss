package handlers

import (
	"database/sql"
	"fmt"
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

	fmt.Printf("Attempting to fetch transcript for video %s\n", videoID)
	var text string
	api := transcript.NewYouTubeTranscriptApi()
	trans, err := api.Fetch(videoID, []string{"en"}, false)
	if err != nil {
		fmt.Printf("Error fetching transcript for %s: %v\n", videoID, err)
		text = ""
	} else {
		fmt.Printf("Transcript fetched successfully, %d snippets\n", len(trans.Snippets))
		if len(trans.Snippets) > 0 {
			fmt.Printf("First snippet: '%s'\n", trans.Snippets[0].Text)
		}
		formatter := &transcript.TextFormatter{}
		text = formatter.FormatTranscript(trans, nil)
		fmt.Printf("Formatted text length: %d characters\n", len(text))
		fmt.Println("Transcript for video", videoID, ":")
		fmt.Println(text)
	}

	var progress int
	err = database.DB.QueryRow("SELECT progress_seconds FROM video_progress WHERE user_id = ? AND video_id = ?", user.ID, videoID).Scan(&progress)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Failed to get progress", http.StatusInternalServerError)
		return
	}

	templates.Layout(user, templates.VideoPage(videoID, progress)).Render(r.Context(), w)
}
