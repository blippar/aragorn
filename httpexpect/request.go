package httpexpect

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

// newHTTPRequest returns an htpp.Request from a base and a request.
func newHTTPRequest(base *base, req *request) (*http.Request, error) {
	// Generate request body.
	var err error
	switch {
	case req.Body != nil:
		req.httpBody, err = fromBody(req.Body)
		if err != nil {
			return nil, err
		}
	case req.Multipart != nil:
		req.httpBody, err = fromMultipart(req.Multipart)
		if err != nil {
			return nil, err
		}
	case req.FormURLEncoded != nil:
		req.httpBody = []byte(req.FormURLEncoded.Encode())
	default:
		req.httpBody = nil
	}

	// Create http.Request.
	url := base.URL + req.Path
	r, err := http.NewRequest(req.Method, url, bytes.NewReader(req.httpBody))
	if err != nil {
		return nil, err
	}

	// Apply base headers.
	for k, v := range base.Headers {
		r.Header.Add(k, v)
	}
	// Apply per-request headers.
	for k, v := range req.Headers {
		r.Header.Add(k, v)
	}

	return r, nil
}

func fromBody(body []byte) ([]byte, error) {
	if bytes.HasPrefix(body, []byte("\"@")) {
		b, err := ioutil.ReadFile(string(body[2 : len(body)-1]))
		if err != nil {
			return nil, err
		}
		// NOTE: not closing the file since http.NewRequest is taking care of that.
		return b, nil
	}
	return body, nil
}

func fromMultipart(multi multipartContent) ([]byte, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for k, v := range multi {
		if strings.HasPrefix(v, "@") { // This is a file.
			f, err := os.Open(v[1:])
			if err != nil {
				return nil, err
			}
			defer f.Close()
			fw, err := w.CreateFormFile(k, v[1:])
			if err != nil {
				return nil, err
			}
			if _, err = io.Copy(fw, f); err != nil {
				return nil, err
			}
		} else { // This is a key-value pair.
			fw, err := w.CreateFormField(k)
			if err != nil {
				return nil, err
			}
			if _, err := fw.Write([]byte(v)); err != nil {
				return nil, err
			}
		}
	}

	w.Close()
	return b.Bytes(), nil
}
