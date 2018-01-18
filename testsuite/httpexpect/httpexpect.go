package httpexpect

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/xeipuuv/gojsonschema"

	"github.com/blippar/aragorn/testsuite"
)

const (
	defaultRetryCount = 1
	defaultRetryWait  = 1 * time.Second
)

// Suite describes an HTTP test suite.
type Suite struct {
	tests []*test

	client *http.Client

	retryCount int
	retryWait  time.Duration
}

type test struct {
	name string
	req  *http.Request // Raw HTTP request generated from the request description.

	statusCode int
	header     Header

	document   interface{}
	jsonSchema *gojsonschema.Schema   // Compiled jsonschema.
	jsonValues map[string]interface{} // Decoded JSONValues.
}

// RunOption is a function that configures a Suite.
type RunOption func(*Suite)

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
	return s, nil
}

// NewSuiteFromJSON returns a `testsuite.Suite` using the cfg to construct the config.
func NewSuiteFromJSON(path string, data []byte) (testsuite.Suite, error) {
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal HTTP test suite: %v", err)
	}
	cfg.Path = path
	return New(cfg)
}

// Run runs all the tests in the suite.
func (s *Suite) Run(r testsuite.Report, failfast bool) {
	for _, t := range s.tests {
		tr := r.AddTest(t.name)
		err := s.runTestWithRetry(t)
		if err != nil {
			r.Log(err)
		}
		tr.Done()
		if err != nil && failfast {
			return
		}
	}
}

// runTestWithRetry will try to run the test t up to n times, waiting for n * wait time
// in between each try. It returns the error of the last tentative if none is sucessful,
// nil otherwise.
func (s *Suite) runTestWithRetry(t *test) error {
	if s.retryCount == 1 {
		_, err := s.runTest(t)
		return err
	}
	for attempt := 1; ; attempt++ {
		retry, err := s.runTest(t)
		if !retry {
			return err
		}
		if attempt >= s.retryCount {
			return fmt.Errorf("could not run test after %d attempts: %v", attempt, err)
		}
		time.Sleep(s.retryWait * time.Duration(attempt))
	}
}

func (s *Suite) runTest(t *test) (bool, error) {
	req := t.cloneRequest()
	resp, err := s.client.Do(req)
	if err != nil {
		return true, fmt.Errorf("could not do HTTP request: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return true, fmt.Errorf("could not read body: %v", err)
	}
	r := NewResponse(t, resp, body)
	return false, r.Check()
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
	testsuite.Register("HTTP", NewSuiteFromJSON)
}
