package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

// isWebURL reports whether s looks like an http:// or https:// URL. The check
// is case-insensitive to match common user input.
func isWebURL(s string) bool {
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")
}

// fetchURLAsMarkdown downloads the given URL, converts the response body from
// HTML to Markdown using the html-to-markdown package, and returns the
// Markdown bytes. The URL is prepended as a brief header so the reader can
// see where the content came from. Non-2xx responses produce an error.
// https:// URLs require the curl executable to be on the PATH; http:// URLs
// are fetched natively.
func fetchURLAsMarkdown(u string) ([]byte, error) {
	headers := map[string]string{
		"User-Agent": httpUserAgent + " text/markdown reader",
		"Accept":     "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5",
	}
	resp, err := httpDo("GET", u, nil, headers, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("fetch %s: HTTP %d", u, resp.StatusCode)
	}

	// Cap response size at 16 MiB to avoid runaway memory on misbehaving servers
	const maxBytes = 16 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(body) > maxBytes {
		return nil, fmt.Errorf("response from %s exceeds %d MiB", u, maxBytes>>20)
	}

	md, err := htmltomarkdown.ConvertString(string(body),
		converter.WithDomain(u),
	)
	if err != nil {
		return nil, fmt.Errorf("convert HTML to Markdown: %w", err)
	}

	var sb strings.Builder
	sb.Grow(len(md) + 64)
	sb.WriteString("# ")
	sb.WriteString(u)
	sb.WriteString("\n\n")
	sb.WriteString(md)
	if !strings.HasSuffix(md, "\n") {
		sb.WriteByte('\n')
	}
	return []byte(sb.String()), nil
}
