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
		return "", fmt.Errorf("%s é obrigatório", name)
	}
	if len([]rune(value)) > max {
		return "", fmt.Errorf("%s deve ter no máximo %d caracteres", name, max)
	}
	return value, nil
}

func ProjectKey(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !projectKey.MatchString(value) {
		return "", fmt.Errorf("a sigla deve ter de 2 a 10 letras maiúsculas ou números e começar com uma letra")
	}
	return value, nil
}

func OneOf(name, value string, allowed ...string) (string, error) {
	for _, candidate := range allowed {
		if value == candidate {
			return value, nil
		}
	}
	return "", fmt.Errorf("%s inválido", name)
}

func OptionalID(value string) (*int64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("identificador inválido")
	}
	return &id, nil
}

func Date(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("data inválida")
	}
	return &date, nil
}

