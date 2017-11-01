package httpexpect

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// newHTTPRequest returns an htpp.Request from a base and a request.
func (cfg *Config) setHTTPRequest(t *test, req *Request) error {
	// Generate request body.
	var cntType string
	var err error
	if req.Body != nil {
		t.body, err = fromBody(req.Body)
		if err != nil {
			return err
		}
	} else if req.Multipart != nil {
		t.body, cntType, err = fromMultipart(req.Multipart)
		if err != nil {
			return err
		}
	} else if req.FormData != nil {
		t.body = fromFormData(req.FormData)
		cntType = "application/x-www-form-urlencoded"
	}

	// Create http request.
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	path := req.Path
	if path == "" {
		path = "/"
	}
	var url string
	if req.URL != "" {
		url = req.URL + path
	} else {
		url = cfg.Base.URL + path
	}
	t.req, err = http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	for k, v := range cfg.Base.Header {
		t.req.Header.Set(k, v)
	}
	for k, v := range req.Header {
		t.req.Header.Set(k, v)
	}
	if cntType != "" {
		t.req.Header.Set("Content-Type", cntType)
	}
	return nil
}

func fromBody(body []byte) ([]byte, error) {
	if bytes.HasPrefix(body, []byte(`"@`)) && bytes.HasSuffix(body, []byte(`"`)) {
		return ioutil.ReadFile(string(body[2 : len(body)-1]))
	}
	return body, nil
}

func fromMultipart(multi map[string]string) ([]byte, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for k, v := range multi {
		if strings.HasPrefix(v, "@") { // This is a file.
			filename := v[1:]
			f, err := os.Open(filename)
			if err != nil {
				return nil, "", err
			}
			defer f.Close()
			fw, err := w.CreateFormFile(k, filename)
			if err != nil {
				return nil, "", err
			}
			if _, err = io.Copy(fw, f); err != nil {
				return nil, "", err
			}
		} else { // This is a key-value pair.
			fw, err := w.CreateFormField(k)
			if err != nil {
				return nil, "", err
			}
			if _, err := fw.Write([]byte(v)); err != nil {
				return nil, "", err
			}
		}
	}

	w.Close()
	return b.Bytes(), w.FormDataContentType(), nil
}

func fromFormData(m map[string]string) []byte {
	formData := url.Values{}
	for k, v := range m {
		formData.Set(k, v)
	}
	return []byte(formData.Encode())
}
