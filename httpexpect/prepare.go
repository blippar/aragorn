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
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema"
)

const (
	defaultRequestPath        = "/"
	defaultRequestMethod      = http.MethodGet
	defaultExpectedStatusCode = http.StatusOK
)

// prepare verifies that an HTTP tests suite is valid. It also create the HTTP requests,
// compiles JSON schemas and unmarshal JSON documents.
func (s *Suite) prepare() error {
	var errs []string
	if s.Base == nil {
		return errors.New("base is required")
	}
	if err := prepareBase(s.Base); err != nil {
		errs = append(errs, err.Error())
	}

	if len(s.Tests) == 0 {
		errs = append(errs, "a tests suite must contain at least one test")
		return concatErrors(errs)
	}
	if err := prepareTests(s.Base, s.Tests); err != nil {
		errs = append(errs, err.Error())
	}

	return concatErrors(errs)
}

func prepareBase(b *base) error {
	if b.URL == "" {
		return errors.New("base: URL is required")
	} else if _, err := url.Parse(b.URL); err != nil {
		return fmt.Errorf("base: URL is invalid: %v", err)
	}

	return nil
}

func prepareTests(b *base, ts []test) error {
	var errs []string
	for _, t := range ts {
		if err := prepareTest(b, &t); err != nil {
			errs = append(errs, fmt.Sprintf("test %q:\n%v", t.Name, err))
		}
	}

	return concatErrors(errs)
}

func prepareTest(b *base, t *test) error {
	var errs []string

	// Request
	if t.Request == nil {
		return errors.New("- request: field required")
	}

	if t.Request.URL != "" {
		if _, err := url.Parse(t.Request.URL); err != nil {
			errs = append(errs, fmt.Sprintf("- request: URL is invalid: %v", err))
		}
	}

	if t.Request.Path == "" {
		t.Request.Path = defaultRequestPath
	}
	if t.Request.Method == "" {
		t.Request.Method = defaultRequestMethod
	}

	set := 0
	if t.Request.Body != nil {
		set++
	}
	if t.Request.Multipart != nil {
		set++
	}
	if t.Request.FormURLEncoded != nil {
		set++
	}
	if set > 1 {
		errs = append(errs, "- request: at most one of body, multipart or formURLEncoded can be set at once")
	}

	var err error
	t.Request.httpReq, err = newHTTPRequest(b, t.Request)
	if err != nil {
		errs = append(errs, fmt.Sprintf("- request: could not create HTTP request: %v", err))
	}

	// Expect
	if t.Expect == nil {
		errs = append(errs, "- request: field is required")
		return concatErrors(errs)
	}

	if t.Expect.StatusCode == 0 {
		t.Expect.StatusCode = defaultExpectedStatusCode
	}

	if t.Expect.Document != nil && t.Expect.JSONSchema != nil {
		errs = append(errs, "- expect: only one of document or schema can be set at once")
	}
	if t.Expect.Document != nil && t.Expect.JSONValues != nil {
		errs = append(errs, "- expect: jsonValues can't be set with document")
	}

	if t.Expect.Document != nil {
		var err error
		d := t.Expect.Document
		if bytes.HasPrefix(d, []byte(`"`)) && bytes.HasSuffix(d, []byte(`"`)) {
			c := d[1 : len(d)-1]
			if bytes.HasPrefix(c, []byte("@")) {
				t.Expect.rawDocument, err = ioutil.ReadFile(string(c[1:]))
				if err != nil {
					errs = append(errs, fmt.Sprintf("- expect: could not read document: %v", err))
				}
			} else {
				t.Expect.rawDocument = c
			}
		} else {
			if err = json.Unmarshal(d, &t.Expect.jsonDocument); err != nil {
				errs = append(errs, fmt.Sprintf("- expect: could not decode expected document: %v", err))
			}
		}
	}

	if t.Expect.JSONSchema != nil {
		r, err := getReadCloser(t.Expect.JSONSchema)
		if err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get reader for JSON schema: %v", err))
		} else {
			defer r.Close()
			cc := jsonschema.NewCompiler()
			// NOTE: the parameter "schema.json" is not relevent.
			cc.AddResource("schema.json", r)
			t.Expect.jsonSchema, err = cc.Compile("schema.json")
			if err != nil {
				errs = append(errs, fmt.Sprintf("- expect: could not compile JSON schema: %v", err))
			}
		}
	}

	if t.Expect.JSONValues != nil {
		r, err := getReadCloser(t.Expect.JSONValues)
		if err != nil {
			errs = append(errs, fmt.Sprintf("- expect: could get reader for JSON values: %v", err))
		} else {
			defer r.Close()
			if err = json.NewDecoder(r).Decode(&t.Expect.jsonValues); err != nil {
				errs = append(errs, fmt.Sprintf("- expect: could decode expected JSON values: %v", err))
			}
		}
	}

	return concatErrors(errs)
}

// getReadCloser returns an io.ReadCloser from a byte slice, or from a file if
// the byte slice starts with the characters '"@'.
func getReadCloser(s []byte) (io.ReadCloser, error) {
	var r io.ReadCloser
	if bytes.HasPrefix(s, []byte(`"@`)) && bytes.HasSuffix(s, []byte(`"`)) { // From file.
		f, err := os.Open(string(s[2 : len(s)-1]))
		if err != nil {
			return nil, err
		}
		r = f
	} else { // Inline.
		r = ioutil.NopCloser(bytes.NewReader(s))
	}
	return r, nil
}

func concatErrors(errs []string) error {
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
