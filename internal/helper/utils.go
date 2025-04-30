package helper

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
)

// GenerateUUID creates a random unique UUID string
func GenerateUUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %v", err)
	}
	return id.String(), nil
}

// pretty print
func PrettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Warn().Msg("Error pretty printing")
	}
	fmt.Println(string(b))
}

