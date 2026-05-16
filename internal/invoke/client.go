package invoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client talks to a local InvokeAI instance.
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

func NewClient(baseURL string, timeout time.Duration, log *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log,
	}
}

// EnqueueBatch submits a workflow graph to the default queue.
func (c *Client) EnqueueBatch(ctx context.Context, graph Graph) (*EnqueueBatchResponse, error) {
	reqBody := EnqueueBatchRequest{
		Batch: Batch{
			Graph: graph,
			Runs:  1,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal enqueue request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/queue/default/enqueue_batch", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	c.log.Debug("enqueuing batch", "url", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("enqueue request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enqueue failed (status %d): %s", resp.StatusCode, string(errBody))
	}

	var result EnqueueBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode enqueue response: %w", err)
	}

	c.log.Debug("batch enqueued", "batch_id", result.Batch.BatchID, "items", result.Enqueued)
	return &result, nil
}

// GetQueueItemStatus polls the status of a queue item.
func (c *Client) GetQueueItemStatus(ctx context.Context, itemID int) (*QueueItemStatus, error) {
	url := fmt.Sprintf("%s/api/v1/queue/default/i/%d", c.baseURL, itemID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get queue item: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get queue item failed (status %d): %s", resp.StatusCode, string(errBody))
	}

	var status QueueItemStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode queue item status: %w", err)
	}
	return &status, nil
}

// GetImageBytes fetches the generated image by name.
func (c *Client) GetImageBytes(ctx context.Context, imageName string) ([]byte, string, error) {
	url := fmt.Sprintf("%s/api/v1/images/i/%s/full", c.baseURL, imageName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetch image failed (status %d)", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read image body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// PollUntilComplete polls a queue item until it reaches a terminal state.
// Returns the final status. Use WaitForCompletion for WebSocket-based waiting.
func (c *Client) PollUntilComplete(ctx context.Context, itemID int, interval time.Duration) (*QueueItemStatus, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.GetQueueItemStatus(ctx, itemID)
			if err != nil {
				c.log.Warn("poll error", "item_id", itemID, "error", err)
				continue
			}

			c.log.Debug("poll status", "item_id", itemID, "status", status.Status)

			switch status.Status {
			case "completed", "failed", "canceled":
				return status, nil
			}
		}
	}
}

// GetQueueItemDetail fetches the full detail of a queue item including session results.
func (c *Client) GetQueueItemDetail(ctx context.Context, itemID int) (*QueueItemDetail, error) {
	url := fmt.Sprintf("%s/api/v1/queue/default/i/%d", c.baseURL, itemID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get queue item detail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get queue item detail failed (status %d): %s", resp.StatusCode, string(errBody))
	}

	var detail QueueItemDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("decode queue item detail: %w", err)
	}
	return &detail, nil
}

// GetImageNames extracts all image names from a completed queue item's results.
func (c *Client) GetImageNames(detail *QueueItemDetail) []string {
	var names []string
	for _, out := range detail.Session.Results {
		if out.Image != nil && out.Image.ImageName != "" {
			names = append(names, out.Image.ImageName)
		}
	}
	return names
}
