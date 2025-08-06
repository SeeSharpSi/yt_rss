package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"yt_rss2/database"
	"yt_rss2/templates"
)

type Channel struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func ChannelsHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(templates.User)
	channels, err := getChannelsByUserID(user.ID)
	if err != nil {
		http.Error(w, "Failed to load channels", http.StatusInternalServerError)
		return
	}

	selectedChannels := make(map[string]bool)
	templates.Channels(channels, selectedChannels, false, "").Render(r.Context(), w)
}

func AddChannelHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(templates.User)
	r.ParseForm()
	handle := r.FormValue("handle")
	showShorts := r.Form.Get("show-shorts") == "true"

	if !strings.HasPrefix(handle, "@") {
		handle = "@" + handle
	}

	channelURL := "https://www.youtube.com/" + handle

	resp, err := http.Get(channelURL)
	if err != nil {
		http.Error(w, "Failed to fetch channel page", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read channel page", http.StatusInternalServerError)
		return
	}

	rssURL, err := extractRSSLink(string(body))
	if err != nil {
		http.Error(w, "Failed to find RSS link", http.StatusInternalServerError)
		return
	}

	var existingID int
	err = database.DB.QueryRow("SELECT id FROM channels WHERE user_id = ? AND url = ?", user.ID, rssURL).Scan(&existingID)
	if err != sql.ErrNoRows && err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if existingID > 0 {
		channels, _ := getChannelsByUserID(user.ID)
		selectedChannels := make(map[string]bool)
		templates.Channels(channels, selectedChannels, showShorts, "Channel already exists.").Render(r.Context(), w)
		return
	}

	channelName, err := extractChannelName(string(body))
	if err != nil {
		http.Error(w, "Failed to find channel name", http.StatusInternalServerError)
		return
	}

	_, err = database.DB.Exec("INSERT INTO channels (user_id, name, url) VALUES (?, ?, ?)", user.ID, channelName, rssURL)
	if err != nil {
		http.Error(w, "Failed to save channel", http.StatusInternalServerError)
		return
	}

	channels, _ := getChannelsByUserID(user.ID)
	w.Header().Set("HX-Trigger", "channelListChanged")
	selectedChannels := make(map[string]bool)
	templates.Channels(channels, selectedChannels, showShorts, "").Render(r.Context(), w)
}

func DeleteChannelHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(templates.User)
	r.ParseForm()
	urlToDelete := r.URL.Query().Get("url")
	showShorts := r.Form.Get("show-shorts") == "true"

	_, err := database.DB.Exec("DELETE FROM channels WHERE user_id = ? AND url = ?", user.ID, urlToDelete)
	if err != nil {
		http.Error(w, "Failed to delete channel", http.StatusInternalServerError)
		return
	}

	channels, _ := getChannelsByUserID(user.ID)
	w.Header().Set("HX-Trigger", "channelListChanged")
	selectedChannels := make(map[string]bool)
	templates.Channels(channels, selectedChannels, showShorts, "").Render(r.Context(), w)
}

func ExportHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(templates.User)
	channels, err := getChannelsByUserID(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch channels for export", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(channels, "", "  ")
	if err != nil {
		http.Error(w, "Failed to generate JSON", http.StatusInternalServerError)
		return
	}

	templates.ExportPopup(string(jsonData)).Render(r.Context(), w)
}

func ImportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		templates.ImportPopup().Render(r.Context(), w)
		return
	}

	user := r.Context().Value("user").(templates.User)
	r.ParseForm()
	jsonData := r.FormValue("json_data")

	var channelsToImport []Channel
	err := json.Unmarshal([]byte(jsonData), &channelsToImport)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	existingChannels, err := getChannelsByUserID(user.ID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	existingUrls := make(map[string]bool)
	for _, ch := range existingChannels {
		existingUrls[ch.URL] = true
	}

	for _, channel := range channelsToImport {
		if !existingUrls[channel.URL] {
			_, err := database.DB.Exec("INSERT INTO channels (user_id, name, url) VALUES (?, ?, ?)", user.ID, channel.Name, channel.URL)
			if err != nil {
				http.Error(w, "Failed to import one or more channels", http.StatusInternalServerError)
				return
			}
		}
	}

	w.Header().Set("HX-Trigger", "channelListChanged")
	channels, _ := getChannelsByUserID(user.ID)
	selectedChannels := make(map[string]bool)
	
	// Render the updated channels list to the main target.
	templates.Channels(channels, selectedChannels, false, "").Render(r.Context(), w)
	// And also render the component that closes the popup.
	templates.ClosePopup("import-popup").Render(r.Context(), w)
}

func getChannelsByUserID(userID int) ([]templates.Channel, error) {
	rows, err := database.DB.Query("SELECT name, url FROM channels WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []templates.Channel
	for rows.Next() {
		var channel templates.Channel
		if err := rows.Scan(&channel.Name, &channel.URL); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

func extractRSSLink(htmlStr string) (string, error) {
	re := regexp.MustCompile(`<link rel="alternate" type="application/rss\+xml" title="RSS" href="([^"]+)">`)
	matches := re.FindStringSubmatch(htmlStr)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find RSS link in the page")
	}
	return matches[1], nil
}

func extractChannelName(htmlStr string) (string, error) {
	re := regexp.MustCompile(`<meta property="og:title" content="([^"]+)">`)
	matches := re.FindStringSubmatch(htmlStr)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find channel name in the page")
	}
	return html.UnescapeString(matches[1]), nil
}

func AddUserIDToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session-name")
		userID, ok := session.Values["user_id"].(int)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
