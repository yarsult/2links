package bot

import (
	"2links/internal/pkg/saving"
	"2links/internal/pkg/shortener"
	"log"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var userStates sync.Map

func StartBot() {

	err := godotenv.Load()
	if err != nil {
		log.Printf("ENVs were loaded not straightly")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")

	url := os.Getenv("MY_DOMAIN")
	if token == "" {
		log.Panic("TELEGRAM_BOT_TOKEN is not set")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
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

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Сократить ссылку"), tgbotapi.NewKeyboardButton("Получить помощь")),
	)

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message != nil {
			var msg tgbotapi.MessageConfig
			chat_id := update.Message.Chat.ID
			state, ok := userStates.Load(chat_id)
			switch update.Message.Text {
			case "/start":
				if !saving.UserInBase(db, chat_id) {
					err = saving.AddUser(db, chat_id)
					if err != nil {
						log.Printf("Не удалось сохранить пользователя")
					}
				}
				msg = tgbotapi.NewMessage(chat_id, "Привет! Я бот для сокращения ссылок 2links")
				msg.ReplyMarkup = keyboard
			case "/help", "Получить помощь":
				msg = tgbotapi.NewMessage(chat_id, "Я могу помочь с сокращением ссылок:\n/start - Запустить\n/help - Узнать, что я умею")
			case "Сократить ссылку":
				msg = tgbotapi.NewMessage(chat_id, "Введи ссылку - и я сокращу её")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				userStates.Store(chat_id, "awaiting_link")
			default:

				if ok && state == "awaiting_link" {
					longLink := update.Message.Text
					if shortener.CheckValidacy(longLink) {
						shortenedLink := url + shortener.СreateShortLink(db, chat_id, longLink)
						msg = tgbotapi.NewMessage(chat_id, "Вот твоя сокращённая ссылка: "+shortenedLink)
						userStates.Delete(chat_id)
					} else {
						msg = tgbotapi.NewMessage(chat_id, "Твоя ссылка не валидна, попробуй другую")

					}

				} else {
					msg = tgbotapi.NewMessage(chat_id, "Такого не знаю(")
				}
			}
			if state == nil {
				msg.ReplyMarkup = keyboard
			}

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending message: %v", err)
			}
		}
	}
}
