package saving

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

const (
	queryCreateDB = `
CREATE DATABASE shortlinks;`

	queryCreateTables = `

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,               
    telegram_id BIGINT UNIQUE NOT NULL   
);


CREATE TABLE IF NOT EXISTS links (
    id SERIAL PRIMARY KEY,                
    user_id INTEGER NOT NULL,             
    original_url TEXT NOT NULL,           
    short_url VARCHAR(255) UNIQUE NOT NULL, 
    expires_at TIMESTAMP,                 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);


CREATE TABLE IF NOT EXISTS clicks (
    id SERIAL PRIMARY KEY,                 
    link_id INTEGER NOT NULL,              
    clicked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    ip_address VARCHAR(45),                
    user_agent TEXT,                        
    FOREIGN KEY (link_id) REFERENCES links(id) ON DELETE CASCADE 
);

CREATE TABLE IF NOT EXISTS suspect_links (
    id SERIAL PRIMARY KEY,                         
    short_url VARCHAR(255) UNIQUE NOT NULL, 
	FOREIGN KEY (id) REFERENCES links(id) ON DELETE CASCADE
);


CREATE TABLE IF NOT EXISTS reviews (
    id SERIAL PRIMARY KEY,               
    user_id INTEGER NOT NULL,   
	review TEXT NOT NULL,                    
    FOREIGN KEY (user_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS feedback (
    id SERIAL PRIMARY KEY,               
    user_id INTEGER NOT NULL,   
	grade INTEGER NOT NULL,                    
    FOREIGN KEY (user_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);`

	queryCheckUser = `SELECT EXISTS (SELECT 1 FROM users WHERE telegram_id = $1)`

	queryUniqueLink = `SELECT EXISTS (SELECT 1 FROM links WHERE short_url = $1);`

	queryShowLink = `SELECT short_url, original_url, created_at, expires_at
					FROM links
					WHERE user_id = $1
					ORDER BY created_at DESC;`

	queryAddUser = `INSERT INTO users (telegram_id) VALUES ($1);`

	queryAddLink = `INSERT INTO links (user_id, original_url, short_url, expires_at) VALUES ($1, $2, $3, $4);`

	queryAddClick = `INSERT INTO clicks (link_id, ip_address, user_agent) VALUES ($1, $2, $3);`

	queryAddSuspect = `INSERT INTO suspect_links (id, short_url) VALUES ($1, $2);`

	queryAddFeedback = `DELETE FROM feedback 
						WHERE user_id = $1;
						INSERT INTO feedback (user_id, grade) VALUES ($1, $2);`

	queryAddReview = `INSERT INTO reviews (user_id, review) VALUES ($1, $2);`

	querySelectLink = `SELECT id FROM links WHERE short_url = $1`

	queryFindDB = `SELECT COUNT(*) = 1 FROM pg_catalog.pg_database WHERE datname = $1`

	queryGetURL = `SELECT original_url, id, expires_at FROM links WHERE short_url = $1`

	queryGetClicks = `
						SELECT l.short_url, l.original_url, COUNT(c.id)
						FROM links l
						LEFT JOIN clicks c ON l.id = c.link_id
						WHERE l.user_id = $1
						GROUP BY l.short_url, l.original_url`

	queryDeleteLink = `DELETE FROM links WHERE short_url = $1`

	queryUpdateExp = `UPDATE links SET expires_at = $1 WHERE short_url = $2 AND user_id = $3`

	queryGetSuspect = `SELECT sl.short_url, l.original_url
						FROM suspect_links sl
						JOIN links l ON sl.id = l.id;`

	queryAllUsers = `SELECT COUNT(*) FROM users`

	queryAllLinks = `SELECT COUNT(*) FROM links`

	queryAllClicks = `SELECT COUNT(*) FROM clicks`

	queryAllExpired = `SELECT COUNT(*) FROM links WHERE expires_at < NOW()`

	queryGetReviews = `SELECT review FROM reviews ORDER BY id DESC LIMIT 5`

	queryGetGrade = `SELECT AVG(grade) FROM feedback`
)

