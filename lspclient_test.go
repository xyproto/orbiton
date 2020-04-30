package main

import (
	"fmt"
	"testing"
)

func TestHello(t *testing.T) {
	result, err := query("hello")
	fmt.Println(result, err)
	if err != nil {
		t.Fail()
	}
}
