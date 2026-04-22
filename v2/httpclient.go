package main

// Minimal HTTP/1.1 client. Plain http:// requests are handled directly over a
// TCP socket. https:// requests shell out to the curl executable, if present,
// so the Go standard library's crypto/tls (and its dependencies crypto/x509,
// crypto/ecdsa, etc) do not have to be linked in.

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xyproto/files"
)

const httpUserAgent = "Orbiton/1.0 (+https://github.com/xyproto/orbiton)"

// httpResponse mimics a subset of http.Response. Body must be closed by the
// caller.
type httpResponse struct {
	StatusCode int
	Status     string
	Header     map[string]string
	Body       io.ReadCloser
}

// parseSimpleURL splits http(s)://host[:port]/path into its pieces. Only the
// parts Orbiton actually needs; no fragment parsing.
func parseSimpleURL(raw string) (scheme, host, port, path string, err error) {
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "https://"):
		scheme = "https"
		raw = raw[len("https://"):]
	case strings.HasPrefix(lower, "http://"):
		scheme = "http"
		raw = raw[len("http://"):]
	default:
		err = errors.New("unsupported URL scheme (only http:// and https:// are supported)")
		return
	}
	slash := strings.IndexByte(raw, '/')
	hostport := raw
	if slash >= 0 {
		hostport = raw[:slash]
		path = raw[slash:]
	} else {
		path = "/"
	}
	if hostport == "" {
		err = errors.New("missing host in URL")
		return
	}
	if colon := strings.LastIndexByte(hostport, ':'); colon >= 0 && !strings.ContainsAny(hostport[colon+1:], "]/") {
		host = hostport[:colon]
		port = hostport[colon+1:]
	} else {
		host = hostport
	}
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return
}

// httpDo performs a request and follows up to 5 redirects. Body may be nil.
// Content-Type must be provided via extraHeaders when relevant.
func httpDo(method, rawURL string, body []byte, extraHeaders map[string]string, timeout time.Duration) (*httpResponse, error) {
	cur := rawURL
	for i := 0; i < 6; i++ {
		scheme, host, port, path, err := parseSimpleURL(cur)
		if err != nil {
			return nil, err
		}
		var resp *httpResponse
		if scheme == "https" {
			resp, err = httpsViaCurl(method, cur, body, extraHeaders, timeout)
		} else {
			resp, err = httpPlain(method, host, port, path, body, extraHeaders, timeout)
		}
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header["location"]
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if loc == "" {
				return resp, nil
			}
			if strings.HasPrefix(loc, "/") {
				cur = scheme + "://" + net.JoinHostPort(host, port) + loc
			} else if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
				cur = loc
			} else {
				// relative path without leading slash
				cur = scheme + "://" + net.JoinHostPort(host, port) + "/" + loc
			}
			continue
		}
		return resp, nil
	}
	return nil, errors.New("too many redirects")
}

// httpGet is a convenience wrapper around httpDo for GET requests.
func httpGet(rawURL string, extraHeaders map[string]string, timeout time.Duration) (*httpResponse, error) {
	return httpDo("GET", rawURL, nil, extraHeaders, timeout)
}

// httpPostJSON is a convenience wrapper for POST requests carrying a JSON body.
func httpPostJSON(rawURL string, body []byte, timeout time.Duration) (*httpResponse, error) {
	return httpDo("POST", rawURL, body, map[string]string{"Content-Type": "application/json"}, timeout)
}

func httpPlain(method, host, port, path string, body []byte, extraHeaders map[string]string, timeout time.Duration) (*httpResponse, error) {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}
	var req bytes.Buffer
	fmt.Fprintf(&req, "%s %s HTTP/1.1\r\n", method, path)
	fmt.Fprintf(&req, "Host: %s\r\n", host)
	// Default headers. Callers can override via extraHeaders.
	headersSet := map[string]bool{"host": true}
	for k, v := range extraHeaders {
		fmt.Fprintf(&req, "%s: %s\r\n", k, v)
		headersSet[strings.ToLower(k)] = true
	}
	if !headersSet["user-agent"] {
		fmt.Fprintf(&req, "User-Agent: %s\r\n", httpUserAgent)
	}
	if !headersSet["accept"] {
		fmt.Fprintf(&req, "Accept: */*\r\n")
	}
	if !headersSet["connection"] {
		fmt.Fprintf(&req, "Connection: close\r\n")
	}
	if body != nil && !headersSet["content-length"] {
		fmt.Fprintf(&req, "Content-Length: %d\r\n", len(body))
	}
	req.WriteString("\r\n")
	if body != nil {
		req.Write(body)
	}
	if _, err := conn.Write(req.Bytes()); err != nil {
		conn.Close()
		return nil, err
	}
	return parseHTTPResponse(conn)
}

