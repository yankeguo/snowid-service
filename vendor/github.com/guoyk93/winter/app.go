package winter

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync/atomic"
)

// HandlerFunc handler func with [Context] as argument
type HandlerFunc func(ctx Context)

// App the main interface of [summer]
type App interface {
	// Handler inherit [http.Handler]
	http.Handler

	// Registry inherit [Registry]
	Registry

	// HandleFunc register an action function with given path pattern
	//
	// This function is similar with [http.ServeMux.HandleFunc]
	HandleFunc(pattern string, fn HandlerFunc)
}

type app struct {
	Registry

	opts options

	mux *http.ServeMux

	hMain http.Handler
	hProm http.Handler
	hProf http.Handler

	cc chan struct{}

	failed int64
}

func (a *app) HandleFunc(pattern string, fn HandlerFunc) {
	a.mux.Handle(
		pattern,
		otelhttp.WithRouteTag(
			pattern,
			http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				c := newContext(rw, req)
				func() {
					defer c.Perform()
					a.Wrap(fn)(c)
				}()
			}),
		),
	)
}

func (a *app) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// alive, ready, metrics
	if req.URL.Path == a.opts.readinessPath {
		// readiness first, works when readinessPath == livenessPath
		sb := &strings.Builder{}
		var failed bool
		a.Check(req.Context(), func(name string, err error) {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(name)
			if err == nil {
				sb.WriteString(": OK")
			} else {
				failed = true
				sb.WriteString(": ")
				sb.WriteString(err.Error())
			}
		})
		if sb.Len() == 0 {
			sb.WriteString("OK")
		}
		status := http.StatusOK
		if failed {
			atomic.AddInt64(&a.failed, 1)
			status = http.StatusInternalServerError
		} else {
			atomic.StoreInt64(&a.failed, 0)
		}
		internalRespond(rw, sb.String(), status)
		return
	} else if req.URL.Path == a.opts.livenessPath {
		if a.opts.readinessCascade > 0 && atomic.LoadInt64(&a.failed) > a.opts.readinessCascade {
			internalRespond(rw, "CASCADED", http.StatusInternalServerError)
		} else {
			internalRespond(rw, "OK", http.StatusOK)
		}
		return
	} else if req.URL.Path == a.opts.metricsPath {
		a.hProm.ServeHTTP(rw, req)
		return
	}

	// pprof
	if strings.HasPrefix(req.URL.Path, "/debug/") {
		a.hProf.ServeHTTP(rw, req)
		return
	}

	// concurrency
	if a.cc != nil {
		<-a.cc
		defer func() {
			a.cc <- struct{}{}
		}()
	}

	a.hMain.ServeHTTP(rw, req)
}

// New create an [App] with [Option]
func New(opts ...Option) App {
	a := &app{
		opts: options{
			concurrency:      128,
			readinessCascade: 5,
			readinessPath:    DefaultReadinessPath,
			livenessPath:     DefaultLivenessPath,
			metricsPath:      DefaultMetricsPath,
		},
	}
	for _, opt := range opts {
		opt(&a.opts)
	}

	a.Registry = NewRegistry()

	a.mux = &http.ServeMux{}

	a.hMain = otelhttp.NewHandler(a.mux, "http")
	a.hProm = promhttp.Handler()
	m := &http.ServeMux{}
	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)
	a.hProf = m

	// create concurrency controller
	if a.opts.concurrency > 0 {
		a.cc = make(chan struct{}, a.opts.concurrency)
		for i := 0; i < a.opts.concurrency; i++ {
			a.cc <- struct{}{}
		}
	}
	return a
}
