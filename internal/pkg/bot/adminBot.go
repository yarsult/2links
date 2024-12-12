package bot

import (
	"2links/internal/pkg/saving"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	buttonSuspiciousLinks = "Подозрительные ссылки"
	buttonLastReviews     = "Последние отзывы"
	buttonMiddleGrade     = "Средняя оценка"
	buttonStatistics      = "Сводная статистика"
)

var adminAuthorized sync.Map

func StartAdminBot(token string, db *saving.DB) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Admin bot authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonSuspiciousLinks)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonLastReviews)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonMiddleGrade)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(buttonStatistics)),
	)

	for update := range updates {
		if update.Message != nil {
			chatID := update.Message.Chat.ID
			text := update.Message.Text
			adminPasswordHash, err := readHashFromFile("internal/pkg/bot/adminPasswordHash.txt")
			if err != nil {
				log.Printf("Error reading file with hash: %v", err)
			}

			isAuthorized, _ := adminAuthorized.Load(chatID)
			if authorized, ok := isAuthorized.(bool); !ok || !authorized {
				if text == "/start" || text == "/help" {
					bot.Send(tgbotapi.NewMessage(chatID, "Введите пароль администратора для доступа."))
					adminAuthorized.Store(chatID, false)
					continue
				}

				if checkPasswordHash(text, adminPasswordHash) {
					adminAuthorized.Store(chatID, true)
					msg := tgbotapi.NewMessage(chatID, "Добро пожаловать, администратор!")
					msg.ReplyMarkup = keyboard
					bot.Send(msg)
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Неверный пароль. Попробуйте снова."))
				}

				continue
			}

			switch {
			case text == buttonSuspiciousLinks:
				handleSuspectLinks(bot, db, chatID)

			case text == buttonStatistics:
				handleStatistics(bot, db, chatID)

			case text == buttonLastReviews:
				handleReviews(bot, db, chatID)

			case text == buttonMiddleGrade:
				handleGrade(bot, db, chatID)

			case strings.HasPrefix(text, "/delete_"):
				link := strings.TrimPrefix(text, "/delete_")
				handleDeleteLink(bot, db, chatID, link)

			default:
				bot.Send(tgbotapi.NewMessage(chatID, "Нет такой команды"))
			}
		}
	}
}

func handleReviews(bot *tgbotapi.BotAPI, db *saving.DB, chatID int64) {
	reviews, err := saving.GetReviews(db.Db)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении списка отзывов"))
		log.Printf("Error fetching reviews: %v", err)
		return
	}

	if len(reviews) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Нет отзывов."))
		return
	}

	message := "Последние 5 отзывов:\n"
	for i, r := range reviews {
		message += fmt.Sprintf("%d. %s\n", i+1, r)
	}

	bot.Send(tgbotapi.NewMessage(chatID, message))
}

func handleGrade(bot *tgbotapi.BotAPI, db *saving.DB, chatID int64) {
	grade, err := saving.GetGrade(db.Db)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении средней оценки."))
		log.Printf("Error fetching grade: %v", err)
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Средняя оценка сервиса: %.3f", grade)))
}

func handleSuspectLinks(bot *tgbotapi.BotAPI, db *saving.DB, chatID int64) {
	links, err := saving.GetSuspectLinks(db.Db)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении списка подозрительных ссылок."))
		log.Printf("Error fetching suspected links: %v", err)
		return
	}

	if len(links) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Нет подозрительных ссылок."))
		return
	}

	message := "Подозрительные ссылки:\n"
	for _, link := range links {
		message += fmt.Sprintf("short url: %s -> %s\nКоманда для удаления: /delete_%s\n\n",
			link.ShortURL, link.OriginalURL, link.ShortURL)
	}

	bot.Send(tgbotapi.NewMessage(chatID, message))
}

func handleDeleteLink(bot *tgbotapi.BotAPI, db *saving.DB, chatID int64, link string) {
	err := saving.DeleteSuspectLink(db.Db, link)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при удалении ссылки."))
		log.Printf("Error deleting link: %v", err)
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, "Ссылка успешно удалена."))
}

func handleStatistics(bot *tgbotapi.BotAPI, db *saving.DB, chatID int64) {
	stats, err := saving.GetSummaryStatistics(db.Db)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении статистики."))
		log.Printf("Error fetching statistics: %v", err)
		return
	}

	message := fmt.Sprintf("Сводная статистика:\n"+
		"Количество пользователей: %d\n"+
		"Созданные ссылки: %d\n"+
		"Переходы по ссылкам: %d\n"+
		"Истёкшие ссылки: %d\n",
		stats.Users, stats.Links, stats.Clicks, stats.ExpiredLinks)

	bot.Send(tgbotapi.NewMessage(chatID, message))
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func readHashFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("Failed to read file: %v", err)
	}

	return strings.TrimSpace(string(data)), nil
}
