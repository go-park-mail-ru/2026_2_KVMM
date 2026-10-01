package fake_data

import (
	"time"

	"kvmm/internal/domain"

	"github.com/brianvoe/gofakeit/v7"
)

// MakePosts создает демо посты с использованием gofakeit.
func MakePosts() map[int64]domain.Post {
	gofakeit.Seed(1)

	createdAt := time.Now().UTC()
	result := make(map[int64]domain.Post, 100)

	for i := 1; i <= 100; i++ {
		id := int64(i)

		avatar := "/api/media/avatar1.png"
		gender := domain.GenderFemale
		mediaURLs := []string{
			"/api/media/post_media1.jpg",
			"/api/media/post_media1.pdf",
			"/api/media/post_media1.pdf.zip",
		}
		if i%2 == 0 {
			avatar = "/api/media/avatar2.png"
			gender = domain.GenderMale
			mediaURLs = []string{
				"/api/media/post_media2.1.png",
				"/api/media/post_media2.2.mp4",
			}
		}

		postText := gofakeit.Sentence(5)
		profile := domain.Profile{
			ID:          id,
			Nickname:    gofakeit.Username(),
			ProfileName: gofakeit.FirstName(),
			Surname:     gofakeit.LastName(),
			Gender:      gender,
			ImageURL:    avatar,
			AvatarURLs:  []string{avatar},
			CreatedAt:   createdAt,
		}

		result[id] = domain.Post{
			ID:              id,
			PostText:        &postText,
			AuthorProfileID: &profile.ID,
			Author:          profile,
			CreatedAt:       createdAt.Add(-time.Duration(i) * time.Hour),
			MediaCount:      int64(len(mediaURLs)),
			MediaURLs:       mediaURLs,
			LikesCount:      int64(gofakeit.Number(0, 100)),
			CommentsCount:   int64(gofakeit.Number(0, 50)),
			RepostsCount:    int64(gofakeit.Number(0, 20)),
		}
	}

	return result
}
