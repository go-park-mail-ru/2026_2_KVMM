package fake_data

import (
	"fmt"
	"time"

	"kvmm/internal/domain"
)

// MakePosts создает демо посты
func MakePosts() map[int64]domain.Post {
	createdAt := time.Now().UTC()
	result := make(map[int64]domain.Post, 100)
	for i := 1; i <= 100; i++ {
		id := int64(i)
		avatar, gender := "/api/media/avatar1.png", domain.GenderFemale
		text := "Тестовая публикация пользователя"
		mediaURLs := []string{"/api/media/post_media1.jpg", "/api/media/post_media1.pdf", "/api/media/post_media1.pdf.zip"}
		likes, comments, reposts := int64(4), int64(1), int64(2)
		if i%2 == 0 {
			avatar, gender = "/api/media/avatar2.png", domain.GenderMale
			text = "Вторая тестовая публикация пользователя"
			mediaURLs = []string{"/api/media/post_media2.1.png", "/api/media/post_media2.2.mp4"}
			likes, comments = int64(7), int64(13)
		}
		profile := domain.Profile{ID: id, Nickname: fmt.Sprintf("demo%d", i), ProfileName: "Демо", Surname: "Пользователь", Gender: gender, ImageURL: avatar, AvatarURLs: []string{avatar}, CreatedAt: createdAt}
		result[id] = domain.Post{ID: id, PostText: &text, AuthorProfileID: &profile.ID, Author: profile, CreatedAt: createdAt.Add(-time.Duration(i) * time.Minute), MediaCount: int64(len(mediaURLs)), MediaURLs: mediaURLs, LikesCount: likes, CommentsCount: comments, RepostsCount: reposts}
	}
	return result
}
