package invoke

import "time"

// EnqueueBatchRequest is the payload for POST /api/v1/queue/{queue_id}/enqueue_batch
type EnqueueBatchRequest struct {
	Batch  Batch `json:"batch"`
	Prepend bool `json:"prepend,omitempty"`
}

type Batch struct {
	Graph Graph  `json:"graph"`
	Runs  int    `json:"runs,omitempty"`
	Data  []any  `json:"data,omitempty"`
}

// Graph is an InvokeAI workflow graph (opaque JSON structure we parameterize externally).
type Graph map[string]any

// EnqueueBatchResponse is the response from enqueue_batch.
type EnqueueBatchResponse struct {
	QueueID   string `json:"queue_id"`
	Enqueued  int    `json:"enqueued"`
	Requested int    `json:"requested"`
	Priority  int    `json:"priority"`
	Batch     struct {
		BatchID string `json:"batch_id"`
	} `json:"batch"`
	ItemIDs []int `json:"item_ids"`
}

// QueueItem represents a single queued invocation.
type QueueItem struct {
	ItemID    int    `json:"item_id"`
	BatchID   string `json:"batch_id"`
	SessionID string `json:"session_id"`
	Status    string `json:"status"` // pending, in_progress, completed, failed, canceled
}

// QueueItemStatus is the response from GET /api/v1/queue/{queue_id}/items/{item_id}
type QueueItemStatus struct {
	ItemID    int       `json:"item_id"`
	BatchID   string    `json:"batch_id"`
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InvocationOutput represents an output from a completed invocation.
type InvocationOutput struct {
	Type  string      `json:"type"`
	Image *ImageField `json:"image,omitempty"`
}

// ImageField is a reference to a generated image.
type ImageField struct {
	ImageName string `json:"image_name"`
}

// QueueItemDetail is the full response from GET /api/v1/queue/{queue_id}/i/{item_id}
type QueueItemDetail struct {
	ItemID      int    `json:"item_id"`
	Status      string `json:"status"`
	BatchID     string `json:"batch_id"`
	SessionID   string `json:"session_id"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Session     struct {
		ID      string                      `json:"id"`
		Graph   Graph                       `json:"graph"`
		Results map[string]InvocationOutput `json:"results"`
	} `json:"session"`
}

// SocketEvent is a WebSocket event from InvokeAI.
type SocketEvent struct {
	Event string         `json:"event"`
	Data  map[string]any `json:"data"`
}
