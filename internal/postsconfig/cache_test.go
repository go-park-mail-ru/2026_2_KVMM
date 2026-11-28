package postsconfig

import (
	"testing"

	"kvmm/internal/domain"
)

func TestLikesPercentage(t *testing.T) {
	likes := domain.OnlineConfFlagSnapshot{Rollout: 50}
	// The specified hash puts user 3 in bucket 49 and user 1 in bucket 86.
	if !likesEnabled(likes, 3) || likesEnabled(likes, 1) {
		t.Fatal("unexpected assignment at 50% rollout")
	}
	likes.Rollout = 49
	if likesEnabled(likes, 3) {
		t.Fatal("a bucket equal to rollout must be disabled")
	}
}
