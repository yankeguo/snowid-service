package main

import (
	"context"
	"github.com/guoyk93/snowid"
	"github.com/guoyk93/summer"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	var err error
	defer func() {
		if err == nil {
			return
		}
		log.Println("exited with error:", err.Error())
		os.Exit(1)
	}()

	type Options struct {
		Bind     string
		Port     string
		WorkerID string
	}

	var opts Options

	// detect bind
	opts.Bind = strings.TrimSpace(os.Getenv("BIND"))

	// detect port
	if opts.Port = strings.TrimSpace(os.Getenv("PORT")); opts.Port == "" {
		opts.Port = "8080"
	}

	// detect workerID
	var workerID uint64

	if opts.WorkerID = strings.TrimSpace(os.Getenv("WORKER_ID")); opts.WorkerID != "" {
		if workerID, err = strconv.ParseUint(strings.TrimSpace(os.Getenv("WORKER_ID")), 10, 64); err != nil {
			return
		}
	} else {
		log.Println("guessing worker id from hostname")
		if workerID, err = SequenceIDFromHostname(); err != nil {
			return
		}
	}

	log.Println("using worker id:", workerID)

	// create the instance
	var idGen snowid.Generator

	if idGen, err = snowid.New(snowid.Options{
		Epoch: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		ID:    workerID,
	}); err != nil {
		return
	}

	a := summer.Basic(
		summer.WithLivenessPath("/healthz"),
		summer.WithReadinessPath("/healthz"),
		summer.WithMetricsPath("/metrics"),
	)

	a.HandleFunc("/", func(ctx summer.Context) {
		args := summer.Bind[struct {
			Size int `json:"size,string"`
		}](ctx)

		if args.Size < 1 {
			args.Size = 1
		}

		var response []string
		for i := 0; i < args.Size; i++ {
			response = append(response, strconv.FormatUint(idGen.NewID(), 10))
		}

		ctx.JSON(response)
	})

	// create server
	s := &http.Server{
		Addr:    opts.Bind + ":" + opts.Port,
		Handler: a,
	}

	log.Println("listening at:", s.Addr)

	// guard
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
