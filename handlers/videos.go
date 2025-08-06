package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"yt_rss2/templates"

	"github.com/mmcdole/gofeed"
)

func VideosHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(templates.User)
	r.ParseForm()
	log.Printf("--- New Request to VideosHandler ---")
	log.Printf("Request Method: %s", r.Method)
	log.Printf("Form values: %v", r.Form)

	// --- State Calculation ---
	selectedChannels := make(map[string]bool)
	if r.Form["channel"] != nil {
		for _, url := range r.Form["channel"] {
			selectedChannels[url] = true
		}
	}

	showShorts := r.Form.Get("show-shorts") == "true"
	log.Printf("Calculated showShorts boolean: %v", showShorts)

	// --- Data Fetching ---
	channels, err := getChannelsByUserID(user.ID)
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
	var videoIDs []string
	for _, feedURL := range feedURLs {
		feed, err := fp.ParseURL(feedURL)
		if err == nil && feed != nil {
			for _, item := range feed.Items {
				videoID, err := extractVideoID(item.Link)
				if err == nil {
					videoIDs = append(videoIDs, videoID)
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

	// --- Live Stream Detection (YouTube API) ---
	liveStatus, err := getLiveStatus(videoIDs)
	if err != nil {
		log.Printf("Error getting live status: %v", err)
		// Don't fail the whole request, just log the error.
	} else {
		for i := range allItems {
			if status, ok := liveStatus[allItems[i].VideoID]; ok && status {
				allItems[i].IsLive = true
			}
		}
	}

	// --- Filtering & Sorting ---
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

	// --- Pagination ---
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page == 0 {
		page = 1 // Default to page 1
	}

	perPage := 6
	start := (page - 1) * perPage
	end := start + perPage

	var nextPage int
	if end < len(filteredItems) {
		nextPage = page + 1
	}

	if start >= len(filteredItems) {
		w.WriteHeader(http.StatusOK) // No more content
		return
	}

	if end > len(filteredItems) {
		end = len(filteredItems)
	}

	videosToShow := filteredItems[start:end]

	// --- Rendering ---
	templates.Videos(videosToShow, nextPage).Render(r.Context(), w)
}

// extractVideoID parses a YouTube URL and returns the video ID.
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

// --- YouTube API Helper ---

type YouTubeResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			LiveBroadcastContent string `json:"liveBroadcastContent"`
		} `json:"snippet"`
	} `json:"items"`
}

func getLiveStatus(videoIDs []string) (map[string]bool, error) {
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("YOUTUBE_API_KEY not set")
	}

	if len(videoIDs) == 0 {
		return make(map[string]bool), nil
	}

	liveStatus := make(map[string]bool)
	
	// Chunk the video IDs into groups of 50.
	chunkSize := 50
	for i := 0; i < len(videoIDs); i += chunkSize {
		end := i + chunkSize
		if end > len(videoIDs) {
			end = len(videoIDs)
		}
		chunk := videoIDs[i:end]

		ids := strings.Join(chunk, ",")
		apiURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=snippet&id=%s&key=%s", ids, apiKey)
		log.Printf("Calling YouTube API: %s", apiURL)

		resp, err := http.Get(apiURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Printf("YouTube API Response: %s", string(body))

		var ytResp YouTubeResponse
		if err := json.Unmarshal(body, &ytResp); err != nil {
			return nil, err
		}

		for _, item := range ytResp.Items {
			if item.Snippet.LiveBroadcastContent == "live" {
				liveStatus[item.ID] = true
			}
		}
	}

	return liveStatus, nil
}
