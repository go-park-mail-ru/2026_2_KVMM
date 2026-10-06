package fake_data

import (
	"fmt"
	"time"

	"kvmm/internal/domain"

	"github.com/brianvoe/gofakeit/v7"
)

func MakePosts() map[int64]domain.Post {
	result := make(map[int64]domain.Post, 100)

	for i := 1; i <= 100; i++ {
		result[int64(i)] = MakeFakePost(i)
	}

	return result
}

func MakeFakePost(i int) domain.Post {
	createdAt := time.Now().UTC()

	var deleteAt *time.Time

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

	if gofakeit.Number(0, 100) < 10 {
		now := time.Now().UTC()
		deleteAt = &now
	}

	postText := fmt.Sprintf(
		"Post ID: %d.\n%s",
		id,
		gofakeit.Sentence(5),
	)

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

	return domain.Post{
		ID:              id,
		PostText:        &postText,
		AuthorProfileID: &profile.ID,
		Author:          profile,
		CreatedAt:       createdAt,
		MediaCount:      int64(len(mediaURLs)),
		MediaURLs:       mediaURLs,
		LikesCount:      int64(gofakeit.Number(0, 100)),
		CommentsCount:   int64(gofakeit.Number(0, 50)),
		RepostsCount:    int64(gofakeit.Number(0, 20)),
		DeletedAt:       deleteAt,
	}
}
