package guessica

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	examinedLinks   []string
	examinedMutex   *sync.Mutex
	clientTimeout   time.Duration
	noStripLetters  = false
	defaultProtocol = "http" // If the protocol is missing

	linkFinder = regexp.MustCompile(`(http|ftp|https):\/\/([\w\-_]+(?:(?:\.[\w\-_]+)+))([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])?`)
)

// unquote will strip a trimmed string from surrounding " or ' quotes
func unquote(s string) string {
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		return s[1 : len(s)-1]
	}
	return s
}

func linkIsPage(url string) bool {
	// If the link ends with an extension, make sure it's .html
	if strings.HasSuffix(url, ".html") || strings.HasSuffix(url, ".htm") {
		return true
	}
	// If there is a question mark in the url, don't bother
	if strings.Contains(url, "?") {
		return false
	}
	// If the last part has no ".", it's assumed to be a page
	if strings.Contains(url, "/") {
		pos := strings.LastIndex(url, "/")
		if !strings.Contains(url[pos:], ".") {
			return true
		}
	}
	// Probably not a page
	return false
}

// For a given URL, return the contents or an empty string.
func get(target string) string {
	var client http.Client
	client.Timeout = clientTimeout
	resp, err := client.Get(target)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(b)
}

// Extract URLs from text
// Relative links are returned as starting with "/"
func getLinks(data string) []string {
	var foundLinks []string

	// Add relative links first
	var quote string
	for _, line := range strings.Split(data, "href=") {
		if len(line) < 1 {
			continue
		}
		quote = string(line[0])
		fields := strings.Split(line, quote)
		if len(fields) > 1 {
			relative := fields[1]
			if !strings.Contains(relative, "://") && !strings.Contains(relative, " ") {
				if strings.HasPrefix(relative, "//") {
					foundLinks = append(foundLinks, defaultProtocol+":"+relative)
				} else if strings.HasPrefix(relative, "/") {
					foundLinks = append(foundLinks, relative)
				} else {
					foundLinks = append(foundLinks, "/"+relative)
				}
			}
		}
	}

	// Then add the absolute links
	return append(foundLinks, linkFinder.FindAllString(data, -1)...)
}

// Extract likely subpages
func getSubPages(data string) []string {
	var subpages []string
	for _, link := range getLinks(data) {
		if linkIsPage(link) {
			subpages = append(subpages, link)
		}
	}
	return subpages
}

// Convert from a host (a.b.c.d.com) to a domain (d.com) or subdomain (c.d.com)
func toDomain(host string, ignoreSubdomain bool) string {
	if strings.Count(host, ".") > 1 {
		parts := strings.Split(host, ".")
		numparts := 3
		if ignoreSubdomain {
			numparts = 2
		}
		return strings.Join(parts[len(parts)-numparts:], ".")
	}
	return host
}

// Filter out links to the same domain (asdf.com) or subdomain (123.asdf.com)
func sameDomain(links []string, host string, ignoreSubdomain bool) []string {
	var result []string
	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil {
			// Invalid URL
			continue
		}
		if toDomain(u.Host, ignoreSubdomain) == toDomain(host, ignoreSubdomain) {
			result = append(result, link)
		}
		// Handle links starting with // or just /
		if strings.HasPrefix(link, "//") {
			result = append(result, defaultProtocol+":"+link)
		} else if strings.HasPrefix(link, "/") {
			result = append(result, defaultProtocol+"://"+host+link)
		}
	}
	return result
}

// Check if a given string slice has a given string
func has(sl []string, s string) bool {
	for _, e := range sl {
		if e == s {
			return true
		}
	}
	return false
}

// Crawl the given URL. Run the examinefunction on the data. Return a list of links to follow.
func crawlOnePage(target string, ignoreSubdomain bool, currentDepth int, examineFunc func(string, string, int)) []string {
	u, err := url.Parse(target)
	if err != nil {
		fmt.Println("invalid url:", target)
		return []string{}
	}
	// Find all links pointing to the same domain or same subdomain
	data := get(target)
	// Don't examine the same target twice
	examinedMutex.Lock()
	if !has(examinedLinks, target) {
		// Update the list of examined urls in a mutex
		examineFunc(target, data, currentDepth)
		examinedLinks = append(examinedLinks, target)
	}
	examinedMutex.Unlock()
	// Return the links to follow next
	return sameDomain(getSubPages(data), u.Host, ignoreSubdomain)
}

// Crawl a given URL recursively. Crawls by domain if ignoreSubdomain is true, else by subdomain.
// Depth is the crawl depth (1 only examines one page, 2 examines 1 page with all subpages, etc)
// wg is a WaitGroup. examineFunc is the function that is executed for the url and contents of every page crawled.
func crawl(target string, ignoreSubdomain bool, depth int, wg *sync.WaitGroup, examineFunc func(string, string, int)) {
	// Finish one wait group when the function returns
	defer wg.Done()
	if depth == 0 {
		return
	}
	links := crawlOnePage(target, ignoreSubdomain, depth, examineFunc)
	for _, link := range links {
		// Go one recursion deeper
		wg.Add(1)
		go crawl(link, ignoreSubdomain, depth-1, wg, examineFunc)
	}
}

// Crawl an URL up to a given depth. Runs the examine function on every page.
// Does not examine the same URL twice. Uses several goroutines.
func crawlDomain(url string, depth int, examineFunc func(string, string, int)) {
	// Set up a mutex and slice to keep track of pages that has already been crawled
	examinedMutex = new(sync.Mutex)
	examinedLinks = []string{}

	// Crawl the given URL to the desired depth, using goroutines and a WaitGroup
	var wg sync.WaitGroup
	wg.Add(1)
	go crawl(url, true, depth, &wg, examineFunc)
	// Wait for all the goroutines to complete
	wg.Wait()
}
