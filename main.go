package main

import (
	"CHIRPY/internal/database"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

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

	}else{

	}
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func main(){
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		errors.New("Database failed to open")
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

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter,req *http.Request){
		
		type requestBody struct {
			Body string `json:"body"`
		}

		// Step 2: Decode JSON from request
		var request requestBody
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Something went wrong"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		// Step 3: Validate chirp length
		if len(request.Body) > 140 {
			w.WriteHeader(http.StatusBadRequest)
			resp, _ := json.Marshal(map[string]string{"error": "Chirp is too long"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		// Step 4: Respond success
		w.WriteHeader(http.StatusOK)
		resp, _ := json.Marshal(map[string]bool{"valid": true})
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
		
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter,req *http.Request){
	
	})


	log.Printf("Server started on port: 8080")
	log.Fatal(server.ListenAndServe()) 
}