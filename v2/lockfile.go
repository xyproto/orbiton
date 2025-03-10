package main

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var defaultLockFile = filepath.Join(userCacheDir, "o", "lockfile.txt")

// LockKeeper keeps track of which files are currently being edited by o
type LockKeeper struct {
	lockedFiles  map[string]time.Time // from filename to lockfilestamp
	mut          *sync.RWMutex
	lockFilename string
}

// NewLockKeeper takes an expanded path (not containing ~) to a lock file
// and creates a new LockKeeper struct, without loading the given lock file.
func NewLockKeeper(lockFilename string) *LockKeeper {
	lockMap := make(map[string]time.Time)
	return &LockKeeper{lockMap, &sync.RWMutex{}, lockFilename}
}

// Load loads the contents of the main lockfile
func (lk *LockKeeper) Load() error {
	f, err := os.Open(lk.lockFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Sync()

	dec := gob.NewDecoder(f)
	lockMap := make(map[string]time.Time)
	err = dec.Decode(&lockMap)
	if err != nil {
		return err
	}
	lk.mut.Lock()
	lk.lockedFiles = lockMap
	lk.mut.Unlock()
	return nil
}

// Save writes the contents of the main lockfile
func (lk *LockKeeper) Save() error {
	if noWriteToCache {
		return nil
	}

	// First create the folder for the lock file overview, if needed
	folderPath := filepath.Dir(lk.lockFilename)
	_ = os.MkdirAll(folderPath, 0o755)

	f, err := os.OpenFile(lk.lockFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)

	lk.mut.RLock()
	err = enc.Encode(lk.lockedFiles)
	lk.mut.RUnlock()

	f.Sync()

	return err
}

// Lock marks the given absolute filename as locked.
// If the file is already locked, an error is returned.
func (lk *LockKeeper) Lock(filename string) error {

	// TODO: Make sure not to lock "-" or "/dev/*" files

	var has bool
	lk.mut.RLock()
	_, has = lk.lockedFiles[filename]
	lk.mut.RUnlock()

	if has {
		return errors.New("already locked: " + filename)
	}

	// Add the file to the map
	lk.mut.Lock()
	lk.lockedFiles[filename] = time.Now()
	lk.mut.Unlock()

	return nil
}

// Unlock marks the given absolute filename as unlocked.
// If the file is already unlocked, an error is returned.
func (lk *LockKeeper) Unlock(filename string) error {
	var has bool
	lk.mut.RLock()
	_, has = lk.lockedFiles[filename]
	lk.mut.RUnlock()

	if !has {
		// Caller can ignore this error if they want
		return errors.New("already unlocked: " + filename)
	}

	// Remove the file from the map
	lk.mut.Lock()
	delete(lk.lockedFiles, filename)
	lk.mut.Unlock()

	return nil
}

// GetTimestamp assumes that the file is locked. A blank timestamp may be returned if not.
func (lk *LockKeeper) GetTimestamp(filename string) time.Time {
	var timestamp time.Time

	lk.mut.RLock()
	timestamp = lk.lockedFiles[filename]
	lk.mut.RUnlock()

	return timestamp
}
