package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/xyproto/vt"
)

// StartMonitoring will start monitoring the current file for changes
// and reload the file whenever it changes.
func (e *Editor) StartMonitoring(c *vt.Canvas, tty *vt.TTY, status *StatusBar) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	absFilename, err := e.AbsFilename()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					// event channel closed, return from goroutine
					return
				}
				// Handle Write, Create, and Remove
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					status.Clear(c, false)
					status.SetMessage("Reloading " + e.filename)
					status.Show(c, e)
					if err := e.Reload(c, tty, status, nil); err != nil {
						status.ClearAll(c, false)
						status.SetError(err)
						status.Show(c, e)
					}
					e.redraw.Store(true)
					e.redrawCursor.Store(true)
					// Re-add the watch in case the file was replaced (atomic write),
					// since the original inode may no longer be watched.
					_ = watcher.Add(absFilename)
				} else if event.Op&(fsnotify.Rename|fsnotify.Remove) != 0 {
					// The file was renamed or removed. Re-add the watch so that
					// when a new file appears at the same path, we pick it up.
					_ = watcher.Add(absFilename)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					// event channel closed, return from goroutine
					return
				}
				status.ClearAll(c, true)
				status.SetError(err)
				status.Show(c, e)
			}
		}
	}()

	return watcher.Add(absFilename)
}
