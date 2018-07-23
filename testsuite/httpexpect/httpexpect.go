package httpexpect

import (
	"bytes"
	"context"
	"crypto/tls"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/Masterminds/sprig"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	ot "github.com/opentracing/opentracing-go"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/oauth2"

	"github.com/blippar/aragorn/plugin"
	"github.com/blippar/aragorn/testsuite"
)

var _ testsuite.Suite = (*Suite)(nil)

// Suite describes an HTTP test suite.
type Suite struct {
	tests []testsuite.Test
}

// New returns a Suite.
func New(cfg *Config) (*Suite, error) {
	client := &http.Client{}
	if cfg.Base.Insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	client.Transport = &nethttp.Transport{RoundTripper: client.Transport}
	if cfg.Base.OAUTH2 != nil {
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		client = cfg.Base.OAUTH2.Client(ctx)
	}
	tests, err := cfg.genTests(client)
	if err != nil {
		return nil, err
	}
	return &Suite{tests: tests}, nil
}

func (s *Suite) Tests() []testsuite.Test { return s.tests }

type test struct {
	id          string
	name        string
	description string
	saveDoc     bool

	client *http.Client
	req    *http.Request // Raw HTTP request generated from the request description.

	statusCode int
	header     testsuite.Header

	document   interface{}
	jsonSchema *gojsonschema.Schema   // Compiled jsonschema.
	jsonValues map[string]interface{} // Decoded JSONValues.
}

func (t *test) Name() string        { return t.name }
func (t *test) Description() string { return t.description }

func (t *test) Run(ctx context.Context, l testsuite.Logger) {
	req := t.cloneRequest().WithContext(ctx)

	md, ok := testsuite.MDFromContext(ctx)
	if ok {
		var b bytes.Buffer
		if req.URL.RawPath != "" {
			if tmpl, err := template.New("URL Path").Funcs(sprig.FuncMap()).Parse(req.URL.RawPath); err == nil {
				if err = tmpl.Execute(&b, md); err == nil {
					req.URL.Path = b.String()
					req.URL.RawPath = ""
				}
				b.Reset()
			}
		}

		if req.URL.RawQuery != "" {
			if tmpl, err := template.New("URL Query").Funcs(sprig.FuncMap()).Parse(req.URL.RawQuery); err == nil {
				if err = tmpl.Execute(&b, md); err == nil {
					req.URL.RawQuery = b.String()
				}
				b.Reset()
			}
		}

		for _, v := range req.Header {
			for idx, hdr := range v {
				if tmpl, err := template.New("Request Header").Funcs(sprig.FuncMap()).Parse(hdr); err == nil {
					if err = tmpl.Execute(&b, md); err != nil {
						v[idx] = b.String()
					}
					b.Reset()
				}
			}
		}
	}
	opName := "HTTP: " + t.Name()
	req, ht := nethttp.TraceRequest(ot.GlobalTracer(), req, nethttp.OperationName(opName))
	defer ht.Finish()

	resp, err := t.client.Do(req)
	if err != nil {
		l.Errorf("could not do HTTP request: %v", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Errorf("could not read body: %v", err)
		return
	}
	checkResponse(t, l, md, resp, body)
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func (t *test) cloneRequest() *http.Request {
	// shallow copy of the struct
	r := t.req
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	if r.Body != nil {
		r2.Body, _ = r.GetBody()
	}
	return r2
}

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.TestSuitePlugin,
		ID:     "HTTP",
		Config: (*Config)(nil),
		InitFn: func(ctx *plugin.InitContext) (interface{}, error) {
			cfg := ctx.Config.(*Config)
			cfg.Path = ctx.Path
			cfg.Root = ctx.Root
			return New(cfg)
		},
	})
}
