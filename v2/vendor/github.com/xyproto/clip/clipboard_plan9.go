// Copyright 2013 @atotto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build plan9
// +build plan9

package clip

import (
	"io/ioutil"
	"os"
)

func readAll(_ ...bool) (string, error) {
	f, err := os.Open("/dev/snarf")
	if err != nil {
		return "", err
	}
	defer f.Close()
	str, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(str), nil
}

func readAllBytes(_ ...bool) ([]byte, error) {
	f, err := os.Open("/dev/snarf")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeAll(text string, _ ...bool) error {
	f, err := os.OpenFile("/dev/snarf", os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(text))
	if err != nil {
		return err
	}
	return nil
}

func writeAllBytes(data []byte, _ ...bool) error {
	f, err := os.OpenFile("/dev/snarf", os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}
