package shortener

import (
	"2links/internal/pkg/saving"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/skip2/go-qrcode"
)

const symbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func СreateShortLink(Db *saving.DB, id int64, longlink string) (string, error) {
	var newlink string
	res := true
	for res != false {
		for range 4 {
			newlink += string(symbols[rand.Intn(len(symbols))])
		}
		res = saving.LinkInBase(Db.Db, newlink)
	}

	saving.SaveLink(Db.Db, id, longlink, newlink, time.Now().Add((24 * time.Hour * 30)))
	return newlink, nil
}

func CheckValidacy(link string) bool {
	re := regexp.MustCompile(`^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(:\d+)?(/[^\s]*)?$`)
	if re.MatchString(link) {
		return true
	}

	parsedURL, err := url.Parse(link)
	if err == nil && parsedURL.Host != "" {
		return true
	}

	return false
}

func GenerateQRCode(url string, short string) (string, error) {
	fileName := fmt.Sprintf("qr_%s.png", filepath.Base(short))
	filePath := filepath.Join(os.TempDir(), fileName)
	fmt.Println(filePath)
	err := qrcode.WriteFile(url+short, qrcode.Medium, 256, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
	} else {
		fmt.Printf("File content size: %d bytes\n", len(content))
	}

	return filePath, nil
}
