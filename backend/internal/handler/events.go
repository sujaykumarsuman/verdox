package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/sse"
)

type EventsHandler struct {
	rdb *redis.Client
}

func NewEventsHandler(rdb *redis.Client) *EventsHandler {
	return &EventsHandler{rdb: rdb}
}

// Stream handles the SSE endpoint. It subscribes to the user's Redis Pub/Sub
// channel and forwards events as Server-Sent Events.
func (h *EventsHandler) Stream(c echo.Context) error {
	userID := mw.GetUserID(c)

	w := c.Response()
	r := c.Request()

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable Nginx buffering

	// Use ResponseController on the underlying http.ResponseWriter for deadline control
	rc := http.NewResponseController(w.Writer)

	// Subscribe to user-specific Redis channel
	ctx := r.Context()
	channel := sse.ChannelForUser(userID)
	sub := h.rdb.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()

	// Heartbeat ticker — keeps the connection alive and detects dead clients
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	// Send initial connection event
	_ = rc.SetWriteDeadline(time.Now().Add(60 * time.Second))
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
	w.Flush()

	for {
		select {
		case <-ctx.Done():
			return nil

		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			_ = rc.SetWriteDeadline(time.Now().Add(60 * time.Second))
			fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
			w.Flush()

		case <-heartbeat.C:
			_ = rc.SetWriteDeadline(time.Now().Add(60 * time.Second))
			fmt.Fprintf(w, ": heartbeat\n\n")
			w.Flush()
		}
	}
}
