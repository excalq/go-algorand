package elastash

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Retrier decides whether to retry a failed HTTP request with Elasticsearch.
type Retrier interface {
	// Retry is called when a request has failed. It decides whether to retry
	// the call, how long to wait for the next call, or whether to return an
	// error (which will be returned to the service that started the HTTP
	// request in the first place).
	//
	// Callers may also use this to inspect the HTTP request/response and
	// the error that happened. Additional data can be passed through via
	// the context.
	Retry(ctx context.Context, retry int, req *http.Request, resp *http.Response, err error) (time.Duration, bool, error)
}

// -- BackoffRetrier --

// BackoffRetrier is an implementation that does nothing but return nil on Retry.
type BackoffRetrier struct {
	backoff Backoff
}

// NewBackoffRetrier returns a retrier that uses the given backoff strategy.
func NewBackoffRetrier() *BackoffRetrier {
	backoff := NewSimpleBackoff()
	return &BackoffRetrier{backoff: *backoff}
}

// Retry calls into the backoff strategy and its wait interval.
func (r *BackoffRetrier) Retry(ctx context.Context, retry int, req *http.Request, resp *http.Response, err error) (time.Duration, bool, error) {
	wait, goahead := r.backoff.Next(retry)
	return wait, goahead, nil
}

// -- Simple Backoff --

// SimpleBackoff takes a list of fixed values for backoff intervals.
// Each call to Next returns the next value from that fixed list.
// After each value is returned, subsequent calls to Next will only return
// the last element.
type Backoff struct {
	sync.Mutex
	ticks  []int
}

// NewSimpleBackoff creates a Backoff algorithm with the specified
// list of fixed intervals in milliseconds.
func NewSimpleBackoff(ticks ...int) *Backoff {
	return &Backoff{
		ticks:  ticks,
	}
}

func (b *Backoff) Next(retry int) (time.Duration, bool) {
	b.Lock()
	defer b.Unlock()

	if retry >= len(b.ticks) {
		return 0, false
	}

	ms := b.ticks[retry]
	return time.Duration(ms) * time.Millisecond, true
}
