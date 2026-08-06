package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
)

// FetchedDoc is raw content plus inferred metadata from a crawl or seed read.
type FetchedDoc struct {
	URL         string
	LocalPath   string
	Body        []byte
	ContentType string
	Meta        models.SeedMeta
}

// Crawler fetches remote URLs or local seed files with rate limiting.
type Crawler struct {
	client  *http.Client
	limiter *rate.Limiter
}

// New creates a crawler that allows roughly rps requests per second.
func New(rps float64) *Crawler {
	if rps <= 0 {
		rps = 2
	}
	return &Crawler{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		limiter: rate.NewLimiter(rate.Limit(rps), 1),
	}
}

// FetchURL downloads a remote document with rate limiting.
func (c *Crawler) FetchURL(ctx context.Context, url string) (*FetchedDoc, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "cloud-search-engine-ingestion/1.0")
	req.Header.Set("Accept", "text/html, text/markdown, text/plain;q=0.9, */*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = guessContentType(url, body)
	} else {
		ct = strings.TrimSpace(strings.Split(ct, ";")[0])
	}

	return &FetchedDoc{
		URL:         url,
		Body:        body,
		ContentType: ct,
		Meta: models.SeedMeta{
			SourceURL: url,
			Title:     filepath.Base(url),
		},
	}, nil
}

// ReadSeedFile reads a local markdown/text seed file and optional sidecar JSON metadata.
func (c *Crawler) ReadSeedFile(path string) (*FetchedDoc, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read seed %s: %w", path, err)
	}

	meta := models.SeedMeta{}
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".json"
	if raw, err := os.ReadFile(metaPath); err == nil {
		if err := decodeSeedMeta(raw, &meta); err != nil {
			return nil, fmt.Errorf("parse seed meta %s: %w", metaPath, err)
		}
	}

	if meta.Title == "" {
		meta.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if meta.SourceURL == "" {
		meta.SourceURL = "file://" + filepath.ToSlash(path)
	}
	if meta.DocumentID == "" {
		meta.DocumentID = slugify(filepath.Base(path))
	}
	if meta.DocumentType == "" {
		meta.DocumentType = "developer_documentation"
	}
	if meta.Version == "" {
		meta.Version = "current"
	}

	return &FetchedDoc{
		LocalPath:   path,
		URL:         meta.SourceURL,
		Body:        body,
		ContentType: guessContentType(path, body),
		Meta:        meta,
	}, nil
}

// LoadSeedDir reads all markdown/text files from a directory.
func (c *Crawler) LoadSeedDir(dir string) ([]*FetchedDoc, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read seed dir: %w", err)
	}

	var docs []*FetchedDoc
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".md", ".markdown", ".txt", ".html", ".htm":
		default:
			continue
		}
		doc, err := c.ReadSeedFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func guessContentType(pathOrURL string, body []byte) string {
	lower := strings.ToLower(pathOrURL)
	switch {
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".markdown"):
		return "text/markdown"
	case strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
		return "text/html"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	}

	trimmed := strings.TrimSpace(string(body))
	if strings.HasPrefix(trimmed, "<!") || strings.HasPrefix(strings.ToLower(trimmed), "<html") {
		return "text/html"
	}
	if strings.Contains(trimmed, "# ") || strings.Contains(trimmed, "## ") {
		return "text/markdown"
	}
	return "text/plain"
}

func slugify(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.ToLower(base)
	var b strings.Builder
	prevDash := false
	for _, r := range base {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
