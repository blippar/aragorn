package httpexpect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema"
)

type Config struct {
	Path  string
	Base  Base
	Tests []*Test
}

type Base struct {
	URL    string // Base URL prepended to all requests' path.
	Header Header // Base set of headers added to all requests.
}

type Test struct {
	Name    string  // Name used to identify this test.
	Request Request // Request describes the HTTP request.
	Expect  Expect  // Expect describes the result of the HTTP request.
}

type Request struct {
	URL    string // If set, will overwrite the base URL.
	Path   string
	Method string
	Header Header

	// Only one of the three following must be set.
	Body      interface{}
	Multipart map[string]string
	FormData  map[string]string
}

type Expect struct {
	StatusCode int
	Header     Header

	Document   interface{}     // Exact document to match. Exclusive with JSONSchema.
	JSONSchema json.RawMessage // Exact JSON schema to match. Exclusive with Document.
	JSONValues interface{}     // Required JSON values. Optional, if JSONSchema is set.
}

// A Header represents the key-value pairs in an HTTP header.
type Header map[string]string

// prepare verifies that an HTTP test suite is valid. It also create the HTTP requests,
// compiles JSON schemas and unmarshal JSON documents.
func (cfg *Config) genTests() ([]*test, error) {
	if cfg.Base.URL == "" {
		return nil, errors.New("base: URL is required")
	}
	if _, err := url.Parse(cfg.Base.URL); err != nil {
		return nil, fmt.Errorf("base: URL is invalid: %v", err)
	}
	if len(cfg.Tests) == 0 {
		return nil, errors.New("a test suite must contain at least one test")
	}
	ts := make([]*test, len(cfg.Tests))
	var errs []string
	for i, test := range cfg.Tests {
		t, err := cfg.prepareTest(test)
		if err != nil {
			errs = append(errs, fmt.Sprintf("test %q:\n%v", test.Name, err))
		}
		ts[i] = t
	}
	if err := concatErrors(errs); err != nil {
		return nil, err
	}
	return ts, nil
}

func (cfg *Config) prepareTest(t *Test) (*test, error) {
	test := &test{
		name:       t.Name,
		statusCode: t.Expect.StatusCode,
		header:     t.Expect.Header,
	}
	var errs []string

	set := 0
	if t.Request.Body != nil {
		set++
	}
	if t.Request.Multipart != nil {
		set++
	}
	if t.Request.FormData != nil {
		set++
	}
	if set > 1 {
		errs = append(errs, "- request: at most one of body, multipart or formURLEncoded can be set at once")
	}

	if err := cfg.setHTTPRequest(test, &t.Request); err != nil {
		errs = append(errs, fmt.Sprintf("- request: could not create HTTP request: %v", err))
	}

	if test.statusCode == 0 {
		test.statusCode = http.StatusOK
	}

	if t.Expect.Document != nil && t.Expect.JSONSchema != nil {
		errs = append(errs, "- expect: only one of document or schema can be set at once")
	}
	if t.Expect.Document != nil && t.Expect.JSONValues != nil {
		errs = append(errs, "- expect: jsonValues can't be set with document")
	}

	if t.Expect.Document != nil {
		doc, err := cfg.getDocumentField(t.Expect.Document)
		if err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get Document: %v", err))
		}
		test.document = doc
	}

	if t.Expect.JSONSchema != nil {
		r, err := cfg.getReaderFromJSONRawMessage(t.Expect.JSONSchema)
		if err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get reader for JSON schema: %v", err))
		} else {
			cc := jsonschema.NewCompiler()
			// NOTE: the parameter "schema.json" is not relevent.
			cc.AddResource("schema.json", r)
			test.jsonSchema, err = cc.Compile("schema.json")
			if err != nil {
				errs = append(errs, fmt.Sprintf("- expect: could not compile JSON schema: %v", err))
			}
		}
	}

	if t.Expect.JSONValues != nil {
		v, err := cfg.getDocumentField(t.Expect.JSONValues)
		if err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get reader for JSON values: %v", err))
		} else if m, ok := v.(map[string]interface{}); !ok {
			errs = append(errs, fmt.Sprintf("- expect: invalid JSON Values type: must be an object"))
		} else {
			test.jsonValues = m
		}
	}

	if err := concatErrors(errs); err != nil {
		return nil, err
	}
	return test, nil
}

func (cfg *Config) getDocumentField(v interface{}) (interface{}, error) {
	s, ok := v.(string)
	if !ok {
		return v, nil
	}
	if !strings.HasPrefix(s, "@") {
		return []byte(s), nil
	}
	path := cfg.getFilePath(s[1:])
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var newVal interface{}
	if err := json.Unmarshal(data, &newVal); err != nil {
		return nil, err
	}
	return newVal, nil
}

// getReaderFromJSONRawMessage returns an io.Reader from a byte slice, or from a file if
// the byte slice starts with the characters '"@'.
func (cfg *Config) getReaderFromJSONRawMessage(val []byte) (io.Reader, error) {
	s := string(val)
	if strings.HasPrefix(s, `"@`) && strings.HasSuffix(s, `"`) {
		path := cfg.getFilePath(s[2 : len(s)-1])
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		val = data
	}
	return bytes.NewReader(val), nil
}

func (cfg *Config) getFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.Path, path)
}

func concatErrors(errs []string) error {
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