type DB struct {
	Db *sql.DB
}

type Link struct {
	ShortURL    string
	OriginalURL string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

func CreateDB(dbtype string, conn string) (*DB, error) {
	db, err := sql.Open(dbtype, conn)
	if err != nil {
		log.Fatal("Error connecting to database")
	}

	_, err = db.Exec(queryCreateTables)
	if err != nil {
		log.Fatal("Error creating tables", err)
	}

	return &DB{Db: db}, nil
}

func SaveLink(db *sql.DB, id int64, orig string, short string, exp time.Time) error {
	_, err := db.Exec(queryAddLink, id, orig, short, exp)
	if err != nil {
		log.Println("Error saving link:", err)
		return err
	}

	return nil
}

func UserInBase(db *sql.DB, id int64) bool {
	var exists bool
	err := db.QueryRow(queryCheckUser, id).Scan(&exists)
	if err != nil {
		log.Println("Error finding user:", err)
		return false
	}

	return exists
}

func LinkInBase(db *sql.DB, link string) bool {
	var exists bool
	err := db.QueryRow(queryUniqueLink, link).Scan(&exists)
	if err != nil {
		log.Println("Error finding link:", err)
		return false
	}

	return exists
}

func AddUser(db *sql.DB, id int64) error {
	_, err := db.Exec(queryAddUser, id)
	if err != nil {
		log.Println("Error saving user:", err)
		return err
	}

	return nil
}

func SaveFeedback(db *sql.DB, ans int, id int64) error {
	_, err := db.Exec(queryAddFeedback, id, ans)
	if err != nil {
		log.Println("Error saving feedback:", err)
		return err
	}

	return nil
}

func SaveReview(db *sql.DB, ans string, id int64) error {
	_, err := db.Exec(queryAddReview, id, ans)
	if err != nil {
		log.Println("Error saving review:", err)
		return err
	}
	return nil
}

func FindLink(db *sql.DB, link string) (int, error) {
	var linkID int
	err := db.QueryRow(querySelectLink, link).Scan(&linkID)
	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("Database query error: %w", err)
	}

	return linkID, nil
}

func SuspectLink(db *sql.DB, id int, link string) error {
	_, err := db.Exec(queryAddSuspect, id, link)
	if err != nil {
		log.Println("Error saving review:", err)
		return err
	}

	return nil
}

func ShowMyLinks(db *sql.DB, id int64) ([]Link, error) {
	rows, err := db.Query(queryShowLink, id)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch user links: %v", err)
	}

	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.ShortURL, &link.OriginalURL, &link.CreatedAt, &link.ExpiresAt); err != nil {
			return nil, fmt.Errorf("Failed to scan row: %v", err)
		}
		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Row iteration error: %v", err)
	}

	return links, nil
}

