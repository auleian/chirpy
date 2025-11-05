package main

import (
	"CHIRPY/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries 
	platform string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	}) 
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hits := cfg.fileserverHits.Load()
	html := fmt.Sprintf(`<html>
							<body>
								<h1>Welcome, Chirpy Admin</h1>
								<p>Chirpy has been visited %d times!</p>
							</body>
						</html>`,
						hits)
	w.Write([]byte(html))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.db.DeleteAllUsers(r.Context())
    if err != nil {
        http.Error(w, "failed to delete users", http.StatusInternalServerError)
        return
    }

	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main(){
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	apiCfg := &apiConfig{
		db: dbQueries,  
		platform: platform,
	}

	mux := http.NewServeMux()

	server := http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request){
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, req *http.Request) {
		type requestBody struct {
			Body   string `json:"body"`
			UserID string `json:"user_id"`
		}

		var request requestBody
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Something went wrong"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		if len(request.Body) > 140 {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Chirp is too long"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		if request.Body == "" {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Chirp body cannot be empty"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		// Parse the user_id UUID
		userID, err := uuid.Parse(request.UserID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Invalid user_id"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		// Create chirp - only pass body and user_id since SQL generates the rest
		chirp, err := apiCfg.db.CreateChirp(req.Context(), database.CreateChirpParams{
			Body:   request.Body,
			UserID: userID,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			resp, _ := json.Marshal(map[string]string{"error": "Failed to create chirp"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		type responseBody struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    string    `json:"user_id"`
		}

		w.WriteHeader(http.StatusCreated)
		resp, _ := json.Marshal(responseBody{
			ID:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, req *http.Request) {
		
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, req *http.Request) {
		type requestBody struct {
			Email string `json:"email"`
		}

		// Decode JSON from request
		var request requestBody
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Invalid request body"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		// Validate email is not empty
		if request.Email == "" {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Email cannot be empty"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		// Create user in database - only pass email since SQL generates id and timestamps
		user, err := apiCfg.db.CreateUser(req.Context(), request.Email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			resp, _ := json.Marshal(map[string]string{"error": "Failed to create user"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		// Response struct
		type responseBody struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
		}

		// Respond with created user
		w.WriteHeader(http.StatusCreated) // 201
		resp, _ := json.Marshal(responseBody{
			ID:        user.ID.String(),
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})

	log.Printf("Server started on port: 8080")
	log.Fatal(server.ListenAndServe()) 
}