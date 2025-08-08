package handler

import (
	"encoding/json"
	"net/http"

	"github.com/alexisvisco/koyebtests/internal/types"
)

type CreateJobRequest struct {
	URL      string `json:"url"`
	IsScript bool   `json:"is_script"`
}

type CreateJobResponse struct {
	URL string `json:"url"`
}

func CreateJob(service types.JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid_json", http.StatusBadRequest)
			return
		}

		if !isValidURL(req.URL) {
			http.Error(w, "invalid_url", http.StatusBadRequest)
			return
		}

		job, err := service.CreateJob(req.URL, req.IsScript)
		if err != nil {
			http.Error(w, "failed_create_job", http.StatusInternalServerError)
			return
		}

		response := CreateJobResponse{
			URL: job.URL,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
