package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"yt_rss2/database"
	"yt_rss2/handlers"
	"yt_rss2/templates"

	"github.com/gorilla/mux"
	_ "yt_rss2/config"
)

func main() {
	port := flag.Int("port", 0, "port to run the server on")
	flag.Parse()

	database.InitDB()

	r := mux.NewRouter()

	r.HandleFunc("/login", handlers.LoginHandler)
	r.HandleFunc("/register", handlers.RegisterHandler)
	r.HandleFunc("/logout", handlers.LogoutHandler)

	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(handlers.AuthMiddleware)

	authRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user").(templates.User)
		templates.Layout(user, templates.IndexPage()).Render(r.Context(), w)
	})
	authRouter.HandleFunc("/videos", handlers.VideosHandler)
	authRouter.HandleFunc("/video/{id}", handlers.VideoPageHandler)
	authRouter.HandleFunc("/channels", handlers.ChannelsHandler)
	authRouter.HandleFunc("/export", handlers.ExportHandler)
	authRouter.HandleFunc("/import", handlers.ImportHandler)
	authRouter.HandleFunc("/cycle-theme", handlers.CycleThemeHandler).Methods("POST")
	authRouter.HandleFunc("/add-channel", handlers.AddChannelHandler).Methods("POST")
	authRouter.HandleFunc("/delete-channel", handlers.DeleteChannelHandler).Methods("POST")

	addr := ":" + strconv.Itoa(*port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Listening on port: " + strconv.Itoa(l.Addr().(*net.TCPAddr).Port))
	log.Fatal(http.Serve(l, r))
}
