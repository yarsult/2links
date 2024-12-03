package server

import (
	"database/sql"
	"log"
	"net/http"
)

type Server struct {
	DB  *sql.DB
	URL string
}

func NewServer(db *sql.DB, url string) *Server {
	return &Server{
		DB:  db,
		URL: url,
	}
}

func (s *Server) Start(port string) {
	http.HandleFunc("/", s.handleRedirect)

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := r.URL.Path[1:]
	if shortCode == "" {
		http.NotFound(w, r)
		return
	}

	var originalURL string
	err := s.DB.QueryRow("SELECT original_url FROM links WHERE short_url = $1", shortCode).Scan(&originalURL)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !startsWithProtocol(originalURL) {
		originalURL = "http://" + originalURL
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func startsWithProtocol(url string) bool {
	return len(url) >= 7 && (url[:7] == "http://" || len(url) >= 8 && url[:8] == "https://")
}
