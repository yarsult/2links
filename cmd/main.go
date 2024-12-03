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

	url := os.Getenv("MY_DOMAIN")
	if token == "" {
		log.Panic("TELEGRAM_BOT_TOKEN is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	domain := os.Getenv("MY_DOMAIN")
	if domain == "" {
		domain = "http://localhost:" + port
	}

	err = saving.CreateDatabaseIfNotExists("shortlinks")
	if err != nil {
		log.Panic(err)
	}
	db, err := saving.CreateDB()
	if err != nil {
		log.Panic("Error connecting to database")
	}

	defer db.Db.Close()
	// db.Db.Close()
	// saving.DropDatabase("shortlinks")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		srv := server.NewServer(db.Db, domain)
		log.Printf("Starting server on port %s", port)
		srv.Start(port)
	}()

	go func() {
		defer wg.Done()
		log.Println("Starting Telegram bot")
		bot.StartBot(url, db, token)
	}()

	wg.Wait()
}
