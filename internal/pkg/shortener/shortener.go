package shortener

import (
	"2links/internal/pkg/saving"
	"math/rand"
	"net/url"
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

	saving.SaveLink(Db, id, longlink, newlink, time.Now())
	return newlink
}

func CheckValidacy(link string) bool {
	_, err1 := url.Parse(link)
	if err1 != nil {
		return false
	}

	return true
}
