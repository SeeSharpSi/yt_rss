package handlers

import (
	"net/http"
	"yt_rss2/templates"

	"github.com/gorilla/mux"
)

func VideoPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]
	user := r.Context().Value("user").(templates.User)

	templates.Layout(user, templates.VideoPage(videoID)).Render(r.Context(), w)
}
