package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
func fetchURLAsMarkdown(u string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	// Many sites return machine-readable or barebones responses to clients
	// without a browser-style User-Agent. Identify as a recognisable
	// Orbiton reader while still being polite.
	req.Header.Set("User-Agent", "Orbiton/1.0 (+https://github.com/xyproto/orbiton) text/markdown reader")
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Drain a small amount so the connection can be reused.
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("fetch %s: HTTP %d %s", u, resp.StatusCode, resp.Status)
	}

	// Cap response size at 16 MiB to avoid runaway memory on misbehaving
	// servers. More than enough for any reasonable article.
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
