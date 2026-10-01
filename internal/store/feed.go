package store

import (
	"sort"
	"strings"

	"kvmm/internal/domain"
)

// ListPosts возвращает посты, пагинация - (offset, offset+limit]
func ListPosts(offset, limit int) ([]domain.Post, bool) {
	if offset < 0 || limit < 1 {
		return []domain.Post{}, false
	}
	result := make([]domain.Post, 0, len(posts))
	for _, post := range posts {
		result = append(result, post)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if offset >= len(result) {
		return []domain.Post{}, false
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], end < len(result)
}

// ReadMediaFile возвращает встроенный файл публикации по безопасному имени
func ReadMediaFile(name string) ([]byte, error) {
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\\`) {
		return nil, ErrNotFound
	}
	return embeddedMediaFiles.ReadFile("media/" + name)
}
