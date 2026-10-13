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

// methodAllowed проверяет HTTP-метод и возвращает ошибку для неподдерживаемого метода
func MethodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
	return false
}
