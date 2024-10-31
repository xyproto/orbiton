package main

import (
	"github.com/fsnotify/fsnotify"
)

// StartMonitoring will start monitoring the current file for changes
// and reload the file whenever it changes.
func (e *Editor) StartMonitoring(c *Canvas, tty *TTY, status *StatusBar) error {

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

				// Handle the received event, for the currently monitored file(s)
				if event.Op&fsnotify.Write == fsnotify.Write { // write event?
					//logf("FILE WRITE\n")

					status.Clear(c, false)
					status.SetMessage("Reloading " + e.filename)
					status.Show(c, e)

					if err := e.Reload(c, tty, status, nil); err != nil {
						status.ClearAll(c, false)
						status.SetError(err)
						status.Show(c, e)
					}

					//const drawLines = true
					//e.FullResetRedraw(c, status, drawLines, false)

					e.redraw.Store(true)
					e.redrawCursor.Store(true)
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
