package rag

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxQueryChars = 1000

var (
	ErrEmptyQuery   = errors.New("query is required")
	ErrQueryTooLong = errors.New("query is too long")
)

func NormalizeQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", ErrEmptyQuery
	}
	if utf8.RuneCountInString(query) > MaxQueryChars {
		return "", fmt.Errorf("%w; max %d characters", ErrQueryTooLong, MaxQueryChars)
	}
	return query, nil
}
