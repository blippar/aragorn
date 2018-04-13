package grpcexpect

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fullstorydev/grpcurl"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/blippar/aragorn/pkg/util/json"
	"github.com/blippar/aragorn/testsuite"
)

type Config struct {
	Path               string                    `json:"path,omitempty"`
	Root               string                    `json:"root,omitempty"`
	Address            string                    `json:"address,omitempty"`
	ProtoSetPath       string                    `json:"protoSetPath,omitempty"`
	TLS                bool                      `json:"tls,omitempty"`
	CAPath             string                    `json:"caPath,omitempty"`
	ServerHostOverride string                    `json:"serverHostOverride,omitempty"`
	Insecure           bool                      `json:"insecure,omitempty"`
	OAUTH2             *clientcredentials.Config `json:"oauth2,omitempty"`
	Header             testsuite.Header          `json:"header,omitempty"`
	Tests              []TestConfig              `json:"tests,omitempty"`
}

type TestConfig struct {
	Name    string        `json:"name,omitempty"`
	Request RequestConfig `json:"request,omitempty"`
	Expect  ExpectConfig  `json:"expect,omitempty"`
}

type RequestConfig struct {
	Method   string           `json:"method,omitempty"`
	Header   testsuite.Header `json:"header,omitempty"`
	Document interface{}      `json:"document,omitempty"`
}

type ExpectConfig struct {
	Code     codes.Code       `json:"code,omitempty"`
	Header   testsuite.Header `json:"header,omitempty"`
	Document interface{}      `json:"document,omitempty"`
}

func (*Config) Example() interface{} {
	return &Config{
		Address:  "localhost:50051",
		Insecure: true,
		Tests: []TestConfig{
			{
				Name: "Simple Call",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService/SimpleCall",
					Header:   testsuite.Header{"hello": "world"},
					Document: map[string]interface{}{"username": "world"},
				},
				Expect: ExpectConfig{
					Code:     codes.OK,
					Header:   testsuite.Header{"hello": "world"},
					Document: map[string]interface{}{"message": "Hello world!"},
				},
			},
		},
	}
}

func (cfg *Config) genTests(cc *grpc.ClientConn, descSource grpcurl.DescriptorSource) ([]testsuite.Test, error) {
	tests := make([]testsuite.Test, len(cfg.Tests))
	for i, tcfg := range cfg.Tests {
		reqMsgs, err := cfg.newDocToMsgs(tcfg.Request.Document)
		if err != nil {
			return nil, fmt.Errorf("test %d %s: request: %v", i, tcfg.Name, err)
		}
		expMsgs, err := cfg.newDocToMsgs(tcfg.Expect.Document)
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
				headers:    testsuite.MergeHeaders(cfg.Header, tcfg.Request.Header).Slice(),
				msgs:       reqMsgs,
			},
			expect: expect{
				code:   tcfg.Expect.Code,
				header: testsuite.MergeHeaders(tcfg.Expect.Header),
				msgs:   expMsgs,
			},
		}
	}
	return tests, nil
}

func (cfg *Config) getFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.Root, path)
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

func (cfg *Config) newDocToMsgs(doc interface{}) ([][]byte, error) {
	d, err := loadDoc(cfg.Path, doc)
	if err != nil {
		return nil, err
	}
	var a []interface{}
	switch v := d.(type) {
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

func loadDoc(path string, i interface{}) (interface{}, error) {
	switch doc := i.(type) {
	case []interface{}:
		for i, item := range doc {
			newVal, err := loadDoc(path, item)
			if err != nil {
				return nil, err
			} else if &newVal != &item {
				doc[i] = newVal
			}
		}
	case map[string]interface{}:
		if ref, ok := doc["$ref"].(string); ok {
			var refPath string
			if filepath.IsAbs(ref) {
				refPath = ref
			} else {
				refPath = filepath.Join(path, ref)
			}
			if raw, _ := doc["$raw"].(bool); raw {
				return loadFileDoc(refPath)
			}
			return loadJSONFileDoc(refPath)
		}
		if raw, ok := doc["$raw"].(string); ok {
			return base64.StdEncoding.EncodeToString([]byte(raw)), nil
		}
		for k, v := range doc {
			if newVal, err := loadDoc(path, v); err != nil {
				return nil, err
			} else if newVal != v {
				doc[k] = newVal
			}
		}
	}
	return i, nil
}

func loadJSONFileDoc(path string) (interface{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var newVal map[string]interface{}
	if err := json.Decode(f, &newVal); err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	return loadDoc(dir, newVal)
}

func loadFileDoc(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
