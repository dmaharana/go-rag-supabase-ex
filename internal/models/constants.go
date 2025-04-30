package models

const (
	ChapterRegex = `(?m)^CHAPTER (.+)$`
	SpeakerRegex = `^(Dhritirashtra|Sanjaya|Krishna|Arjuna)[\.:]\s*`
	TitleRegex   = `(?m)^Entitled (.+)$`
	ExpandedTitleRegex = `(?m)^Or (.+)$`
	FnRegex      = `^\[FN#\d+\]`
)