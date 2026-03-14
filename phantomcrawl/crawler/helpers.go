package crawler

import "net/url"

func resolveURL(base, href string) string {
	if href == "" {
		return ""
	}
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	r, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := b.ResolveReference(r).String()
	// Filter out non-http schemes
	u, err := url.Parse(resolved)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return resolved
}

func getBaseURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Scheme + "://" + u.Host
}
