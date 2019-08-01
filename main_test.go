package main

import (
	"strings"
	"testing"
)

func TestExtractPoilcyNumber(t *testing.T) {
	key := "unprocessed_1234567890.pdf"
	polNumber := key[strings.Index(key, "_"):strings.Index(key, ".")]
	if polNumber == "1234567890" {
		t.Errorf("policy number not found")
	}
}
