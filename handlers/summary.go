package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
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

	// Fetch transcript
	api := transcript.NewYouTubeTranscriptApi()
	trans, err := api.Fetch(videoID, []string{"en"}, false)
	if err != nil {
		fmt.Printf("Error fetching transcript for summary %s: %v\n", videoID, err)
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
		fmt.Printf("Error creating Gemini client: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error initializing AI client"))
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")
	prompt := `You are creating a summary of a Youtube video in the form of a research paper's abstract. The transcript below includes timestamps in [MM:SS] format.

Please provide a concise but comprehensive summary of this video transcript. Your summary should:

1. Include relevant timestamps as clickable links using this EXACT format: <a href="#" class="timestamp-link" data-time="SECONDS">MM:SS</a>

CRITICAL TIMESTAMP CONVERSION RULES:
- Convert [MM:SS] format to TOTAL SECONDS: minutes * 60 + seconds
- Examples:
  - [0:15] = 0*60 + 15 = 15 seconds → data-time="15"
  - [1:10] = 1*60 + 10 = 70 seconds → data-time="70"
  - [2:30] = 2*60 + 30 = 150 seconds → data-time="150"
  - [10:45] = 10*60 + 45 = 645 seconds → data-time="645"

2. Use HTML formatting with appropriate tags like <p>, <strong>, <em>, <br> for readability
3. Highlight key points and main topics discussed
4. Keep the summary engaging and suitable for an abstract

IMPORTANT: Always use the clickable link format for timestamps, never plain text timestamps. Double-check your timestamp calculations!

Transcript:
` + text

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		fmt.Printf("Error generating summary: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error generating summary"))
		return
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		w.Write([]byte("No summary generated"))
		return
	}

	summary := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`<div class="summary-title">Video Summary</div>
<div class="summary-content">%s</div>
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
