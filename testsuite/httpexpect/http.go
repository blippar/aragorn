package httpexpect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/santhosh-tekuri/jsonschema"

	"github.com/blippar/aragorn/testsuite"
)

const (
	defaultRetryCount = 3
	defaultRetryWait  = 30 * time.Second
)

// Suite describes an HTTP test suite.
type Suite struct {
	tests []*test

	client     *http.Client
	retryCount int
	retryWait  time.Duration
}

type test struct {
	name string
	req  *http.Request // Raw HTTP request generated from the request description.
	body []byte        // Bytes buffer from which the httpReq Body will be read from on each request.

	statusCode int
	header     Header

	rawDocument  []byte                 // Raw document
	jsonDocument map[string]interface{} // Decoded json document.

	jsonSchema *jsonschema.Schema     // Compiled jsonschema.
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

		retryCount: defaultRetryCount,
		retryWait:  defaultRetryWait,
		client:     http.DefaultClient,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func newFromJSONData(data []byte) (testsuite.Suite, error) {
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal HTTP test suite: %v", err)
	}
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
	t.req.Body = ioutil.NopCloser(bytes.NewReader(t.body))
	resp, err := s.client.Do(t.req)
	if err != nil {
		return fmt.Errorf("could not do HTTP request: %v", err)
	}
	// NOTE: not closing the body since NewResponse is taking care of that.

	r, err := NewResponse(tr, resp)
	if err != nil {
		return err
	}

	r.StatusCode(t.statusCode)
	r.ContainsHeader(t.header)
	if t.rawDocument != nil {
		if t.jsonDocument != nil {
			r.MatchJSONDocument(t.jsonDocument)
		} else {
			r.MatchRawDocument(t.rawDocument)
		}
	} else if t.jsonSchema != nil {
		r.MatchJSONSchema(t.jsonSchema)
		if t.jsonValues != nil {
			r.ContainsJSONValues(t.jsonValues)
		}
	}
	return nil
}

func init() {
	testsuite.Register("HTTP", newFromJSONData)
}
