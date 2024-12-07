package bot

import (
	"2links/internal/pkg/saving"
	"2links/internal/pkg/shortener"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Сократить ссылку")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Посмотреть свои ссылки")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Оставить обратную связь")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Получить помощь")),
	)

	question := "Как вам наш сервис?"
	options := []string{"Плохо", "Средне", "Хорошо", "Отлично"}
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
			case callbackData == "back":
				message = "Возвращаемся в основное меню"
			}

			msg := tgbotapi.NewMessage(chatID, message)
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		} else if update.PollAnswer != nil {
			var msg tgbotapi.MessageConfig
			answer := update.PollAnswer
			userChoiceIndex := answer.OptionIDs[0]
			err = saving.SaveFeedback(db, userChoiceIndex+1, answer.User.ID)
			if userChoiceIndex == 4 {
				msg = tgbotapi.NewMessage(answer.User.ID, "Спасибо за вашу оценку!")
			} else {
				userStates.Store(answer.User.ID, "awaiting_feedback_details")
				msg = tgbotapi.NewMessage(answer.User.ID, "Расскажите подробнее, что пошло не так?")
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
				if !saving.UserInBase(db, chatID) {
					err = saving.AddUser(db, chatID)
					if err != nil {
						log.Printf("Error saving user %v", err)
					}
				}
				msg = tgbotapi.NewMessage(chatID, "Привет! Я бот для сокращения ссылок 2links")
				msg.ReplyMarkup = keyboard

			case "/help", "Получить помощь":
				msg = tgbotapi.NewMessage(chatID, "Я могу помочь с сокращением ссылок:\n/start - Запустить\n/feedback - Поделиться мнением о боте\n/help - Узнать, что я умею")

			case "/feedback", "Оставить обратную связь":
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
			case "Посмотреть свои ссылки":
				var links []saving.Link
				var message string
				inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup()
				links, err = saving.ShowMyLinks(db, chatID)
				if err != nil {
					log.Printf("Error showing links: %v", err)
				}
				if len(links) == 0 {
					message = "У вас пока нет ссылок"
					msg = tgbotapi.NewMessage(chatID, message)
				} else {
					message = "Вот ваши ссылки:\n"
					for _, link := range links {
						formattedTime := link.CreatedAt.Add(3 * time.Hour).Format("02.01.2006, 15:04")
						message += fmt.Sprintf("%s -> %s : %s\n", url+link.ShortURL, link.OriginalURL, formattedTime)
						button := tgbotapi.NewInlineKeyboardButtonData(
							fmt.Sprintf("Удалить ссылку на %s", link.OriginalURL),
							fmt.Sprintf("delete:%s", link.ShortURL),
						)
						inlineKeyboard.InlineKeyboard = append(inlineKeyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))

					}
					backButton := tgbotapi.NewInlineKeyboardButtonData("Назад", "back")
					inlineKeyboard.InlineKeyboard = append(inlineKeyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(backButton))
					removeKeyboard := tgbotapi.NewMessage(chatID, "Если хотите удалить ссылку, воспользуйтесь инлайн-кнопками")
					removeKeyboard.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
					bot.Send(removeKeyboard)
					msg = tgbotapi.NewMessage(chatID, message)
					msg.ReplyMarkup = inlineKeyboard
				}

			case "Сократить ссылку":
				msg = tgbotapi.NewMessage(chatID, "Введите ссылку - и я сокращу её")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				userStates.Store(chatID, "awaiting_link")

			default:
				if ok && state == "awaiting_link" {
					longLink := update.Message.Text
					if shortener.CheckValidacy(longLink) {
						shortenedLink := url + shortener.СreateShortLink(db, chatID, longLink)
						msg = tgbotapi.NewMessage(chatID, "Вот ваша сокращённая ссылка: "+shortenedLink)
						msg.ReplyMarkup = keyboard
						userStates.Delete(chatID)
					} else {
						msg = tgbotapi.NewMessage(chatID, "Ваша ссылка не действительня, попробуй другую")
					}

				} else if ok && state == "awaiting_feedback_details" {
					msg = tgbotapi.NewMessage(chatID, "Спасибо за ваш отзыв!")
					err = saving.SaveReview(db, update.Message.Text, chatID)
					if err != nil {
						log.Printf("Error saving review: %v", err)
					}
					userStates.Delete(chatID)

				} else {
					msg = tgbotapi.NewMessage(chatID, "Такого не знаю(")
				}
			}
			// if state == nil {
			// 	msg.ReplyMarkup = keyboard
			// }

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending message: %v", err)
			}
		}
	}
}
