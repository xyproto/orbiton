package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const nreplTimeout = 10 * time.Second

var (
	nreplClients = make(map[string]*NREPLClient) // keyed by "host:port"
	nreplMu      sync.Mutex
)

// NREPLClient holds an active connection to a running nREPL server together
// with the session ID that was assigned when the connection was opened.
type NREPLClient struct {
	conn    net.Conn
	reader  *bufio.Reader
	session string
	mu      sync.Mutex
}

// findNREPLPort searches for a .nrepl-port file starting from startDir and
// walking up the directory tree. It returns the port number written in that
// file, or an error when no such file is found.
func findNREPLPort(startDir string) (int, error) {
	dir := startDir
	for {
		data, err := os.ReadFile(filepath.Join(dir, ".nrepl-port"))
		if err == nil {
			port, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				return 0, fmt.Errorf("invalid .nrepl-port content: %w", err)
			}
			return port, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return 0, errors.New(".nrepl-port not found")
}

// bencodeStr returns the bencode encoding of a single string.
func bencodeStr(s string) []byte {
	return fmt.Appendf(nil, "%d:%s", len(s), s)
}

// bencodeMsg encodes a map[string]string as a bencode dict with sorted keys.
func bencodeMsg(m map[string]string) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b := []byte{'d'}
	for _, k := range keys {
		b = append(b, bencodeStr(k)...)
		b = append(b, bencodeStr(m[k])...)
	}
	return append(b, 'e')
}

// bencodeRead reads one bencode value from r and returns it as a string,
// []any (list), or map[string]any (dict).
func bencodeRead(r *bufio.Reader) (any, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch b {
	case 'i': // integer: iNe
		s, err := r.ReadString('e')
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(s[:len(s)-1], 10, 64)
	case 'l': // list: l<items>e
		var items []any
		for {
			next, err := r.Peek(1)
			if err != nil {
				return nil, err
			}
			if next[0] == 'e' {
				r.ReadByte() //nolint:errcheck
				break
			}
			item, err := bencodeRead(r)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	case 'd': // dict: d<key><value>...e
		m := make(map[string]any)
		for {
			next, err := r.Peek(1)
			if err != nil {
				return nil, err
			}
			if next[0] == 'e' {
				r.ReadByte() //nolint:errcheck
				break
			}
			key, err := bencodeRead(r)
			if err != nil {
				return nil, err
			}
			val, err := bencodeRead(r)
			if err != nil {
				return nil, err
			}
			if ks, ok := key.(string); ok {
				m[ks] = val
			}
		}
		return m, nil
	default: // string: <len>:<data>  (b is the first digit of the length)
		var lenStr strings.Builder
		lenStr.WriteString(string(b))
		for {
			ch, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			if ch == ':' {
				break
			}
			lenStr.WriteString(string(ch))
		}
		n, err := strconv.Atoi(lenStr.String())
		if err != nil {
			return nil, err
		}
		data := make([]byte, n)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		return string(data), nil
	}
}

// nreplConnect opens a TCP connection to addr, clones a new nREPL session
// and returns the ready-to-use client.
func nreplConnect(addr string) (*NREPLClient, error) {
	conn, err := net.DialTimeout("tcp", addr, nreplTimeout)
	if err != nil {
		return nil, err
	}
	nc := &NREPLClient{conn: conn, reader: bufio.NewReader(conn)}
	conn.SetDeadline(time.Now().Add(nreplTimeout)) //nolint:errcheck
	if _, err := conn.Write(bencodeMsg(map[string]string{"op": "clone"})); err != nil {
		conn.Close()
		return nil, err
	}
	for {
		val, err := bencodeRead(nc.reader)
		if err != nil {
			conn.Close()
			return nil, err
		}
		resp, ok := val.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := resp["new-session"].(string); ok && s != "" {
			nc.session = s
		}
		done := false
		if statuses, ok := resp["status"].([]any); ok {
			for _, item := range statuses {
				if str, ok := item.(string); ok && str == "done" {
					done = true
					break
				}
			}
		}
		if done {
			break
		}
	}
	if nc.session == "" {
		conn.Close()
		return nil, errors.New("nREPL clone did not return a session id")
	}
	conn.SetDeadline(time.Time{}) //nolint:errcheck // clear deadline
	return nc, nil
}

// nreplClientFor returns a connected NREPLClient for addr, reusing an
// existing connection when one is available.
func nreplClientFor(addr string) (*NREPLClient, error) {
	nreplMu.Lock()
	defer nreplMu.Unlock()
	if nc, ok := nreplClients[addr]; ok {
		return nc, nil
	}
	nc, err := nreplConnect(addr)
	if err != nil {
		return nil, err
	}
	nreplClients[addr] = nc
	return nc, nil
}

// nreplDrop closes and removes the cached connection for addr so that the
// next call to nreplClientFor will open a fresh one.
func nreplDrop(addr string) {
	nreplMu.Lock()
	if nc, ok := nreplClients[addr]; ok {
		nc.conn.Close()
		delete(nreplClients, addr)
	}
	nreplMu.Unlock()
}

// Eval sends code to the nREPL session, waits for the evaluation to finish
// and returns the result value, or the accumulated stderr output when the
// evaluation produced an error.
func (nc *NREPLClient) Eval(code string) (string, error) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.conn.SetDeadline(time.Now().Add(nreplTimeout)) //nolint:errcheck
	defer nc.conn.SetDeadline(time.Time{})            //nolint:errcheck
	msg := bencodeMsg(map[string]string{
		"id":      "1",
		"op":      "eval",
		"session": nc.session,
		"code":    code,
	})
	if _, err := nc.conn.Write(msg); err != nil {
		return "", err
	}
	var (
		value  string
		outBuf strings.Builder
		errBuf strings.Builder
	)
	for {
		val, err := bencodeRead(nc.reader)
		if err != nil {
			return "", err
		}
		resp, ok := val.(map[string]any)
		if !ok {
			continue
		}
		if v, ok := resp["value"].(string); ok {
			value = v
		}
		if v, ok := resp["out"].(string); ok {
			outBuf.WriteString(v)
		}
		if v, ok := resp["err"].(string); ok {
			errBuf.WriteString(v)
		}
		// "ex" carries the exception class name as a fallback error hint
		if v, ok := resp["ex"].(string); ok && errBuf.Len() == 0 {
			errBuf.WriteString(v)
		}
		// keep reading until the server signals that evaluation is done
		done := false
		if statuses, ok := resp["status"].([]any); ok {
			for _, item := range statuses {
				if str, ok := item.(string); ok && str == "done" {
					done = true
					break
				}
			}
		}
		if done {
			break
		}
	}
	if errBuf.Len() > 0 {
		return "", errors.New(strings.TrimSpace(errBuf.String()))
	}
	if value != "" {
		return value, nil
	}
	return strings.TrimSpace(outBuf.String()), nil
}

