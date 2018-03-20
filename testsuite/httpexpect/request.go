package httpexpect

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/blippar/aragorn/pkg/util/json"
	"github.com/blippar/aragorn/testsuite"
)

// toHTTPRequest returns an http.Request from a base and a request.
func (req *Request) toHTTPRequest(cfg *Config) (*http.Request, error) {
	body, cntType, err := req.getBody(cfg)
	if err != nil {
		return nil, err
	}
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	path := req.Path
	if path == "" {
		path = "/"
	}
	var urlStr string
	if req.URL != "" {
		urlStr = req.URL + path
	} else {
		urlStr = cfg.Base.URL + path
	}
	httpReq, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	setHeaderinHTTPRequest(cfg.Base.Header, httpReq)
	setHeaderinHTTPRequest(req.Header, httpReq)
	if cntType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", cntType)
	}
	return httpReq, nil
}

func setHeaderinHTTPRequest(h testsuite.Header, req *http.Request) {
	for k, v := range h {
		req.Header.Set(k, v)
	}
}

func (req *Request) getBody(cfg *Config) (io.Reader, string, error) {
	var (
		body    []byte
		cntType string
	)
	if req.Body != nil {
		v, err := cfg.getDocumentField(req.Body)
		if err != nil {
			return nil, "", err
		}
		var ok bool
		body, ok = v.([]byte)
		if !ok {
			body, err = json.Marshal(v)
			if err != nil {
				return nil, "", err
			}
			cntType = "application/json; charset=utf-8"
		}
	} else if req.Multipart != nil {
		var err error
		body, cntType, err = cfg.fromMultipart(req.Multipart)
		if err != nil {
			return nil, "", err
		}
	} else if req.FormData != nil {
		body = fromFormData(req.FormData)
		cntType = "application/x-www-form-urlencoded"
	} else {
		return nil, "", nil
	}
	return bytes.NewReader(body), cntType, nil
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
