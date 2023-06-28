package ufx

import (
	"net/http"
)

// HandlerFunc handler func with [Context] as argument
type HandlerFunc func(c Context)

// Router router interface
type Router interface {
	http.Handler

	HandleFunc(pattern string, fn HandlerFunc)
}

type router struct {
	RouterParams
	m  *http.ServeMux
	cc chan struct{}
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// concurrency
	if r.cc != nil {
		<-r.cc
		defer func() {
			r.cc <- struct{}{}
		}()
	}

	r.m.ServeHTTP(w, req)
}

func (r *router) HandleFunc(pattern string, fn HandlerFunc) {
	r.m.Handle(
		pattern,
		instrumentHTTPHandler(
			pattern,
			http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				c := newContext(rw, req)
				c.loggingResponse = r.Logging.Response
				c.loggingRequest = r.Logging.Request
				func() {
					defer c.Perform()
					fn(c)
				}()
			}),
		),
	)
}

type RouterParams struct {
	Concurrency int `json:"concurrency" default:"128" validate:"min=1"`
	Logging     struct {
		Response bool `json:"response"`
		Request  bool `json:"request"`
	} `json:"logging"`
}

func RouterParamsFromConf(conf Conf) (params RouterParams, err error) {
	err = conf.Bind(&params, "router")
	return
}

func NewRouter(opts RouterParams) Router {
	r := &router{
		RouterParams: opts,
		m:            &http.ServeMux{},
	}

	if opts.Concurrency > 0 {
		r.cc = make(chan struct{}, opts.Concurrency)
		for i := 0; i < opts.Concurrency; i++ {
			r.cc <- struct{}{}
		}
	}

	return r
}
