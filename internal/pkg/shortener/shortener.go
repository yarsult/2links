package shortener

import (
	"2links/internal/pkg/saving"
	"math/rand"
	"net/url"
	"time"
)

const symbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func СreateShortLink(Db *saving.DB, id int64, longlink string) string {
	var res string
	for range 4 {
		res += string(symbols[rand.Intn(len(symbols))])
	}
	saving.SaveLink(Db, id, longlink, res, time.Now())
	return res
}

func CheckValidacy(link string) bool {
	_, err1 := url.Parse(link)
	if err1 != nil {
		return false
	}
	return true
}
