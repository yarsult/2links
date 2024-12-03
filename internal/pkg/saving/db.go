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

	queryShowLink = `SELECT short_url, original_url, created_at
					FROM links
					WHERE user_id = $1
					ORDER BY created_at DESC;`

	queryAddUser = `INSERT INTO users (telegram_id) VALUES ($1);`

	queryAddLink = `INSERT INTO links (user_id, original_url, short_url, expires_at) VALUES ($1, $2, $3, $4);`

	queryAddClick = `INSERT INTO clicks (link_id, ip_address, user_agent) VALUES ($1, '$2', '$3');`

	queryAddFeedback = `INSERT INTO feedback (user_id, grade) VALUES ($1, $2);`

	queryAddReview = `INSERT INTO reviews (user_id, review) VALUES ($1, $2);`

	QuerySelectLink = `SELECT original_url FROM links WHERE short_url = $1`

	queryFindDB = `SELECT COUNT(*) = 1 FROM pg_catalog.pg_database WHERE datname = $1`
)

type DB struct {
	Db *sql.DB
}

type Link struct {
	ShortURL    string
	OriginalURL string
	CreatedAt   time.Time
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

func SaveLink(Database *DB, id int64, orig string, short string, exp time.Time) error {
	_, err := Database.Db.Exec(queryAddLink, id, orig, short, exp)
	if err != nil {
		log.Println("Error saving link:", err)
		return err
	}
	return nil
}

func UserInBase(Database *DB, id int64) bool {
	var exists bool
	err := Database.Db.QueryRow(queryCheckUser, id).Scan(&exists)
	if err != nil {
		log.Println("Error finding user:", err)
		return false
	}
	return exists
}

func LinkInBase(Database *DB, link string) bool {
	var exists bool
	err := Database.Db.QueryRow(queryUniqueLink, link).Scan(&exists)
	if err != nil {
		log.Println("Error finding link:", err)
		return false
	}
	return exists
}

func AddUser(Database *DB, id int64) error {
	_, err := Database.Db.Exec(queryAddUser, id)
	if err != nil {
		log.Println("Error saving user:", err)
		return err
	}
	return nil
}

func SaveFeedback(Database *DB, ans int, id int64) error {
	_, err := Database.Db.Exec(queryAddFeedback, id, ans)
	if err != nil {
		log.Println("Error saving feedback:", err)
		return err
	}
	return nil
}

func SaveReview(Database *DB, ans string, id int64) error {
	_, err := Database.Db.Exec(queryAddReview, id, ans)
	if err != nil {
		log.Println("Error saving review:", err)
		return err
	}
	return nil
}

func ShowMyLinks(Database *DB, id int64) ([]Link, error) {
	rows, err := Database.Db.Query(queryShowLink, id)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch user links: %v", err)
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.ShortURL, &link.OriginalURL, &link.CreatedAt); err != nil {
			return nil, fmt.Errorf("Failed to scan row: %v", err)
		}
		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("Row iteration error: %v", err)
	}

	return links, nil
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
