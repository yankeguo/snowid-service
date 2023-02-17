package main

import (
	"context"
	"github.com/guoyk93/snowid"
	"github.com/guoyk93/winter"
	"github.com/guoyk93/winter/wboot"
	"log"
	"strconv"
	"time"
)

const (
	KeySnowIDGenerator = "snowid.Generator"
)

func main() {
	wboot.Main(func() (a winter.App, err error) {
		var (
			id uint64
		)

		if id, err = extractWorkerID(); err != nil {
			return
		}

		log.Println("using worker id:", id)

		a = winter.New(
			winter.WithLivenessPath("/healthz"),
			winter.WithReadinessPath("/healthz"),
			winter.WithMetricsPath("/metrics"),
		)

		{
			var idGen snowid.Generator

			a.Component("snowid").
				Startup(func(ctx context.Context) (err error) {
					var workerID uint64
					if workerID, err = extractWorkerID(); err != nil {
						return
					}
					if idGen, err = snowid.New(snowid.Options{
						Epoch: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
						ID:    workerID,
					}); err != nil {
						return
					}
					return
				}).
				Middleware(func(h winter.HandlerFunc) winter.HandlerFunc {
					return func(ctx winter.Context) {
						ctx.Inject(func(ctx context.Context) context.Context {
							return context.WithValue(ctx, KeySnowIDGenerator, idGen)
						})
						h(ctx)
					}
				}).
				Shutdown(func(ctx context.Context) (err error) {
					idGen.Stop()
					return
				})
		}

		a.HandleFunc("/", func(ctx winter.Context) {
			idGen := ctx.Value(KeySnowIDGenerator).(snowid.Generator)

			args := winter.Bind[struct {
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

		return
	})

}
