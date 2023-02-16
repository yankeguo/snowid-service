package snowid

import (
	"errors"
	"time"
)

const (
	Uint10Mask = (uint64(1) << 10) - 1
	Uint12Mask = (uint64(1) << 12) - 1
	Uint41Mask = (uint64(1) << 41) - 1
)

var (
	ErrInvalidEpoch = errors.New("failed to create snowid.Generator: invalid Epoch")
	ErrInvalidID    = errors.New("failed to create snowid.Generator: invalid ID")
	ErrStopped      = errors.New("failed to retrieve ID: snowid.Generator stopped")
)

// Clock abstract the standard time package
type Clock interface {
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
}

type defaultClock struct{}

func (defaultClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (defaultClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func DefaultClock() Clock {
	return defaultClock{}
}

// Options options for Generator
type Options struct {
	// Clock custom implementation of clock, default to standard library
	Clock Clock
	// Epoch pre-defined zero time in Snowflake algorithm, required
	Epoch time.Time
	// ID unique unsigned integer indicate the ID of current Generator instance, maximum 10 bits wide, default to 0
	ID uint64
}

// Generator the main interface
type Generator interface {
	// Stop shutdown the instance, release all related resources
	// can not stop twice, NewID() invocation will panic after stopped
	Stop()

	// Count returns the count of generated ids
	Count() uint64

	// NewID returns a new id
	NewID() uint64
}

type generator struct {
	chReq     chan struct{}
	chResp    chan uint64
	chStop    chan struct{}
	epoch     time.Time
	shiftedID uint64
	count     uint64
	clock     Clock
}

// New create a new instance of Generator
func New(opts Options) (Generator, error) {
	if opts.Clock == nil {
		opts.Clock = DefaultClock()
	}
	if opts.Epoch.IsZero() {
		return nil, ErrInvalidEpoch
	}
	if opts.ID&Uint10Mask != opts.ID {
		return nil, ErrInvalidID
	}
	sf := &generator{
		chReq:     make(chan struct{}),
		chResp:    make(chan uint64),
		chStop:    make(chan struct{}),
		epoch:     opts.Epoch,
		shiftedID: opts.ID << 12,
		clock:     opts.Clock,
	}
	go sf.run()
	return sf, nil
}

func (sf *generator) Stop() {
	close(sf.chStop)
}

func (sf *generator) run() {
	var nowT, lastT, seqID uint64
	for {
		select {
		case <-sf.chReq:
		retry:
			nowT = uint64(sf.clock.Since(sf.epoch) / time.Millisecond)
			if nowT == lastT {
				seqID = seqID + 1
				if seqID > Uint12Mask {
					sf.clock.Sleep(time.Millisecond)
					goto retry
				}
			} else {
				lastT = nowT
				seqID = 0
			}
			sf.count++
			sf.chResp <- ((nowT & Uint41Mask) << 22) | sf.shiftedID | seqID
		case <-sf.chStop:
			return
		}
	}
}

func (sf *generator) Count() uint64 {
	return sf.count
}

func (sf *generator) NewID() uint64 {
	select {
	case sf.chReq <- struct{}{}:
		select {
		case v := <-sf.chResp:
			return v
		case <-sf.chStop:
			panic(ErrStopped)
		}
		return <-sf.chResp
	case <-sf.chStop:
		panic(ErrStopped)
	}
}
