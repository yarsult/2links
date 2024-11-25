package bot

import (
	"2links/internal/shortener"
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
		log.Panic("Error loading .env file")
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
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			var msg tgbotapi.MessageConfig
			state, ok := userStates.Load(update.Message.Chat.ID)
			switch update.Message.Text {
			case "/start":
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! Я бот для сокращения ссылок 2links")
				msg.ReplyMarkup = keyboard
			case "/help", "Получить помощь":
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Я могу помочь с сокращением ссылок:\n/start - Запустить\n/help - Узнать, что я умею")
			case "Сократить ссылку":
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Введи ссылку - и я сокращу её")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				userStates.Store(update.Message.Chat.ID, "awaiting_link")
			default:

				if ok && state == "awaiting_link" {
					longLink := update.Message.Text
					if shortener.CheckValidacy(longLink) {
						shortenedLink := url + shortener.СreateShortLink()
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вот твоя сокращённая ссылка: "+shortenedLink)
						userStates.Delete(update.Message.Chat.ID)
					} else {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Твоя ссылка не валидна, попробуй другую")

					}

				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Такого не знаю(")
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
