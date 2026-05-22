package hub

import (
	"context"
	"sync/atomic"

	"github.com/echo-app/echo/internal/metrics"
)

type Hub struct {
	clients     map[*Client]struct{}
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	quit        chan struct{}
	clientCount atomic.Int32
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		quit:       make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for c := range h.clients {
				delete(h.clients, c)
				close(c.send)
				_ = c.conn.Close()
			}
			h.clientCount.Store(0)
			metrics.WSConnections.Set(0)
			close(h.quit)
			return
		case c := <-h.register:
			h.clients[c] = struct{}{}
			h.clientCount.Add(1)
			metrics.WSConnections.Set(float64(h.clientCount.Load()))
		case c := <-h.unregister:
			h.remove(c)
		case payload := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- payload:
				default:
					h.remove(c)
				}
			}
		}
	}
}

func (h *Hub) Register(c *Client) {
	select {
	case <-h.quit:
		_ = c.conn.Close()
	case h.register <- c:
	}
}

func (h *Hub) Unregister(c *Client) {
	select {
	case <-h.quit:
	case h.unregister <- c:
	}
}

func (h *Hub) Broadcast(payload []byte) {
	select {
	case <-h.quit:
	case h.broadcast <- payload:
	default:
	}
}

func (h *Hub) remove(c *Client) {
	if _, ok := h.clients[c]; !ok {
		return
	}

	delete(h.clients, c)
	h.clientCount.Add(-1)
	metrics.WSConnections.Set(float64(h.clientCount.Load()))
	close(c.send)
	_ = c.conn.Close()
}

func (h *Hub) ClientCount() int {
	return int(h.clientCount.Load())
}
