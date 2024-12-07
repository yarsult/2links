package server

import (
	"2links/internal/pkg/saving"
	"database/sql"
	"log"
	"net/http"
)

type Server struct {
}

func NewServer(db *sql.DB, url string) *Server {
	return &Server{}
}

func (s *Server) Start(port string, db *sql.DB) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.handleRedirect(w, r, db)
	})

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	shortCode := r.URL.Path[1:]
	if shortCode == "" {
		http.NotFound(w, r)
		return
	}

	originalURL, err := saving.GetOriginalURL(db, shortCode)
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
