package parser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"document-rag/internal/config"
	"document-rag/internal/models"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/tealeg/xlsx"
	"github.com/xuri/excelize/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// // Chunk represents a parsed chunk with metadata
// type Chunk struct {
// 	Content    string
// 	PageNumber *int
// }

type Parser interface {
	ParseToMarkdown(filePath string) ([]models.Chunk, error)
}

type ParserConfig struct {
	Config *config.Config
}

const (
	defaultChunkSize    = 1000 // bytes
	defaultChunkOverlap = 500  // bytes
	defaultPageNumber   = 1
)

func ParseToMarkdown(filePath string, cfg *config.Config) ([]models.Chunk, error) {

	// if config is nil, use default values
	if cfg == nil {
		cfg = &config.Config{
			RAG: config.RAGConfig{
				ChunkSize:    defaultChunkSize,
				ChunkOverlap: defaultChunkOverlap,
			},
		}
	} else if cfg.RAG.ChunkSize == 0 || cfg.RAG.ChunkOverlap == 0 {
		cfg.RAG.ChunkSize = defaultChunkSize
		cfg.RAG.ChunkOverlap = defaultChunkOverlap
	}

	p := ParserConfig{
		Config: cfg,
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return p.parsePDF(filePath)
	case ".docx":
		return parseDOCX(filePath)
	case ".pptx":
		return parsePPTX(filePath)
	case ".xlsx":
		return parseXLSX(filePath)
	case ".ods":
		return parseODS(filePath)
	case ".txt":
		return parseText(filePath)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

func (p *ParserConfig) parsePDF(filePath string) ([]models.Chunk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Get file size for reader initialization
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	reader, err := pdf.NewReader(f, stat.Size())
	if err != nil {
		return nil, err
	}

	var chunks []models.Chunk
	numPages := reader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			return nil, err
		}
		pageNum := i
		markdown, err := convertToMarkdown(pageText)
		if err != nil {
			return nil, err
		}

		chunks = append(chunks, p.getChunks(markdown, pageNum)...)
	}
	return chunks, nil
}

