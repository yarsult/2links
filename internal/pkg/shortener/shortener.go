package shortener

import (
	"2links/internal/pkg/saving"
	"math/rand"
	"net/url"
	"regexp"
	"time"
)

const symbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func СreateShortLink(Db *saving.DB, id int64, longlink string) string {
	var newlink string
	res := true
	for res != false {
		for range 4 {
			newlink += string(symbols[rand.Intn(len(symbols))])
		}
		res = saving.LinkInBase(Db, newlink)
	}

	saving.SaveLink(Db, id, longlink, newlink, time.Now().Add((24 * time.Hour * 30)))
	return newlink
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
