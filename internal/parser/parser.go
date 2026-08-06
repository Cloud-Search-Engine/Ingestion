package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
)

var (
	mdHeadingRe = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	multiSpace  = regexp.MustCompile(`[ \t]+`)
	multiNL     = regexp.MustCompile(`\n{3,}`)
)

// Parse converts raw bytes into cleaned text plus heading metadata.
func Parse(body []byte, contentType, fallbackTitle string) (models.ParsedDocument, error) {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case strings.Contains(ct, "html"), looksLikeHTML(body):
		return parseHTML(body, fallbackTitle)
	case strings.Contains(ct, "markdown"), strings.HasSuffix(ct, "md"):
		return parseMarkdown(string(body), fallbackTitle), nil
	default:
		return parsePlain(string(body), fallbackTitle), nil
	}
}

func looksLikeHTML(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	lower := bytes.ToLower(trimmed)
	return bytes.HasPrefix(lower, []byte("<!doctype")) ||
		bytes.HasPrefix(lower, []byte("<html")) ||
		bytes.Contains(lower[:min(512, len(lower))], []byte("<body"))
}

func parseHTML(body []byte, fallbackTitle string) (models.ParsedDocument, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return models.ParsedDocument{}, fmt.Errorf("parse html: %w", err)
	}

	var title string
	var headings []models.Heading
	var text strings.Builder
	var skip bool

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "script", "style", "noscript", "svg", "nav", "footer", "header":
				prev := skip
				skip = true
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				skip = prev
				return
			case "title":
				title = strings.TrimSpace(textContent(n))
			case "h1", "h2", "h3", "h4", "h5", "h6":
				level := int(n.Data[1] - '0')
				htext := collapseWS(textContent(n))
				if htext != "" {
					headings = append(headings, models.Heading{
						Level:     level,
						Text:      htext,
						CharStart: text.Len(),
					})
					if text.Len() > 0 {
						text.WriteByte('\n')
					}
					text.WriteString(htext)
					text.WriteByte('\n')
				}
				return
			case "p", "div", "section", "article", "li", "tr", "br":
				if text.Len() > 0 && !strings.HasSuffix(text.String(), "\n") {
					text.WriteByte('\n')
				}
			}
		}

		if n.Type == html.TextNode && !skip {
			t := collapseWS(n.Data)
			if t != "" {
				if text.Len() > 0 && !strings.HasSuffix(text.String(), "\n") && !strings.HasSuffix(text.String(), " ") {
					text.WriteByte(' ')
				}
				text.WriteString(t)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	content := normalizeText(text.String())
	if title == "" {
		title = fallbackTitle
	}
	if title == "" && len(headings) > 0 {
		title = headings[0].Text
	}
	if content == "" {
		return models.ParsedDocument{}, fmt.Errorf("empty content after html parse")
	}
	return models.ParsedDocument{Title: title, Content: content, Headings: headings}, nil
}

func parseMarkdown(raw, fallbackTitle string) models.ParsedDocument {
	lines := strings.Split(raw, "\n")
	var headings []models.Heading
	var body strings.Builder
	title := fallbackTitle
	pos := 0

	for i, line := range lines {
		if m := mdHeadingRe.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			htext := strings.TrimSpace(m[2])
			headings = append(headings, models.Heading{
				Level:     level,
				Text:      htext,
				CharStart: pos,
			})
			if level == 1 && (title == "" || title == fallbackTitle) {
				title = htext
			}
			body.WriteString(htext)
			body.WriteByte('\n')
			pos += len(htext) + 1
			continue
		}
		// Strip simple markdown emphasis while keeping content.
		cleaned := stripMDInline(line)
		body.WriteString(cleaned)
		if i < len(lines)-1 {
			body.WriteByte('\n')
			pos += len(cleaned) + 1
		} else {
			pos += len(cleaned)
		}
	}

	content := normalizeText(body.String())
	if title == "" && len(headings) > 0 {
		title = headings[0].Text
	}
	return models.ParsedDocument{Title: title, Content: content, Headings: headings}
}

func parsePlain(raw, fallbackTitle string) models.ParsedDocument {
	content := normalizeText(raw)
	title := fallbackTitle
	if title == "" {
		// First non-empty line as title.
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				title = line
				if len(title) > 120 {
					title = title[:120]
				}
				break
			}
		}
	}
	return models.ParsedDocument{Title: title, Content: content}
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func stripMDInline(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "`", "")
	// [text](url) -> text
	for {
		start := strings.Index(s, "[")
		mid := strings.Index(s, "](")
		end := strings.Index(s, ")")
		if start < 0 || mid < 0 || end < 0 || mid < start || end < mid {
			break
		}
		s = s[:start] + s[start+1:mid] + s[end+1:]
	}
	return s
}

func collapseWS(s string) string {
	return strings.TrimSpace(multiSpace.ReplaceAllString(s, " "))
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = multiNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
