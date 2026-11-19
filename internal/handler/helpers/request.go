package helpers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func IsUUID(value string) bool {
	return uuidPattern.MatchString(strings.TrimSpace(value))
}

// UUIDQuery читает UUID из query-параметра
func UUIDQuery(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if !IsUUID(value) {
		return "", fmt.Errorf("%s must be a valid UUID", name)
	}
	return strings.ToLower(value), nil
}

// Pagination читает cursor и размер страницы из query-параметров
func Pagination(r *http.Request, max_limit int) (int, int, error) {
	cursor := 0
	limit := max_limit
	query := r.URL.Query()
	if value := query.Get("cursor"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return 0, 0, fmt.Errorf("cursor must be a non-negative integer")
		}
		cursor = parsed
	}
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 10 {
			return 0, 0, fmt.Errorf("limit must be between 1 and %d", max_limit)
		}
		limit = parsed
	}
	return cursor, limit, nil
}

// UUIDPagination читает UUID-курсор и размер страницы
func UUIDPagination(r *http.Request, maxLimit int) (string, int, error) {
	limit := maxLimit
	value := r.URL.Query().Get("cursor")
	if value != "" && !IsUUID(value) {
		return "", 0, fmt.Errorf("cursor must be a valid UUID")
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > maxLimit {
			return "", 0, fmt.Errorf("limit must be between 1 and %d", maxLimit)
		}
		limit = parsed
	}
	return strings.ToLower(strings.TrimSpace(r.URL.Query().Get("cursor"))), limit, nil
}

// PositiveQueryInt64 читает положительное число из query-параметра
func PositiveQueryInt64(r *http.Request, name string) (int64, error) {
	value := r.URL.Query().Get(name)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}
