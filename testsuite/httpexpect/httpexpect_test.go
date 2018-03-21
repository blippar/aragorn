package httpexpect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/blippar/aragorn/testsuite"
)

var cfgTest = &Config{
	Path: "./testdata/test.suite.json",
	Root: "./testdata/",
	Base: Base{
		URL: "http://localhost:3000",
		Header: testsuite.Header{
			"X-Custom-Test": "test",
		},
	},
}

var (
	userData = map[string]interface{}{"name": "John Doe"}
	userRaw  = []byte("{ \"name\": \"John Doe\" }\n")
	userRef  = map[string]interface{}{"$ref": "user.json"}
)

type mockLogger struct {
	errs []string
}

func (tr *mockLogger) Error(args ...interface{}) {
	tr.errs = append(tr.errs, fmt.Sprint(args...))
}

func (tr *mockLogger) Errorf(format string, args ...interface{}) {
	tr.errs = append(tr.errs, fmt.Sprintf(format, args...))
}

func TestRequestToHTTPRequest(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		req := &Request{}
		httpReq, err := req.toHTTPRequest(cfgTest)
		if err != nil {
			t.Fatalf("can't convert to http request: %v", err)
		}
		if got, want := httpReq.URL.String(), "http://localhost:3000/"; got != want {
			t.Fatalf("invalid request url (got %v; want %v)", got, want)
		}
		if got, want := httpReq.Method, "GET"; got != want {
			t.Fatalf("invalid request Method (got %v; want %v)", got, want)
		}
		if got, want := httpReq.Header.Get("X-Custom-Test"), "test"; got != want {
			t.Fatalf("invalid request Header (got %v; want %v)", got, want)
		}
		if httpReq.Body != nil {
			t.Fatalf("invalid request body not nil")
		}
	})
	t.Run("RAW Body", func(t *testing.T) {
		rawBody := []byte("RAWDATA")
		req := &Request{
			URL:    "https://google.com",
			Path:   "/search",
			Method: "POST",
			Body:   rawBody,
		}
		httpReq, err := req.toHTTPRequest(cfgTest)
		if err != nil {
			t.Fatalf("can't convert to http request: %v", err)
		}
		if got, want := httpReq.URL.String(), "https://google.com/search"; got != want {
			t.Fatalf("invalid request url (got %v; want %v)", got, want)
		}
		if got, want := httpReq.Method, "POST"; got != want {
			t.Fatalf("invalid request Method (got %v; want %v)", got, want)
		}
		body, _ := ioutil.ReadAll(httpReq.Body)
		if got, want := body, rawBody; !bytes.Equal(got, want) {
			t.Fatalf("invalid request body (got %v; want %v)", got, want)
		}
	})
	t.Run("JSON body", func(t *testing.T) {
		req := &Request{
			Body: map[string]string{
				"hello": "world",
			},
		}
		httpReq, err := req.toHTTPRequest(cfgTest)
		if err != nil {
			t.Fatalf("can't convert to http request: %v", err)
		}
		body, _ := ioutil.ReadAll(httpReq.Body)
		if got, want := body, []byte(`{"hello":"world"}`); !bytes.Equal(got, want) {
			t.Fatalf("invalid request body (got %v; want %v)", got, want)
		}
		if got, want := httpReq.Header.Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Fatalf("invalid request Content-Type (got %v; want %v)", got, want)
		}
	})
	t.Run("Multipart body", func(t *testing.T) {
		req := &Request{
			Multipart: map[string]string{
				"hello": "world",
				"file":  "@user.json",
			},
		}
		httpReq, err := req.toHTTPRequest(cfgTest)
		if err != nil {
			t.Fatalf("can't convert to http request: %v", err)
		}
		if httpReq.ContentLength == 0 {
			t.Fatalf("invalid request body length == 0")
		}
		if got, want := httpReq.Header.Get("Content-Type"), "multipart/form-data"; !strings.HasPrefix(got, want) {
			t.Fatalf("invalid request Content-Type (got %v; want prefix %v)", got, want)
		}
	})
	t.Run("FormData body", func(t *testing.T) {
		req := &Request{
			FormData: map[string]string{
				"hello": "world",
			},
		}
		httpReq, err := req.toHTTPRequest(cfgTest)
		if err != nil {
			t.Fatalf("can't convert to http request: %v", err)
		}
		body, _ := ioutil.ReadAll(httpReq.Body)
		if got, want := body, []byte(`hello=world`); !bytes.Equal(got, want) {
			t.Fatalf("invalid request body (got %v; want %v)", got, want)
		}
		if got, want := httpReq.Header.Get("Content-Type"), "application/x-www-form-urlencoded"; got != want {
			t.Fatalf("invalid request Content-Type (got %v; want %v)", got, want)
		}
	})
}

