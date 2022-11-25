package main

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	regexpSequenceSuffix = regexp.MustCompile(`[0-9]+$`)
)

func SequenceIDFromHostname() (id uint64, err error) {
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
