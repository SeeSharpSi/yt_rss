package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"yt_rss2/templates"

	"github.com/mmcdole/gofeed"
)

func VideosHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	r.ParseForm()
	log.Printf("--- New Request to VideosHandler ---")
	log.Printf("Request Method: %s", r.Method)
	log.Printf("Form values: %v", r.Form)

	selectedChannels := make(map[string]bool)
	if r.Form["channel"] != nil {
		for _, url := range r.Form["channel"] {
			selectedChannels[url] = true
		}
	}

	showShorts := r.Form.Get("show-shorts") == "true"
	log.Printf("Calculated showShorts boolean: %v", showShorts)

	channels, err := getChannelsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to load channels", http.StatusInternalServerError)
		return
	}

	var feedURLs []string
	for _, channel := range channels {
		if len(selectedChannels) == 0 || selectedChannels[channel.URL] {
			feedURLs = append(feedURLs, channel.URL)
		}
	}

	fp := gofeed.NewParser()
	var allItems []templates.VideoWithChannel
	for _, feedURL := range feedURLs {
		feed, err := fp.ParseURL(feedURL)
		if err == nil && feed != nil {
			for _, item := range feed.Items {
				videoID, err := extractVideoID(item.Link)
				if err == nil {
					uploadDate := item.PublishedParsed.Format("01/02/06")
					allItems = append(allItems, templates.VideoWithChannel{
						Item:        item,
						ChannelName: feed.Title,
						VideoID:     videoID,
						UploadDate:  uploadDate,
					})
				}
			}
		}
	}

	var filteredItems []templates.VideoWithChannel
	if !showShorts {
		log.Println("Filtering shorts...")
		for _, item := range allItems {
			if !strings.Contains(item.Item.Link, "/shorts/") {
				filteredItems = append(filteredItems, item)
			}
		}
	} else {
		log.Println("Not filtering shorts.")
		filteredItems = allItems
	}

	sort.Slice(filteredItems, func(i, j int) bool {
		return filteredItems[i].Item.PublishedParsed.After(*filteredItems[j].Item.PublishedParsed)
	})

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page == 0 {
		page = 1
	}

	perPage := 6
	start := (page - 1) * perPage
	end := start + perPage

	var nextPage int
	if end < len(filteredItems) {
		nextPage = page + 1
	}

	if start >= len(filteredItems) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if end > len(filteredItems) {
		end = len(filteredItems)
	}

	videosToShow := filteredItems[start:end]

	templates.Videos(videosToShow, nextPage).Render(r.Context(), w)
}

func extractVideoID(videoURL string) (string, error) {
	parsedURL, err := url.Parse(videoURL)
	if err != nil {
		return "", err
	}

	if parsedURL.Host == "youtu.be" {
		return strings.TrimPrefix(parsedURL.Path, "/"), nil
	}

	if strings.Contains(parsedURL.Path, "/shorts/") {
		parts := strings.Split(parsedURL.Path, "/")
		return parts[len(parts)-1], nil
	}

	videoID := parsedURL.Query().Get("v")
	if videoID == "" {
		return "", fmt.Errorf("could not find video ID in URL: %s", videoURL)
	}
	return videoID, nil
}