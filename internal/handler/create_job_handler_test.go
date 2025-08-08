package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexisvisco/koyebtests/internal/types"
	"github.com/alexisvisco/koyebtests/mocks"
)

func TestCreateJob(t *testing.T) {
	jobService := mocks.NewJobService(t)
	expectedURL := "http://job.example.com"

	jobService.EXPECT().
		CreateJob("http://example.com", true).
		Return(&types.CreateJobOutput{URL: expectedURL}, nil)

	handler := CreateJob(jobService)

	req := httptest.NewRequest(http.MethodPost, "/create-job", strings.NewReader(`{"url":"http://example.com","is_script":true}`))
	w := httptest.NewRecorder()

	handler(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var resp CreateJobResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.URL != expectedURL {
		t.Fatalf("expected URL %s, got %s", expectedURL, resp.URL)
	}
}
