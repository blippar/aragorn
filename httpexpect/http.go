package httpexpect

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/santhosh-tekuri/jsonschema"
	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
)

const (
	defaultRetryCount = 3
	defaultRetryWait  = 30 * time.Second
)

// Suite describes an HTTP tests suite.
type Suite struct {
	Base  *base
	Tests []test

	name       string
	client     *http.Client
	retryCount int
	retryWait  time.Duration
	notifier   notifier.Notifier
}

type base struct {
	URL     string  // Base URL prepended to all requests' path.
	Headers headers // Base set of headers added to all requests.
}

type test struct {
	Name    string   // Name used to identify this test.
	Request *request // Request describes the HTTP request.
	Expect  *expect  // Expect describes the result of the HTTP request.
}

type request struct {
	URL     string // If set, will overwrite the base URL.
	Path    string
	Method  string
	Headers headers

	// Only one of the three following must be set.
	Body           json.RawMessage
	Multipart      multipartContent
	FormURLEncoded url.Values

	httpReq  *http.Request // Raw HTTP request generated from the request description.
	httpBody []byte        // Bytes buffer from which the httpReq Body will be read from on each request.
}

type expect struct {
	StatusCode int
	Headers    headers

	Document     json.RawMessage        // Exact document to match. Exclusive with JSONSchema.
	jsonDocument map[string]interface{} // Decoded JSONDocument.
	rawDocument  []byte                 // Raw document

	JSONSchema json.RawMessage    // Exact JSON schema to match. Exclusive with Document.
	jsonSchema *jsonschema.Schema // Compiled JSONSchema.

	JSONValues json.RawMessage        // Required JSON values. Optional, if JSONSchema is set.
	jsonValues map[string]interface{} // Decoded JSONValues.
}

type (
	headers          map[string]string
	multipartContent map[string]string

	// RunOption is a function that configures a Suite.
	RunOption func(*Suite)
)

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

// Init initializes an HTTP tests suite.
func (s *Suite) Init(name string, n notifier.Notifier, opts ...RunOption) error {
	s.name = name
	s.notifier = n
	s.retryCount = defaultRetryCount
	s.retryWait = defaultRetryWait
	s.client = http.DefaultClient

	for _, opt := range opts {
		opt(s)
	}

	if err := s.prepare(); err != nil {
		return err
	}
	return nil
}

// runTestWithRetry will try to run the test t up to n times, waiting for n * wait time
// in between each try. It returns the error of the last tentative if none is sucessful,
// nil otherwise.
func (s *Suite) runTestWithRetry(t *test, n int, wait time.Duration) error {
	var err error
	for i := 0; i < n; i++ {
		err = s.runTest(t)
		if err == nil {
			return nil
		}
		log.Error("run test error",
			zap.String("name", t.Name),
			zap.Error(err),
			zap.Int("attempt", i+1),
			zap.Int("max_attempt", n),
			zap.Duration("backoff", wait*time.Duration(i+1)))
		time.Sleep(wait * time.Duration(i+1))
	}
	return err
}

// Run runs all the tests in the suite.
func (s *Suite) Run() {
	log.Info("running tests suite", zap.String("name", s.name))
	start := time.Now()
	for _, t := range s.Tests {
		s.notifier.BeforeTest(t.Name)
		if err := s.runTestWithRetry(&t, s.retryCount, s.retryWait); err != nil {
			s.notifier.TestError(err)
		}
		s.notifier.AfterTest()
	}
	s.notifier.SuiteDone()
	log.Info("ran tests suite", zap.String("name", s.name), zap.Duration("took", time.Since(start)))
}
