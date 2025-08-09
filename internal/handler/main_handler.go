package handler

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"

	"github.com/alexisvisco/koyebtests/internal/types"
)

type MainParams struct {
	Host       string
	ApiHost    string
	JobService types.JobService
}

func Main(params MainParams) http.HandlerFunc {
	logger := slog.With("component", "main_handler")
	jobIDPattern := regexp.MustCompile(`^([^.]+)\.` + regexp.QuoteMeta(params.Host) + `$`)
	return func(w http.ResponseWriter, r *http.Request) {
		hostHeader := r.Host
		logger.Info("incoming request", "host", hostHeader, "path", r.URL.Path, "method", r.Method)

		if hostHeader == params.ApiHost {
			http.DefaultServeMux.ServeHTTP(w, r)
			return
		}

		matches := jobIDPattern.FindStringSubmatch(hostHeader)
		if len(matches) > 1 {
			mayJobID := matches[1]
			jobPort, ok := params.JobService.GetJobPort(mayJobID)
			if !ok {
				http.Error(w, "unable_to_find_job", http.StatusNotFound)
				return
			}

			target, err := url.Parse("http://localhost:" + strconv.Itoa(jobPort))
			if err != nil {
				logger.Error("error parsing target URL", "error", err)
				http.Error(w, "internal_server_error", http.StatusInternalServerError)
				return
			}

			proxy := httputil.NewSingleHostReverseProxy(target)

			originalDirector := proxy.Director
			proxy.Director = func(req *http.Request) {
				originalDirector(req)
				req.Header.Set("X-Original-Subdomain", mayJobID)
				req.Header.Set("X-Original-Host", hostHeader)
			}

			proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
				logger.Error("reverse proxy error", "host", hostHeader, "job_id", mayJobID, "error", err)
				http.Error(w, "service_unavailable", http.StatusServiceUnavailable)
			}

			logger.Info("proxying request", "host", hostHeader, "job_id", mayJobID, "target", "localhost:"+strconv.Itoa(jobPort))
			proxy.ServeHTTP(w, r)
			return
		}

		logger.Warn("unknown host", "host", hostHeader)
		http.Error(w, "not_found", http.StatusNotFound)
	}
}
