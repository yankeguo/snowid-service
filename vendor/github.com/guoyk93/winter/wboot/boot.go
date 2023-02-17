package wboot

import (
	"context"
	"github.com/guoyk93/rg"
	"github.com/guoyk93/winter"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Main best practice of running a [winter.App]
func Main(fn func() (a winter.App, err error)) {
	var err error
	defer func() {
		if err == nil {
			return
		}
		log.Println("exited with error:", err.Error())
		os.Exit(1)
	}()
	defer rg.Guard(&err)

	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(rg.Must(zipkin.New(""))),
	))
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
			b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader|b3.B3SingleHeader)),
		),
	)

	ctx := context.Background()

	a := rg.Must(fn())
	rg.Must0(a.Startup(ctx))
	defer a.Shutdown(ctx)

	s := &http.Server{
		Addr:    envOr("BIND", "") + ":" + envOr("PORT", "8080"),
		Handler: a,
	}

	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		chErr <- s.ListenAndServe()
	}()

	select {
	case err = <-chErr:
		return
	case sig := <-chSig:
		log.Println("signal caught:", sig.String())
		time.Sleep(time.Second * 3)
	}

	err = s.Shutdown(ctx)
}
