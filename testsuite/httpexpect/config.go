package httpexpect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/oauth2/clientcredentials"
)

type Config struct {
	Path  string
	Base  Base
	Tests []*Test
}

type Base struct {
	URL        string // Base URL prepended to all requests' path.
	Header     Header // Base set of headers added to all requests.
	OAUTH2     clientcredentials.Config
	RetryCount int
	RetryWait  int
}

type Test struct {
	Name    string  // Name used to identify this test.
	Request Request // Request describes the HTTP request.
	Expect  Expect  // Expect describes the expected result of the HTTP request.
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

	Document   interface{}            // Exact document to match. Exclusive with JSONSchema.
	JSONSchema map[string]interface{} // Exact JSON schema to match. Exclusive with Document.
	JSONValues map[string]interface{} // Required JSON values. Optional, if JSONSchema is set.
}

// A Header represents the key-value pairs in an HTTP header.
type Header map[string]string

func (h Header) addToRequest(req *http.Request) {
	for k, v := range h {
		req.Header.Set(k, v)
	}
}

// genTest verifies that an HTTP test suite is valid. It also create the HTTP requests,
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
		t, err := test.prepare(cfg)
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

func (t *Test) prepare(cfg *Config) (*test, error) {
	test := &test{
		name:       t.Name,
		statusCode: t.Expect.StatusCode,
		header:     make(Header),
	}
	var errs []string

	copyHeader(test.header, t.Expect.Header)

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
		errs = append(errs, "- request: at most one of body, multipart or formData can be set at once")
	}

	if httpReq, err := t.Request.toHTTPRequest(cfg); err != nil {
		errs = append(errs, fmt.Sprintf("- request: could not create HTTP request: %v", err))
	} else {
		test.req = httpReq
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
		m, err := cfg.getObjectField(t.Expect.JSONSchema)
		if err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get JSON schema: %v", err))
		} else {
			schemaLoader := gojsonschema.NewGoLoader(m)
			test.jsonSchema, err = gojsonschema.NewSchema(schemaLoader)
			if err != nil {
				errs = append(errs, fmt.Sprintf("- expect: could not load JSON schema: %v", err))
			}
		}
	}

	if t.Expect.JSONValues != nil {
		m, err := cfg.getObjectField(t.Expect.JSONValues)
		if err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get JSON values: %v", err))
		} else {
			test.jsonValues = m
		}
	}

	if test.req != nil {
		_, isRawDoc := test.document.([]byte)
		if test.req.Header.Get("Accept") == "" && (test.jsonSchema != nil || test.jsonValues != nil || (test.document != nil && !isRawDoc)) {
			test.req.Header.Set("Accept", "application/json")
		}
	}

	if err := concatErrors(errs); err != nil {
		return nil, err
	}
	return test, nil
}

func (cfg *Config) getDocumentField(v interface{}) (interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return v, nil
	}
	rawI, _ := m["$raw"]
	if rawStr, ok := rawI.(string); ok {
		return []byte(rawStr), nil
	}
	ref, ok := m["$ref"].(string)
	if !ok {
		return v, nil
	}
	path := cfg.getFilePath(ref)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if raw, _ := m["$raw"].(bool); raw {
		return ioutil.ReadAll(f)
	}
	var newVal interface{}
	err = json.NewDecoder(f).Decode(&newVal)
	return newVal, err
}

func (cfg *Config) getObjectField(m map[string]interface{}) (map[string]interface{}, error) {
	ref, ok := m["$ref"].(string)
	if !ok {
		return m, nil
	}
	path := cfg.getFilePath(ref)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var newVal map[string]interface{}
	err = json.NewDecoder(f).Decode(&newVal)
	return newVal, err
}

func (cfg *Config) getFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.Path, path)
}

func copyHeader(dest, src Header) {
	for k, v := range src {
		dest[k] = v
	}
}

func concatErrors(errs []string) error {
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
