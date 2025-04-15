package parser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/tealeg/xlsx"
	"github.com/xuri/excelize/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func ParseToMarkdown(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return parsePDF(filePath)
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
		return "", fmt.Errorf("unsupported file format: %s", ext)
	}
}

func parsePDF(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Get file size for reader initialization
	stat, err := f.Stat()
	if err != nil {
		return "", err
	}

	reader, err := pdf.NewReader(f, stat.Size())
	if err != nil {
		return "", err
	}

	var text strings.Builder
	numPages := reader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i) // Use Page(i) instead of GetPage
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			return "", err
		}
		text.WriteString(pageText + "\n\n")
	}
	return convertToMarkdown(text.String())
}

func parseDOCX(filePath string) (string, error) {
	r, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Access the underlying Docx struct
	doc := r.Editable()
	content := doc.GetContent() // GetContent returns a string
	// Split content into paragraphs based on newlines for Markdown formatting
	paragraphs := strings.Split(content, "\n")
	var text strings.Builder
	for _, p := range paragraphs {
		if p != "" {
			text.WriteString(p + "\n\n")
		}
	}
	return convertToMarkdown(text.String())
}

func parsePPTX(filePath string) (string, error) {
	f, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var text strings.Builder
	for _, file := range f.File {
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
			text.WriteString(slideText + "\n\n")
		}
	}
	return convertToMarkdown(text.String())
}

func parseXLSX(filePath string) (string, error) {
	f, err := xlsx.OpenFile(filePath)
	if err != nil {
		return "", err
	}

	var text strings.Builder
	for _, sheet := range f.Sheets {
		text.WriteString(fmt.Sprintf("## Sheet: %s\n", sheet.Name))
		for _, row := range sheet.Rows {
			for _, cell := range row.Cells {
				text.WriteString(cell.String() + "\t")
			}
			text.WriteString("\n")
		}
		text.WriteString("\n")
	}
	return convertToMarkdown(text.String())
}

func parseODS(filePath string) (string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var text strings.Builder
	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}
		text.WriteString(fmt.Sprintf("## Sheet: %s\n", sheetName))
		for _, row := range rows {
			for _, cell := range row {
				text.WriteString(cell + "\t")
			}
			text.WriteString("\n")
		}
		text.WriteString("\n")
	}
	return convertToMarkdown(text.String())
}

func parseText(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return convertToMarkdown(string(data))
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

