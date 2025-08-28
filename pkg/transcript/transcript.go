package transcript

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// FetchedTranscriptSnippet represents a single snippet of transcript text with timing
type FetchedTranscriptSnippet struct {
	Text     string  `json:"text"`
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
}

// FetchedTranscript represents a complete fetched transcript
type FetchedTranscript struct {
	Snippets     []FetchedTranscriptSnippet `json:"snippets"`
	VideoID      string                     `json:"video_id"`
	Language     string                     `json:"language"`
	LanguageCode string                     `json:"language_code"`
	IsGenerated  bool                       `json:"is_generated"`
}

// ToRawData returns the transcript as a slice of maps
func (ft *FetchedTranscript) ToRawData() []map[string]interface{} {
	data := make([]map[string]interface{}, len(ft.Snippets))
	for i, snippet := range ft.Snippets {
		data[i] = map[string]interface{}{
			"text":     snippet.Text,
			"start":    snippet.Start,
			"duration": snippet.Duration,
		}
	}
	return data
}

// Transcript represents a transcript that can be fetched
type Transcript struct {
	httpClient           *http.Client
	videoID              string
	url                  string
	language             string
	languageCode         string
	isGenerated          bool
	translationLanguages []TranslationLanguage
}

// TranslationLanguage represents a language that the transcript can be translated to
type TranslationLanguage struct {
	Language     string `json:"language"`
	LanguageCode string `json:"language_code"`
}

// Fetch retrieves the actual transcript data
func (t *Transcript) Fetch(preserveFormatting bool) (*FetchedTranscript, error) {
	req, err := http.NewRequest("GET", t.url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "en-US")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch transcript: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parser := &TranscriptParser{PreserveFormatting: preserveFormatting}
	snippets, err := parser.Parse(string(data))
	if err != nil {
		return nil, err
	}

	return &FetchedTranscript{
		Snippets:     snippets,
		VideoID:      t.videoID,
		Language:     t.language,
		LanguageCode: t.languageCode,
		IsGenerated:  t.isGenerated,
	}, nil
}

// TranscriptList represents a list of available transcripts for a video
type TranscriptList struct {
	VideoID                    string
	ManuallyCreatedTranscripts map[string]*Transcript
	GeneratedTranscripts       map[string]*Transcript
	TranslationLanguages       []TranslationLanguage
}

// FindTranscript finds a transcript for the given language codes
func (tl *TranscriptList) FindTranscript(languageCodes []string) (*Transcript, error) {
	return tl.findTranscript(languageCodes, []map[string]*Transcript{
		tl.ManuallyCreatedTranscripts,
		tl.GeneratedTranscripts,
	})
}

// FindGeneratedTranscript finds a generated transcript for the given language codes
func (tl *TranscriptList) FindGeneratedTranscript(languageCodes []string) (*Transcript, error) {
	return tl.findTranscript(languageCodes, []map[string]*Transcript{
		tl.GeneratedTranscripts,
	})
}

// FindManuallyCreatedTranscript finds a manually created transcript for the given language codes
func (tl *TranscriptList) FindManuallyCreatedTranscript(languageCodes []string) (*Transcript, error) {
	return tl.findTranscript(languageCodes, []map[string]*Transcript{
		tl.ManuallyCreatedTranscripts,
	})
}

func (tl *TranscriptList) findTranscript(languageCodes []string, transcriptDicts []map[string]*Transcript) (*Transcript, error) {
	for _, languageCode := range languageCodes {
		for _, transcriptDict := range transcriptDicts {
			if transcript, exists := transcriptDict[languageCode]; exists {
				return transcript, nil
			}
		}
	}
	return nil, &NoTranscriptFoundError{VideoID: tl.VideoID, RequestedLanguageCodes: languageCodes}
}

// TranscriptParser parses raw transcript data
type TranscriptParser struct {
	PreserveFormatting bool
}

