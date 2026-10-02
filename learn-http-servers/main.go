package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"sync/atomic"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/thuyencode/backend-bootdev/learn-http-servers/internals/auth"
	"github.com/thuyencode/backend-bootdev/learn-http-servers/internals/database"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries      *database.Queries
	jwtSecret      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

const PORT = "8080"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		log.Fatal(`The "JWT_SECRET" enviroment variable is required to not be empty`)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
		dbQueries:      database.New(db),
		jwtSecret:      jwtSecret,
	}
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
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if platform != "dev" {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		if err := apiCfg.dbQueries.PruneUsers(r.Context()); err != nil {
			slog.Error("Failed to prune users table", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		apiCfg.fileServerHits.Store(0)
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var deserialzed struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&deserialzed); err != nil {
			httpError(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusBadRequest)
			return
		}

		parsedEmail, err := mail.ParseAddress(deserialzed.Email)
		if err != nil {
			httpError(w, `{"error":"Invalid email address"}`, http.StatusBadRequest)
			return
		}

		hashedPassword, err := auth.HashPassword(deserialzed.Password)
		if err != nil {
			slog.Error("Failed to hash password", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		newUser, err := apiCfg.dbQueries.CreateUser(
			r.Context(),
			database.CreateUserParams{Email: parsedEmail.Address, HashedPassword: hashedPassword},
		)
		if err != nil {
			if pqErr := pq.As(err, pqerror.UniqueViolation); pqErr != nil {
				httpError(w, `{"error":"Email address already registered"}`, http.StatusBadRequest)
				return
			}

			slog.Error("Failed to insert new db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		resBody, err := json.Marshal(newUser)
		if err != nil {
			slog.Error("Failed to marshal db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if _, err = w.Write(resBody); err != nil {
			slog.Error("Failed to write response body", "err", err)
			return
		}
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		type responseBody struct {
			database.User
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}

		var deserialzed struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&deserialzed); err != nil {
			httpError(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusBadRequest)
			return
		}

		parsedEmail, err := mail.ParseAddress(deserialzed.Email)
		if err != nil {
			httpError(w, `{"error":"Invalid email address"}`, http.StatusBadRequest)
			return
		}

		user, _ := apiCfg.dbQueries.SelectUserByEmail(r.Context(), parsedEmail.Address)
		if user == (database.User{}) {
			httpError(w, `{"error":"Invalid email address or password"}`, http.StatusUnauthorized)
			return
		}

		match, err := auth.CheckPasswordHash(deserialzed.Password, user.HashedPassword)
		if err != nil {
			slog.Error("Failed to hash password", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		if !match {
			httpError(w, `{"error":"Invalid email address or password"}`, http.StatusUnauthorized)
			return
		}

		accessToken, err := auth.MakeJWT(user.ID, apiCfg.jwtSecret, time.Hour)
		if err != nil {
			slog.Error("Failed to create a JWT token", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		refreshToken := auth.MakeRefreshToken()
		_, err = apiCfg.dbQueries.CreateRefreshToken(
			r.Context(),
			database.CreateRefreshTokenParams{
				Token:     refreshToken,
				UserID:    user.ID,
				ExpiresAt: time.Now().AddDate(0, 0, 60),
			},
		)
		if err != nil {
			slog.Error("Failed to insert new db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		v := responseBody{user, accessToken, refreshToken}
		resBody, err := json.Marshal(v)
		if err != nil {
			slog.Error("Failed to marshal a value", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err = w.Write(resBody); err != nil {
			slog.Error("Failed to write response body", "err", err)
			return
		}
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var deserialized struct {
			Body string `json:"body"`
		}

		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			httpError(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusBadRequest)
			return
		}

		userId, err := auth.ValidateJWT(token, apiCfg.jwtSecret)
		if err != nil {
			httpError(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusUnauthorized)
			return
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&deserialized); err != nil {
			httpError(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusBadRequest)
			return
		}

		if utf8.RuneCountInString(deserialized.Body) > 140 {
			httpError(w, `{"error":"Chirp is too long"}`, http.StatusBadRequest)
			return
		}

		_, err = apiCfg.dbQueries.SelectUserById(r.Context(), userId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				httpError(w, `{"error":"User not found"}`, http.StatusNotFound)
				return
			}
			slog.Error("Failed to retrieve db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		newChirp, err := apiCfg.dbQueries.CreateChirp(
			r.Context(),
			database.CreateChirpParams{UserID: userId, Body: deserialized.Body},
		)
		if err != nil {
			slog.Error("Failed to insert new db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		resBody, err := json.Marshal(newChirp)
		if err != nil {
			slog.Error("Failed to marshal db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if _, err = w.Write(resBody); err != nil {
			slog.Error("Failed to write response body", "err", err)
			return
		}
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		chirps, err := apiCfg.dbQueries.SelectChirps(r.Context())
		if err != nil {
			slog.Error("Failed to retrieve db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		resBody, err := json.Marshal(chirps)
		if err != nil {
			slog.Error("Failed to marshal db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err = w.Write(resBody); err != nil {
			slog.Error("Failed to write response body", "err", err)
			return
		}
	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		chirpID, err := uuid.Parse(r.PathValue("chirpID"))
		if err != nil {
			httpError(w, `{"error":"A valid ID is required in the path"}`, http.StatusBadRequest)
			return
		}

		chirp, err := apiCfg.dbQueries.SelectChirp(r.Context(), chirpID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				httpError(w, `{"error":"Chirp not found"}`, http.StatusNotFound)
				return
			}
			slog.Error("Failed to retrieve db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		resBody, err := json.Marshal(chirp)
		if err != nil {
			slog.Error("Failed to marshal db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err = w.Write(resBody); err != nil {
			slog.Error("Failed to write response body", "err", err)
			return
		}
	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		type responseBody struct {
			Token string `json:"token"`
		}

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			httpError(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusBadRequest)
			return
		}

		refreshToken, err := apiCfg.dbQueries.SelectRefreshToken(r.Context(), bearerToken)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				httpError(w, `{"error":"Refresh token not found"}`, http.StatusUnauthorized)
				return
			}
			slog.Error("Failed to retrieve db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		if refreshToken.RevokedAt.Valid && refreshToken.RevokedAt.Time.Before(time.Now()) {
			httpError(w, `{"error":"Refresh token revoked"}`, http.StatusUnauthorized)
			return
		}

		if refreshToken.ExpiresAt.Before(time.Now()) {
			httpError(w, `{"error":"Refresh token expired"}`, http.StatusUnauthorized)
			return
		}

		accessToken, err := auth.MakeJWT(refreshToken.UserID, jwtSecret, time.Hour)
		if err != nil {
			slog.Error("Failed to create a JWT token", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		v := responseBody{accessToken}
		resBody, err := json.Marshal(v)
		if err != nil {
			slog.Error("Failed to marshal a value", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err = w.Write(resBody); err != nil {
			slog.Error("Failed to write response body", "err", err)
			return
		}
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			httpError(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusBadRequest)
			return
		}

		refreshToken, err := apiCfg.dbQueries.SelectRefreshToken(r.Context(), bearerToken)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				httpError(w, `{"error":"Refresh token not found"}`, http.StatusUnauthorized)
				return
			}
			slog.Error("Failed to retrieve db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		if refreshToken.ExpiresAt.Before(time.Now()) {
			httpError(w, `{"error":"Refresh token expired"}`, http.StatusUnauthorized)
			return
		}

		err = apiCfg.dbQueries.RevokeRefreshToken(r.Context(), bearerToken)
		if err != nil {
			slog.Error("Failed to update db record(s)", "err", err)
			httpError(w, `{"error":"Something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	fmt.Printf("Server is listening on localhost:%s\n", PORT)
	log.Fatal(server.ListenAndServe())
}

// This is http.Error but is doesn't override Content-Type header
func httpError(w http.ResponseWriter, error string, code int) {
	h := w.Header()

	h.Del("Content-Length")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, error)
}
