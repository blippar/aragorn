package grpcexpect

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	otgrpc "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"

	"github.com/blippar/aragorn/pkg/util/json"
	"github.com/blippar/aragorn/plugin"
	"github.com/blippar/aragorn/testsuite"
)

var _ testsuite.Suite = (*Suite)(nil)

// Suite describes a GRPC test suite.
type Suite struct {
	tests []testsuite.Test
}

// New returns a Suite.
func New(cfg *Config) (*Suite, error) {
	ctx := context.Background()
	tcOpt, err := cfg.transportDialOption()
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(otgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otgrpc.StreamClientInterceptor()),
		tcOpt,
	}
	if cfg.OAUTH2 != nil {
		httpClient := &http.Client{Transport: &nethttp.Transport{}}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
		ts := &oauth.TokenSource{TokenSource: cfg.OAUTH2.TokenSource(ctx)}
		opts = append(opts, grpc.WithPerRPCCredentials(ts))
	}
	cc, err := grpc.Dial(cfg.Address, opts...)
	if err != nil {
		return nil, err
	}
	var descSource grpcurl.DescriptorSource
	if cfg.ProtoSetPath != "" {
		psPath := cfg.getFilePath(cfg.ProtoSetPath)
		descSource, err = grpcurl.DescriptorSourceFromProtoSets(psPath)
		if err != nil {
			return nil, err
		}
	} else {
		refClient := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(cc))
		descSource = grpcurl.DescriptorSourceFromServer(ctx, refClient)
	}
	tests, err := cfg.genTests(cc, descSource)
	if err != nil {
		return nil, err
	}
	return &Suite{tests: tests}, nil
}

func (s *Suite) Tests() []testsuite.Test { return s.tests }

type test struct {
	cc         *grpc.ClientConn
	descSource grpcurl.DescriptorSource

	name        string
	description string
	req         request
	expect      expect
}

type request struct {
	methodName string
	headers    []string
	msgs       [][]byte
}

type expect struct {
	code   codes.Code
	header testsuite.Header
	msgs   [][]byte
}

func (t *test) Name() string        { return t.name }
func (t *test) Description() string { return t.description }

func (t *test) Run(ctx context.Context, logger testsuite.Logger) {
	h := &handler{reqs: t.req.msgs}
	err := grpcurl.InvokeRpc(ctx, t.descSource, t.cc, t.req.methodName, t.req.headers, h, h.getRequestData)
	if err != nil {
		logger.Errorf("could not invoke method: %v", err)
		return
	}
	if got, want := h.status.Code(), t.expect.code; got != want {
		logger.Errorf("wrong status code (got %s; want %s) message=%q", got, want, h.status.Message())
		return
	}
	for k, want := range t.expect.header {
		vs := h.md[k]
		if len(vs) == 0 {
			logger.Errorf("missing header %s", k)
			continue
		}
		if got := vs[0]; got != want {
			logger.Errorf("wrong value for header %q (got %q; want %q)", k, got, want)
		}
	}
	if len(t.expect.msgs) != len(h.resps) {
		logger.Errorf("wrong number of response (got %d; want %d)", len(h.resps), len(t.expect.msgs))
	}
	for i, wantRaw := range t.expect.msgs {
		if i >= len(h.resps) {
			break
		}
		got := h.resps[i]
		var want proto.Message
		switch v := got.(type) {
		case *dynamic.Message:
			gotCopy := *v
			gotCopy.Reset()
			want = &gotCopy
		default:
			wantRef := reflect.New(reflect.TypeOf(got).Elem())
			want = wantRef.Interface().(proto.Message)
		}
		if err := json.Unmarshal(wantRaw, want); err != nil {
			logger.Errorf("could not unmarshal expected document: %v", err)
			continue
		}
		if !dynamic.MessagesEqual(got, want) {
			logger.Errorf("wrong response\ngot: %s\nwant: %s", got, want)
		}
	}
}

type handler struct {
	reqs   [][]byte
	curReq int
	mu     sync.Mutex

	methodDesc *desc.MethodDescriptor
	md         metadata.MD
	status     *status.Status
	resps      []proto.Message
}

func (h *handler) OnResolveMethod(md *desc.MethodDescriptor)               { h.methodDesc = md }
func (*handler) OnSendHeaders(md metadata.MD)                              {}
func (h *handler) OnReceiveHeaders(md metadata.MD)                         { h.md = md }
func (h *handler) OnReceiveTrailers(status *status.Status, md metadata.MD) { h.status = status }
func (h *handler) OnReceiveResponse(resp proto.Message)                    { h.resps = append(h.resps, resp) }

func (h *handler) getRequestData() ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.curReq >= len(h.reqs) {
		return nil, io.EOF
	}
	req := h.reqs[h.curReq]
	h.curReq++
	return req, nil
}

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.TestSuitePlugin,
		ID:     "GRPC",
		Config: (*Config)(nil),
		InitFn: func(ctx *plugin.InitContext) (interface{}, error) {
			cfg := ctx.Config.(*Config)
			cfg.Path = ctx.Path
			cfg.Root = ctx.Root
			return New(cfg)
		},
	})
}
