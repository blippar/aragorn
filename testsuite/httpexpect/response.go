package httpexpect

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/jsonq"
	"github.com/xeipuuv/gojsonschema"
)

// Logger logs error.
type Logger interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// Response wraps an http.Response and allows you to have expectations on it.
type Response struct {
	resp   *http.Response
	logger Logger

	// HTTP response body, this is required because some
	// verifications need to read the body multiple times.
	body []byte

	dataJSON      interface{}
	dataJSONError bool
}

// NewResponse returns a new response on which you can have expectations.
// Any failed expectation will be reported to the given reporter.
// It reads and closes the response body.
func NewResponse(logger Logger, resp *http.Response, body []byte) *Response {
	return &Response{
		logger: logger,
		resp:   resp,
		body:   body,
	}
}

// StatusCode checks whether the response has the given status code.
func (r *Response) StatusCode(code int) {
	if r.resp.StatusCode != code {
		r.logger.Errorf("wrong http status code (got %d; expected %d)", r.resp.StatusCode, code)
	}
}

// ContainsHeader checks whether the response contains the given headers.
func (r *Response) ContainsHeader(h Header) {
	for k, v := range h {
		if val := r.resp.Header.Get(k); val == "" {
			r.logger.Errorf("missing header %s", k)
		} else if val != v {
			r.logger.Errorf("wrong value for header %q (got %q; expected %q)", k, val, v)
		}
	}
}

// MatchRawDocument checks whether the raw response body matches the given document.
func (r *Response) MatchRawDocument(doc []byte) {
	if !bytes.Equal(r.body, doc) {
		r.logger.Error("request body does not match document")
	}
}

// MatchJSONDocument checks whether the JSON response body matches the given document.
func (r *Response) MatchJSONDocument(doc interface{}) {
	if !r.unmarshalJSONBody() {
		return
	}
	if !cmp.Equal(doc, r.dataJSON) {
		r.logger.Error("request body does not match document")
	}
}

// MatchJSONSchema checks whether the JSON formated response body matches the given JSON schema.
func (r *Response) MatchJSONSchema(schema *gojsonschema.Schema) {
	if !r.unmarshalJSONBody() {
		return
	}
	data := gojsonschema.NewGoLoader(r.dataJSON)
	result, err := schema.Validate(data)
	if err != nil {
		r.logger.Errorf("JSON schema validation failed: %v", err)
		return
	}
	if !result.Valid() {
		for _, err := range result.Errors() {
			r.logger.Errorf("JSON schema validation failed: %v", err)
		}
	}
}

// ContainsJSONValues checks that the JSON formated response body contains
// specific values at given keys.
func (r *Response) ContainsJSONValues(values map[string]interface{}) {
	if !r.unmarshalJSONBody() {
		return
	}
	jq := jsonq.NewQuery(r.dataJSON)
	for key, expected := range values {
		val, err := jq.Interface(strings.Split(key, ".")...)
		if err != nil {
			r.logger.Errorf("missing JSON key: %q", key)
			continue
		}
		if !cmp.Equal(val, expected) {
			r.logger.Errorf("wrong value for key %q (got %v; want %v)", key, val, expected)
		}
	}
}

func (r *Response) unmarshalJSONBody() bool {
	if r.dataJSON != nil {
		return true
	}
	if r.dataJSONError {
		return false
	}
	if err := json.Unmarshal(r.body, &r.dataJSON); err != nil {
		r.logger.Errorf("could not decode JSON response body: %v", err)
		r.dataJSONError = true
		return false
	}
	return true
}
