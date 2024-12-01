package saving

import (
	"database/sql"
	"fmt"
	"log"
	"os"
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

	queryAddUser = `INSERT INTO users (telegram_id) VALUES ($1);`

	queryAddLink = `INSERT INTO links (user_id, original_url, short_url, expires_at) VALUES ($1, $2, $3, $4);`

	queryAddClick = `INSERT INTO clicks (link_id, ip_address, user_agent) VALUES ($1, '$2', '$3');`

	queryAddFeedback = `INSERT INTO feedback (user_id, grade) VALUES ($1, $2);`

	queryAddReview = `INSERT INTO reviews (user_id, review) VALUES ($1, $2);`
)

type DB struct {
	Db *sql.DB
}

func CreateDB() (*DB, error) {
	db, err := sql.Open(os.Getenv("DB"), os.Getenv("POSTGRES"))
	if err != nil {
		_, err = db.Exec(queryCreateDB)
		if err == nil {
			log.Println("Database is being created")
			db, err = sql.Open(os.Getenv("DB"), os.Getenv("POSTGRES"))
		} else {
			log.Fatal("Error connecting to database", err)
		}
	}

	_, err = db.Exec(queryCreateTables)
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

func DropDatabase(dbName string) error {
	connStr := "host=postgres port=5432 user=user password=password dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
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

func CreateDatabaseIfNotExists(dbName string) error {
	connStr := "host=postgres port=5432 user=user password=password dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	defer db.Close()

	var exists bool
	query := "SELECT COUNT(*) = 1 FROM pg_catalog.pg_database WHERE datname = $1"
	err = db.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Failed to check database existence: %v", err)
	}

	if !exists {
		_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			return fmt.Errorf("Failed to create database: %v", err)
		}
		log.Printf("Database %s created successfully.", dbName)
	} else {
		log.Printf("Database %s already exists.", dbName)
	}

	return nil
}
