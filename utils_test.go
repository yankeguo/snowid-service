package main

import (
	"os"
	"testing"
)

func TestExtractWorkerID(t *testing.T) {
	os.Setenv("WORKER_ID", "2")

	id, err := extractWorkerID()
	if err != nil {
		t.Fatal("found error")
	}
	if id != 2 {
		t.Fatal("id not 2")
	}

	os.Setenv("WORKER_ID", "")
	os.Setenv("HOSTNAME", "example-4")

	id, err = extractWorkerID()
	if err != nil {
		t.Fatal("found error")
	}
	if id != 4 {
		t.Fatal("id not 4")
	}
}

func TestSequenceIDFromHostname(t *testing.T) {
	os.Setenv("HOSTNAME", "example-3")

	id, err := sequenceIDFromHostname()
	if err != nil {
		t.Fatal("found error")
	}
	if id != 3 {
		t.Fatal("id not 3")
	}
}
