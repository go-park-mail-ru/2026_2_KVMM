package helpers

import (
	"fmt"
	"net/http"
	"strconv"
)

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

// PositiveQueryInt64 читает положительное число из query-параметра
func PositiveQueryInt64(r *http.Request, name string) (int64, error) {
	value := r.URL.Query().Get(name)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

// methodAllowed проверяет HTTP-метод и возвращает ошибку для неподдерживаемого метода
func MethodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
	return false
}
