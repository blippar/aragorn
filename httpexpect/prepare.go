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

// prepare verifies that an HTTP tests suite is valid. It also create the HTTP requests,
// compiles JSON schemas and unmarshal JSON documents.
func (s *Suite) prepare() error {
	var errs []string
	if s.Base == nil {
		errs = append(errs, "base is required")
	} else {
		if err := prepareBase(s.Base); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(s.Tests) < 1 {
		errs = append(errs, "a tests suite must contain at least one test")
	} else {
		if err := prepareTests(s.Base, s.Tests); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func prepareBase(b *base) error {
	var errs []string
	if b.URL == "" {
		errs = append(errs, "base: URL is required")
	} else if _, err := url.Parse(b.URL); err != nil {
		errs = append(errs, "base: URL is invalid: "+err.Error())
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func prepareTests(b *base, ts []test) error {
	var errs []string
	for _, t := range ts {
		if err := prepareTest(b, &t); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func prepareTest(b *base, t *test) error {
	var errs []string
	if t.Request == nil {
		errs = append(errs, fmt.Sprintf("test %q: request: field required", t.Name))
	} else {
		if t.Request.Path == "" {
			t.Request.Path = "/"
		}
		if t.Request.Method == "" {
			t.Request.Method = http.MethodGet
		}
		var err error
		t.Request.httpReq, err = newHTTPRequest(b, t.Request)
		if err != nil {
			errs = append(errs, fmt.Sprintf("test %q: request: could not create HTTP request: %v", t.Name, err))
		}
	}

	if t.Expect == nil {
		errs = append(errs, fmt.Sprintf("test %q: request: field is required", t.Name))
	} else {
		if t.Expect.JSONDocument != nil && t.Expect.JSONSchema != nil {
			errs = append(errs, fmt.Sprintf("test %q: expect: only one of document or schema can be set at once", t.Name))
		}
		if t.Expect.JSONDocument != nil && t.Expect.JSONValues != nil {
			errs = append(errs, fmt.Sprintf("test %q: expect: jsonValues can't be set with document", t.Name))
		}

		if t.Expect.JSONDocument != nil {
			r, err := getReadCloser(t.Expect.JSONDocument)
			if err != nil {
				errs = append(errs, fmt.Sprintf("test %q: expect: could get reader for JSON document: %v", t.Name, err))
			}
			defer r.Close()
			if err = json.NewDecoder(r).Decode(&t.Expect.jsonDocument); err != nil {
				errs = append(errs, fmt.Sprintf("test %q: expect: could decode expected JSON document: %v", t.Name, err))
			}
		}

		if t.Expect.JSONSchema != nil {
			r, err := getReadCloser(t.Expect.JSONSchema)
			if err != nil {
				errs = append(errs, fmt.Sprintf("test %q: expect: could get reader for JSON schema: %v", t.Name, err))
			}
			defer r.Close()
			cc := jsonschema.NewCompiler()
			// NOTE: the parameter "schema.json" is not relevent.
			cc.AddResource("schema.json", r)
			t.Expect.jsonSchema, err = cc.Compile("schema.json")
			if err != nil {
				errs = append(errs, fmt.Sprintf("test %q: expect: could not compile JSON schema: %v", t.Name, err))
			}
		}

		if t.Expect.JSONValues != nil {
			r, err := getReadCloser(t.Expect.JSONValues)
			if err != nil {
				errs = append(errs, fmt.Sprintf("test %q: expect: could get reader for JSON values: %v", t.Name, err))
			}
			defer r.Close()
			if err = json.NewDecoder(r).Decode(&t.Expect.jsonValues); err != nil {
				errs = append(errs, fmt.Sprintf("test %q: expect: could decode expected JSON values: %v", t.Name, err))
			}
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

// getReadCloser returns an io.ReadCloser from a byte slice, or from a file if
// the byte slice starts with the characters '"@'.
func getReadCloser(s []byte) (io.ReadCloser, error) {
	var r io.ReadCloser
	if bytes.HasPrefix(s, []byte(`"@`)) { // From file.
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
