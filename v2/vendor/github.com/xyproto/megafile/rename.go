package megafile

import (
	"os"
	"path/filepath"
	"strings"
)

type renameUIHooks struct {
	clearAndPrepare func()
	clearWritten    func()
	drawWritten     func()
}

type renameSession struct {
	s             *State
	active        bool
	original      string
	selectedIndex int
}

func newRenameSession(s *State) *renameSession {
	return &renameSession{
		s:             s,
		selectedIndex: -1,
	}
}

func (r *renameSession) isActive() bool {
	return r.active
}

func (r *renameSession) reset() {
	r.active = false
	r.original = ""
	r.selectedIndex = -1
}

func (r *renameSession) redrawUI(errText string, hooks renameUIHooks) {
	hooks.clearAndPrepare()
	r.s.ls(r.s.Directories[r.s.dirIndex])
	if r.selectedIndex >= 0 && r.selectedIndex < len(r.s.fileEntries) {
		r.s.clearHighlight()
		r.s.setSelectedIndex(r.selectedIndex)
		r.s.highlightSelection()
	}
	if errText != "" {
		r.s.drawError(errText)
	}
	hooks.clearWritten()
	hooks.drawWritten()
}

func (r *renameSession) redrawAndSelect(targetName string, fallbackIndex int, index *uint, hooks renameUIHooks) {
	hooks.clearAndPrepare()
	r.s.ls(r.s.Directories[r.s.dirIndex])
	r.s.written = []rune{}
	*index = 0
	hooks.clearWritten()
	hooks.drawWritten()
	if len(r.s.fileEntries) == 0 {
		return
	}
	r.s.clearHighlight()
	if targetName != "" {
		for i, entry := range r.s.fileEntries {
			if entry.realName == targetName {
				r.s.setSelectedIndex(i)
				r.s.highlightSelection()
				return
			}
		}
	}
	if fallbackIndex >= 0 && fallbackIndex < len(r.s.fileEntries) {
		r.s.setSelectedIndex(fallbackIndex)
		r.s.highlightSelection()
		return
	}
	if r.s.selectedIndex() >= 0 && r.s.selectedIndex() < len(r.s.fileEntries) {
		r.s.highlightSelection()
	}
}

func (r *renameSession) enter(index *uint, hooks renameUIHooks) {
	selectedIndex := r.s.selectedIndex()
	if selectedIndex < 0 || selectedIndex >= len(r.s.fileEntries) {
		return
	}
	r.active = true
	r.selectedIndex = selectedIndex
	r.original = r.s.fileEntries[selectedIndex].realName
	r.s.filterPattern = ""
	r.s.written = []rune(r.original)
	*index = ulen(r.s.written)
	r.redrawUI("", hooks)
}

func (r *renameSession) handleKey(key string, index *uint, hooks renameUIHooks) (handled bool, shouldDraw bool) {
	if !r.active {
		return false, false
	}

	switch key {
	case "c:27", "c:3", "c:17": // esc, ctrl-c or ctrl-q: cancel rename
		r.active = false
		r.redrawAndSelect(r.original, r.selectedIndex, index, hooks)
		r.original = ""
		r.selectedIndex = -1
		return true, true
	case "c:13": // return: apply rename
		newName := string(r.s.written)
		switch {
		case newName == "":
			r.redrawUI("rename: empty filename", hooks)
			return true, true
		case newName == "." || newName == "..":
			r.redrawUI("rename: invalid filename", hooks)
			return true, true
		case strings.ContainsRune(newName, os.PathSeparator):
			r.redrawUI("rename: filename cannot contain path separator", hooks)
			return true, true
		case newName == r.original:
			r.active = false
			r.redrawAndSelect(r.original, r.selectedIndex, index, hooks)
			r.original = ""
			r.selectedIndex = -1
			return true, true
		}
		currentDir := r.s.Directories[r.s.dirIndex]
		oldPath := filepath.Join(currentDir, r.original)
		newPath := filepath.Join(currentDir, newName)
		if _, err := os.Stat(newPath); err == nil {
			r.redrawUI("rename: target already exists", hooks)
			return true, true
		} else if !os.IsNotExist(err) {
			r.redrawUI(err.Error(), hooks)
			return true, true
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			r.redrawUI(err.Error(), hooks)
			return true, true
		}
		r.active = false
		r.redrawAndSelect(newName, r.selectedIndex, index, hooks)
		r.original = ""
		r.selectedIndex = -1
		return true, true
	case "c:127": // backspace
		if *index > 0 && len(r.s.written) > 0 {
			hooks.clearWritten()
			r.s.written = append(r.s.written[:*index-1], r.s.written[*index:]...)
			*index = *index - 1
			hooks.drawWritten()
		}
		return true, true
	case deleteKey, "c:4": // delete / ctrl-d
		if *index < ulen(r.s.written) {
			hooks.clearWritten()
			r.s.written = append(r.s.written[:*index], r.s.written[*index+1:]...)
			hooks.drawWritten()
		}
		return true, true
	case leftArrow:
		hooks.clearWritten()
		if *index > 0 {
			*index = *index - 1
		}
		hooks.drawWritten()
		return true, true
	case rightArrow:
		hooks.clearWritten()
		if *index < ulen(r.s.written) {
			*index = *index + 1
		}
		hooks.drawWritten()
		return true, true
	case "c:1", homeKey: // ctrl-a, home
		hooks.clearWritten()
		*index = 0
		hooks.drawWritten()
		return true, true
	case "c:5", endKey: // ctrl-e, end
		hooks.clearWritten()
		*index = ulen(r.s.written)
		hooks.drawWritten()
		return true, true
	case "c:11": // ctrl-k
		hooks.clearWritten()
		if len(r.s.written) > 0 {
			r.s.written = r.s.written[:*index]
		}
		hooks.drawWritten()
		return true, true
	case "":
		return true, false
	default:
		if key != " " && strings.TrimSpace(key) == "" {
			return true, false
		}
		hooks.clearWritten()
		tmp := append(r.s.written[:*index], []rune(key)...)
		r.s.written = append(tmp, r.s.written[*index:]...)
		*index += ulen([]rune(key))
		hooks.drawWritten()
		return true, true
	}
}
