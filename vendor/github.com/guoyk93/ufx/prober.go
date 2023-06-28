package ufx

import (
	"context"
	"strings"
	"sync/atomic"
)

type CheckerFunc func(ctx context.Context) error

// Prober is a check prober
type Prober interface {
	// CheckLiveness check liveness
	CheckLiveness() bool

	// CheckReadiness check readiness
	CheckReadiness(ctx context.Context) (s string, failed bool)

	// AddChecker add checker
	AddChecker(name string, fn CheckerFunc)
}

type proberItem struct {
	name string
	fn   CheckerFunc
}

type ProberParams struct {
	Readiness struct {
		Cascade int `json:"cascade" default:"5" validate:"min=1"`
	} `json:"readiness"`
}

func ProberParamsFromConf(conf Conf) (params ProberParams, err error) {
	err = conf.Bind(&params, "prober")
	return
}

type prober struct {
	ProberParams

	checkers []proberItem
	failed   int64
}

func NewProber(params ProberParams) Prober {
	return &prober{
		ProberParams: params,
	}
}

func (m *prober) CheckLiveness() bool {
	if m.Readiness.Cascade > 0 {
		return m.failed < int64(m.Readiness.Cascade)
	} else {
		return true
	}
}

func (m *prober) CheckReadiness(ctx context.Context) (result string, ready bool) {
	var results []string

	ready = true

	for _, checker := range m.checkers {
		var (
			name = checker.name
			err  = checker.fn(ctx)
		)
		if err == nil {
			results = append(results, name+": OK")
		} else {
			results = append(results, name+": "+err.Error())
			ready = false
		}
	}

	result = strings.Join(results, "\n")

	if ready {
		atomic.StoreInt64(&m.failed, 0)
	} else {
		atomic.AddInt64(&m.failed, 1)
	}
	return
}

func (m *prober) AddChecker(name string, fn CheckerFunc) {
	m.checkers = append(m.checkers, proberItem{name: name, fn: fn})
}