func parseHTTPResponse(rc io.ReadCloser) (*httpResponse, error) {
	br := bufio.NewReader(rc)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		rc.Close()
		return nil, err
	}
	statusLine = strings.TrimRight(statusLine, "\r\n")
	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "HTTP/") {
		rc.Close()
		return nil, fmt.Errorf("bad status line: %q", statusLine)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil {
		rc.Close()
		return nil, fmt.Errorf("bad status code: %q", parts[1])
	}
	resp := &httpResponse{StatusCode: code, Status: statusLine, Header: map[string]string{}}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			rc.Close()
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:colon]))
		val := strings.TrimSpace(line[colon+1:])
		resp.Header[key] = val
	}
	if strings.EqualFold(resp.Header["transfer-encoding"], "chunked") {
		resp.Body = &chunkedReader{br: br, closer: rc}
	} else if cl, ok := resp.Header["content-length"]; ok {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
			resp.Body = readCloser{Reader: io.LimitReader(br, n), Closer: rc}
		} else {
			resp.Body = readCloser{Reader: br, Closer: rc}
		}
	} else {
		resp.Body = readCloser{Reader: br, Closer: rc}
	}
	return resp, nil
}

type readCloser struct {
	io.Reader
	io.Closer
}

// chunkedReader decodes HTTP/1.1 chunked transfer encoding.
type chunkedReader struct {
	br     *bufio.Reader
	closer io.Closer
	remain int64
	eof    bool
}

func (c *chunkedReader) Read(p []byte) (int, error) {
	if c.eof {
		return 0, io.EOF
	}
	if c.remain == 0 {
		line, err := c.br.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")
		if semi := strings.IndexByte(line, ';'); semi >= 0 {
			line = line[:semi]
		}
		n, err := strconv.ParseInt(strings.TrimSpace(line), 16, 64)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			_, _ = c.br.ReadString('\n')
			c.eof = true
			return 0, io.EOF
		}
		c.remain = n
	}
	read := int64(len(p))
	if read > c.remain {
		read = c.remain
	}
	m, err := c.br.Read(p[:read])
	c.remain -= int64(m)
	if c.remain == 0 && err == nil {
		_, _ = c.br.Discard(2)
	}
	return m, err
}

func (c *chunkedReader) Close() error {
	return c.closer.Close()
}

// httpsViaCurl fetches an https URL by invoking curl. The response is parsed
// with the same parseHTTPResponse used for plain http, so callers see the
// same httpResponse surface.
func httpsViaCurl(method, url string, body []byte, extraHeaders map[string]string, timeout time.Duration) (*httpResponse, error) {
	curlPath := files.WhichCached("curl")
	if curlPath == "" {
		return nil, errors.New("https support requires the curl executable, but it was not found in the PATH")
	}
	args := []string{
		"-sS", "-i",
		"-X", method,
		"-A", httpUserAgent,
	}
	if timeout > 0 {
		args = append(args, "--max-time", strconv.FormatFloat(timeout.Seconds(), 'f', 0, 64))
	}
	hasAccept := false
	for k, v := range extraHeaders {
		if strings.EqualFold(k, "Accept") {
			hasAccept = true
		}
		args = append(args, "-H", k+": "+v)
	}
	if !hasAccept {
		args = append(args, "-H", "Accept: */*")
	}
	if body != nil {
		args = append(args, "--data-binary", "@-")
	}
	args = append(args, url)
	cmd := exec.Command(curlPath, args...)
	if body != nil {
		cmd.Stdin = bytes.NewReader(body)
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("curl %s: %w", url, err)
	}
	return parseHTTPResponse(io.NopCloser(bytes.NewReader(out)))
}
