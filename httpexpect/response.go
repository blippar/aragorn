package httpexpect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/jsonq"
	"github.com/santhosh-tekuri/jsonschema"

	"github.com/blippar/aragorn/reporter"
)

// Response wraps an http.Response and allows you to have expectations on it.
type Response struct {
	httpResp *http.Response
	reporter reporter.Reporter

	// HTTP response body, this is required because some
	// verifications need to read the body multiple times.
	body []byte
}

// NewResponse returns a new response on which you can have expectations.
// Any failed expectation will be reported to the given reporter.
// It reads and closes the response body.
func NewResponse(reporter reporter.Reporter, resp *http.Response) (*Response, error) {
	r := &Response{
		reporter: reporter,
		httpResp: resp,
	}

	if err := r.readAndCloseBody(); err != nil {
		return nil, err
	}
	return r, nil
}

// readAndCloseBody reads and closes the underlying http.Response's body and stores
// it in a temporary byte slice.
// This is required because we can only read this body once and some checks might
// need to read it multiple times.
func (r *Response) readAndCloseBody() error {
	var err error
	r.body, err = ioutil.ReadAll(r.httpResp.Body)
	if err != nil {
		return err
	}
	r.httpResp.Body.Close()
	return nil
}

// StatusCode checks whether the response has the given status code.
func (r *Response) StatusCode(code int) {
	if r.httpResp.StatusCode != code {
		r.reporter.Report(fmt.Errorf("wrong http status code (got %d; expected %d)", r.httpResp.StatusCode, code))
	}
}

// ContainsHeaders checks whether the response contains the given headers.
func (r *Response) ContainsHeaders(h headers) {
	for k, v := range h {
		if val := r.httpResp.Header.Get(k); val == "" {
			r.reporter.Report(fmt.Errorf("missing header %s", k))
		} else if val != v {
			r.reporter.Report(fmt.Errorf("wrong value for header %q (got %q; expected %q)", k, val, v))
		}
	}
}

// MatchJSONDocument checks whether the JSON response body matches the given document.
func (r *Response) MatchJSONDocument(doc map[string]interface{}) {
	resp := map[string]interface{}{}
	if err := json.Unmarshal(r.body, &resp); err != nil {
		r.reporter.Report(fmt.Errorf("could not decode JSON response body: %v", err))
		return
	}

	if !cmp.Equal(resp, doc) {
		r.reporter.Report(fmt.Errorf("request body does not match document"))
	}
}

// MatchJSONSchema checks whether the JSON formated response body matches the given JSON schema.
func (r *Response) MatchJSONSchema(schema *jsonschema.Schema) {
	if err := schema.Validate(bytes.NewReader(r.body)); err != nil {
		if e, ok := err.(*jsonschema.ValidationError); ok {
			r.reporter.Report(fmt.Errorf("wrong JSON schema: %v", e))
		} else {
			r.reporter.Report(fmt.Errorf("JSON schema validation failed: %v", err))
		}
	}
}

// ContainsJSONValues checks that the JSON formated response body contains
// specific values at given keys.
func (r *Response) ContainsJSONValues(values map[string]interface{}) {
	d := map[string]interface{}{}
	if err := json.NewDecoder(bytes.NewReader(r.body)).Decode(&d); err != nil {
		r.reporter.Report(fmt.Errorf("could not decode JSON body: %v", err))
		return
	}

	jq := jsonq.NewQuery(d)

	for key, expected := range values {
		val, err := jq.Interface(strings.Split(key, ".")...)
		if err != nil {
			r.reporter.Report(fmt.Errorf("missing JSON key: %q", key))
			continue
		}

		if !cmp.Equal(val, expected) {
			r.reporter.Report(fmt.Errorf("wrong value for key %q (got %v; want %v)", key, val, expected))
		}
	}
}
