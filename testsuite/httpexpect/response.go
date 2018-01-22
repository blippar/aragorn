package httpexpect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/xeipuuv/gojsonschema"
)

var (
	errInvalidArrayIndex   = errors.New("invalid array index")
	errIndexOutOfBounds    = errors.New("array index out of bounds")
	errObjectFieldNotFound = errors.New("object does not contain field")
	errInvalidType         = errors.New("invalid type")
)

// Logger logs error.
type Logger interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// Response wraps an http.Response and allows you to have expectations on it.
type Response struct {
	test   *test
	resp   *http.Response
	logger Logger

	// HTTP response body, this is required because some
	// verifications need to read the body multiple times.
	body []byte

	dataJSON      interface{}
	dataJSONError bool
	errs          []string
}

// NewResponse returns a new response on which you can have expectations.
// Any failed expectation will be reported to the given reporter.
// It reads and closes the response body.
func NewResponse(test *test, resp *http.Response, body []byte) *Response {
	return &Response{
		test: test,
		resp: resp,
		body: body,
	}
}

func (r *Response) Check() error {
	r.StatusCode()
	r.ContainsHeader()
	if doc := r.test.document; doc != nil {
		raw, ok := doc.([]byte)
		if ok {
			r.MatchRawDocument(raw)
		} else {
			r.MatchJSONDocument(doc)
		}
	}
	if r.test.jsonSchema != nil {
		r.MatchJSONSchema()
	}
	if r.test.jsonValues != nil {
		r.ContainsJSONValues()
	}
	if len(r.errs) > 0 {
		return errors.New(strings.Join(r.errs, "\n"))
	}
	return nil
}

// StatusCode checks whether the response has the given status code.
func (r *Response) StatusCode() {
	if r.resp.StatusCode != r.test.statusCode {
		r.logger.Errorf("wrong http status code (got %d; expected %d)", r.resp.StatusCode, r.test.statusCode)
	}
}

// ContainsHeader checks whether the response contains the given headers.
func (r *Response) ContainsHeader() {
	for k, v := range r.test.header {
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
func (r *Response) MatchJSONSchema() {
	if !r.unmarshalJSONBody() {
		return
	}
	data := gojsonschema.NewGoLoader(r.dataJSON)
	result, err := r.test.jsonSchema.Validate(data)
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
func (r *Response) ContainsJSONValues() {
	if !r.unmarshalJSONBody() {
		return
	}
	for query, expected := range r.test.jsonValues {
		val, err := queryJSONData(query, r.dataJSON)
		if err != nil {
			r.logger.Errorf("could not get value for query %q: %v", query, err)
			continue
		}
		if !cmp.Equal(val, expected) {
			r.logger.Errorf("wrong value for query %q (got %v; want %v)", query, val, expected)
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
		r.logger.Errorf("could not decode json response body: %v", err)
		r.dataJSONError = true
		return false
	}
	return true
}

// queryJSONData returns the value for the given query in the decoded json data.
func queryJSONData(q string, v interface{}) (interface{}, error) {
	var err error
	ks := strings.Split(q, ".")
	for i, k := range ks {
		v, err = lookupJSONData(k, v)
		if err != nil {
			pq := strings.Join(ks[:i+1], ".")
			return nil, fmt.Errorf("%s: %v", pq, err)
		}
	}
	return v, nil
}

// lookupJSONData returns the value for the key in the decoded json data.
func lookupJSONData(k string, i interface{}) (interface{}, error) {
	switch v := i.(type) {
	case []interface{}:
		i, err := strconv.Atoi(k)
		if err != nil {
			return nil, errInvalidArrayIndex
		}
		if i >= len(v) {
			return nil, errIndexOutOfBounds
		}
		return v[i], nil
	case map[string]interface{}:
		val, ok := v[k]
		if !ok {
			return nil, errObjectFieldNotFound
		}
		return val, nil
	}
	return nil, errInvalidType
}
