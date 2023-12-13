package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxSearchHistoryEntries = 1024

// SearchHistory is a map from timestap to search term (string).
// Assume no timestamp collisions for when the user is adding search terms, because the user is not that fast.
type SearchHistory map[time.Time]string

var (
	searchHistoryFilename = filepath.Join(userCacheDir, "o", "search.txt")
	searchHistory         SearchHistory
	searchHistoryMutex    sync.RWMutex
)

// Empty checks if the search history has no entries
func (sh SearchHistory) Empty() bool {
	searchHistoryMutex.RLock()
	defer searchHistoryMutex.RUnlock()

	return len(sh) == 0
}

// AddWithTimestamp adds a new line number for the given absolute path, and also records the current time
func (sh SearchHistory) AddWithTimestamp(searchTerm string, timestamp int64) {
	searchHistoryMutex.Lock()
	defer searchHistoryMutex.Unlock()

	sh[time.Unix(timestamp, 0)] = searchTerm
}

// Add adds a search term to the search history, and also sets the timestamp
func (sh SearchHistory) Add(searchTerm string) {
	sh.AddWithTimestamp(searchTerm, time.Now().Unix())
}

// Set adds or updates the given search term
func (sh SearchHistory) Set(searchTerm string) {
	// First check if an existing entry can be removed
	searchHistoryMutex.RLock()
	for k, v := range sh {
		if v == searchTerm {
			searchHistoryMutex.RUnlock()
			searchHistoryMutex.Lock()
			delete(sh, k)
			searchHistoryMutex.Unlock()
			searchHistoryMutex.RLock() // to be unlocked after the loop
			break
		}
	}
	searchHistoryMutex.RUnlock()

	// If not, just add the new entry
	sh.Add(searchTerm)
}

// SetWithTimestamp adds or updates the given search term
func (sh SearchHistory) SetWithTimestamp(searchTerm string, timestamp int64) {
	// First check if an existing entry can be removed
	searchHistoryMutex.RLock()
	for k, v := range sh {
		if v == searchTerm {
			searchHistoryMutex.RUnlock()
			searchHistoryMutex.Lock()
			delete(sh, k)
			searchHistoryMutex.Unlock()
			searchHistoryMutex.RLock() // to be unlocked after the loop
			break
		}
	}
	searchHistoryMutex.RUnlock()

	// If not, just add the new entry
	sh.AddWithTimestamp(searchTerm, timestamp)
}

