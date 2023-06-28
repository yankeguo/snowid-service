package ufx

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"
)

// ServerParams params
type ServerParams struct {
	Listen string `json:"listen" default:":8080" validate:"required"`

	Path struct {
		Readiness string `json:"readiness" default:"/debug/ready" validate:"required"`
		Liveness  string `json:"liveness" default:"/debug/alive" validate:"required"`
		Metrics   string `json:"metrics" default:"/debug/metrics" validate:"required"`
	} `json:"path"`

	Delay struct {
		Start time.Duration `json:"start" default:"3s"`
		Stop  time.Duration `json:"stop" default:"3s"`
	} `json:"delay"`
}

// ServerParamsFromConf create ServerParams from flag.FlagSet
func ServerParamsFromConf(conf Conf) (opts ServerParams, err error) {
	err = conf.Bind(&opts, "server")
	return
}

// Server the main interface of [summer]
type Server interface {
	// Handler inherit [http.Handler]
	http.Handler
}

type server struct {
	ServerParams

	Prober
	Router

	hProm http.Handler
}

func (a *server) serveReadiness(rw http.ResponseWriter, req *http.Request) {
	c := newContext(rw, req)
	defer c.Perform()

	s, ready := a.CheckReadiness(c)

	status := http.StatusInternalServerError
	if ready {
		status = http.StatusOK
	}

	c.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	c.Code(status)
	c.Text(s)
}

func (a *server) serveLiveness(rw http.ResponseWriter, req *http.Request) {
	c := newContext(rw, req)
	defer c.Perform()

	c.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	if a.CheckLiveness() {
		c.Code(http.StatusOK)
		c.Text("OK")
	} else {
		c.Code(http.StatusInternalServerError)
		c.Text("CASCADED FAILURE")
	}
}

func (a *server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// alive, ready, metrics
	if req.URL.Path == a.Path.Readiness {
		// support readinessPath == livenessPath
		a.serveReadiness(rw, req)
		return
	} else if req.URL.Path == a.Path.Liveness {
		a.serveLiveness(rw, req)
		return
	} else if req.URL.Path == a.Path.Metrics {
		a.hProm.ServeHTTP(rw, req)
		return
	}

	// pprof
	if strings.HasPrefix(req.URL.Path, "/debug/pprof") {
		http.DefaultServeMux.ServeHTTP(rw, req)
		return
	}

	// serve with main handler
	a.Router.ServeHTTP(rw, req)
}

type ServerOptions struct {
	fx.In
	fx.Lifecycle

	ServerParams
	Prober
	Router
}

// NewServer create an [Server] with [Option]
func NewServer(opts ServerOptions) Server {
	a := &server{
		ServerParams: opts.ServerParams,
		Prober:       opts.Prober,
		Router:       opts.Router,
		hProm:        promhttp.Handler(),
	}
	if opts.Lifecycle != nil {
		hs := &http.Server{
			Addr:    opts.Listen,
			Handler: a,
		}
		opts.Lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				chErr := make(chan error, 1)
				go func() {
					chErr <- hs.ListenAndServe()
				}()
				select {
				case err := <-chErr:
					return err
				case <-ctx.Done():
					return hs.Shutdown(ctx)
				case <-time.After(opts.Delay.Start):
					return nil
				}
			},
			OnStop: func(ctx context.Context) error {
				time.Sleep(opts.Delay.Stop)
				return hs.Shutdown(ctx)
			},
		})
	}
	return a
}