// clojureTopLevelFormStart returns the line index of the nearest top-level
// Clojure form that contains or precedes the cursor. Top-level forms are
// lines that begin with '(' at column 0. Returns -1 when none is found.
func (e *Editor) clojureTopLevelFormStart() int {
	y := int(e.DataY())
	for y >= 0 {
		if line, ok := e.lines[y]; ok && len(line) > 0 && line[0] == '(' {
			return y
		}
		y--
	}
	return -1
}

// clojureTopLevelForm returns the source text of the top-level Clojure form
// that contains or immediately precedes the cursor. String literals and
// line comments (;) are taken into account so their parentheses do not affect
// the depth count. Returns an empty string when no form is found.
func (e *Editor) clojureTopLevelForm() string {
	startY := e.clojureTopLevelFormStart()
	if startY < 0 {
		return ""
	}
	var sb strings.Builder
	depth := 0
	inString := false
	escape := false
	for y := startY; y < e.Len(); y++ {
		line := e.Line(LineIndex(y))
		sb.WriteString(line)
		sb.WriteByte('\n')
		inComment := false
		for _, r := range line {
			if escape {
				escape = false
				continue
			}
			if inString {
				switch r {
				case '\\':
					escape = true
				case '"':
					inString = false
				}
				continue
			}
			if inComment {
				continue
			}
			switch r {
			case '"':
				inString = true
			case ';':
				inComment = true
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return sb.String()
				}
			}
		}
	}
	return sb.String()
}
