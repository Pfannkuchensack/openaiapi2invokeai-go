package invoke_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/invoke"
)

// Run with: INVOKE_URL=http://192.168.178.58:9090 go test -v ./internal/invoke/

func getClient(t *testing.T) *invoke.Client {
	t.Helper()
	invokeURL := os.Getenv("INVOKE_URL")
	if invokeURL == "" {
		t.Skip("INVOKE_URL not set, skipping integration test")
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return invoke.NewClient(invokeURL, 5*time.Minute, log)
}

func TestIntegrationFetchExistingImage(t *testing.T) {
	client := getClient(t)
	ctx := context.Background()

	// Fetch detail of a known completed item to get image names
	detail, err := client.GetQueueItemDetail(ctx, 11853)
	if err != nil {
		t.Fatalf("get item detail: %v", err)
	}

	if detail.Status != "completed" {
		t.Fatalf("expected completed, got %s", detail.Status)
	}

	names := client.GetImageNames(detail)
	if len(names) == 0 {
		t.Fatal("no images found in results")
	}

	t.Logf("found %d images", len(names))

	// Fetch first image
	imgData, contentType, err := client.GetImageBytes(ctx, names[0])
	if err != nil {
		t.Fatalf("fetch image: %v", err)
	}
	t.Logf("image %s: %d bytes, content-type: %s", names[0], len(imgData), contentType)
}

func TestIntegrationEnqueueAndWait(t *testing.T) {
	client := getClient(t)

	workflowFile := os.Getenv("INVOKE_TEST_WORKFLOW")
	if workflowFile == "" {
		t.Skip("INVOKE_TEST_WORKFLOW not set, skipping enqueue test")
	}

	data, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("read workflow file: %v", err)
	}

	var graph invoke.Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		t.Fatalf("parse workflow: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Enqueue
	resp, err := client.EnqueueBatch(ctx, graph)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if len(resp.ItemIDs) == 0 {
		t.Fatal("no items enqueued")
	}

	itemID := resp.ItemIDs[0]
	batchID := resp.Batch.BatchID
	t.Logf("enqueued item_id=%d batch_id=%s", itemID, batchID)

	// Wait for completion
	status, err := client.WaitForCompletion(ctx, batchID, itemID)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}

	t.Logf("final status: %s", status.Status)
	if status.Status != "completed" {
		t.Fatalf("expected completed, got %s (error: %s)", status.Status, status.Error)
	}

	// Fetch results
	detail, err := client.GetQueueItemDetail(ctx, itemID)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}

	names := client.GetImageNames(detail)
	t.Logf("produced %d images", len(names))
	for _, name := range names {
		imgData, ct, err := client.GetImageBytes(ctx, name)
		if err != nil {
			t.Errorf("fetch image %s: %v", name, err)
			continue
		}
		t.Logf("  %s: %d bytes (%s)", name, len(imgData), ct)
	}
}
