package httpexpect

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// newHTTPRequest returns an htpp.Request from a base and a request.
func (cfg *Config) setHTTPRequest(t *test, req *Request) error {
	// Generate request body.
	var cntType string
	var err error
	if req.Body != nil {
		v, err := cfg.getDocumentField(req.Body)
		if err != nil {
			return err
		}
		body, ok := v.([]byte)
		if !ok {
			body, err = json.Marshal(v)
			if err != nil {
				return err
			}
			cntType = "application/json; charset=utf-8"
		}
		t.body = body
	} else if req.Multipart != nil {
		t.body, cntType, err = cfg.fromMultipart(req.Multipart)
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
	if cntType != "" && t.req.Header.Get("Content-Type") == "" {
		t.req.Header.Set("Content-Type", cntType)
	}
	return nil
}

func (cfg *Config) fromBody(body interface{}) ([]byte, error) {
	s, ok := body.(string)
	if !ok {
		return json.Marshal(body)
	}
	if !strings.HasPrefix(s, "@") {
		return []byte(s), nil
	}
	path := cfg.getFilePath(s[1:])
	return ioutil.ReadFile(path)
}

func (cfg *Config) fromMultipart(m map[string]string) ([]byte, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for k, v := range m {
		if err := cfg.addMultipartKV(w, k, v); err != nil {
			return nil, "", err
		}
	}
	w.Close()
	return b.Bytes(), w.FormDataContentType(), nil
}

func (cfg *Config) addMultipartKV(w *multipart.Writer, k, v string) error {
	if strings.HasPrefix(v, "@") {
		path := cfg.getFilePath(v[1:])
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		filename := filepath.Base(path)
		fw, err := w.CreateFormFile(k, filename)
		if err != nil {
			return err
		}
		if _, err := io.Copy(fw, f); err != nil {
			return err
		}
		return nil
	}
	fw, err := w.CreateFormField(k)
	if err != nil {
		return err
	}
	_, err = fw.Write([]byte(v))
	return err
}

func fromFormData(m map[string]string) []byte {
	formData := url.Values{}
	for k, v := range m {
		formData.Set(k, v)
	}
	return []byte(formData.Encode())
}
