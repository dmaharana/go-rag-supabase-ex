package models

const (
	ChapterRegex       = `(?m)^CHAPTER (.+)$`
	SpeakerRegex       = `^(Dhritirashtra|Sanjaya|Krishna|Arjuna)[\.:]\s*`
	TitleRegex         = `(?m)^Entitled (.+)$`
	ExpandedTitleRegex = `(?m)^Or (.+)$`
	FnRegex            = `^\[FN#\d+\]`
	ContextSeparator   = "\n---\n"
	ThinkTag           = `(?s)<think>.*?</think>`
)

var (
	ContextPromptTemplate = `<document>
%s
</document>
Here is the chunk we want to situate within the whole document
<chunk>
%s
</chunk>
Please give a short succinct context to situate this chunk within the overall document for the purposes of improving search retrieval of the chunk. Answer only with the succinct context and nothing else.
`
)
