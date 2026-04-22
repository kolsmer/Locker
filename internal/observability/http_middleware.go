package observability

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func requestPathLabel(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if tpl, err := route.GetPathTemplate(); err == nil && tpl != "" {
			return tpl
		}
	}
	if r.URL == nil || r.URL.Path == "" {
		return "unknown"
	}
	return r.URL.Path
}

func ensureRequestID(r *http.Request) string {
	if rid := strings.TrimSpace(r.Header.Get("X-Request-Id")); rid != "" {
		return rid
	}
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := ensureRequestID(r)
		w.Header().Set("X-Request-Id", rid)
		ctx := ContextWithRequestID(r.Context(), rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RecoveryMiddleware(logger *Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					rid, _ := RequestIDFromContext(r.Context())
					logger.Error("panic recovered",
						"request_id", rid,
						"method", r.Method,
						"path", r.URL.Path,
						"panic", rec,
					)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"INTERNAL_ERROR","message":"internal error"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func LoggingMiddleware(logger *Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			rw := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			status := rw.statusCode
			if status == 0 {
				status = http.StatusOK
			}

			rid, _ := RequestIDFromContext(r.Context())
			logger.Info("http_request",
				"request_id", rid,
				"method", r.Method,
				"path", r.URL.Path,
				"route", requestPathLabel(r),
				"status", status,
				"duration_ms", time.Since(started).Milliseconds(),
				"bytes", rw.bytes,
			)
		})
	}
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rw := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		status := rw.statusCode
		if status == 0 {
			status = http.StatusOK
		}

		method := r.Method
		path := requestPathLabel(r)
		statusStr := fmt.Sprintf("%d", status)

		httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
		httpRequestDurationSeconds.WithLabelValues(method, path, statusStr).Observe(time.Since(started).Seconds())
	})
}
