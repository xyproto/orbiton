package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/xyproto/vt100"
)

// StartMonitoring will start monitoring the current file for changes
// and reload the file whenever it changes.
func (e *Editor) StartMonitoring(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		status.Clear(c)
		status.SetError(err)
		status.Show(c, e)
	}
	defer watcher.Close()

	absFilename, err := e.AbsFilename()
	if err != nil {
		status.ClearAll(c)
		status.SetError(err)
		status.Show(c, e)
	}

	go func() {

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Handle the received event, for the currently monitored file(s)
				if event.Has(fsnotify.Write) { // is it a write event?
					status.Clear(c)
					status.SetMessage("Reloading " + e.filename)
					status.Show(c, e)

					if err := e.Reload(c, tty, status, nil); err != nil {
						status.ClearAll(c)
						status.SetError(err)
						status.Show(c, e)
					}

					const drawLines = true
					e.FullResetRedraw(c, status, drawLines)
					e.redraw = true
					e.redrawCursor = true
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				status.ClearAll(c)
				status.SetError(err)
				status.Show(c, e)
			}
		}

	}()

	_ = watcher.Add(absFilename)
}
