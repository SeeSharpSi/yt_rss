package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"yt_rss2/database"
	"yt_rss2/pkg/transcript"

	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

func VideoSummaryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	// Validate video ID
	youtubeVideoRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)
	if !youtubeVideoRegex.MatchString(videoID) {
		http.Error(w, "Invalid video ID", http.StatusBadRequest)
		return
	}

	// Check if this is a refresh request
	isRefresh := r.URL.Query().Get("refresh") == "1"

	// Check for existing summary first (unless it's a refresh request)
	if !isRefresh {
		existingSummary, err := database.GetSummary(videoID)
		if err == nil {
			// Summary exists, return it
			w.Header().Set("Content-Type", "text/html")
			html := fmt.Sprintf(`<div class="summary-content">%s</div>
<script>
document.addEventListener('click', function(e) {
	if (e.target.classList.contains('timestamp-link')) {
		e.preventDefault();
		const time = e.target.dataset.time;
		if (window.player && typeof window.player.seekTo === 'function') {
			window.player.seekTo(parseFloat(time), true);
			console.log('Seeking to timestamp:', time);
		} else {
			console.log('YouTube player not available');
		}
	}
});
</script>`, existingSummary)
			w.Write([]byte(html))
			return
		}
	}

	// No existing summary, generate new one
	// Fetch transcript
	api := transcript.NewYouTubeTranscriptApi()
	trans, err := api.Fetch(videoID, []string{"en"}, false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching transcript"))
		return
	}

	formatter := &transcript.TextFormatter{}
	text := formatter.FormatTranscript(trans, nil)

	if len(text) == 0 {
		w.Write([]byte("No transcript available for summary"))
		return
	}

	// Get Gemini summary
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error initializing AI client"))
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash-lite")
	prompt := `You are creating a summary of a Youtube video in the form of a research paper's abstract. The transcript below includes timestamps in [MM:SS] format.

Please provide a concise but comprehensive summary of this video transcript, without mentioning that it's an abstract. Your summary should:

1. Include relevant timestamps as clickable links using this EXACT format: <a href="#" class="timestamp-link" data-time="SECONDS">MM:SS</a>

	CRITICAL TIMESTAMP CONVERSION RULES:
	- Convert [MM:SS] format to TOTAL SECONDS: minutes * 60 + seconds
	- Convert [HH:MM:SS] format to TOTAL SECONDS: (hours * 3600) + (minutes * 60) + seconds
	- Examples for [MM:SS] format:
	  - [0:05] = 0*60 + 5 = 5 seconds → data-time="5"
	  - [0:15] = 0*60 + 15 = 15 seconds → data-time="15"
	  - [0:30] = 0*60 + 30 = 30 seconds → data-time="30"
	  - [0:45] = 0*60 + 45 = 45 seconds → data-time="45"
	  - [1:00] = 1*60 + 0 = 60 seconds → data-time="60"
	  - [1:10] = 1*60 + 10 = 70 seconds → data-time="70"
	  - [1:30] = 1*60 + 30 = 90 seconds → data-time="90"
	  - [2:15] = 2*60 + 15 = 135 seconds → data-time="135"
	  - [2:30] = 2*60 + 30 = 150 seconds → data-time="150"
	  - [5:20] = 5*60 + 20 = 320 seconds → data-time="320"
	  - [10:45] = 10*60 + 45 = 645 seconds → data-time="645"
	  - [15:30] = 15*60 + 30 = 930 seconds → data-time="930"
	- Examples for [HH:MM:SS] format:
	  - [0:00:30] = (0*3600) + (0*60) + 30 = 30 seconds → data-time="30"
	  - [0:01:15] = (0*3600) + (1*60) + 15 = 75 seconds → data-time="75"
	  - [0:02:45] = (0*3600) + (2*60) + 45 = 165 seconds → data-time="165"
	  - [0:05:30] = (0*3600) + (5*60) + 30 = 330 seconds → data-time="330"
	  - [0:10:15] = (0*3600) + (10*60) + 15 = 615 seconds → data-time="615"
	  - [1:00:00] = (1*3600) + (0*60) + 0 = 3600 seconds → data-time="3600"
	  - [1:15:30] = (1*3600) + (15*60) + 30 = 4530 seconds → data-time="4530"
	  - [2:30:45] = (2*3600) + (30*60) + 45 = 9045 seconds → data-time="9045"

2. Use HTML formatting with appropriate tags like <p>, <strong>, <em>, <br> for readability
3. Highlight key points and main topics discussed
4. Keep the summary engaging and suitable for an abstract
5. At the end of the summary, include a list of notable events with their corresponding time stamps

** CONSTRAINTS ** 
1. You MUST NOT use crude language or profanity 
2. You MUST NOT use Markdown to format text 
3. You MUST use HTML to format text

IMPORTANT: Always use the clickable link format for timestamps, never plain text timestamps. Double-check your timestamp calculations!

Transcript:
` + text

	print("starting...")
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	print("done")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error generating summary"))
		print("whoops")
		return
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		w.Write([]byte("No summary generated"))
		print("whoops2")
		return
	}

	summary := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	// Save the summary to database
	fmt.Printf("Saving summary for video %s (length: %d)\n", videoID, len(summary))
	err = database.SaveSummary(videoID, summary)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Error saving summary to database: %v\n", err)
	} else {
		fmt.Printf("Successfully saved summary for video %s\n", videoID)
	}

	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`<div class="summary-content">%s</div>
<script>
document.addEventListener('click', function(e) {
	if (e.target.classList.contains('timestamp-link')) {
		e.preventDefault();
		const time = e.target.dataset.time;
		if (window.player && typeof window.player.seekTo === 'function') {
			window.player.seekTo(parseFloat(time), true);
			console.log('Seeking to timestamp:', time);
		} else {
			console.log('YouTube player not available');
		}
	}
});
</script>`, summary)
	w.Write([]byte(html))
}
