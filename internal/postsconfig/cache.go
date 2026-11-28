package postsconfig

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"kvmm/internal/domain"
)

const DefaultMaxTextLength = 256

type Values struct {
	MaxTextLength int  `json:"max_text_length"`
	LikesEnabled  bool `json:"likes_enabled"`
}

func likesEnabled(f domain.OnlineConfFlagSnapshot, userID int64) bool {
	if slices.Contains(f.ExcludeIDs, userID) {
		return false
	}
	if slices.Contains(f.IncludeIDs, userID) {
		return true
	}
	hash := sha256.Sum256([]byte("posts/likes_enabled:" + strconv.FormatInt(userID, 10)))
	return binary.BigEndian.Uint64(hash[:8])%100 < uint64(f.Rollout)
}

type snapshot struct {
	maxTextLength int
	likes         *domain.OnlineConfFlagSnapshot
}

type Cache struct {
	fetch   func(context.Context, domain.OnlineConfScope) (domain.OnlineConfSnapshot, error)
	current atomic.Pointer[snapshot]
}

func New(fetch func(context.Context, domain.OnlineConfScope) (domain.OnlineConfSnapshot, error)) *Cache {
	return &Cache{fetch: fetch}
}

func (c *Cache) Values(userID int64) Values {
	current := c.current.Load()
	if current == nil {
		return Values{MaxTextLength: DefaultMaxTextLength, LikesEnabled: true}
	}
	return Values{
		MaxTextLength: current.maxTextLength,
		LikesEnabled:  current.likes == nil || likesEnabled(*current.likes, userID),
	}
}

func (c *Cache) Refresh(ctx context.Context) error {
	data, err := c.fetch(ctx, domain.OnlineConfScopePosts)
	if err != nil {
		return fmt.Errorf("fetch posts snapshot: %w", err)
	}
	if data.Scope != domain.OnlineConfScopePosts {
		return fmt.Errorf("unexpected snapshot scope %q", data.Scope)
	}
	next := &snapshot{maxTextLength: DefaultMaxTextLength}
	if raw, ok := data.Configs["max_text_length"]; ok {
		var limit int
		if err := json.Unmarshal(raw, &limit); err != nil || limit < 1 || limit > 2048 {
			return fmt.Errorf("invalid posts.max_text_length in snapshot")
		}
		next.maxTextLength = limit
	}
	if likes, ok := data.Flags["likes_enabled"]; ok {
		if likes.Rollout < 0 || likes.Rollout > 100 {
			return fmt.Errorf("invalid posts.likes_enabled in snapshot")
		}
		next.likes = &likes
	}
	c.current.Store(next)
	return nil
}

func (c *Cache) Poll(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Refresh(ctx); err != nil && ctx.Err() == nil {
				onError(err)
			}
		}
	}
}
