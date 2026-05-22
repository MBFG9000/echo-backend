package service

import (
	"testing"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
)

func TestLatestFeedETag_stable(t *testing.T) {
	id := uuid.New()
	posts := []domain.Post{{ID: id, CreatedAt: time.Unix(0, 1), Score: 1}}
	a := LatestFeedETag(20, "", posts)
	b := LatestFeedETag(20, "", posts)
	if a != b {
		t.Fatalf("expected stable etag, got %s vs %s", a, b)
	}
}

func TestLatestFeedETag_changes(t *testing.T) {
	posts := []domain.Post{{ID: uuid.New(), CreatedAt: time.Unix(0, 1), Score: 1}}
	other := []domain.Post{{ID: uuid.New(), CreatedAt: time.Unix(0, 2), Score: 2}}
	if LatestFeedETag(20, "", posts) == LatestFeedETag(20, "", other) {
		t.Fatal("expected different etags for different feeds")
	}
}
