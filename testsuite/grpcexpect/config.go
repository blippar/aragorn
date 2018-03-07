package grpcexpect

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fullstorydev/grpcurl"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/blippar/aragorn/testsuite"
)

type Config struct {
	Path string

	Address      string
	ProtoSetPath string

	TLS                bool
	CAPath             string
	ServerHostOverride string
	Insecure           bool
	OAUTH2             clientcredentials.Config

	Header Header

	Tests []TestConfig
}

type TestConfig struct {
	Name    string
	Request RequestConfig
	Expect  ExpectConfig
}

type RequestConfig struct {
	Method   string
	Header   Header
	Document interface{}
}

type ExpectConfig struct {
	Code     codes.Code
	Header   Header
	Document interface{}
}

// A Header represents the key-value pairs in an GRPC header.
type Header map[string]string

func mergeHeaders(hs ...Header) Header {
	res := make(Header)
	for _, h := range hs {
		res.copy(h)
	}
	return res
}

func (h Header) copy(src Header) {
	for k, v := range src {
		h[k] = v
	}
}

func (h Header) slice() []string {
	res := make([]string, 0, len(h))
	for k, v := range h {
		res = append(res, k+":"+v)
	}
	return res
}

func (cfg *Config) genTests(cc *grpc.ClientConn, descSource grpcurl.DescriptorSource) ([]testsuite.Test, error) {
	tests := make([]testsuite.Test, len(cfg.Tests))
	for i, tcfg := range cfg.Tests {
		reqMsgs, err := newDocToMsgs(tcfg.Request.Document)
		if err != nil {
			return nil, fmt.Errorf("test %d %s: request: %v", i, tcfg.Name, err)
		}
		expMsgs, err := newDocToMsgs(tcfg.Expect.Document)
		if err != nil {
			return nil, fmt.Errorf("test %d %s: expect: %v", i, tcfg.Name, err)
		}
		tests[i] = &test{
			cc:          cc,
			descSource:  descSource,
			name:        tcfg.Name,
			description: fmt.Sprintf("grpc://%s/%s", cfg.Address, tcfg.Request.Method),
			req: request{
				methodName: tcfg.Request.Method,
				headers:    mergeHeaders(cfg.Header, tcfg.Request.Header).slice(),
				msgs:       reqMsgs,
			},
			expect: expect{
				code:   tcfg.Expect.Code,
				header: mergeHeaders(tcfg.Expect.Header),
				msgs:   expMsgs,
			},
		}
	}
	return tests, nil
}

func (cfg *Config) getDocumentField(v interface{}) (interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return v, nil
	}
	ref, ok := m["$ref"].(string)
	if !ok {
		return v, nil
	}
	path := cfg.getFilePath(ref)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var newVal interface{}
	err = decodeReaderJSON(f, &newVal)
	return newVal, err
}

func (cfg *Config) getFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.Path, path)
}

func (cfg *Config) transportDialOption() (grpc.DialOption, error) {
	if !cfg.TLS {
		return grpc.WithInsecure(), nil
	}
	cp := x509.NewCertPool()
	if cfg.CAPath != "" {
		caPath := cfg.getFilePath(cfg.CAPath)
		b, err := ioutil.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("could not read CA file: %v", err)
		}
		if !cp.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("credentials: failed to append certificates")
		}
	}
	tlsCfg := &tls.Config{
		ServerName:         cfg.ServerHostOverride,
		RootCAs:            cp,
		InsecureSkipVerify: cfg.Insecure,
	}
	tc := credentials.NewTLS(tlsCfg)
	return grpc.WithTransportCredentials(tc), nil
}

func newDocToMsgs(doc interface{}) ([][]byte, error) {
	var a []interface{}
	switch v := doc.(type) {
	case []interface{}:
		a = v
	case map[string]interface{}:
		a = []interface{}{v}
	case nil:
		a = []interface{}{struct{}{}}
	default:
		return nil, errors.New("invalid document type")
	}
	res := make([][]byte, len(a))
	for i, v := range a {
		res[i], _ = json.Marshal(v)
	}
	return res, nil
}

func decodeReaderJSON(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(v)
}

func decodeJSON(b []byte, v interface{}) error {
	return decodeReaderJSON(bytes.NewReader(b), v)
}
