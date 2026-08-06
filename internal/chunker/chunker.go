package chunker

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/tokenizer"
)

// Options controls chunk sizing.
type Options struct {
	SizeTokens int // target size: 256, 512, or 1024
	Overlap    int // overlapping tokens between consecutive chunks
}

// Chunk splits parsed content into overlapping chunks with heading/section metadata.
func Chunk(docID string, meta models.IngestMessage, parsed models.ParsedDocument, opts Options) []models.Chunk {
	size := opts.SizeTokens
	switch size {
	case 256, 512, 1024:
	default:
		size = 512
	}
	overlap := opts.Overlap
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 8
	}

	title := parsed.Title
	if title == "" {
		title = meta.Title
	}

	sections := splitByHeadings(parsed)
	var chunks []models.Chunk
	pos := 0

	for _, sec := range sections {
		parts := splitTokens(sec.Content, size, overlap)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			tc := tokenizer.CountTokens(part)
			chunkID := fmt.Sprintf("%s#c%04d", docID, pos)
			chunks = append(chunks, models.Chunk{
				ChunkID:    chunkID,
				DocumentID: docID,
				Provider:   meta.Provider,
				Service:    meta.Service,
				Category:   meta.Category,
				Title:      title,
				Heading:    sec.Heading,
				Section:    sec.Section,
				SourceURL:  meta.SourceURL,
				Content:    part,
				TokenCount: tc,
				Position:   pos,
			})
			pos++
		}
	}

	if len(chunks) == 0 && strings.TrimSpace(parsed.Content) != "" {
		chunks = append(chunks, models.Chunk{
			ChunkID:    fmt.Sprintf("%s#c%04d", docID, 0),
			DocumentID: docID,
			Provider:   meta.Provider,
			Service:    meta.Service,
			Category:   meta.Category,
			Title:      title,
			SourceURL:  meta.SourceURL,
			Content:    strings.TrimSpace(parsed.Content),
			TokenCount: tokenizer.CountTokens(parsed.Content),
			Position:   0,
		})
	}
	return chunks
}

type section struct {
	Heading string
	Section string
	Content string
}

func splitByHeadings(parsed models.ParsedDocument) []section {
	content := parsed.Content
	if len(parsed.Headings) == 0 {
		return []section{{Content: content}}
	}

	// Rebuild sections by walking heading markers already inlined as text lines.
	lines := strings.Split(content, "\n")
	headingSet := map[string]models.Heading{}
	for _, h := range parsed.Headings {
		headingSet[strings.ToLower(strings.TrimSpace(h.Text))] = h
	}

	var sections []section
	var cur section
	var buf strings.Builder
	sectionPath := []string{}

	flush := func() {
		text := strings.TrimSpace(buf.String())
		if text == "" && cur.Heading == "" {
			buf.Reset()
			return
		}
		cur.Content = text
		sections = append(sections, cur)
		buf.Reset()
	}

	for _, line := range lines {
		key := strings.ToLower(strings.TrimSpace(line))
		if h, ok := headingSet[key]; ok && strings.TrimSpace(line) != "" {
			flush()
			// Maintain breadcrumb path by heading level.
			for len(sectionPath) >= h.Level && len(sectionPath) > 0 {
				sectionPath = sectionPath[:len(sectionPath)-1]
			}
			sectionPath = append(sectionPath, h.Text)
			cur = section{
				Heading: h.Text,
				Section: strings.Join(sectionPath, " > "),
			}
			buf.WriteString(line)
			buf.WriteByte('\n')
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	flush()

	if len(sections) == 0 {
		return []section{{Content: content}}
	}
	return sections
}

func splitTokens(text string, size, overlap int) []string {
	tokens := tokenizeKeepWS(text)
	if len(tokens) == 0 {
		return nil
	}
	if len(tokens) <= size {
		return []string{joinTokens(tokens)}
	}

	var parts []string
	step := size - overlap
	if step < 1 {
		step = 1
	}
	for start := 0; start < len(tokens); start += step {
		end := start + size
		if end > len(tokens) {
			end = len(tokens)
		}
		parts = append(parts, joinTokens(tokens[start:end]))
		if end == len(tokens) {
			break
		}
	}
	return parts
}

// tokenizeKeepWS splits on whitespace but keeps each word as a token unit.
func tokenizeKeepWS(text string) []string {
	var tokens []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		tokens = append(tokens, cur.String())
		cur.Reset()
	}
	for _, r := range text {
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		cur.WriteRune(r)
	}
	flush()
	return tokens
}

func joinTokens(tokens []string) string {
	return strings.Join(tokens, " ")
}