// Parse parses the raw transcript XML data
func (tp *TranscriptParser) Parse(rawData string) ([]FetchedTranscriptSnippet, error) {
	var snippets []FetchedTranscriptSnippet

	// Parse XML
	type Transcript struct {
		Texts []struct {
			Text  string `xml:",innerxml"`
			Start string `xml:"start,attr"`
			Dur   string `xml:"dur,attr"`
		} `xml:"text"`
	}

	var transcript Transcript
	err := xml.Unmarshal([]byte(rawData), &transcript)
	if err != nil {
		return nil, err
	}

	htmlRegex := tp.getHTMLRegex()

	for _, p := range transcript.Texts {
		text := p.Text
		if !tp.PreserveFormatting {
			text = htmlRegex.ReplaceAllString(text, "")
		}
		text = html.UnescapeString(text)

		start := 0.0
		duration := 0.0
		if p.Start != "" {
			fmt.Sscanf(p.Start, "%f", &start)
		}
		if p.Dur != "" {
			fmt.Sscanf(p.Dur, "%f", &duration)
		}

		snippets = append(snippets, FetchedTranscriptSnippet{
			Text:     text,
			Start:    start,
			Duration: duration,
		})
	}

	return snippets, nil
}

func (tp *TranscriptParser) getHTMLRegex() *regexp.Regexp {
	if tp.PreserveFormatting {
		// Keep some formatting tags
		formattingTags := []string{"strong", "em", "b", "i", "mark", "small", "del", "ins", "sub", "sup"}
		pattern := `<\/?(?!\/?(` + strings.Join(formattingTags, "|") + `)\b).*?\b>`
		return regexp.MustCompile(pattern)
	}
	return regexp.MustCompile(`<[^>]*>`)
}

// YouTubeTranscriptApi is the main API class
type YouTubeTranscriptApi struct {
	httpClient *http.Client
}

