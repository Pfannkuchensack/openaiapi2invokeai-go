package invoke

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// WaitForCompletion connects via Socket.IO to get real-time completion events.
// Falls back to polling if the socket connection fails.
func (c *Client) WaitForCompletion(ctx context.Context, batchID string, itemID int) (*QueueItemStatus, error) {
	status, err := c.waitViaSocketIO(ctx, batchID, itemID)
	if err != nil {
		c.log.Warn("socket.io wait failed, falling back to polling", "error", err)
		return c.PollUntilComplete(ctx, itemID, 2*time.Second)
	}
	return status, nil
}

func (c *Client) waitViaSocketIO(ctx context.Context, batchID string, itemID int) (*QueueItemStatus, error) {
	wsURL, err := c.socketIOURL()
	if err != nil {
		return nil, err
	}

	c.log.Debug("socket.io connecting", "url", wsURL)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()

	// 1. Receive Engine.IO OPEN packet: 0{"sid":"...","upgrades":[],...}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read EIO open: %w", err)
	}
	if len(msg) == 0 || msg[0] != '0' {
		return nil, fmt.Errorf("expected EIO open, got: %s", truncate(msg, 80))
	}
	c.log.Debug("socket.io EIO open", "data", truncate(msg[1:], 80))

	// 2. Send Socket.IO CONNECT to default namespace "/"
	// Format: "40" (EIO message + SIO connect)
	if err := conn.WriteMessage(websocket.TextMessage, []byte("40")); err != nil {
		return nil, fmt.Errorf("send SIO connect: %w", err)
	}

	// 3. Receive Socket.IO CONNECT ack: 40{"sid":"..."}
	_, msg, err = conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read SIO connect ack: %w", err)
	}
	if len(msg) < 2 || msg[0] != '4' || msg[1] != '0' {
		return nil, fmt.Errorf("expected SIO connect ack (40...), got: %s", truncate(msg, 80))
	}
	c.log.Debug("socket.io namespace connected")

	// 4. Emit "subscribe_queue" event: 42["subscribe_queue",{"queue_id":"default"}]
	subscribeEvent := `42["subscribe_queue",{"queue_id":"default"}]`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(subscribeEvent)); err != nil {
		return nil, fmt.Errorf("emit subscribe_queue: %w", err)
	}
	c.log.Debug("socket.io subscribed to queue")

	// 5. Listen for queue_item_status_changed events
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, msg, err = conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("read event: %w", err)
		}

		msgStr := string(msg)
		if len(msgStr) == 0 {
			continue
		}

		// Handle Engine.IO PING (2) → respond PONG (3)
		if msgStr[0] == '2' {
			conn.WriteMessage(websocket.TextMessage, []byte("3"))
			continue
		}

		// Only process Socket.IO EVENT messages: "42[...]"
		// 4 = EIO message, 2 = SIO event
		if len(msgStr) < 3 || msgStr[0] != '4' || msgStr[1] != '2' {
			continue
		}

		// Parse: ["event_name", {data}]
		var eventArr []json.RawMessage
		if err := json.Unmarshal([]byte(msgStr[2:]), &eventArr); err != nil {
			continue
		}
		if len(eventArr) < 2 {
			continue
		}

		var eventName string
		if err := json.Unmarshal(eventArr[0], &eventName); err != nil {
			continue
		}

		// We only care about queue_item_status_changed
		if eventName != "queue_item_status_changed" {
			continue
		}

		var eventData struct {
			ItemID  int    `json:"item_id"`
			BatchID string `json:"batch_id"`
			Status  string `json:"status"`
			Error   string `json:"error_message"`
		}
		if err := json.Unmarshal(eventArr[1], &eventData); err != nil {
			c.log.Debug("failed to parse status event", "error", err)
			continue
		}

		// Filter for our item
		if eventData.ItemID != itemID && eventData.BatchID != batchID {
			continue
		}

		c.log.Debug("queue status event", "item_id", eventData.ItemID, "status", eventData.Status)

		switch eventData.Status {
		case "completed", "failed", "canceled", "cancelled":
			return &QueueItemStatus{
				ItemID:  itemID,
				Status:  eventData.Status,
				BatchID: batchID,
				Error:   eventData.Error,
			}, nil
		}
	}
}

func (c *Client) socketIOURL() (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	return fmt.Sprintf("%s://%s/ws/socket.io/?EIO=4&transport=websocket", scheme, u.Host), nil
}

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
