package output

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ScrapedData struct {
	URL       string            `json:"url"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	Links     []string          `json:"links"`
	Images    []string          `json:"images"`
	Videos    []string          `json:"videos"`
	Documents []string          `json:"documents"`
	Emails    []string          `json:"emails"`
	Phones    []string          `json:"phones"`
	Metadata  map[string]string `json:"metadata"`
	LayerUsed string            `json:"layer_used"`
	CrawledAt time.Time         `json:"crawled_at"`
	Raw       string            `json:"raw,omitempty"`
	Cleaned   string            `json:"cleaned,omitempty"`
}

type rawFile struct {
	URL       string            `json:"url"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	Links     []string          `json:"links"`
	Images    []string          `json:"images"`
	Videos    []string          `json:"videos"`
	Documents []string          `json:"documents"`
	Emails    []string          `json:"emails"`
	Phones    []string          `json:"phones"`
	Metadata  map[string]string `json:"metadata"`
	LayerUsed string            `json:"layer_used"`
	CrawledAt time.Time         `json:"crawled_at"`
	Raw       string            `json:"raw,omitempty"`
}

type cleanedFile struct {
	URL       string            `json:"url"`
	Title     string            `json:"title"`
	CrawledAt time.Time         `json:"crawled_at"`
	Cleaned   string            `json:"cleaned"`
	Links     []string          `json:"links,omitempty"`
	Images    []string          `json:"images,omitempty"`
	Videos    []string          `json:"videos,omitempty"`
	Documents []string          `json:"documents,omitempty"`
	Emails    []string          `json:"emails,omitempty"`
	Phones    []string          `json:"phones,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Writer struct {
	outputDir string
}

func New(outputDir string) *Writer {
	return &Writer{outputDir: expandHome(outputDir)}
}

func (w *Writer) Save(data *ScrapedData) error {
	folder := w.buildFolderPath(data.URL, data.Title)

	if err := os.MkdirAll(folder, 0755); err != nil {
		return fmt.Errorf("could not create output folder: %w", err)
	}

	// Write raw.json
	raw := rawFile{
		URL:       data.URL,
		Title:     data.Title,
		Content:   data.Content,
		Links:     data.Links,
		Images:    data.Images,
		Videos:    data.Videos,
		Documents: data.Documents,
		Emails:    data.Emails,
		Phones:    data.Phones,
		Metadata:  data.Metadata,
		LayerUsed: data.LayerUsed,
		CrawledAt: data.CrawledAt,
		Raw:       data.Raw,
	}
	if err := writeJSON(filepath.Join(folder, "raw.json"), raw); err != nil {
		return err
	}

	// Write cleaned.json only if cleaned content exists
	if data.Cleaned != "" {
		cleaned := cleanedFile{
			URL:       data.URL,
			Title:     data.Title,
			CrawledAt: data.CrawledAt,
			Cleaned:   data.Cleaned,
			Links:     data.Links,
			Images:    data.Images,
			Videos:    data.Videos,
			Documents: data.Documents,
			Emails:    data.Emails,
			Phones:    data.Phones,
			Metadata:  data.Metadata,
		}
		if err := writeJSON(filepath.Join(folder, "cleaned.json"), cleaned); err != nil {
			return err
		}
	}

	return nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("could not write %s: %w", filepath.Base(path), err)
	}
	return nil
}

func (w *Writer) buildFolderPath(rawURL, title string) string {
	domain := extractDomain(rawURL)
	slug := slugify(title)
	if slug == "" {
		slug = slugify(rawURL)
	}
	return filepath.Join(w.outputDir, domain, slug)
}

// URLToPath returns the raw.json path for a given URL
func URLToPath(outputDir string, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	domain := u.Hostname()
	slug := slugify(u.Path + " " + u.Fragment)
	if slug == "" {
		slug = domain
	}

	dir := filepath.Join(expandHome(outputDir), domain, slug)
	return filepath.Join(dir, "raw.json")
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func extractDomain(rawURL string) string {
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	rawURL = strings.TrimPrefix(rawURL, "www.")
	parts := strings.Split(rawURL, "/")
	return parts[0]
}

func slugify(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}
