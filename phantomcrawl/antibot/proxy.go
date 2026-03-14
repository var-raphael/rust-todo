package antibot

import (
	"math/rand"
	"net/url"
)

type ProxyRotator struct {
	urls     []string
	strategy string
	index    int
}

func NewProxyRotator(urls []string, strategy string) *ProxyRotator {
	return &ProxyRotator{
		urls:     urls,
		strategy: strategy,
	}
}

func (p *ProxyRotator) Next() *url.URL {
	if len(p.urls) == 0 {
		return nil
	}

	var raw string

	switch p.strategy {
	case "round_robin":
		raw = p.urls[p.index%len(p.urls)]
		p.index++
	case "fallback":
		raw = p.urls[0]
	default:
		// random
		raw = p.urls[rand.Intn(len(p.urls))]
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil
	}

	return parsed
}