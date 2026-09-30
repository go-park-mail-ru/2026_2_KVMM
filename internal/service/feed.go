package service

import (
	"kvmm/internal/domain"
	"kvmm/internal/store"
)

type FeedService struct{ store *store.Memory }

// NewFeedService создаёт сервис работы с лентой публикаций.
func NewFeedService(s *store.Memory) *FeedService { return &FeedService{store: s} }

// List возвращает часть ленты в обратном хронологическом порядке.
func (s *FeedService) List(offset, limit int) ([]domain.Post, bool) {
	return s.store.Feed(offset, limit)
}
