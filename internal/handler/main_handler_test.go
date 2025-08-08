package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/alexisvisco/koyebtests/mocks"
)

func TestMainHandlerReverseProxy(t *testing.T) {
	backendReceived := make(chan http.Header, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendReceived <- r.Header
		io.WriteString(w, "ok")
	}))
	defer backend.Close()

	backendPort, err := strconv.Atoi(strings.Split(backend.Listener.Addr().String(), ":")[1])
	if err != nil {
		t.Fatalf("failed to parse backend port: %v", err)
	}

	jobID := "jobid"
	host := "example.com"

	jobService := mocks.NewJobService(t)
	jobService.EXPECT().
		GetJobPort(jobID).
		Return(backendPort, true)

	mainHandler := Main(MainParams{Host: host, ApiHost: "api." + host, JobService: jobService})

	server := httptest.NewServer(mainHandler)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Host = jobID + "." + host

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("unexpected body: %s", string(body))
	}

	headers := <-backendReceived
	if headers.Get("X-Original-Subdomain") != jobID {
		t.Fatalf("expected X-Original-Subdomain %s, got %s", jobID, headers.Get("X-Original-Subdomain"))
	}
	if headers.Get("X-Original-Host") != jobID+"."+host {
		t.Fatalf("expected X-Original-Host %s, got %s", jobID+"."+host, headers.Get("X-Original-Host"))
	}
}