func TestGetDocumentField(t *testing.T) {
	tt := []struct {
		name string
		val  interface{}
		want interface{}
	}{
		{
			"object",
			map[string]interface{}{"id": "fake-id"},
			map[string]interface{}{"id": "fake-id"},
		},
		{
			"objects",
			[]map[string]interface{}{{"id": "fake-id"}},
			[]map[string]interface{}{{"id": "fake-id"}},
		},
		{
			"ref to json file",
			userRef,
			userData,
		},
		{
			"ref invalid",
			map[string]interface{}{"$ref": "invalid_file"},
			nil,
		},
		{
			"raw data",
			map[string]interface{}{"$raw": "raw data"},
			[]byte("raw data"),
		},
		{
			"ref to raw data",
			map[string]interface{}{"$ref": "user.json", "$raw": true},
			userRaw,
		},
		{
			"ref to raw data invalid",
			map[string]interface{}{"$ref": "invalid_file", "$raw": true},
			nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cfgTest.getDocumentField(tc.val)
			if tc.want == nil {
				if err == nil {
					t.Fatalf("no error returned")
				}
				return
			}
			if err != nil {
				t.Fatalf("get document field: %v", err)
			}
			if !cmp.Equal(got, tc.want) {
				t.Fatalf("invalid document (got %v; want %v)", got, tc.want)
			}
		})
	}
}

func TestGetObjectField(t *testing.T) {
	tt := []struct {
		name string
		val  map[string]interface{}
		want map[string]interface{}
	}{
		{
			"default",
			map[string]interface{}{"id": "fake-id"},
			map[string]interface{}{"id": "fake-id"},
		},
		{
			"ref",
			userRef,
			userData,
		},
		{
			"ref invalid",
			map[string]interface{}{"$ref": "invalid_file"},
			nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cfgTest.getObjectField(tc.val)
			if tc.want == nil {
				if err == nil {
					t.Fatalf("no error returned")
				}
				return
			}
			if err != nil {
				t.Fatalf("get object field: %v", err)
			}
			if !cmp.Equal(got, tc.want) {
				t.Fatalf("invalid object (got %v; want %v)", got, tc.want)
			}
		})
	}
}

func TestNewWithEmptyConfig(t *testing.T) {
	if _, err := New(&Config{}); err == nil {
		t.Error("new test suite with empty config should return an error")
	}
}

func TestNewWithInvalidRequestConfig(t *testing.T) {
	if _, err := New(&Config{
		Base: Base{
			URL: "invalid_url",
		},
		Tests: []*Test{
			{
				Name: "index",
				Request: Request{
					Body: map[string]interface{}{"$ref": "invalid_file"},
				},
			},
		},
	}); err == nil {
		t.Error("new test suite with invalid request should return an error")
	}
}

func TestSuiteRunTestSimple(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, client")
	}))
	defer ts.Close()
	cfg := &Config{
		Base: Base{
			URL: ts.URL,
		},
		Tests: []*Test{
			{
				Name: "index",
				Expect: Expect{
					StatusCode: http.StatusOK,
					Document: map[string]interface{}{
						"$raw": "Hello, client",
					},
				},
			},
		},
	}
	suite, err := New(cfg)
	if err != nil {
		t.Fatalf("can't create suite: %v", err)
	}
	ctx := context.Background()
	tr := &mockLogger{}
	suite.tests[0].Run(ctx, tr)
	if len(tr.errs) > 0 {
		t.Fatalf("unexpected test report errors: %v", tr.errs)
	}
}

func TestSuiteRunTestJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("invalid request Content-Type (got %v; want %v)", got, want)
		}
		if got, want := r.Header.Get("Accept"), "application/json"; got != want {
			t.Errorf("invalid request Accept (got %v; want %v)", got, want)
		}
		w.Header().Set("X-Custom-Header", "1")
		json.NewEncoder(w).Encode(userData)
	}))
	defer ts.Close()
	cfg := &Config{
		Path: "./testdata/test.suite.json",
		Root: "./testdata/",
		Base: Base{
			URL: ts.URL,
		},
		Tests: []*Test{
			{
				Name: "index",
				Request: Request{
					Body: userData,
				},
				Expect: Expect{
					StatusCode: http.StatusOK,
					Document:   userRef,
					Header: testsuite.Header{
						"X-Custom-Header": "1",
					},
				},
			},
		},
	}
	suite, err := New(cfg)
	if err != nil {
		t.Fatalf("can't create suite: %v", err)
	}
	ctx := context.Background()
	tr := &mockLogger{}
	suite.tests[0].Run(ctx, tr)
	if len(tr.errs) > 0 {
		t.Fatalf("unexpected test report errors: %v", tr.errs)
	}
}

func TestQueryJSONData(t *testing.T) {
	m := map[string]interface{}{
		"hello": "world",
		"arr":   []interface{}{42, 0, 1},
		"sub": map[string]interface{}{
			"a": "b",
			"c": "d",
		},
	}
	arr := []interface{}{1, 3, 6, "test", "123", map[string]interface{}{"abc": "def"}}
	tt := []struct {
		name   string
		val    interface{}
		query  string
		want   interface{}
		errStr string
	}{
		{"obj simple string", m, "hello", "world", ""},
		{"obj field not found", m, "invalid_key", nil, "invalid_key: object does not contain field"},
		{"obj sub obj field a", m, "sub.a", "b", ""},
		{"obj sub obj field not found", m, "sub.d", "", "sub.d: object does not contain field"},
		{"obj sub arr", m, "arr.0", 42, ""},
		{"obj sub arr length check", m, "arr.length", json.Number("3"), ""},
		{"obj sub arr out of bounds", m, "arr.125", 42, "arr.125: array index out of bounds"},
		{"arr simple int", arr, "0", 1, ""},
		{"arr simple string", arr, "3", "test", ""},
		{"arr simple length check", arr, "length", json.Number("6"), ""},
		{"arr invalid index", arr, "invalid_index", nil, "invalid_index: invalid array index"},
		{"arr out of bounds", arr, "1234", nil, "1234: array index out of bounds"},
		{"arr sub obj field a", arr, "5.abc", "def", ""},
		{"arr sub obj field not found", arr, "5.d", "", "5.d: object does not contain field"},
		{"invalid type", nil, "key", nil, "key: invalid type"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := queryJSONData(tc.query, tc.val)
			if err != nil {
				if errStr := err.Error(); errStr != tc.errStr {
					t.Fatalf("invalid error (got %v; want %v)", errStr, tc.errStr)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("invalid lookup value (got %v; want %v)", got, tc.want)
			}
		})
	}
}

func TestLookupJSONData(t *testing.T) {
	m := map[string]interface{}{"hello": "world"}
	arr := []interface{}{1, 3, 6, "test", "123"}
	tt := []struct {
		name string
		val  interface{}
		key  string
		want interface{}
	}{
		{"obj simple string", m, "hello", "world"},
		{"obj field not found", m, "invalid_key", errObjectFieldNotFound},
		{"arr simple int", arr, "0", 1},
		{"arr simple string", arr, "3", "test"},
		{"arr simple length check", arr, "length", json.Number("5")},
		{"arr invalid index", arr, "invalid_index", errInvalidArrayIndex},
		{"arr out of bounds", arr, "1234", errIndexOutOfBounds},
		{"invalid type", nil, "key", errInvalidType},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := lookupJSONData(tc.key, tc.val)
			if err != nil {
				e, _ := tc.want.(error)
				if err != e {
					t.Fatalf("invalid error (got %v; want %v)", err, e)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("invalid lookup value (got %v; want %v)", got, tc.want)
			}
		})
	}
}
