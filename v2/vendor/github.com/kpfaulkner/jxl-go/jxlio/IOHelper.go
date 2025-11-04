package jxlio

import (
	"errors"
	"io"
)

const (
	tempBufSize = 10000
)

// returns number of bytes NOT read (remaining) and error.
func ReadFullyWithOffset(in io.ReadSeeker, buffer []byte, offset int, len int) (int, error) {
	remaining := len

	_, err := in.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return 0, err
	}

	// potentially stupidly large buffer... but will leave for now. TODO(kpfaulkner) revisit
	if len > 2*1024*1024*1024 {
		return 0, errors.New("length of read too large")
	}

	tempBuf := make([]byte, len)
	var tempBuffer []byte
	for remaining > 0 {

		//count := in.Read(buffer, offset+len-remaining, remaining)
		count, err := in.Read(tempBuf)
		if err != nil {
			return 0, err
		}

		if count <= 0 {
			break
		}

		// copy tempBuf to buffer.
		tempBuffer = append(tempBuffer, tempBuf[:count]...)
		remaining -= count
	}
	copy(buffer, tempBuffer[:len])
	return remaining, nil
}

func ReadFully(in io.ReadSeeker, buffer []byte) (int, error) {
	return ReadFullyWithOffset(in, buffer, 0, len(buffer))
}

// FIXME(kpfaulkner) really unsure what this is supposed to do. Skip some content... then read more?
func SkipFully(in io.ReadSeeker, n int64) (int, error) {
	remaining := n
	var sz int64
	if n < tempBufSize {
		sz = n
	} else {
		sz = tempBufSize
	}

	tempBuf := make([]byte, sz)
	for remaining > 0 {
		skipped, err := in.Read(tempBuf)
		if err != nil {
			return 0, err
		}

		remaining -= int64(skipped)
		if skipped == 0 {
			break
		}
	}
	if remaining == 0 {
		return 0, nil
	}
	buffer := make([]byte, 4096)
	for remaining > int64(len(buffer)) {
		k, err := ReadFully(in, buffer)
		if err != nil {
			return 0, err
		}
		remaining = remaining - int64(len(buffer)) + int64(k)
		if k != 0 {
			return int(remaining), nil
		}
	}
	return ReadFullyWithOffset(in, buffer, 0, int(remaining))
}
