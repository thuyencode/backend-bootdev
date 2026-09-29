package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"sync/atomic"
	"unicode/utf8"

	"github.com/thuyencode/backend-bootdev/learn-http-servers/internals/filter"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

const PORT = "8080"

func main() {
	apiCfg := apiConfig{fileServerHits: atomic.Int32{}}
	mux := http.NewServeMux()
	server := http.Server{Handler: mux, Addr: ":" + PORT}

	mux.Handle(
		"/app/",
		apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))),
	)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprint(w, http.StatusText(http.StatusOK))
		if err != nil {
			slog.Error("Failed to write response body", "err", err)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	})

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, `
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, apiCfg.fileServerHits.Load())
		if err != nil {
			slog.Error("Failed to write response body", "err", err)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.fileServerHits.Store(0)
	})

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var chirp struct {
			Body string `json:"body"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&chirp); err != nil {
			slog.Error("Failed to decode request body", "err", err)
			http.Error(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		if utf8.RuneCountInString(chirp.Body) > 140 {
			http.Error(w, `{"error":"Chirp is too long"}`, http.StatusBadRequest)
			return
		}

		_, err := fmt.Fprintf(w, `{"cleaned_body":%q}`, filter.Censor(chirp.Body))
		if err != nil {
			slog.Error("Failed to write response body", "err", err)
			http.Error(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}
	})

	fmt.Printf("Server is listening on localhost:%s\n", PORT)
	log.Fatal(server.ListenAndServe())
}
