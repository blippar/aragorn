package httpexpect

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/xeipuuv/gojsonschema"

	"github.com/blippar/aragorn/plugin"
	"github.com/blippar/aragorn/testsuite"
)

const (
	defaultRetryCount = 1
	defaultRetryWait  = 1 * time.Second
	defaultTimeout    = 30 * time.Second
)

var _ testsuite.Suite = (*Suite)(nil)

// Suite describes an HTTP test suite.
type Suite struct {
	tests []*test

	client *http.Client

	retryCount int
	retryWait  time.Duration
	timeout    time.Duration
}

type test struct {
	name    string
	req     *http.Request // Raw HTTP request generated from the request description.
	timeout time.Duration

	statusCode int
	header     Header

	document   interface{}
	jsonSchema *gojsonschema.Schema   // Compiled jsonschema.
	jsonValues map[string]interface{} // Decoded JSONValues.
}

func (t *test) Name() string        { return t.name }
func (t *test) Description() string { return t.req.Method + " " + t.req.URL.String() }

// New returns a Suite.
func New(cfg *Config) (*Suite, error) {
	tests, err := cfg.genTests()
	if err != nil {
		return nil, err
	}
	s := &Suite{
		tests:      tests,
		client:     http.DefaultClient,
		retryCount: defaultRetryCount,
		retryWait:  defaultRetryWait,
	}
	if cfg.Base.OAUTH2.ClientID != "" && cfg.Base.OAUTH2.ClientSecret != "" {
		s.client = cfg.Base.OAUTH2.Client(context.Background())
	}
	if cfg.Base.RetryCount > 0 {
		s.retryCount = cfg.Base.RetryCount
	}
	if cfg.Base.RetryWait > 0 {
		s.retryWait = time.Duration(cfg.Base.RetryWait) * time.Second
	}
	if cfg.Base.Insecure {
		s.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
	return s, nil
}

// Run runs all the tests in the suite.
func (s *Suite) Run(ctx context.Context, r testsuite.Report) {
	failfast := false
	if rpcInfo, ok := testsuite.RPCInfoFromContext(ctx); ok {
		failfast = rpcInfo.FailFast
	}
	for _, t := range s.tests {
		tr := r.AddTest(t)
		s.runTestWithRetry(ctx, t, tr)
		if tr.Done() && failfast {
			return
		}
	}
}

func (s *Suite) Tests() []testsuite.Test {
	tests := make([]testsuite.Test, len(s.tests))
	for i, t := range s.tests {
		tests[i] = t
	}
	return tests
}

// runTestWithRetry will try to run the test t up to n times, waiting for n * wait time
// in between each try. It returns the error of the last tentative if none is sucessful,
// nil otherwise.
func (s *Suite) runTestWithRetry(ctx context.Context, t *test, l Logger) {
	if s.retryCount == 1 {
		if err := s.runTest(ctx, t, l); err != nil {
			l.Error(err)
		}
		return
	}
	for attempt := 1; ; attempt++ {
		err := s.runTest(ctx, t, l)
		if err == nil {
			return
		}
		if attempt >= s.retryCount {
			l.Errorf("could not run test after %d attempts: %v", attempt, err)
			return
		}
		select {
		case <-ctx.Done():
			err = ctx.Err()
			l.Errorf("could not run test after %d attempts: %v", attempt, err)
		case <-time.After(s.retryWait):
		}
	}
}

func (s *Suite) runTest(ctx context.Context, t *test, l Logger) error {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	req := t.cloneRequest()
	req = req.WithContext(ctx)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not do HTTP request: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read body: %v", err)
	}
	checkResponse(t, l, resp, body)
	return nil
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func (t *test) cloneRequest() *http.Request {
	// shallow copy of the struct
	r := t.req
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	if r.Body != nil {
		r2.Body, _ = r.GetBody()
	}
	return r2
}

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.TestSuitePlugin,
		ID:     "HTTP",
		Config: (*Config)(nil),
		InitFn: func(ctx *plugin.InitContext) (interface{}, error) {
			cfg := ctx.Config.(*Config)
			cfg.Path = ctx.Root
			return New(cfg)
		},
	})
}
