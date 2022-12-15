package elastash

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	// DefaultURL is the default endpoint of Elasticsearch on the local machine.
	// It is used e.g. when initializing a new Client without a specific URL.
	DefaultURL = "http://127.0.0.1:9200"

	// DefaultScheme is the default protocol scheme to use when sniffing
	// the Elasticsearch cluster.
	DefaultScheme = "http"

	// DefaultHealthcheckEnabled specifies if healthchecks are enabled by default.
	DefaultHealthcheckEnabled = true

	// DefaultHealthcheckTimeoutStartup is the time the healthcheck waits
	// for a response from Elasticsearch on startup, i.e. when creating a
	// client. After the client is started, a shorter timeout is commonly used
	// (its default is specified in DefaultHealthcheckTimeout).
	DefaultHealthcheckTimeoutStartup = 5 * time.Second

	// DefaultHealthcheckTimeout specifies the time a running client waits for
	// a response from Elasticsearch. Notice that the healthcheck timeout
	// when a client is created is larger by default (see DefaultHealthcheckTimeoutStartup).
	DefaultHealthcheckTimeout = 1 * time.Second

	// DefaultHealthcheckInterval is the default interval between
	// two health checks of the nodes in the cluster.
	DefaultHealthcheckInterval = 60 * time.Second

	// DefaultGzipEnabled specifies if gzip compression is enabled by default.
	DefaultGzipEnabled = false
)

var (
	// ErrNoClient is raised when no Elasticsearch node is available.
	ErrNoClient = errors.New("no Elasticsearch node available")

	// ErrRetry is raised when a request cannot be executed after the configured
	// number of retries.
	ErrRetry = errors.New("cannot connect after several retries")

	// ErrTimeout is raised when a request timed out, e.g. when WaitForStatus
	// didn't return in time.
	ErrTimeout = errors.New("timeout")

)

// Client was an Elasticsearch client, but heavily simplified.
type Client struct {
	c *http.Client // net/http Client to use for requests

	connsMu sync.RWMutex // connsMu guards the next block
	conns   []*conn      // all connections
	cindex  int          // index into conns

	mu                        sync.RWMutex    // guards the next block
	url                       string          // set of URLs passed initially to the client
	log         	          Logger          // logger passed from outter application
	healthcheckEnabled        bool            // healthchecks enabled or disabled
	healthcheckTimeout        time.Duration   // time the healthcheck waits for a response from Elasticsearch
	decoder					  Decoder         // used to decode data sent from Elasticsearch
	basicAuth                 bool            // indicates whether to send HTTP Basic Auth credentials
	basicAuthUsername         string          // username for HTTP Basic Auth
	basicAuthPassword         string          // password for HTTP Basic Auth
	gzipEnabled               bool            // gzip compression enabled or disabled (default)
	retrier                   Retrier         // strategy for retries
}

