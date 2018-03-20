package httpexpect

import (
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

	"github.com/blippar/aragorn/pkg/util/json"
	"github.com/blippar/aragorn/testsuite"
)

type Config struct {
	Path  string  `json:"path,omitempty"`
	Root  string  `json:"root,omitempty"`
	Base  Base    `json:"base,omitempty"`
	Tests []*Test `json:"tests,omitempty"`
}

type Base struct {
	URL      string                    `json:"url,omitempty"`    // Base URL prepended to all requests' path.
	Header   testsuite.Header          `json:"header,omitempty"` // Base set of headers added to all requests.
	OAUTH2   *clientcredentials.Config `json:"oauth2,omitempty"`
	Insecure bool                      `json:"insecure,omitempty"`
}

type Test struct {
	Name    string  `json:"name,omitempty"`    // Name used to identify this test.
	Request Request `json:"request,omitempty"` // Request describes the HTTP request.
	Expect  Expect  `json:"expect,omitempty"`  // Expect describes the expected result of the HTTP request.
}

type Request struct {
	URL    string           `json:"url,omitempty"` // If set, will overwrite the base URL.
	Path   string           `json:"path,omitempty"`
	Method string           `json:"method,omitempty"`
	Header testsuite.Header `json:"header,omitempty"`

	// Only one of the three following must be set.
	Body      interface{}       `json:"body,omitempty"`
	Multipart map[string]string `json:"multipart,omitempty"`
	FormData  map[string]string `json:"formData,omitempty"`
}

type Expect struct {
	StatusCode int              `json:"statusCode,omitempty"`
	Header     testsuite.Header `json:"header,omitempty"`

	Document   interface{}            `json:"document,omitempty"`   // Exact document to match. Exclusive with JSONSchema.
	JSONSchema map[string]interface{} `json:"jsonSchema,omitempty"` // Exact JSON schema to match. Exclusive with Document.
	JSONValues map[string]interface{} `json:"jsonValues,omitempty"` // Required JSON values. Optional, if JSONSchema is set.
}

func (*Config) Example() interface{} {
	return &Config{
		Base: Base{
			URL: "localhost:8000",
		},
		Tests: []*Test{
			{
				Name: "Index",
				Request: Request{
					Path: "/",
					FormData: map[string]string{
						"name": "John Doe",
					},
				},
				Expect: Expect{
					StatusCode: http.StatusNotFound,
					Header: testsuite.Header{
						"X-Custom-Header": "1",
					},
					Document: map[string]interface{}{
						"message": "User not found!",
					},
				},
			},
		},
	}
}

// genTest verifies that an HTTP test suite is valid. It also create the HTTP requests,
// compiles JSON schemas and unmarshal JSON documents.
func (cfg *Config) genTests(client *http.Client) ([]testsuite.Test, error) {
	if cfg.Base.URL == "" {
		return nil, errors.New("base: URL is required")
	}
	if _, err := url.Parse(cfg.Base.URL); err != nil {
		return nil, fmt.Errorf("base: URL is invalid: %v", err)
	}
	if len(cfg.Tests) == 0 {
		return nil, errors.New("a test suite must contain at least one test")
	}
	ts := make([]testsuite.Test, len(cfg.Tests))
	var errs []string
	for i, testcfg := range cfg.Tests {
		t, err := testcfg.prepare(cfg, client)
		if err != nil {
			errs = append(errs, fmt.Sprintf("test %q:\n%v", testcfg.Name, err))
		}
		ts[i] = t
	}
	if err := concatErrors(errs); err != nil {
		return nil, err
	}
	return ts, nil
}

func (t *Test) prepare(cfg *Config, client *http.Client) (*test, error) {
	test := &test{
		name:       t.Name,
		client:     client,
		statusCode: t.Expect.StatusCode,
		header:     testsuite.MergeHeaders(t.Expect.Header),
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
		errs = append(errs, "- request: at most one of body, multipart or formData can be set at once")
	}

	if httpReq, err := t.Request.toHTTPRequest(cfg); err != nil {
		errs = append(errs, fmt.Sprintf("- request: could not create HTTP request: %v", err))
	} else {
		test.req = httpReq
		test.description = httpReq.Method + " " + httpReq.URL.String()
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

	expectJSONBody := false

	if t.Expect.Document != nil {
		if doc, err := cfg.getDocumentField(t.Expect.Document); err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get Document: %v", err))
		} else {
			test.document = doc
			if _, ok := doc.([]byte); !ok {
				expectJSONBody = true
			}
		}
	}

	if t.Expect.JSONSchema != nil {
		if ref, ok := t.Expect.JSONSchema["$ref"].(string); ok {
			if !strings.Contains(ref, "://") {
				relRef := cfg.getFilePath(ref)
				if absRef, err := filepath.Abs(relRef); err == nil {
					t.Expect.JSONSchema["$ref"] = "file://" + absRef
				}
			}
		}
		schemaLoader := newJSONGoLoader(t.Expect.JSONSchema)
		if jsonSchema, err := gojsonschema.NewSchema(schemaLoader); err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could not load JSON schema: %v", err))
		} else {
			test.jsonSchema = jsonSchema
		}
		expectJSONBody = true
	}

	if t.Expect.JSONValues != nil {
		if m, err := cfg.getObjectField(t.Expect.JSONValues); err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get JSON values: %v", err))
		} else {
			test.jsonValues = m
		}
		expectJSONBody = true
	}

	if err := concatErrors(errs); err != nil {
		return nil, err
	}

	if expectJSONBody && test.req.Header.Get("Accept") == "" {
		test.req.Header.Set("Accept", "application/json")
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
	err = json.Decode(f, &newVal)
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
	err = json.Decode(f, &newVal)
	return newVal, err
}

func (cfg *Config) getFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.Root, path)
}

func concatErrors(errs []string) error {
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
