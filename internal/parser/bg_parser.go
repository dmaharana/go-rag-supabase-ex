package parser

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"document-rag/internal/config"
	"document-rag/internal/embedding"
	"document-rag/internal/models"

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

type bgParserState struct {
	chapter, title, expandedTitle  string
	currentSpeaker, currentContent string
	result                         []BGSection
	cfg                            *config.Config
}

// ParseBGText parses the BG text according to custom rules and returns a slice of BGSection
func ParseBGText(input string, cfg *config.Config) []BGSection {
	log.Debug().Msgf("Parsing BG text from file: %s", input)
	var state bgParserState

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

	chapterRe := regexp.MustCompile(models.ChapterRegex)
	speakerRe := regexp.MustCompile(models.SpeakerRegex)
	titleRe := regexp.MustCompile(models.TitleRegex)
	expandedTitleRe := regexp.MustCompile(models.ExpandedTitleRegex)
	fnRe := regexp.MustCompile(models.FnRegex)

	f, err := os.Open(input)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to open file: %s", input)
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if fnRe.MatchString(line) {
			break
		}
		processBGLine(line, &state, chapterRe, speakerRe, titleRe, expandedTitleRe, cfg)
	}
	handleSpeakerChange(&state, cfg, true) // store last section if needed
	return state.result
}

// processBGLine handles a single line of BG text, updating the parser state accordingly
func processBGLine(line string, state *bgParserState, chapterRe, speakerRe, titleRe, expandedTitleRe *regexp.Regexp, cfg *config.Config) {
	if m := chapterRe.FindStringSubmatch(line); m != nil {
		handleSpeakerChange(state, cfg, false)
		state.chapter = m[1]
		state.currentContent = ""
		return
	}
	if m := titleRe.FindStringSubmatch(line); m != nil {
		state.title = strings.ReplaceAll(m[1], "\"", "")
		state.result = updateTitleOfPreviousRecords(state.result, state.chapter, state.title)
		return
	}
	if m := expandedTitleRe.FindStringSubmatch(line); m != nil {
		state.expandedTitle = strings.ReplaceAll(m[1], "\"", "")
		state.result = updateExpandedTitleOfPreviousRecords(state.result, state.chapter, state.expandedTitle)
		return
	}
	if m := speakerRe.FindStringSubmatch(line); m != nil {
		handleSpeakerChange(state, cfg, false)
		state.currentSpeaker = m[1]
		state.currentContent = ""
		return
	}
	// If inside a speaker section, accumulate content
	if state.currentSpeaker != "" {
		if state.currentContent != "" {
			state.currentContent += "\n"
		}
		state.currentContent += line
	}
}

// handleSpeakerChange saves the current speaker's content if present, and resets content if not final
func handleSpeakerChange(state *bgParserState, cfg *config.Config, final bool) {
	if state.currentSpeaker != "" && state.currentContent != "" {
		entry := BGSection{
			Chapter:       state.chapter,
			Title:         state.title,
			ExpandedTitle: state.expandedTitle,
			Speaker:       state.currentSpeaker,
			Content:       strings.TrimSpace(state.currentContent),
		}
		// chunks := chunkSaveContent(entry, cfg)
		chunksWithContext := chunkSaveContentWithContext(entry, cfg)
		if len(chunksWithContext) > 0 {
			state.result = append(state.result, chunksWithContext...)
		}
	}
	if !final {
		state.currentContent = ""
	}
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
// chunkSaveContent breaks the content of a BGSection into chunks based on the
// maxChars and overlapChars parameters. It returns a slice of BGSection, each
// with a chunk of the content, and the same chapter, title, expanded title, and speaker
// as the original contentEntry. The ChunkID is set to the index of the chunk
// in the slice.
func chunkSaveContent(contentEntry BGSection, cfg *config.Config) []BGSection {
	var result []BGSection
	for i, chunk := range chunkContent(contentEntry.Content, cfg.RAG.ChunkSize, cfg.RAG.ChunkOverlap) {
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

func chunkSaveContentWithContext(contentEntry BGSection, cfg *config.Config) []BGSection {
	log.Debug().Msgf("Chunking content with context for chapter: %s, title: %s, expanded title: %s, speaker: %s", contentEntry.Chapter, contentEntry.Title, contentEntry.ExpandedTitle, contentEntry.Speaker)
	var result []BGSection
	for i, chunk := range chunkContent(contentEntry.Content, cfg.RAG.ChunkSize, cfg.RAG.ChunkOverlap) {

		// based on anthropic research, adding context to chunks can improve search retrieval
		// https://www.anthropic.com/news/contextual-retrieval
		context, err := embedding.GenerateContext(context.Background(), &cfg.QueryLLM, contentEntry.Content, chunk)
		if err != nil {
			log.Error().Err(err).Msg("Error generating context")
			return nil
		}

		log.Debug().Msgf("Before Context: %s", context)

		if context != "" {
			// remove <think> tag and all content between it and </think>
			re := regexp.MustCompile(models.ThinkTag)
			context = re.ReplaceAllString(context, "")
			log.Debug().Msgf("After Context: %s", context)
			chunk = context + models.ContextSeparator + chunk
		}

		log.Info().Msgf("Chunk %d: %s", i+1, chunk)
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

// chunk save with context
func AddContextByChapter(content []BGSection, cfg *config.Config) []BGSection {
	var result []BGSection
	ctx := context.Background()
	docs := generateDocumentByChapter(content, cfg)

	// print docs
	for k, v := range docs {
		log.Info().Msgf("Chapter: %s", k)
		log.Info().Msgf("Content: %s", v)
	}

	for _, chunk := range content {
		context, err := embedding.GenerateContext(ctx, &cfg.QueryLLM, docs[chunk.Chapter], chunk.Content)
		if err != nil {
			log.Error().Err(err).Msg("Error generating context")
			return nil
		}
		result = append(result, BGSection{
			Chapter:       chunk.Chapter,
			Title:         chunk.Title,
			ExpandedTitle: chunk.ExpandedTitle,
			Speaker:       chunk.Speaker,
			Content:       context + models.ContextSeparator + chunk.Content,
			ChunkID:       chunk.ChunkID,
		})
	}
	return result
}

// generate full document by chapter
func generateDocumentByChapter(chunks []BGSection, cfg *config.Config) map[string]string {
	documentByChapter := map[string]string{}

	overlapLen := cfg.RAG.ChunkOverlap
	for _, chunk := range chunks {
		if _, ok := documentByChapter[chunk.Chapter]; !ok {
			var content strings.Builder
			truncatedContent := ""
			for _, c := range chunks {
				if c.Chapter == chunk.Chapter {
					if len(c.Content) > overlapLen {
						truncatedContent = c.Content[:overlapLen]
						content.WriteString(c.Content[overlapLen:])
					} else {
						truncatedContent = ""
						content.WriteString(c.Content)
					}
				}
			}
			documentByChapter[chunk.Chapter] = content.String() + truncatedContent
		}
	}
	return documentByChapter
}

// create metadata map[string]string from BGSection
func CreateMetadata(contentEntry BGSection) map[string]string {
	return map[string]string{
		"chapter":        contentEntry.Chapter,
		"title":          contentEntry.Title,
		"expanded_title": contentEntry.ExpandedTitle,
		"speaker":        contentEntry.Speaker,
		"chunk_id":       fmt.Sprintf("%d", contentEntry.ChunkID),
	}
}
