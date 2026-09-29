package filter

import (
	"slices"
	"strings"
)

var BadWords = []string{"kerfuffle", "sharbert", "fornax"}

func Censor(text string) string {
	beeps := "****"
	words := strings.Fields(text)
	cleanedWords := []string{}

	for _, w := range words {
		if slices.Contains(BadWords, strings.ToLower(w)) {
			cleanedWords = append(cleanedWords, beeps)
		} else {
			cleanedWords = append(cleanedWords, w)
		}
	}

	return strings.Join(cleanedWords, " ")
}
