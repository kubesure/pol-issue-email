package main

import (
	"strings"
	"testing"

	e "github.com/aws/aws-lambda-go/events"
)

func TestExtractPoilcyNumber(t *testing.T) {
	key := "unprocessed/1234567890.pdf"
	polNumber := key[strings.Index(key, "/")+1 : strings.Index(key, ".")]
	if polNumber != "1234567890" {
		t.Errorf("policy number not found")
	}
}

func TestProcessEvent(t *testing.T) {
	bucket := e.S3Bucket{Name: "kubesure-cs-1"}
	object := e.S3Object{Key: "unprocessed/1234567890.pdf"}
	r := e.S3EventRecord{}
	r.S3.Bucket = bucket
	r.S3.Object = object
	err := processEvent(r)

	if err != nil {
		t.Errorf("S3 Event Processed %v", err)
	}
}
