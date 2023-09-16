package main

import (
	"testing"
	"time"
)

func TestGetIndex(t *testing.T) {
	sh := make(SearchHistory)
	now := time.Now()

	// Add some entries to the search history
	sh[now.Add(-1*time.Hour)] = "search1"
	sh[now.Add(-2*time.Hour)] = "search2"
	sh[now.Add(-3*time.Hour)] = "search3"

	tests := []struct {
		index         int
		expectedTerm  string
		expectedExist bool
	}{
		{index: 0, expectedTerm: "search1", expectedExist: true},
		{index: 1, expectedTerm: "search2", expectedExist: true},
		{index: 2, expectedTerm: "search3", expectedExist: true},
		{index: 3, expectedTerm: "", expectedExist: false},
	}

	const newestFirst = true
	for i, test := range tests {
		got := sh.GetIndex(test.index, newestFirst)
		if got != test.expectedTerm {
			t.Errorf("Test %d: expected %q, got %q", i, test.expectedTerm, got)
		}
	}
}

func TestKeepNewest(t *testing.T) {
	sh := make(SearchHistory)
	now := time.Now()

	// Add some entries to the search history
	sh[now.Add(-1*time.Hour)] = "search1"
	sh[now.Add(-2*time.Hour)] = "search2"
	sh[now.Add(-3*time.Hour)] = "search3"

	tests := []struct {
		n           int
		expectedLen int
	}{
		{n: 2, expectedLen: 2},
		{n: 1, expectedLen: 1},
		{n: 3, expectedLen: 3},
		{n: 0, expectedLen: 0},
	}

	for i, test := range tests {
		newSh := sh.KeepNewest(test.n)
		gotLen := newSh.Len()
		if gotLen != test.expectedLen {
			t.Errorf("Test %d: expected length %d, got %d", i, test.expectedLen, gotLen)
		}
	}
}
