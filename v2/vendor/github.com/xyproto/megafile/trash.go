package megafile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
)

type trashEntry struct {
	original string
	trash    string
	hash     string
}

func uniqueTrashPath(trashDir, base string) (string, error) {
	target := filepath.Join(trashDir, base)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target, nil
	}
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for i := 2; i < 1000; i++ {
		candidate := filepath.Join(trashDir, fmt.Sprintf("%s%d%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", errors.New("could not find a free name in trash")
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *State) moveToTrash(path string) (string, string, error) {
	var fileHash string
	if files.File(path) {
		if hash, err := hashFile(path); err == nil {
			fileHash = hash
		}
	}
	trashDir := env.TrashPath()
	if trashDir == "" {
		return "", "", errors.New("trash path unavailable")
	}
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return "", "", err
	}
	target, err := uniqueTrashPath(trashDir, filepath.Base(path))
	if err != nil {
		return "", "", err
	}
	if err := os.Rename(path, target); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return "", "", err
		}
		// Cross-device: copy then remove
		if err := copyFileOrDir(path, target); err != nil {
			return "", "", err
		}
		if err := os.RemoveAll(path); err != nil {
			return "", "", err
		}
	}
	return target, fileHash, nil
}

func (s *State) restoreTrashEntry(entry trashEntry) error {
	if _, err := os.Stat(entry.trash); err != nil {
		if os.IsNotExist(err) {
			return errors.New("trashed item no longer exists")
		}
		return err
	}
	if _, err := os.Stat(entry.original); err == nil {
		return fmt.Errorf("original path already exists: %s", entry.original)
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(filepath.Dir(entry.original)); err != nil {
		if os.IsNotExist(err) {
			return errors.New("original directory no longer exists")
		}
		return err
	}
	if entry.hash != "" {
		hash, err := hashFile(entry.trash)
		if err != nil {
			return err
		}
		if hash != entry.hash {
			return errors.New("trashed item has changed since deletion")
		}
	}
	if err := os.Rename(entry.trash, entry.original); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return err
		}
		if err := copyFileOrDir(entry.trash, entry.original); err != nil {
			return err
		}
		return os.RemoveAll(entry.trash)
	}
	return nil
}

// copyFileOrDir copies a file or directory tree from src to dst.
func copyFileOrDir(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst, info)
	}
	return copyFile(src, dst, info)
}

func copyFile(src, dst string, info os.FileInfo) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		eInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath, eInfo); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath, eInfo); err != nil {
				return err
			}
		}
	}
	return nil
}
