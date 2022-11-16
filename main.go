package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/guoyk93/snowid"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	hostnameSequenceIDRegexp = regexp.MustCompile(`[0-9]+$`)
)

func hostnameSequenceID() (id uint64, err error) {
	var hostname string
	if hostname = strings.TrimSpace(os.Getenv("HOSTNAME")); hostname == "" {
		if hostname, err = os.Hostname(); err != nil {
			return
		}
	}

	if match := hostnameSequenceIDRegexp.FindStringSubmatch(hostname); len(match) == 0 {
		err = errors.New("no sequence id in hostname")
	} else {
		id, err = strconv.ParseUint(match[0], 10, 64)
	}

	return
}

func main() {
	var err error
	defer func() {
		if err == nil {
			return
		}
		log.Println("exited with error:", err.Error())
		os.Exit(1)
	}()

	envPort := strings.TrimSpace(os.Getenv("PORT"))

	if envPort == "" {
		envPort = "8080"
	}

	var workerID uint64

	if envWorkerID := strings.TrimSpace(os.Getenv("WORKER_ID")); envWorkerID != "" {
		if workerID, err = strconv.ParseUint(strings.TrimSpace(os.Getenv("WORKER_ID")), 10, 64); err != nil {
			return
		}
	} else {
		log.Println("missing $WORKER_ID, trying hostname")
		if workerID, err = hostnameSequenceID(); err != nil {
			return
		}
	}

	log.Println("using worker id:", workerID)

	var idGen snowid.Generator

	if idGen, err = snowid.New(snowid.Options{
		Epoch: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		ID:    workerID,
	}); err != nil {
		return
	}

	s := &http.Server{
		Addr: ":" + envPort,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/healthz" {
				buf := []byte("OK")
				rw.Header().Set("Content-Type", "text/plain")
				rw.Header().Set("Content-Length", strconv.Itoa(len(buf)))
				_, _ = rw.Write(buf)
				return
			}

			size, _ := strconv.Atoi(req.URL.Query().Get("size"))
			if size < 1 {
				size = 1
			}
			var response []string
			for i := 0; i < size; i++ {
				response = append(response, strconv.FormatUint(idGen.NewID(), 10))
			}
			buf, err := json.Marshal(response)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusServiceUnavailable)
				return
			}
			rw.Header().Set("Content-Type", "application/json")
			rw.Header().Set("Content-Length", strconv.Itoa(len(buf)))
			_, _ = rw.Write(buf)
		}),
	}

	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(chSig)

	go func() {
		chErr <- s.ListenAndServe()
	}()

	select {
	case err = <-chErr:
		return
	case sig := <-chSig:
		log.Println("signal caught:", sig.String())
	}

	time.Sleep(time.Second * 3)

	err = s.Shutdown(context.Background())
	<-chErr

	return
}
