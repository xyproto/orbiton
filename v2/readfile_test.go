package main

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

// createDummyFiles generates dummy files with specified size and returns their paths.
func createDummyFiles(numFiles int, fileSize int) ([]string, error) {
	var paths []string
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for range numFiles {
		content := make([]byte, fileSize)
		for j := range content {
			content[j] = charset[rand.Intn(len(charset))]
		}

		tmpfile, err := os.CreateTemp("", "dummyfile")
		if err != nil {
			return nil, err
		}
		_, err = tmpfile.Write(content)
		if err != nil {
			return nil, err
		}
		tmpfile.Close()

		paths = append(paths, tmpfile.Name())
	}

	return paths, nil
}

func BenchmarkReadFileAndProcessLines(b *testing.B) {
	numFiles := 5
	fileSizes := []int{32, 64, 128, 256, 512, 1024, 2 * 1024, 3 * 1024, 7 * 512, 4 * 1024, 5 * 1024, 6 * 1024, 7 * 1024, 8 * 1024, 10 * 1024, 12 * 1024, 16 * 1024, 32 * 1024, 64 * 1024}

	for _, fileSize := range fileSizes {
		filePaths, err := createDummyFiles(numFiles, fileSize)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(fmt.Sprintf("FileSize%dKB", fileSize/1024), func(b *testing.B) {
			e := NewSimpleEditor(80) // Assuming your Editor constructor is named NewSimpleEditor
			for i := 0; i < b.N; i++ {
				for _, filePath := range filePaths {
					_ = e.ReadFileAndProcessLines(filePath)
				}
			}
		})

		// Clean up dummy files
		for _, filePath := range filePaths {
			os.Remove(filePath)
		}
	}
}
