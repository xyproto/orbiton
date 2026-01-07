package megafile

import (
	"os"

	"github.com/dustin/go-humanize"
)

func fileSizeHuman(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return humanize.IBytes(uint64(fi.Size())), nil
}