// NewYouTubeTranscriptApi creates a new instance
func NewYouTubeTranscriptApi() *YouTubeTranscriptApi {
	return &YouTubeTranscriptApi{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Fetch retrieves the transcript for a single video
func (api *YouTubeTranscriptApi) Fetch(videoID string, languages []string, preserveFormatting bool) (*FetchedTranscript, error) {
	transcriptList, err := api.List(videoID)
	if err != nil {
		return nil, err
	}

	transcript, err := transcriptList.FindTranscript(languages)
	if err != nil {
		return nil, err
	}

	return transcript.Fetch(preserveFormatting)
}

// List retrieves the list of available transcripts for a video
func (api *YouTubeTranscriptApi) List(videoID string) (*TranscriptList, error) {
	fetcher := &TranscriptListFetcher{httpClient: api.httpClient}
	return fetcher.Fetch(videoID)
}

// TranscriptListFetcher fetches transcript lists
type TranscriptListFetcher struct {
	httpClient *http.Client
}

// Fetch fetches the transcript list for a video
func (tlf *TranscriptListFetcher) Fetch(videoID string) (*TranscriptList, error) {
	captionsJSON, err := tlf.fetchCaptionsJSON(videoID)
	if err != nil {
		return nil, err
	}

	return tlf.build(videoID, captionsJSON)
}

// build creates a TranscriptList from captions JSON
func (tlf *TranscriptListFetcher) build(videoID string, captionsJSON map[string]interface{}) (*TranscriptList, error) {
	translationLanguages := []TranslationLanguage{}
	if tlData, ok := captionsJSON["translationLanguages"].([]interface{}); ok {
		for _, item := range tlData {
			if langData, ok := item.(map[string]interface{}); ok {
				if name, ok := langData["languageName"].(map[string]interface{}); ok {
					if runs, ok := name["runs"].([]interface{}); ok && len(runs) > 0 {
						if run, ok := runs[0].(map[string]interface{}); ok {
							if text, ok := run["text"].(string); ok {
								if code, ok := langData["languageCode"].(string); ok {
									translationLanguages = append(translationLanguages, TranslationLanguage{
										Language:     text,
										LanguageCode: code,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	manuallyCreatedTranscripts := make(map[string]*Transcript)
	generatedTranscripts := make(map[string]*Transcript)

	if captionTracks, ok := captionsJSON["captionTracks"].([]interface{}); ok {
		for _, track := range captionTracks {
			if trackData, ok := track.(map[string]interface{}); ok {
				baseURL, _ := trackData["baseUrl"].(string)
				baseURL = strings.Replace(baseURL, "&fmt=srv3", "", 1)

				name, _ := trackData["name"].(map[string]interface{})
				var language, languageCode string
				if name != nil {
					if runs, ok := name["runs"].([]interface{}); ok && len(runs) > 0 {
						if run, ok := runs[0].(map[string]interface{}); ok {
							language, _ = run["text"].(string)
						}
					}
				}
				languageCode, _ = trackData["languageCode"].(string)

				isGenerated := false
				if kind, ok := trackData["kind"].(string); ok && kind == "asr" {
					isGenerated = true
				}

				var transLangs []TranslationLanguage
				if isTranslatable, ok := trackData["isTranslatable"].(bool); ok && isTranslatable {
					transLangs = translationLanguages
				}

				transcript := &Transcript{
					httpClient:           tlf.httpClient,
					videoID:              videoID,
					url:                  baseURL,
					language:             language,
					languageCode:         languageCode,
					isGenerated:          isGenerated,
					translationLanguages: transLangs,
				}

				if isGenerated {
					generatedTranscripts[languageCode] = transcript
				} else {
					manuallyCreatedTranscripts[languageCode] = transcript
				}
			}
		}
	}

	return &TranscriptList{
		VideoID:                    videoID,
		ManuallyCreatedTranscripts: manuallyCreatedTranscripts,
		GeneratedTranscripts:       generatedTranscripts,
		TranslationLanguages:       translationLanguages,
	}, nil
}

// fetchCaptionsJSON fetches the captions JSON data from YouTube
func (tlf *TranscriptListFetcher) fetchCaptionsJSON(videoID string) (map[string]interface{}, error) {
	html := tlf.fetchVideoHTML(videoID)
	apiKey := tlf.extractInnertubeAPIKey(html, videoID)
	innertubeData := tlf.fetchInnertubeData(videoID, apiKey)
	return tlf.extractCaptionsJSON(innertubeData, videoID)
}

func (tlf *TranscriptListFetcher) extractInnertubeAPIKey(html, videoID string) string {
	re := regexp.MustCompile(`"INNERTUBE_API_KEY":\s*"([a-zA-Z0-9_-]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return matches[1]
	}
	panic(fmt.Sprintf("Could not extract INNERTUBE_API_KEY for video %s", videoID))
}

func (tlf *TranscriptListFetcher) extractCaptionsJSON(innertubeData map[string]interface{}, videoID string) (map[string]interface{}, error) {
	if captions, ok := innertubeData["captions"].(map[string]interface{}); ok {
		if playerCaptions, ok := captions["playerCaptionsTracklistRenderer"].(map[string]interface{}); ok {
			if _, hasCaptionTracks := playerCaptions["captionTracks"]; hasCaptionTracks {
				return playerCaptions, nil
			}
		}
	}
	return nil, &TranscriptsDisabledError{VideoID: videoID}
}

func (tlf *TranscriptListFetcher) fetchVideoHTML(videoID string) string {
	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := tlf.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return html.UnescapeString(string(data))
}

func (tlf *TranscriptListFetcher) fetchInnertubeData(videoID, apiKey string) map[string]interface{} {
	url := fmt.Sprintf("https://www.youtube.com/youtubei/v1/player?key=%s", apiKey)
	payload := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"clientName":    "ANDROID",
				"clientVersion": "20.10.38",
			},
		},
		"videoId": videoID,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := tlf.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	return data
}

// Error types
type YouTubeTranscriptApiError struct {
	VideoID string
	Message string
}

func (e *YouTubeTranscriptApiError) Error() string {
	return fmt.Sprintf("Error for video %s: %s", e.VideoID, e.Message)
}

type NoTranscriptFoundError struct {
	VideoID                string
	RequestedLanguageCodes []string
}

func (e *NoTranscriptFoundError) Error() string {
	return fmt.Sprintf("No transcript found for video %s with languages %v", e.VideoID, e.RequestedLanguageCodes)
}

type TranscriptsDisabledError struct {
	VideoID string
}

func (e *TranscriptsDisabledError) Error() string {
	return fmt.Sprintf("Transcripts are disabled for video %s", e.VideoID)
}

// Formatter interface for formatting transcripts
type Formatter interface {
	FormatTranscript(transcript *FetchedTranscript, kwargs map[string]interface{}) string
	FormatTranscripts(transcripts []*FetchedTranscript, kwargs map[string]interface{}) string
}

// JSONFormatter formats transcripts as JSON
type JSONFormatter struct{}

// FormatTranscript formats a single transcript as JSON
func (f *JSONFormatter) FormatTranscript(transcript *FetchedTranscript, kwargs map[string]interface{}) string {
	data := transcript.ToRawData()
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// FormatTranscripts formats multiple transcripts as JSON
func (f *JSONFormatter) FormatTranscripts(transcripts []*FetchedTranscript, kwargs map[string]interface{}) string {
	allData := make([][]map[string]interface{}, len(transcripts))
	for i, transcript := range transcripts {
		allData[i] = transcript.ToRawData()
	}
	jsonData, _ := json.Marshal(allData)
	return string(jsonData)
}

// TextFormatter formats transcripts as plain text
type TextFormatter struct{}

// FormatTranscript formats a single transcript as plain text
func (f *TextFormatter) FormatTranscript(transcript *FetchedTranscript, kwargs map[string]interface{}) string {
	var text strings.Builder
	for _, snippet := range transcript.Snippets {
		// Format timestamp as MM:SS or HH:MM:SS
		timestamp := f.formatTimestamp(snippet.Start)
		text.WriteString("[")
		text.WriteString(timestamp)
		text.WriteString("] ")
		text.WriteString(snippet.Text)
		text.WriteString(" ")
	}
	return strings.TrimSpace(text.String())
}

// formatTimestamp converts seconds to MM:SS or HH:MM:SS format
func (f *TextFormatter) formatTimestamp(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours*3600)) / 60)
	secs := int(seconds - float64(hours*3600) - float64(minutes*60))

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

// FormatTranscripts formats multiple transcripts as plain text
func (f *TextFormatter) FormatTranscripts(transcripts []*FetchedTranscript, kwargs map[string]interface{}) string {
	var text strings.Builder
	for _, transcript := range transcripts {
		text.WriteString(f.FormatTranscript(transcript, kwargs))
		text.WriteString("\n\n")
	}
	return strings.TrimSpace(text.String())
}

// WebVTTFormatter formats transcripts as WebVTT
type WebVTTFormatter struct{}

// FormatTranscript formats a single transcript as WebVTT
func (f *WebVTTFormatter) FormatTranscript(transcript *FetchedTranscript, kwargs map[string]interface{}) string {
	var vtt strings.Builder
	vtt.WriteString("WEBVTT\n\n")
	for _, snippet := range transcript.Snippets {
		start := f.formatTime(snippet.Start)
		end := f.formatTime(snippet.Start + snippet.Duration)
		vtt.WriteString(fmt.Sprintf("%s --> %s\n%s\n\n", start, end, snippet.Text))
	}
	return vtt.String()
}

// FormatTranscripts formats multiple transcripts as WebVTT
func (f *WebVTTFormatter) FormatTranscripts(transcripts []*FetchedTranscript, kwargs map[string]interface{}) string {
	// For multiple transcripts, combine them
	var allSnippets []FetchedTranscriptSnippet
	for _, transcript := range transcripts {
		allSnippets = append(allSnippets, transcript.Snippets...)
	}
	combined := &FetchedTranscript{
		Snippets:     allSnippets,
		VideoID:      transcripts[0].VideoID,
		Language:     transcripts[0].Language,
		LanguageCode: transcripts[0].LanguageCode,
		IsGenerated:  transcripts[0].IsGenerated,
	}
	return f.FormatTranscript(combined, kwargs)
}

func (f *WebVTTFormatter) formatTime(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours*3600)) / 60)
	secs := seconds - float64(hours*3600) - float64(minutes*60)
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, secs)
}

// CLI function for command-line usage
func RunCLI(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: go run main.go <video_id> [options]")
		fmt.Println("Options:")
		fmt.Println("  --languages <lang1,lang2>    Specify language codes (default: en)")
		fmt.Println("  --format <json|text|vtt>    Output format (default: text)")
		fmt.Println("  --list-transcripts          List available transcripts")
		os.Exit(1)
	}

	videoID := args[1]
	languages := []string{"en"}
	format := "text"
	listTranscripts := false

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--languages":
			if i+1 < len(args) {
				languages = strings.Split(args[i+1], ",")
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--list-transcripts":
			listTranscripts = true
		}
	}

	api := NewYouTubeTranscriptApi()

	if listTranscripts {
		transcriptList, err := api.List(videoID)
		if err != nil {
			os.Exit(1)
		}
		fmt.Println(transcriptList)
		return
	}

	transcript, err := api.Fetch(videoID, languages, false)
	if err != nil {
		os.Exit(1)
	}

	var formatter Formatter
	switch format {
	case "json":
		formatter = &JSONFormatter{}
	case "vtt":
		formatter = &WebVTTFormatter{}
	default:
		formatter = &TextFormatter{}
	}

	output := formatter.FormatTranscript(transcript, nil)
	fmt.Println(output)
}
