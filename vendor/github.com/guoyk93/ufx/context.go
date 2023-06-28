package ufx

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/guoyk93/halt"
	"go.opentelemetry.io/otel/trace"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Bind a generic version of [Context.Bind]
//
// demo:
//
//		func actionValidate(c summer.Context) {
//			args := summer.Bind[struct {
//	       		Tenant       string `json:"header_x_tenant"`
//				Username     string `json:"username"`
//				Age 		 int    `json:"age,string"`
//			}](c)
//	        _ = args.Tenant
//	        _ = args.Username
//	        _ = args.Age
//		}
func Bind[T any](c Context) (o T) {
	c.Bind(&o)
	return
}

// Context context of an incoming request and corresponding response writer
type Context interface {
	// Context extend the [context.Context] interface by proxying to [http.Request.Context]
	context.Context

	// Inject inject underlying [context.Context]
	Inject(fn func(ctx context.Context) context.Context)

	// Req returns the underlying *http.Request
	Req() *http.Request

	// Header returns the headers of underlying [http.ResponseWriter]
	Header() http.Header

	// Bind unmarshal the request data into any struct with json tags
	//
	// HTTP header is prefixed with "header_"
	//
	// HTTP query is prefixed with "query_"
	//
	// both JSON and Form are supported
	Bind(data interface{})

	// Files returns the multipart file headers
	Files() map[string][]*multipart.FileHeader

	// Code set the response code, can be called multiple times
	Code(code int)

	// Body set the response body with content type, can be called multiple times
	Body(contentType string, buf []byte)

	// Text set the response body to plain text
	Text(s string)

	// JSON set the response body to json
	JSON(data interface{})

	// Perform actually perform the response
	// it is suggested to use in defer, recover() is included to recover from any panics
	Perform()
}

type winterContext struct {
	req *http.Request
	rw  http.ResponseWriter

	buf   []byte
	files map[string][]*multipart.FileHeader

	code int
	body []byte

	recvOnce *sync.Once
	sendOnce *sync.Once

	loggingResponse bool
	loggingRequest  bool
}

func (c *winterContext) Deadline() (deadline time.Time, ok bool) {
	return c.req.Context().Deadline()
}

func (c *winterContext) Done() <-chan struct{} {
	return c.req.Context().Done()
}

func (c *winterContext) Err() error {
	return c.req.Context().Err()
}

func (c *winterContext) Value(key any) any {
	return c.req.Context().Value(key)
}

func (c *winterContext) Inject(fn func(ctx context.Context) context.Context) {
	ctx := c.req.Context()
	neo := fn(ctx)
	if neo != nil && neo != ctx {
		c.req = c.req.WithContext(neo)
	}
}

func (c *winterContext) Req() *http.Request {
	return c.req
}

func (c *winterContext) Header() http.Header {
	return c.rw.Header()
}

func (c *winterContext) receive() {
	var (
		m   = map[string]any{}
		f   = map[string][]*multipart.FileHeader{}
		err error
	)
	if err = extractRequest(m, f, c.req); err != nil {
		halt.Error(err, halt.WithStatusCode(http.StatusBadRequest))
	}
	if c.buf, err = json.Marshal(m); err != nil {
		halt.Error(err, halt.WithStatusCode(http.StatusBadRequest))
	}
	c.files = f
}

func (c *winterContext) send() {
	if c.loggingResponse {
		var traceID string
		if sp := trace.SpanFromContext(c); sp != nil {
			traceID = sp.SpanContext().TraceID().String()
		}
		log.Printf("trace_id=%s, code=%d; %s", traceID, c.code, string(c.body))
	}
	c.rw.WriteHeader(c.code)
	_, _ = c.rw.Write(c.body)
}

func (c *winterContext) Bind(data interface{}) {
	c.recvOnce.Do(c.receive)
	if err := json.Unmarshal(c.buf, data); err != nil {
		halt.Error(err, halt.WithStatusCode(http.StatusBadRequest))
	}
}

func (c *winterContext) Files() map[string][]*multipart.FileHeader {
	c.recvOnce.Do(c.receive)
	return c.files
}

func (c *winterContext) Code(code int) {
	c.code = code
}

func (c *winterContext) Body(contentType string, buf []byte) {
	c.rw.Header().Set("Content-Type", contentType)
	c.rw.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	c.rw.Header().Set("X-Content-Type-Options", "nosniff")
	c.body = buf
}

func (c *winterContext) Text(s string) {
	c.Body(ContentTypeTextPlainUTF8, []byte(s))
}

func (c *winterContext) JSON(data interface{}) {
	if buf, err := json.Marshal(data); err != nil {
		halt.Error(err, halt.WithStatusCode(http.StatusBadRequest))
	} else {
		c.Body(ContentTypeApplicationJSONUTF8, buf)
	}
}

func (c *winterContext) Perform() {
	if r := recover(); r != nil {
		var (
			e  error
			ok bool
		)
		if e, ok = r.(error); !ok {
			e = fmt.Errorf("panic: %v", r)
		}
		c.Code(halt.GetStatusCode(e))
		c.JSON(halt.GetExtras(e))
		c.loggingResponse = true
	}
	c.sendOnce.Do(c.send)
}

func newContext(rw http.ResponseWriter, req *http.Request) *winterContext {
	return &winterContext{
		req:      req,
		rw:       rw,
		code:     http.StatusOK,
		recvOnce: &sync.Once{},
		sendOnce: &sync.Once{},
	}
}
