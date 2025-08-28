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
	prompt := `You are an expert academic research assistant, skilled in summarizing complex information into clear and concise abstracts. Your task is to create a summary of a YouTube video transcript.

## Summary Guidelines

Your summary should be structured like a research paper's abstract and adhere to the following guidelines:

* **Concise and Comprehensive**: Provide a summary that is both brief and covers the main points of the video.
* **No Introductory Phrases**: Do not begin with 'This video is about...' or 'In this video...'. Jump directly into the summary.
* **Formal Tone**: Maintain an academic and objective tone throughout the summary.
* **HTML Formatting**: Use HTML tags like <p>, <strong>, and <em> to structure the summary and emphasize key points. Do not use Markdown.
* **Clickable Timestamps**: Include relevant timestamps in the format: <a href="#" class="timestamp-link" data-time="SECONDS">MM:SS</a>.

## Timestamp Conversion Rules

You must convert all timestamps from the [MM:SS] or [HH:MM:SS] format in the transcript to total seconds for the 'data-time' attribute.

### [MM:SS] to Seconds
**Formula**: 'minutes * 60 + seconds'
* **[0:05]** = 0*60 + 5 = 5 seconds → 'data-time="5"'
* **[0:15]** = 0*60 + 15 = 15 seconds → 'data-time="15"'
* **[0:30]** = 0*60 + 30 = 30 seconds → 'data-time="30"'
* **[0:45]** = 0*60 + 45 = 45 seconds → 'data-time="45"'
* **[1:00]** = 1*60 + 0 = 60 seconds → 'data-time="60"'
* **[1:10]** = 1*60 + 10 = 70 seconds → 'data-time="70"'
* **[2:30]** = 2*60 + 30 = 150 seconds → 'data-time="150"'
* **[5:20]** = 5*60 + 20 = 320 seconds → 'data-time="320"'
* **[10:45]** = 10*60 + 45 = 645 seconds → 'data-time="645"'
* **[15:30]** = 15*60 + 30 = 930 seconds → 'data-time="930"'
* **[25:10]** = 25*60 + 10 = 1510 seconds → 'data-time="1510"'
* **[59:59]** = 59*60 + 59 = 3599 seconds → 'data-time="3599"'

### [HH:MM:SS] to Seconds
**Formula**: '(hours * 3600) + (minutes * 60) + seconds'
* **[0:00:30]** = (0*3600) + (0*60) + 30 = 30 seconds → 'data-time="30"'
* **[0:01:15]** = (0*3600) + (1*60) + 15 = 75 seconds → 'data-time="75"'
* **[0:05:30]** = (0*3600) + (5*60) + 30 = 330 seconds → 'data-time="330"'
* **[0:10:15]** = (0*3600) + (10*60) + 15 = 615 seconds → 'data-time="615"'
* **[1:00:00]** = (1*3600) + (0*60) + 0 = 3600 seconds → 'data-time="3600"'
* **[1:15:30]** = (1*3600) + (15*60) + 30 = 4530 seconds → 'data-time="4530"'
* **[2:30:45]** = (2*3600) + (30*60) + 45 = 9045 seconds → 'data-time="9045"'
* **[3:01:05]** = (3*3600) + (1*60) + 5 = 10865 seconds → 'data-time="10865"'

## Key Events

After the main summary, create a section with a heading <h3>Key Moments</h3> and list the most notable events or topics with their corresponding clickable timestamps.

**Example of a good summary structure:**

<p>This is a paragraph of the summary with <strong>key terms</strong> emphasized. It includes a clickable timestamp right here: <a href="#" class="timestamp-link" data-time="90">1:30</a>.</p>
<p>This is another paragraph that continues to explain the main topics of the video, with another important timestamp here: <a href="#" class="timestamp-link" data-time="320">5:20</a>.</p>
<h3>Key Moments</h3>
<ul>
    <li><a href="#" class="timestamp-link" data-time="90">1:30</a> - First major point is introduced.</li>
    <li><a href="#" class="timestamp-link" data-time="320">5:20</a> - A critical demonstration or example is shown.</li>
</ul>

## Transcript to Summarize

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
