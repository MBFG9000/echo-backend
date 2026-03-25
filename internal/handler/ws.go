package handler

import (
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/echo-app/echo/internal/hub"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WS struct {
	hub      *hub.Hub
	upgrader websocket.Upgrader
}

func NewWS(h *hub.Hub) *WS {
	return &WS{
		hub: h,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(*http.Request) bool {
				return true
			},
		},
	}
}

func (w *WS) Register(rg *gin.RouterGroup) {
	rg.GET("/feed", w.feed)
}

func (w *WS) feed(c *gin.Context) {
	conn, err := w.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	client := hub.NewClient(w.hub, conn)
	w.hub.Register(client)
	client.Start()
}
