package parser

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"document-rag/internal/config"
	"github.com/rs/zerolog/log"
)

type BGSection struct {
	Chapter       string `json:"chapter"`
	Title         string `json:"title"`
	ExpandedTitle string `json:"expanded_title"`
	Speaker       string `json:"speaker"`
	Content       string `json:"content"`
	ChunkID       int    `json:"chunk_id"`
}

// ParseBGText parses the BG text according to custom rules and returns a slice of BGSection
func ParseBGText(input string, cfg *config.Config) []BGSection {
	var result []BGSection
	var chapter, title, expandedTitle string
	var currentSpeaker, currentContent string

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

	chapterRe := regexp.MustCompile(`(?m)^CHAPTER (.+)$`)
	speakerRe := regexp.MustCompile(`^(Dhritirashtra|Sanjaya|Krishna|Arjuna)[\.:]\s*`)
	titleRe := regexp.MustCompile(`(?m)^Entitled (.+)$`)
	expandedTitleRe := regexp.MustCompile(`(?m)^Or (.+)$`)
	fnRe := regexp.MustCompile(`^\[FN#\d+\]`)

	content, err := os.ReadFile(input)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to read file: %s", input)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if fnRe.MatchString(line) {
			break
		}

		if m := chapterRe.FindStringSubmatch(line); m != nil {
			
			// save previous record
			if currentSpeaker != "" && currentContent != "" {
				entry := BGSection{
					Chapter:       chapter,
					Title:         title,
					ExpandedTitle: expandedTitle,
					Speaker:       currentSpeaker,
					Content:       strings.TrimSpace(currentContent),
				}
				chunks := chunkSaveContent(entry, cfg.RAG.ChunkSize, cfg.RAG.ChunkOverlap)
				if len(chunks) > 0 {
					result = append(result, chunks...)
				}
			}

			chapter = m[1]
			// reset current content
			currentContent = ""
			continue
		}
		if m := titleRe.FindStringSubmatch(line); m != nil {
			title = m[1]
			// remove double quotes
			title = strings.ReplaceAll(title, "\"", "")
			result = updateTitleOfPreviousRecords(result, chapter, title)
			continue
		}
		if m := expandedTitleRe.FindStringSubmatch(line); m != nil {
			expandedTitle = m[1]
			// remove double quotes
			expandedTitle = strings.ReplaceAll(expandedTitle, "\"", "")
			result = updateExpandedTitleOfPreviousRecords(result, chapter, expandedTitle)
			continue
		}
		if m := speakerRe.FindStringSubmatch(line); m != nil {
			// Store previous content if exists
			if currentSpeaker != "" && currentContent != "" {
				entry := BGSection{
					Chapter:       chapter,
					Title:         "",
					ExpandedTitle: "",
					Speaker:       currentSpeaker,
					Content:       strings.TrimSpace(currentContent),
				}
				chunks := chunkSaveContent(entry, cfg.RAG.ChunkSize, cfg.RAG.ChunkOverlap)
				if len(chunks) > 0 {
					result = append(result, chunks...)
				}
			}

			// New speaker
			currentSpeaker = m[1]
			// Reset current content
			currentContent = ""
			continue
		}
		// If inside a speaker section, accumulate content
		if currentSpeaker != "" {
			if currentContent != "" {
				currentContent += "\n"
			}
			currentContent += line
		}
	}
	// Store last section
	if currentSpeaker != "" && currentContent != "" {
		entry := BGSection{
			Chapter:       chapter,
			Title:         title,
			ExpandedTitle: expandedTitle,
			Speaker:       currentSpeaker,
			Content:       strings.TrimSpace(currentContent),
		}
		chunks := chunkSaveContent(entry, cfg.RAG.ChunkSize, cfg.RAG.ChunkOverlap)
		if len(chunks) > 0 {
			result = append(result, chunks...)
		}
	}
	return result
}


// update previous records with title if it is empty
func updateTitleOfPreviousRecords(result []BGSection, chapter string, title string) []BGSection {
	for row := range result {
		if row >= 0 && result[row].Chapter == chapter {
			if result[row].Title == "" && title != "" {
				result[row].Title = title
			}
		}
	}
	return result
}

// update previous records with expanded title if it is empty
func updateExpandedTitleOfPreviousRecords(result []BGSection, chapter string, expandedTitle string) []BGSection {
	for row := range result {
		if row >= 0 && result[row].Chapter == chapter {
			if result[row].ExpandedTitle == "" && expandedTitle != "" {
				result[row].ExpandedTitle = expandedTitle
			}
		}
	}
	return result
}


// chunk content
func chunkSaveContent(contentEntry BGSection, maxChars, overlapChars int) []BGSection {
	var result []BGSection
	for i, chunk := range chunkContent(contentEntry.Content, maxChars, overlapChars) {
		result = append(result, BGSection{
			Chapter:       contentEntry.Chapter,
			Title:         contentEntry.Title,
			ExpandedTitle: contentEntry.ExpandedTitle,
			Speaker:       contentEntry.Speaker,
			Content:       chunk,
			ChunkID:       i + 1,
		})
	}
	return result
}
