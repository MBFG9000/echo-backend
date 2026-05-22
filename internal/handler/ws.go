package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/echo-app/echo/internal/hub"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WS struct {
	hub      *hub.Hub
	allowAll bool
	origins  map[string]struct{}
	upgrader websocket.Upgrader
}

func NewWS(h *hub.Hub, allowedOrigins []string) *WS {
	origins := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, origin := range allowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			allowAll = true
			continue
		}
		origins[trimmed] = struct{}{}
	}

	ws := &WS{
		hub:      h,
		allowAll: allowAll,
		origins:  origins,
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	upgrader.CheckOrigin = ws.checkOrigin
	ws.upgrader = upgrader

	return ws
}

func (w *WS) checkOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || w.allowAll {
		return true
	}

	if _, ok := w.origins[origin]; ok {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	return strings.EqualFold(parsedOrigin.Host, r.Host)
}

func (w *WS) Register(rg *gin.RouterGroup) {
	rg.GET("/feed", w.feed)
}

func (w *WS) feed(c *gin.Context) {
	conn, err := w.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := hub.NewClient(w.hub, conn)
	w.hub.Register(client)
	client.Start()
}
