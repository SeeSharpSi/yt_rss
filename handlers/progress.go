package handlers

import (
	"encoding/json"
	"net/http"
	"yt_rss2/database"
	"yt_rss2/templates"
)

type ProgressRequest struct {
	VideoID  string `json:"video_id"`
	Progress int    `json:"progress"`
}

func ProgressHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(templates.User)

	var p ProgressRequest
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = database.DB.Exec(`
		INSERT INTO video_progress (user_id, video_id, progress_seconds)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, video_id) DO UPDATE SET progress_seconds = excluded.progress_seconds;
	`, user.ID, p.VideoID, p.Progress)

	if err != nil {
		http.Error(w, "Failed to save progress", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
