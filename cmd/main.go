package main

import (
	"2links/internal/pkg/bot"
	"2links/internal/pkg/saving"
	"2links/internal/pkg/server"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("ENVs were loaded not straightly")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Panic("TELEGRAM_BOT_TOKEN is not set")
	}

	admToken := os.Getenv("ADMIN_BOT_TOKEN")
	if admToken == "" {
		log.Panic("ADMIN_BOT_TOKEN is not set")
	}

	url := os.Getenv("MY_DOMAIN")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	domain := os.Getenv("MY_DOMAIN")
	if domain == "" {
		domain = "http://localhost:" + port
	}

	dbType := os.Getenv("DB")
	postgresDefault := os.Getenv("POSTGRES_DEFAULT")
	postgresConn := os.Getenv("POSTGRES")
	if dbType == "" || postgresDefault == "" || postgresConn == "" {
		log.Panic("Envs weren't loaded")
	}

	err = saving.CreateDatabaseIfNotExists("shortlinks", dbType, postgresDefault)
	if err != nil {
		log.Panic(err)
	}

	db, err := saving.CreateDB(dbType, postgresConn)
	if err != nil {
		log.Panic("Error connecting to database")
	}

	defer db.Db.Close()
	// to drop db
	// db.Db.Close()
	// saving.DropDatabase("shortlinks", dbType, postgresDefault)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		srv := server.NewServer(db.Db, domain)
		log.Printf("Starting server on port %s", port)
		srv.Start(port, db.Db)
	}()

	go func() {
		defer wg.Done()
		log.Println("Starting Telegram bot")
		bot.StartBot(url, db, token)
	}()

	go func() {
		defer wg.Done()
		log.Println("Starting Admin bot")
		bot.StartAdminBot(admToken, db)
	}()

	wg.Wait()
}
