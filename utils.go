package main

import (
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	regexpSequenceSuffix = regexp.MustCompile(`[0-9]+$`)
)

func extractWorkerID() (id uint64, err error) {
	var (
		envWorkerID string
	)

	if envWorkerID = strings.TrimSpace(os.Getenv("WORKER_ID")); envWorkerID != "" {
		if id, err = strconv.ParseUint(strings.TrimSpace(os.Getenv("WORKER_ID")), 10, 64); err != nil {
			return
		}
	} else {
		log.Println("guessing worker id from hostname")
		if id, err = sequenceIDFromHostname(); err != nil {
			return
		}
	}
	return
}

func sequenceIDFromHostname() (id uint64, err error) {
	var hostname string
	if hostname = strings.TrimSpace(os.Getenv("HOSTNAME")); hostname == "" {
		if hostname, err = os.Hostname(); err != nil {
			return
		}
	}

	if match := regexpSequenceSuffix.FindStringSubmatch(hostname); len(match) == 0 {
		err = errors.New("no sequence id in hostname")
	} else {
		id, err = strconv.ParseUint(match[0], 10, 64)
	}

	return
}