func parseDOCX(filePath string) ([]models.Chunk, error) {
	r, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	doc := r.Editable()
	content := doc.GetContent()
	paragraphs := strings.Split(content, "\n")
	var chunks []models.Chunk
	for _, p := range paragraphs {
		if p == "" {
			continue
		}
		chunk := models.Chunk{
			Content:    p,
			PageNumber: defaultPageNumber, // DOCX has no page numbers
		}
		markdown, err := convertToMarkdown(chunk.Content)
		if err != nil {
			return nil, err
		}
		chunk.Content = markdown
		if strings.TrimSpace(chunk.Content) != "" {
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}

func parsePPTX(filePath string) ([]models.Chunk, error) {
	f, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var chunks []models.Chunk
	for slideNum, file := range f.File {
		if strings.HasPrefix(file.Name, "ppt/slides/slide") {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			slideText := extractTextFromXML(string(data))
			chunk := models.Chunk{
				Content:    slideText,
				PageNumber: defaultPageNumber, // Treat slides as non-paged
			}
			markdown, err := convertToMarkdown(chunk.Content)
			if err != nil {
				return nil, err
			}
			chunk.Content = markdown
			if strings.TrimSpace(chunk.Content) != "" {
				slideNumCopy := slideNum + 1 // 1-based indexing
				chunk.PageNumber = slideNumCopy
				chunks = append(chunks, chunk)
			}
		}
	}
	return chunks, nil
}

func parseXLSX(filePath string) ([]models.Chunk, error) {
	f, err := xlsx.OpenFile(filePath)
	if err != nil {
		return nil, err
	}

	var chunks []models.Chunk
	for sheetNum, sheet := range f.Sheets {
		var text strings.Builder
		text.WriteString(fmt.Sprintf("## Sheet: %s\n", sheet.Name))
		for _, row := range sheet.Rows {
			for _, cell := range row.Cells {
				text.WriteString(cell.String() + "\t")
			}
			text.WriteString("\n")
		}
		chunk := models.Chunk{
			Content:    text.String(),
			PageNumber: defaultPageNumber, // XLSX has no pages
		}
		markdown, err := convertToMarkdown(chunk.Content)
		if err != nil {
			return nil, err
		}
		chunk.Content = markdown
		if strings.TrimSpace(chunk.Content) != "" {
			sheetNumCopy := sheetNum + 1 // 1-based indexing
			chunk.PageNumber = sheetNumCopy
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}

func parseODS(filePath string) ([]models.Chunk, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var chunks []models.Chunk
	for sheetNum, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}
		var text strings.Builder
		text.WriteString(fmt.Sprintf("## Sheet: %s\n", sheetName))
		for _, row := range rows {
			for _, cell := range row {
				text.WriteString(cell + "\t")
			}
			text.WriteString("\n")
		}
		chunk := models.Chunk{
			Content:    text.String(),
			PageNumber: defaultPageNumber, // ODS has no pages
		}
		markdown, err := convertToMarkdown(chunk.Content)
		if err != nil {
			return nil, err
		}
		chunk.Content = markdown
		if strings.TrimSpace(chunk.Content) != "" {
			sheetNumCopy := sheetNum + 1 // 1-based indexing
			chunk.PageNumber = sheetNumCopy
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}

func parseText(filePath string) ([]models.Chunk, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	chunk := models.Chunk{
		Content:    string(data),
		PageNumber: defaultPageNumber, // TXT has no pages
	}
	markdown, err := convertToMarkdown(chunk.Content)
	if err != nil {
		return nil, err
	}
	chunk.Content = markdown
	if strings.TrimSpace(chunk.Content) == "" {
		return nil, nil
	}
	return []models.Chunk{chunk}, nil
}

func convertToMarkdown(text string) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(text), &buf); err != nil {
		return "", err
	}

	// Trim leading and trailing newlines
	buf.WriteString("\n")
	// Trim leading and trailing spaces
	buf.WriteString(strings.Trim(buf.String(), " \t\n\r"))

	return buf.String(), nil
}

func extractTextFromXML(xmlContent string) string {
	var text strings.Builder
	parts := strings.Split(xmlContent, "<a:t>")
	for i, part := range parts {
		if i == 0 {
			continue
		}
		endIdx := strings.Index(part, "</a:t>")
		if endIdx >= 0 {
			text.WriteString(part[:endIdx] + " ")
		}
	}
	return text.String()
}

// chunk content into chunks with maxChars and overlapChars
func chunkContent(content string, maxChars, overlapChars int) []string {
	// Handle edge cases
	if maxChars <= 0 {
		return nil
	}
	if overlapChars < 0 {
		overlapChars = 0
	}
	if overlapChars >= maxChars {
		overlapChars = maxChars / 2 // Reasonable default to avoid excessive overlap
	}
	if len(content) == 0 {
		return nil
	}

	var chunks []string
	content = strings.TrimSpace(content)
	contentLen := len(content)

	// If content is shorter than maxChars, return it as a single chunk
	if contentLen <= maxChars {
		return []string{content}
	}

	// Iterate through content, creating chunks with overlap
	start := 0
	for start < contentLen {
		// Calculate end index, ensuring it doesn't exceed content length
		end := min(start+maxChars, contentLen)

		// Find a clean break point (e.g., end of a word or sentence) if possible
		if end < contentLen {
			// Look for a space or punctuation within the last 10% of the chunk
			lookBack := min(maxChars/10, end-start)
			for i := end - 1; i >= end-lookBack && i > start; i-- {
				if content[i] == ' ' || content[i] == '\n' || content[i] == '.' {
					end = i + 1
					break
				}
			}
		}

		// Extract the chunk and append it
		chunk := strings.TrimSpace(content[start:end])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		// Move start forward, accounting for overlap
		start += maxChars - overlapChars
		if start >= contentLen {
			break
		}
	}

	return chunks
}

// get chunks from content and page number
func (p *ParserConfig) getChunks(content string, pageNumber int) []models.Chunk {
	var chunks []models.Chunk

	// generate chunk strings from content
	chunkStrings := chunkContent(content, p.Config.RAG.ChunkSize, p.Config.RAG.ChunkOverlap)
	for i, chunkString := range chunkStrings {
		chunks = append(chunks, models.Chunk{
			Content:    chunkString,
			PageNumber: pageNumber,
			ChunkID:    i + 1,
		})
	}

	return chunks
}

// return complete content from chunks based on related chunkIDs and overlapCharLen
//
//	logic is to trim the content of each chunk to the last overlapCharLen characters
//	and then append the content of the next chunk, last chunk is appended as is
func getCompleteContent(chunks []models.Chunk, overlapCharLen int) string {
	var content strings.Builder
	for i, chunk := range chunks {
		chunkContent := chunk.Content
		if i < len(chunks)-1 {
			contentLen := len(chunkContent)
			if contentLen > 0 && contentLen > overlapCharLen {
				chunkContent = chunkContent[contentLen-overlapCharLen:]
			}
		}
		content.WriteString(chunkContent)
	}
	return content.String()
}
