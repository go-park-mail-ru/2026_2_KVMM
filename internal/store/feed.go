package store

import (
	"kvmm/internal/domain"
)

// ListPosts возвращает посты
// limit - количесво постов
// cursor — ID последнего отданного поста
// cursor = 0 — первая страница cursor станет самым новым постом
func ListPosts(cursor int, limit int) ([]domain.Post, bool, int) {
	nextCursor := 0
	if cursor < 0 || limit < 1 {
		return []domain.Post{}, false, -1
	}

	result := make([]domain.Post, 0, len(posts))

	startID := int64(cursor) - 1
	if cursor == 0 {
		startID = int64(len(posts))
	}

	postsCounter := 0
	hasMore := true

	for postsCounter < limit {
		if startID < 1 {
			hasMore = false
			break
		}
		post := posts[int64(startID)]
		if post.DeletedAt == nil {
			result = append(result, post)
			postsCounter++
		}
		startID--
	}
	nextCursor = int(startID)

	return result, hasMore, nextCursor
}
