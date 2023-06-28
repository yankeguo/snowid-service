package halt

import (
	"errors"
	"net/http"
)

const (
	ExtraKeyMessage = "message"
)

// Option configuration function for [Halt]
type Option func(h *Halt)

// WithStatusCode a [Option] setting status code
func WithStatusCode(code int) Option {
	return func(h *Halt) {
		h.statusCode = code
	}
}

// WithBadRequest alias to [WithStatusCode] with [http.StatusBadRequest]
func WithBadRequest() Option {
	return WithStatusCode(http.StatusBadRequest)
}

// WithMessage a [Option] overriding message key
func WithMessage(m string) Option {
	return WithExtra(ExtraKeyMessage, m)
}

// WithExtra a [Option] setting extras with a key-value
func WithExtra(k string, v any) Option {
	return func(h *Halt) {
		if h.extras == nil {
			h.extras = map[string]any{}
		}
		h.extras[k] = v
	}
}

// WithExtras a [Option] setting extras with key-values
func WithExtras(m map[string]any) Option {
	return func(h *Halt) {
		if h.extras == nil {
			h.extras = map[string]any{}
		}
		for k, v := range m {
			h.extras[k] = v
		}
	}
}

type withStatusCode interface {
	StatusCode() int
}

type withExtract interface {
	ExtractExtras(m map[string]any)
}

type withUnwrap interface {
	Unwrap() error
}

var (
	_ withStatusCode = &Halt{}
	_ withExtract    = &Halt{}
	_ withUnwrap     = &Halt{}
)

type Halt struct {
	error
	statusCode int
	extras     map[string]any
}

func (h *Halt) Unwrap() error {
	return h.error
}

func (h *Halt) StatusCode() int {
	return h.statusCode
}

func (h *Halt) ExtractExtras(m map[string]any) {
	for k, v := range h.extras {
		m[k] = v
	}
}

// New create a new [Halt] as an error
func New(err error, opts ...Option) error {
	he := &Halt{
		error:      err,
		statusCode: http.StatusInternalServerError,
	}
	for _, opt := range opts {
		opt(he)
	}
	return he
}

// Error panic with new [Halt]
func Error(err error, opts ...Option) {
	panic(New(err, opts...))
}

// String panic with new [Halt] with plain string
func String(s string, opts ...Option) {
	panic(New(errors.New(s), opts...))
}

// GetStatusCode get status code from previous created halt error
func GetStatusCode(err error) int {
	for {
		if err == nil {
			return http.StatusInternalServerError
		}
		if eh, ok := err.(withStatusCode); ok {
			return eh.StatusCode()
		}
		if eu, ok := err.(withUnwrap); ok {
			err = eu.Unwrap()
		} else {
			break
		}
	}
	return http.StatusInternalServerError
}

// GetExtras get extras from previous created halt error
func GetExtras(err error) (m map[string]any) {
	for {
		if err == nil {
			return
		}
		if m == nil {
			m = map[string]any{
				ExtraKeyMessage: err.Error(),
			}
		}
		if eh, ok := err.(withExtract); ok {
			eh.ExtractExtras(m)
		}
		if eu, ok := err.(withUnwrap); ok {
			err = eu.Unwrap()
		} else {
			break
		}
	}
	return
}