func DeleteLink(db *sql.DB, shortCode string) error {
	result, err := db.Exec(queryDeleteLink, shortCode)
	if err != nil {
		return fmt.Errorf("Error deleting link: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("No link found to delete")
	}

	return nil
}

func SaveClick(db *sql.DB, linkID int, ipAddress, userAgent string) error {
	_, err := db.Exec(queryAddClick, linkID, ipAddress, userAgent)
	if err != nil {
		return fmt.Errorf("Failed to save click: %w", err)
	}

	return nil
}

func GetClicksByUser(db *sql.DB, userID int64) (map[string]struct {
	OriginalURL string
	Clicks      int
}, error) {
	rows, err := db.Query(queryGetClicks, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch clicks: %w", err)
	}

	defer rows.Close()

	clicks := make(map[string]struct {
		OriginalURL string
		Clicks      int
	})
	for rows.Next() {
		var shortURL, originalURL string
		var count int
		if err := rows.Scan(&shortURL, &originalURL, &count); err != nil {
			return nil, fmt.Errorf("Failed to scan row: %w", err)
		}
		clicks[shortURL] = struct {
			OriginalURL string
			Clicks      int
		}{
			OriginalURL: originalURL,
			Clicks:      count,
		}
	}

	return clicks, nil
}

func GetOriginalURL(db *sql.DB, shortLink string) (string, int, time.Time, error) {
	var originalURL string
	var linkID int
	var expires_at time.Time
	err := db.QueryRow(queryGetURL, shortLink).Scan(&originalURL, &linkID, &expires_at)
	if err == sql.ErrNoRows {
		return "", 0, expires_at, fmt.Errorf("Short link not found")
	} else if err != nil {
		return "", 0, expires_at, fmt.Errorf("Database query error: %w", err)
	}

	return originalURL, linkID, expires_at, nil
}

func UpdateLinkExpiry(db *sql.DB, userID int64, shortURL string, newExpiry time.Time) error {
	result, err := db.Exec(queryUpdateExp, newExpiry, shortURL, userID)
	if err != nil {
		return fmt.Errorf("Error updating link expiry: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("No link found or not authorized")
	}

	return nil
}

func GetReviews(db *sql.DB) ([]string, error) {
	rows, err := db.Query(queryGetReviews)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch user links: %v", err)
	}

	defer rows.Close()

	var reviews []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, fmt.Errorf("Failed to scan row: %v", err)
		}
		reviews = append(reviews, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Row iteration error: %v", err)
	}

	return reviews, nil
}

func GetGrade(db *sql.DB) (float32, error) {
	var grade float32
	err := db.QueryRow(queryGetGrade).Scan(&grade)
	if err != nil {
		log.Println("Error counting grade:", err)
		return 0, err
	}

	return grade, nil
}

func GetSuspectLinks(db *sql.DB) ([]Link, error) {
	rows, err := db.Query(queryGetSuspect)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var links []Link

	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.ShortURL, &link.OriginalURL); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, nil
}

func GetSummaryStatistics(db *sql.DB) (struct {
	Users        int
	Links        int
	Clicks       int
	ExpiredLinks int
}, error) {
	var stats struct {
		Users        int
		Links        int
		Clicks       int
		ExpiredLinks int
	}

	err := db.QueryRow(queryAllUsers).Scan(&stats.Users)
	if err != nil {
		return stats, err
	}

	err = db.QueryRow(queryAllLinks).Scan(&stats.Links)
	if err != nil {
		return stats, err
	}

	err = db.QueryRow(queryAllClicks).Scan(&stats.Clicks)
	if err != nil {
		return stats, err
	}

	err = db.QueryRow(queryAllExpired).Scan(&stats.ExpiredLinks)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func DeleteSuspectLink(db *sql.DB, link string) error {
	_, err := db.Exec(queryDeleteLink, link)
	return err
}

func DropDatabase(dbName string, dbtype string, postgres string) error {
	db, err := sql.Open(dbtype, postgres)
	if err != nil {
		return fmt.Errorf("Failed to connect to postgres: %v", err)
	}

	defer db.Close()
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName)
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("Failed to drop database: %v", err)
	}

	log.Printf("Database %s successfully dropped.", dbName)
	return nil
}

func CreateDatabaseIfNotExists(dbName string, dbtype string, postgres string) error {
	db, err := sql.Open(dbtype, postgres)
	if err != nil {
		return fmt.Errorf("Failed to open default database: %v", err)
	}

	defer db.Close()

	var exists bool
	err = db.QueryRow(queryFindDB, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Failed to check database existence: %v", err)
	}

	if !exists {
		_, err := db.Exec(queryCreateDB)
		if err != nil {
			return fmt.Errorf("Failed to create database: %v", err)
		}
		log.Printf("Database %s created successfully.", dbName)
	} else {
		log.Printf("Database %s already exists.", dbName)
	}

	return nil
}
