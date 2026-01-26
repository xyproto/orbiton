package megafile

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
)

var errNoUndoForDir = errors.New("nothing to undo in current directory")

func encodeUndoField(value string) string {
	if value == "" {
		return ""
	}
	return url.PathEscape(value)
}

func decodeUndoField(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	return url.PathUnescape(value)
}

func (s *State) loadUndoHistory() {
	if s.undoHistoryPath == "" {
		return
	}
	file, err := os.Open(s.undoHistoryPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}
		original, err := decodeUndoField(parts[0])
		if err != nil {
			continue
		}
		hash, err := decodeUndoField(parts[1])
		if err != nil {
			continue
		}
		trash, err := decodeUndoField(parts[2])
		if err != nil {
			continue
		}
		if original == "" || trash == "" {
			continue
		}
		s.trashUndo = append(s.trashUndo, trashEntry{
			original: original,
			trash:    trash,
			hash:     hash,
		})
	}
}

func (s *State) appendUndoHistory(entry trashEntry) error {
	if s.undoHistoryPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.undoHistoryPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(s.undoHistoryPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "%s\t%s\t%s\n",
		encodeUndoField(entry.original),
		encodeUndoField(entry.hash),
		encodeUndoField(entry.trash),
	)
	return err
}

func (s *State) writeUndoHistory() error {
	if s.undoHistoryPath == "" {
		return nil
	}
	if len(s.trashUndo) == 0 {
		return files.RemoveFile(s.undoHistoryPath)
	}
	if err := os.MkdirAll(filepath.Dir(s.undoHistoryPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(s.undoHistoryPath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, entry := range s.trashUndo {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n",
			encodeUndoField(entry.original),
			encodeUndoField(entry.hash),
			encodeUndoField(entry.trash),
		); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func (s *State) undoTrash(currentDir string) (trashEntry, error) {
	if len(s.trashUndo) == 0 {
		return trashEntry{}, errNoUndoForDir
	}
	currentDir = filepath.Clean(currentDir)
	for i := len(s.trashUndo) - 1; i >= 0; i-- {
		entry := s.trashUndo[i]
		entryDir := filepath.Clean(filepath.Dir(entry.original))
		if entryDir != currentDir {
			continue
		}
		if err := s.restoreTrashEntry(entry); err != nil {
			return trashEntry{}, err
		}
		s.trashUndo = append(s.trashUndo[:i], s.trashUndo[i+1:]...)
		_ = s.writeUndoHistory()
		return entry, nil
	}
	return trashEntry{}, errNoUndoForDir
}
