package engine

import (
	"net/http"
	"net/url"
)

func Client(u string) (http.Client, error) {
	if u == "" {
		return http.Client{}, nil

	}
	proxy, err := url.Parse(u)
	if err != nil {
		return http.Client{}, err
	}

	client := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}
	return client, nil
}
