package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func main(){
	apiCfg := &apiConfig{}
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
		
	})


	log.Printf("Server started on port: 8080")
	log.Fatal(server.ListenAndServe()) 
}