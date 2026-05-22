package realtime

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	EventPostCreated = "post.created"
	EventPostDeleted = "post.deleted"
	EventPostUpdated = "post.updated"
	EventPostHidden  = "post.hidden"
	EventReplyCreated = "reply.created"
	EventReplyUpdated = "reply.updated"
	EventReplyDeleted = "reply.deleted"
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type PostIDPayload struct {
	PostID string `json:"postId"`
}

type PostUpdatedPayload struct {
	PostID     string `json:"postId"`
	Score      int    `json:"score"`
	ReplyCount int    `json:"replyCount"`
}

type ReplyDeletedPayload struct {
	PostID  string `json:"postId"`
	ReplyID string `json:"replyId"`
}

type ReplyScorePayload struct {
	PostID  string `json:"postId"`
	ReplyID string `json:"replyId"`
	Score   int    `json:"score"`
}

func MarshalEnvelope(eventType string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return json.Marshal(Envelope{Type: eventType, Payload: raw})
}

func BroadcastBytes(eventType string, payload []byte) ([]byte, error) {
	var peek Envelope
	if err := json.Unmarshal(payload, &peek); err == nil && peek.Type != "" {
		return payload, nil
	}

	return json.Marshal(Envelope{Type: eventType, Payload: json.RawMessage(payload)})
}

func Publish(publisher interface {
	Enqueue(ctx context.Context, eventType string, payload []byte) error
}, ctx context.Context, eventType string, payload any) error {
	if publisher == nil {
		return nil
	}

	body, err := MarshalEnvelope(eventType, payload)
	if err != nil {
		return fmt.Errorf("marshal feed event: %w", err)
	}

	return publisher.Enqueue(ctx, eventType, body)
}
