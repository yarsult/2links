package shortener

import (
	"math/rand"
	"net/http"
	"net/url"
)

const symbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func СreateShortLink() string {
	var res string
	for range 4 {
		res += string(symbols[rand.Intn(len(symbols))])
	}
	return res
}

func CheckValidacy(link string) bool {
	addr, err1 := url.Parse(link)
	if err1 != nil {
		return false
	}
	if !addr.IsAbs() {
		link = "http://" + link
	}

	resp, err2 := http.Get(link)
	if err2 != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
