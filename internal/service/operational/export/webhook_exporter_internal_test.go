package export

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage: lines 185-187 — json.Marshal error in doPost (unmarshalable body)
func TestWebhookExporter_doPost_MarshalError(t *testing.T) {
	we := &WebhookExporter{
		baseURL:     "http://localhost",
		tokenGetter: func() string { return "" },
	}

	// channels cannot be marshaled to JSON
	_, err := we.doPost(context.Background(), "/test", make(chan int), 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal request body")
}

// Coverage: lines 190-192 — http.NewRequestWithContext error (invalid URL)
func TestWebhookExporter_doPost_InvalidURL(t *testing.T) {
	we := &WebhookExporter{
		baseURL:     "http://host\x00invalid",
		tokenGetter: func() string { return "" },
	}

	_, err := we.doPost(context.Background(), "/test", map[string]string{}, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}
