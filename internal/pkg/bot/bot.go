package bot

import (
	"2links/internal/pkg/saving"
	"2links/internal/pkg/shortener"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	buttonShorten   = "Сократить ссылку"
	buttonMyLinks   = "Мои ссылки"
	buttonComplaint = "Пожаловаться на ссылку"
	buttonFeedback  = "Оставить обратную связь"
	buttonHelp      = "Получить помощь"
)

var userStates sync.Map

func StartBot(url string, db *saving.DB, token string) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonShorten)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonMyLinks)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonComplaint)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonFeedback)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonHelp)),
	)

	question := "Как вам наш сервис?"
	options := []string{"Плохо", "Так себе", "Хорошо", "Здорово", "Супер"}
	isAnonymous := false

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.CallbackQuery != nil {
			callbackData := update.CallbackQuery.Data
			chatID := update.CallbackQuery.Message.Chat.ID
			var message string

			switch {
			case len(callbackData) > 7 && callbackData[:7] == "delete:":
				shortLink := callbackData[7:]
				err := saving.DeleteLink(db.Db, shortLink)
				if err != nil {
					log.Printf("Error deleting link: %v", err)
				}

				message = "Ссылка удалена"
				msg := tgbotapi.NewMessage(chatID, message)
				msg.ReplyMarkup = keyboard
				bot.Send(msg)

			case callbackData == "back":
				message = "Возвращаемся в основное меню"
				msg := tgbotapi.NewMessage(chatID, message)
				msg.ReplyMarkup = keyboard
				bot.Send(msg)

			case len(callbackData) > 7 && callbackData[:7] == "update:":
				shortURL := callbackData[7:]
				userStates.Store(chatID, fmt.Sprintf("awaiting_expiry_%s", shortURL))
				message = fmt.Sprintf("Введите новый срок хранения для ссылки %s в формате DD-MM-YYYY:", shortURL)
				msg := tgbotapi.NewMessage(chatID, message)
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				bot.Send(msg)
			}

		} else if update.PollAnswer != nil {
			var msg tgbotapi.MessageConfig
			answer := update.PollAnswer
			userChoiceIndex := answer.OptionIDs[0]
			err = saving.SaveFeedback(db.Db, userChoiceIndex+2, answer.User.ID)
			fmt.Println(userChoiceIndex)
			if userChoiceIndex == 4 {
				msg = tgbotapi.NewMessage(answer.User.ID, "Спасибо за вашу оценку!")
			} else {
				userStates.Store(answer.User.ID, "awaiting_feedback_details")
				msg = tgbotapi.NewMessage(answer.User.ID, "Расскажите подробнее, что можно улучшить?")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			}

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending message: %v", err)
			}

		} else if update.Message != nil {
			var msg tgbotapi.MessageConfig
			chatID := update.Message.Chat.ID
			state, ok := userStates.Load(chatID)
			switch update.Message.Text {

			case "/start":
				if !saving.UserInBase(db.Db, chatID) {
					err = saving.AddUser(db.Db, chatID)
					if err != nil {
						log.Printf("Error saving user %v", err)
					}
				}

				msg = tgbotapi.NewMessage(chatID, "Привет! Я бот для сокращения ссылок 2links")
				msg.ReplyMarkup = keyboard

			case "/help", buttonHelp:
				msg = tgbotapi.NewMessage(chatID, "Я могу помочь с сокращением ссылок:\n/start - Запустить\n/feedback - Поделиться мнением о боте\n/help - Узнать, что я умею")

			case "/feedback", buttonFeedback:
				poll := tgbotapi.SendPollConfig{
					BaseChat:    tgbotapi.BaseChat{ChatID: chatID},
					Question:    question,
					Options:     options,
					IsAnonymous: isAnonymous,
				}

				_, err := bot.Send(poll)
				if err != nil {
					log.Printf("Failed to send poll: %v", err)
				}

			case buttonMyLinks:
				stats, err := saving.GetClicksByUser(db.Db, chatID)
				if err != nil {
					log.Printf("Error fetching clicks: %v", err)
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении статистики. Попробуйте позже."))
				}

				links, err := saving.ShowMyLinks(db.Db, chatID)
				if err != nil {
					log.Printf("Error fetching links: %v", err)
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении ваших ссылок. Попробуйте позже."))
				}

				if len(links) == 0 {
					bot.Send(tgbotapi.NewMessage(chatID, "У вас пока нет ссылок."))
				}

				message := "Ваши ссылки:\n"
				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup()

				for _, link := range links {
					statsData, ok := stats[link.ShortURL]
					clicks := 0
					if ok {
						clicks = statsData.Clicks
					}

					formattedTime := link.CreatedAt.Add(3 * time.Hour).Format("02.01.2006, 15:04")
					expiryTime := link.ExpiresAt.Format("02.01.2006, 15:04")
					daysLeft := int(time.Until(link.ExpiresAt).Hours() / 24)
					if daysLeft < 0 {
						message += fmt.Sprintf(
							"Ссылка: %s\nОригинал: %s\nПереходов: %d\nСоздана: %s\nСтатус: Просрочена\n\n",
							url+link.ShortURL, link.OriginalURL, clicks, formattedTime,
						)
					} else {
						message += fmt.Sprintf(
							"Ссылка: [%s](%s)\nОригинал: %s\nПереходов: %d\nСоздана: %s\nИстекает: %s\nОсталось: %d дней\nQR-код: [/qr_%s](%s)\n\n",
							url+link.ShortURL, url+link.ShortURL, link.OriginalURL, clicks, formattedTime, expiryTime, daysLeft, link.ShortURL, fmt.Sprintf("/qr/%s", link.ShortURL),
						)

					}

					deleteButton := tgbotapi.NewInlineKeyboardButtonData(
						fmt.Sprintf("Удалить %s", link.ShortURL),
						fmt.Sprintf("delete:%s", link.ShortURL),
					)

					updateButton := tgbotapi.NewInlineKeyboardButtonData(
						fmt.Sprintf("Изменить срок %s", link.ShortURL),
						fmt.Sprintf("update:%s", link.ShortURL),
					)

					inlineKeyboard.InlineKeyboard = append(
						inlineKeyboard.InlineKeyboard,
						tgbotapi.NewInlineKeyboardRow(deleteButton, updateButton),
					)
				}

				msg := tgbotapi.NewMessage(chatID, message)
				msg.ReplyMarkup = inlineKeyboard
				msg.ParseMode = "Markdown"
				bot.Send(msg)

			case buttonShorten:
				msg = tgbotapi.NewMessage(chatID, "Введите ссылку - и я сокращу её")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				userStates.Store(chatID, "awaiting_link")

			case buttonComplaint:
				msg = tgbotapi.NewMessage(chatID, "Введите ссылку, на которую хотите пожаловаться в формате 2lnx.ru/xxxx")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				userStates.Store(chatID, "awaiting_bad_link")

			default:
				if ok && state == "awaiting_link" {
					longLink := update.Message.Text
					if shortener.CheckValidacy(longLink) {
						shortLink, err := shortener.СreateShortLink(db, chatID, longLink)
						if err != nil {
							bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании короткой ссылки. Попробуйте позже."))
							log.Printf("Error creating short link: %v", err)
							continue
						}

						msg = tgbotapi.NewMessage(chatID, "Вот ваша сокращённая ссылка: "+url+shortLink)

					} else {
						msg = tgbotapi.NewMessage(chatID, "Эта ссылка не действительня, попробуйте другую")
					}

					msg.ReplyMarkup = keyboard
					userStates.Delete(chatID)

				} else if ok && state == "awaiting_feedback_details" {
					msg = tgbotapi.NewMessage(chatID, "Спасибо за ваш отзыв!")
					err = saving.SaveReview(db.Db, update.Message.Text, chatID)
					if err != nil {
						log.Printf("Error saving review: %v", err)
					}

					msg.ReplyMarkup = keyboard
					userStates.Delete(chatID)

				} else if ok && state == "awaiting_bad_link" {
					var linkID int
					var message string
					if strings.HasPrefix(update.Message.Text, "2lnx.ru/") {
						badLink := update.Message.Text[8:]
						linkID, err = saving.FindLink(db.Db, badLink)
						if err != nil {
							log.Printf("Error finding link: %v", err)
						} else if linkID == 0 {
							message = "Ссылка не найдена"
						} else {
							err = saving.SuspectLink(db.Db, linkID, badLink)
							message = "Спасибо за обращение, мы проверим эту ссылку"
						}
					} else {
						message = "Неверный формат ссылки"
					}

					msg = tgbotapi.NewMessage(chatID, message)
					msg.ReplyMarkup = keyboard
					userStates.Delete(chatID)

				} else if ok && strings.HasPrefix(state.(string), "awaiting_expiry_") {
					threashold := os.Getenv("MAX_LIFETIME")
					if threashold == "" {
						threashold = "730"
					}

					threasholdDays, err := strconv.Atoi(threashold)
					if err != nil {
						log.Printf("Error converting lifetime: %v", err)
						break
					}

					shortURL := strings.TrimPrefix(state.(string), "awaiting_expiry_")
					newExpiry, err := time.Parse("02-01-2006", update.Message.Text)
					if err != nil {
						msg := tgbotapi.NewMessage(chatID, "Неверный формат даты. Используйте формат: DD-MM-YYYY.")
						bot.Send(msg)
						break
					}

					differenceInDays := int(newExpiry.Sub(time.Now()).Hours() / 24)
					if newExpiry.Before(time.Now()) || differenceInDays > threasholdDays {
						msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Нельзя установить прошедшую дату, и срок жизни не может превышать %d дней. Введите заново", threasholdDays))
						bot.Send(msg)
						break
					}

					var message string

					err = saving.UpdateLinkExpiry(db.Db, chatID, shortURL, newExpiry)
					if err != nil {
						message = "Не удалось обновить срок хранения. Убедитесь, что ссылка существует."
					} else {
						message = "Срок хранения успешно обновлён."
					}

					userStates.Delete(chatID)
					msg = tgbotapi.NewMessage(chatID, message)
					msg.ReplyMarkup = keyboard
				} else if strings.HasPrefix(update.Message.Text, "/qr_") {
					shortURL := strings.TrimPrefix(update.Message.Text, "/qr_")
					qrFilePath, err := shortener.GenerateQRCode(url, shortURL)
					if err != nil {
						msg = tgbotapi.NewMessage(chatID, "Ошибка при генерации QR-кода. Убедитесь, что ссылка существует.")
						bot.Send(msg)
						log.Printf("Error generating QR code: %v", err)
						break
					}

					photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(qrFilePath))
					photo.Caption = fmt.Sprintf("QR-код для ссылки: %s%s", url, shortURL)
					bot.Send(photo)
					break

				} else {
					msg = tgbotapi.NewMessage(chatID, "Такого не знаю(")
				}
			}

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending message: %v", err)
			}
		}
	}
}