// Save will attempt to save the per-absolute-filename recording of which line is active
func (sh SearchHistory) Save(path string) error {
	if noWriteToCache {
		return nil
	}

	searchHistoryMutex.RLock()
	defer searchHistoryMutex.RUnlock()

	// First create the folder, if needed, in a best effort attempt
	folderPath := filepath.Dir(path)
	os.MkdirAll(folderPath, os.ModePerm)

	var sb strings.Builder
	for timeStamp, searchTerm := range sh {
		sb.WriteString(fmt.Sprintf("%d:%s\n", timeStamp.Unix(), searchTerm))
	}

	// Write the search history and return the error, if any.
	// The permissions are a bit stricter for this one.
	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

// Len returns the current search history length
func (sh SearchHistory) Len() int {
	searchHistoryMutex.RLock()
	defer searchHistoryMutex.RUnlock()

	return len(sh)
}

// GetIndex sorts the timestamps and indexes into that.
// An empty string is returned if no element is found.
// Indexes from oldest to newest entry if asc is true,
// and from newest to oldest if asc is false.
func (sh SearchHistory) GetIndex(index int, newestFirst bool) string {
	searchHistoryMutex.RLock()
	defer searchHistoryMutex.RUnlock()

	l := len(sh)

	if l == 0 || index < 0 || index >= l {
		return ""
	}

	type timeEntry struct {
		timeObj  time.Time
		unixTime int64
	}

	timeEntries := make([]timeEntry, 0, l)

	for timestamp := range sh {
		timeEntries = append(timeEntries, timeEntry{timeObj: timestamp, unixTime: timestamp.Unix()})
	}

	if newestFirst {
		// Reverse sort
		sort.Slice(timeEntries, func(i, j int) bool {
			return timeEntries[i].unixTime > timeEntries[j].unixTime
		})
	} else {
		// Regular sort
		sort.Slice(timeEntries, func(i, j int) bool {
			return timeEntries[i].unixTime < timeEntries[j].unixTime
		})

	}

	selectedTimestampKey := timeEntries[index].timeObj
	return sh[selectedTimestampKey]
}

// LoadSearchHistory will attempt to load the map[time.Time]string from the given filename.
// The returned map can be empty.
func LoadSearchHistory(path string) (SearchHistory, error) {
	sh := make(SearchHistory, 0)

	contents, err := os.ReadFile(path)
	if err != nil {
		// Could not read file, return an empty map and an error
		return sh, err
	}

	// The format of the file is, per line:
	// timeStamp:searchTerm
	for _, filenameSearch := range strings.Split(string(contents), "\n") {
		fields := strings.Split(filenameSearch, ":")

		if len(fields) == 2 {

			// Retrieve an unquoted filename in the filename variable
			timeStampString := strings.TrimSpace(fields[0])
			searchTerm := strings.TrimSpace(fields[1])

			timestamp, err := strconv.ParseInt(timeStampString, 10, 64)
			if err != nil {
				// Could not convert timestamp to a number, skip this one
				continue
			}

			// Build the search history by setting the search term (SetWithTimestamp deals with the mutex on its own)
			sh.SetWithTimestamp(searchTerm, timestamp)
		}

	}

	// Return the search history map. It could be empty, which is fine.
	return sh, nil
}

// KeepNewest removes all entries from the searchHistory except the N entries with the highest UNIX timestamp
func (sh SearchHistory) KeepNewest(n int) SearchHistory {
	searchHistoryMutex.RLock()
	l := len(sh)
	searchHistoryMutex.RUnlock()

	if l <= n {
		return sh
	}

	keys := make([]int64, 0, l)

	// Note that if there are timestamp collisions, the loss of rembembering a search in a file is acceptable.
	// Collisions are unlikely, though.

	searchHistoryMutex.RLock()
	for timestamp := range sh {
		keys = append(keys, timestamp.Unix())
	}

	// Reverse sort
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	keys = keys[:n] // Keep only 'n' newest timestamps

	newSearchHistory := make(SearchHistory, n)

	for _, timestamp := range keys {
		t := time.Unix(timestamp, 0)
		newSearchHistory[t] = sh[t]
	}
	searchHistoryMutex.RUnlock()

	return newSearchHistory
}

// LastAdded returns the search entry that was added last
func (sh SearchHistory) LastAdded() string {
	searchHistoryMutex.RLock()
	l := len(sh)
	searchHistoryMutex.RUnlock()

	if l == 0 {
		return ""
	}

	const newestFirst = true
	return sh.GetIndex(0, newestFirst)
}

// AddAndSave culls the search history, adds the given search term and then
// saves the current search history in the background, ignoring any errors.
func (sh *SearchHistory) AddAndSave(searchTerm string) {
	if sh.LastAdded() == searchTerm {
		return
	}

	// Set the given search term, overwriting the previous timestamp if needed
	sh.Set(searchTerm)

	// Cull the history
	searchHistoryMutex.RLock()
	l := len(*sh)
	searchHistoryMutex.RUnlock()

	if l > maxSearchHistoryEntries {
		culledSearchHistory := sh.KeepNewest(maxSearchHistoryEntries)

		searchHistoryMutex.Lock()
		*sh = culledSearchHistory
		searchHistoryMutex.Unlock()
	}

	// Save the search history in the background
	go func() {
		// Ignore any errors, since saving the search history is not that important
		_ = sh.Save(searchHistoryFilename)
	}()
}
