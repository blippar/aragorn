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

	"github.com/blippar/aragorn/testsuite"
)

const maxErrorBodySize = 512

var (
	errInvalidArrayIndex   = errors.New("invalid array index")
	errIndexOutOfBounds    = errors.New("array index out of bounds")
	errObjectFieldNotFound = errors.New("object does not contain field")
	errInvalidType         = errors.New("invalid type")
)

// response wraps an http.Response and allows you to have expectations on it.
type response struct {
	test          *test
	logger        testsuite.Logger
	resp          *http.Response
	body          []byte
	dataJSON      interface{}
	dataJSONError bool
}

// checkResponse checks a response on which you can have expectations.
// Any failed expectation will be logged on the logger.
func checkResponse(test *test, logger testsuite.Logger, resp *http.Response, body []byte) {
	if resp.StatusCode != test.statusCode {
		str := string(body)
		if len(str) > maxErrorBodySize {
			str = fmt.Sprintf("%s... (%d)", str[:maxErrorBodySize], len(str))
		}
		logger.Errorf("wrong http status code (got %d; expected %d)\n%s", resp.StatusCode, test.statusCode, str)
		return
	}
	r := response{
		test:   test,
		logger: logger,
		resp:   resp,
		body:   body,
	}
	r.checkHeader()
	if test.document != nil {
		r.matchDocument()
	}
	if test.jsonSchema != nil {
		r.matchJSONSchema()
	}
	if test.jsonValues != nil {
		r.containsJSONValues()
	}
}

// checkHeader checks whether the response contains the given headers.
func (r *response) checkHeader() {
	for k, v := range r.test.header {
		if val := r.resp.Header.Get(k); val == "" {
			r.logger.Errorf("missing header %s", k)
		} else if val != v {
			r.logger.Errorf("wrong value for header %q (got %q; expected %q)", k, val, v)
		}
	}
}

// matchDocument checks whether the test document against the body.
func (r *response) matchDocument() {
	if raw, ok := r.test.document.([]byte); ok {
		if !bytes.Equal(r.body, raw) {
			r.logger.Error("request body does not match document")
		}
		return
	}
	if !r.unmarshalJSONBody() {
		return
	}
	if !cmp.Equal(r.test.document, r.dataJSON) {
		r.logger.Error("request body does not match document")
	}
}

// matchJSONSchema checks whether the JSON formated response body matches the given JSON schema.
func (r *response) matchJSONSchema() {
	if !r.unmarshalJSONBody() {
		return
	}
	data := newJSONGoLoader(r.dataJSON)
	result, err := r.test.jsonSchema.Validate(data)
	if err != nil {
		r.logger.Errorf("JSON schema validation failed: %v", err)
		return
	}
	if !result.Valid() {
		errs := result.Errors()
		b := &bytes.Buffer{}
		for _, err := range errs {
			b.WriteString("\n\t- ")
			b.WriteString(err.String())
		}
		r.logger.Errorf("JSON schema validation failed:%s", b.String())
	}
}

// containsJSONValues checks that the JSON formated response body contains
// specific values at given keys.
func (r *response) containsJSONValues() {
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

func (r *response) unmarshalJSONBody() bool {
	if r.dataJSON != nil {
		return true
	}
	if r.dataJSONError {
		return false
	}
	if err := decodeJSON(r.body, &r.dataJSON); err != nil {
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
		if k == "length" {
			return json.Number(strconv.Itoa(len(v))), nil
		}
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
