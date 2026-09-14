package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var projectKey = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)

func Required(name, value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	if len([]rune(value)) > max {
		return "", fmt.Errorf("%s must be at most %d characters", name, max)
	}
	return value, nil
}

func ProjectKey(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !projectKey.MatchString(value) {
		return "", fmt.Errorf("key must contain 2-10 uppercase letters or numbers and start with a letter")
	}
	return value, nil
}

func OneOf(name, value string, allowed ...string) (string, error) {
	for _, candidate := range allowed {
		if value == candidate {
			return value, nil
		}
	}
	return "", fmt.Errorf("invalid %s", name)
}

func OptionalID(value string) (*int64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid identifier")
	}
	return &id, nil
}

func Date(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("invalid date")
	}
	return &date, nil
}