// Do executes the operation. Body is a JSON string
func (s *Client) Post(ctx context.Context, urlPath string, body interface{}) (*IndexResponse, error) {

	// Get HTTP response
	res, err := s.PerformRequest(ctx, PerformRequestOptions{
		Method: "POST",
		Path:   urlPath,
		Params: url.Values{},
		Body:   body,
	})
	if err != nil {
		return nil, err
	}

	// Return operation response
	ret := new(IndexResponse)
	if err := s.decoder.Decode(res.Body, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// IndexResponse is the result of indexing a document in Elasticsearch. [Simplified]
type IndexResponse struct {
	Index         string      `json:"_index,omitempty"`
	Type          string      `json:"_type,omitempty"`
	Id            string      `json:"_id,omitempty"`
	Version       int64       `json:"_version,omitempty"`
	Result        string      `json:"result,omitempty"`
	Status        int         `json:"status,omitempty"`
}


// -- PerformRequest --

// PerformRequestOptions must be passed into PerformRequest.
type PerformRequestOptions struct {
	Method          string
	Path            string
	Params          url.Values
	Body            interface{}
	ContentType     string
	IgnoreErrors    []int
	Retrier         Retrier
	Headers         http.Header
	MaxResponseSize int64
}



// PerformRequest does a HTTP request to Elasticsearch.
// It returns a response (which might be nil) and an error on failure.
//
// Optionally, a list of HTTP error codes to ignore can be passed.
// This is necessary for services that expect e.g. HTTP status 404 as a
// valid outcome (Exists, IndicesExists, IndicesTypeExists).
func (c *Client) PerformRequest(ctx context.Context, opt PerformRequestOptions) (*Response, error) {
	start := time.Now().UTC()

	c.mu.RLock()
	timeout := c.healthcheckTimeout
	basicAuth := c.basicAuth
	basicAuthUsername := c.basicAuthUsername
	basicAuthPassword := c.basicAuthPassword
	gzipEnabled := c.gzipEnabled
	retrier := c.retrier
	if opt.Retrier != nil {
		retrier = opt.Retrier
	}
	c.mu.RUnlock()

	var err error
	var conn *conn
	var req *Request
	var resp *Response
	var retried bool
	var n int

	for {
		pathWithParams := opt.Path
		if len(opt.Params) > 0 {
			pathWithParams += "?" + opt.Params.Encode()
		}

		// Get a connection
		conn, err = c.next()
		if errors.Cause(err) == ErrNoClient {
			n++
			if !retried {
				// Force a healtcheck as all connections seem to be dead.
				c.healthcheck(ctx, timeout, false)
			}
			wait, ok, rerr := retrier.Retry(ctx, n, nil, nil, err)
			if rerr != nil {
				return nil, rerr
			}
			if !ok {
				return nil, err
			}
			retried = true
			time.Sleep(wait)
			continue // try again
		}
		if err != nil {
			c.log.Error("elastic: cannot get connection from pool")
			return nil, err
		}

		req, err = NewRequest(opt.Method, conn.URL()+pathWithParams)
		if err != nil {
			c.log.Errorf("elastic: cannot create request for %s %s: %v", strings.ToUpper(opt.Method), conn.URL()+pathWithParams, err)
			return nil, err
		}

		if basicAuth {
			req.SetBasicAuth(basicAuthUsername, basicAuthPassword)
		}
		if opt.ContentType != "" {
			req.Header.Set("Content-Type", opt.ContentType)
		}

		if len(opt.Headers) > 0 {
			for key, value := range opt.Headers {
				for _, v := range value {
					req.Header.Add(key, v)
				}
			}
		}

		// Set body
		if opt.Body != nil {
			err = req.SetBody(opt.Body, gzipEnabled)
			if err != nil {
				c.log.Errorf("elastic: couldn't set body %+v for request: %v", opt.Body, err)
				return nil, err
			}
		}

		// Tracing
		c.dumpRequest((*http.Request)(req))

		// Get response
		res, err := c.c.Do((*http.Request)(req).WithContext(ctx))
		if IsContextErr(err) {
			// Proceed, but don't mark the node as dead
			return nil, err
		}
		if err != nil {
			n++
			wait, ok, rerr := retrier.Retry(ctx, n, (*http.Request)(req), res, err)
			if rerr != nil {
				c.log.Errorf("elastic: %s is dead", conn.URL())
				conn.MarkAsDead()
				return nil, rerr
			}
			if !ok {
				c.log.Errorf("elastic: %s is dead", conn.URL())
				conn.MarkAsDead()
				return nil, err
			}
			retried = true
			time.Sleep(wait)
			continue // try again
		}
		if res.Body != nil {
			defer res.Body.Close()
		}

		// Tracing
		c.dumpResponse(res)

		// Log deprecation warnings as errors
		if s := res.Header.Get("Warning"); s != "" {
			c.log.Error(s)
		}

		// Check for errors
		if err := checkResponse((*http.Request)(req), res, opt.IgnoreErrors...); err != nil {
			// No retry if request succeeded
			// We still try to return a response.
			resp, _ = c.newResponse(res, opt.MaxResponseSize)
			return resp, err
		}

		// We successfully made a request with this connection
		conn.MarkAsHealthy()

		resp, err = c.newResponse(res, opt.MaxResponseSize)
		if err != nil {
			return nil, err
		}

		break
	}

	duration := time.Now().UTC().Sub(start)
	c.log.Infof("%s %s [status:%d, request:%.3fs]",
		strings.ToUpper(opt.Method),
		req.URL,
		resp.StatusCode,
		float64(int64(duration/time.Millisecond))/1000)

	return resp, nil
}

// next returns the next available connection, or ErrNoClient.
func (c *Client) next() (*conn, error) {
	// We do round-robin here.
	// TODO(oe) This should be a pluggable strategy, like the Selector in the official clients.
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	i := 0
	numConns := len(c.conns)
	for {
		i++
		if i > numConns {
			break // we visited all conns: they all seem to be dead
		}
		c.cindex++
		if c.cindex >= numConns {
			c.cindex = 0
		}
		conn := c.conns[c.cindex]
		if !conn.IsDead() {
			return conn, nil
		}
	}


	// We have a deadlock here: All nodes are marked as dead.
	// So we are marking them as alive
	// They'll then be picked up in the next call to PerformRequest.
	c.log.Errorf("elastic: all %d nodes marked as dead; resurrecting them to prevent deadlock", len(c.conns))
	for _, conn := range c.conns {
		conn.MarkAsAlive()
	}

	// We tried hard, but there is no node available
	return nil, errors.Wrap(ErrNoClient, "no available connection")
}

// dumpRequest dumps the given HTTP request to the trace log.
func (c *Client) dumpRequest(r *http.Request) {
	if c.log != nil {
		out, err := httputil.DumpRequestOut(r, true)
		if err == nil {
			c.log.Debugf("%s\n", string(out))
		}
	}
}

// dumpResponse dumps the given HTTP response to the trace log.
func (c *Client) dumpResponse(resp *http.Response) {
	if c.log != nil {
		out, err := httputil.DumpResponse(resp, true)
		if err == nil {
			c.log.Debug("%s\n", string(out))
		}
	}
}

// healthcheck does a health check on all nodes in the cluster. Depending on
// the node state, it marks connections as dead, sets them alive etc.
// If healthchecks are disabled and force is false, this is a no-op.
// The timeout specifies how long to wait for a response from Elasticsearch.
func (c *Client) healthcheck(parentCtx context.Context, timeout time.Duration, force bool) {
	c.mu.RLock()
	if !c.healthcheckEnabled && !force {
		c.mu.RUnlock()
		return
	}
	basicAuth := c.basicAuth
	basicAuthUsername := c.basicAuthUsername
	basicAuthPassword := c.basicAuthPassword
	c.mu.RUnlock()

	c.connsMu.RLock()
	conns := c.conns
	c.connsMu.RUnlock()

	for _, conn := range conns {
		// Run the HEAD request against ES with a timeout
		ctx, cancel := context.WithTimeout(parentCtx, timeout)
		defer cancel()

		// Goroutine executes the HTTP request, returns an error and sets status
		var status int
		errc := make(chan error, 1)
		go func(url string) {
			req, err := NewRequest("HEAD", url)
			if err != nil {
				errc <- err
				return
			}
			if basicAuth {
				req.SetBasicAuth(basicAuthUsername, basicAuthPassword)
			}
			res, err := c.c.Do((*http.Request)(req).WithContext(ctx))
			if res != nil {
				status = res.StatusCode
				if res.Body != nil {
					res.Body.Close()
				}
			}
			errc <- err
		}(conn.URL())

		// Wait for the Goroutine (or its timeout)
		select {
		case <-ctx.Done(): // timeout
			c.log.Errorf("elastic: %s is dead", conn.URL())
			conn.MarkAsDead()
		case err := <-errc:
			if err != nil {
				c.log.Errorf("elastic: %s is dead", conn.URL())
				conn.MarkAsDead()
				break
			}
			if status >= 200 && status < 300 {
				conn.MarkAsAlive()
			} else {
				conn.MarkAsDead()
				c.log.Errorf("elastic: %s is dead [status=%d]", conn.URL(), status)
			}
		}
	}
}
