package httpexpect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/xeipuuv/gojsonschema"
	
	"github.com/blippar/aragorn/testsuite"
)

const (
	defaultRetryCount = 3
	defaultRetryWait  = 30 * time.Second
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
	body []byte        // Bytes buffer from which the httpReq Body will be read from on each request.

	statusCode int
	header     Header

	document   interface{}
	jsonSchema *gojsonschema.Schema   // Compiled jsonschema.
	jsonValues map[string]interface{} // Decoded JSONValues.
}

// RunOption is a function that configures a Suite.
type RunOption func(*Suite)

// New returns a Suite.
func New(cfg *Config, opts ...RunOption) (*Suite, error) {
	tests, err := cfg.genTests()
	if err != nil {
		return nil, err
	}
	s := &Suite{
		tests: tests,

		client: http.DefaultClient,

		retryCount: defaultRetryCount,
		retryWait:  defaultRetryWait,
	}
	for _, opt := range opts {
		opt(s)
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

// WithHTTPClient can be used to specify the http.Client to use when making HTTP requests.
func WithHTTPClient(c *http.Client) RunOption {
	return func(s *Suite) {
		s.client = c
	}
}

// WithRetryPolicy can be used to specify the retry policy to use when making HTTP requests.
func WithRetryPolicy(n int, wait time.Duration) RunOption {
	return func(s *Suite) {
		s.retryCount = n
		s.retryWait = wait
	}
}

// Run runs all the tests in the suite.
func (s *Suite) Run(r testsuite.Report) {
	for _, t := range s.tests {
		tr := r.AddTest(t.name)
		s.runTestWithRetry(t, tr)
		tr.Done()
	}
}

// runTestWithRetry will try to run the test t up to n times, waiting for n * wait time
// in between each try. It returns the error of the last tentative if none is sucessful,
// nil otherwise.
func (s *Suite) runTestWithRetry(t *test, tr testsuite.TestReport) {
	for attempt := 1; ; attempt++ {
		err := s.runTest(t, tr)
		if err == nil {
			return
		}
		if attempt > s.retryCount {
			tr.Errorf("could not run test after %d attempts: %v", attempt, err)
			return
		}
		time.Sleep(s.retryWait * time.Duration(attempt))
	}
}

func (s *Suite) runTest(t *test, tr testsuite.TestReport) error {
	req := t.cloneRequest()
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not do HTTP request: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read body: %v", err)
	}
	r := NewResponse(tr, resp, body)
	r.StatusCode(t.statusCode)
	r.ContainsHeader(t.header)
	if t.document != nil {
		raw, ok := t.document.([]byte)
		if ok {
			r.MatchRawDocument(raw)
		} else {
			r.MatchJSONDocument(t.document)
		}
	}
	if t.jsonSchema != nil {
		r.MatchJSONSchema(t.jsonSchema)
	}
	if t.jsonValues != nil {
		r.ContainsJSONValues(t.jsonValues)
	}
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
	if len(t.body) > 0 {
		r2.Body = ioutil.NopCloser(bytes.NewReader(t.body))
	}
	return r2
}

func init() {
	testsuite.Register("HTTP", NewSuiteFromJSON)
}
