//go:generate protoc --descriptor_set_out=grpctesting/test.protoset --include_imports --go_out=plugins=grpc:. grpctesting/test.proto

package grpcexpect

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"github.com/blippar/aragorn/testsuite/grpcexpect/grpctesting"
)

func TestNewProtoset(t *testing.T) {
	l, err := newGRPCTestServer(false)
	if err != nil {
		t.Errorf("grpc server init: %v", err)
	}
	defer l.Close()
	cfg := &Config{
		Address:      l.Addr().String(),
		ProtoSetPath: "./grpctesting/test.protoset",
		Header:       Header{"hello": "world"},
		Tests: []TestConfig{
			{
				Name:    "Empty Call",
				Request: RequestConfig{Method: "grpcexpect.testing.TestService/EmptyCall"},
				Expect:  ExpectConfig{Code: codes.OK},
			},
			{
				Name: "Simple Call",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService/SimpleCall",
					Document: map[string]interface{}{"username": "world"},
				},
				Expect: ExpectConfig{
					Code:     codes.OK,
					Header:   Header{"hello": "world"},
					Document: map[string]interface{}{"message": "Hello world!"},
				},
			},
		},
	}
	testsErrs := [][]string{nil, nil}
	checkSuite(t, cfg, testsErrs)
}

func TestNewProtosetNotFound(t *testing.T) {
	cfg := &Config{ProtoSetPath: "invalid.protoset"}
	want := `could not load protoset file "invalid.protoset": open invalid.protoset: no such file or directory`
	if _, err := New(cfg); err == nil || err.Error() != want {
		t.Fatalf("new suite invalid error (got %v; want %v)", err, want)
	}
}

func TestNewReflect(t *testing.T) {
	l, err := newGRPCTestServer(true)
	if err != nil {
		t.Errorf("grpc server init: %v", err)
	}
	defer l.Close()
	cfg := &Config{
		Address: l.Addr().String(),
		Tests: []TestConfig{
			{
				Name:    "Empty Call",
				Request: RequestConfig{Method: "grpcexpect.testing.TestService/EmptyCall"},
				Expect:  ExpectConfig{Code: codes.OK},
			},
			{
				Name:    "Empty Call with bad code",
				Request: RequestConfig{Method: "grpcexpect.testing.TestService/EmptyCall"},
				Expect:  ExpectConfig{Code: codes.NotFound},
			},
			{
				Name:    "Empty Call with missing header",
				Request: RequestConfig{Method: "grpcexpect.testing.TestService.EmptyCall"},
				Expect:  ExpectConfig{Code: codes.OK, Header: Header{"test": "123"}},
			},
			{
				Name: "Simple Call",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService/SimpleCall",
					Header:   Header{"hello": "world"},
					Document: map[string]interface{}{"username": "world"},
				},
				Expect: ExpectConfig{
					Code:     codes.OK,
					Header:   Header{"hello": "world"},
					Document: map[string]interface{}{"message": "Hello world!"},
				},
			},
			{
				Name: "Simple Call with invalid response header",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService.SimpleCall",
					Header:   Header{"hello": "123"},
					Document: map[string]interface{}{"username": "world"},
				},
				Expect: ExpectConfig{
					Code:     codes.OK,
					Header:   Header{"hello": "world"},
					Document: map[string]interface{}{"message": "Hello world!"},
				},
			},
			{
				Name: "Simple Call with invalid expected document",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService.SimpleCall",
					Document: map[string]interface{}{"username": "test"},
				},
				Expect: ExpectConfig{
					Code:     codes.OK,
					Document: map[string]interface{}{"message": "Hello world!"},
				},
			},
			{
				Name: "Simple Call with invalid request document field",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService.SimpleCall",
					Document: map[string]interface{}{"invalid_field": "test"},
				},
			},
			{
				Name: "Simple Call with invalid expected document field",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService.SimpleCall",
					Document: map[string]interface{}{"username": "test"},
				},
				Expect: ExpectConfig{
					Code:     codes.OK,
					Document: map[string]interface{}{"test": "Hello world!"},
				},
			},
			{
				Name: "Simple Call with invalid nb of responses",
				Request: RequestConfig{
					Method:   "grpcexpect.testing.TestService.SimpleCall",
					Document: map[string]interface{}{"username": "world"},
				},
				Expect: ExpectConfig{
					Code: codes.OK,
					Document: []interface{}{
						map[string]interface{}{"message": "Hello world!"},
						map[string]interface{}{"message": "Hello world!"},
					},
				},
			},
			{
				Name:    "Method not found",
				Request: RequestConfig{Method: "invalid_service/invalid_method"},
			},
		},
	}
	testsErrs := [][]string{
		nil,
		{`wrong status code (got OK; want NotFound) message=""`},
		{"missing header test"},
		nil,
		{`wrong value for header "hello" (got "123"; want "world")`},
		{"wrong response\ngot: message:\"Hello test!\"\nwant: message:\"Hello world!\""},
		{`could not invoke method: could not parse given request body as message of type "grpcexpect.testing.SimpleRequest": Message type grpcexpect.testing.SimpleRequest has no known field named invalid_field`},
		{`could not unmarshal expected document: Message type grpcexpect.testing.SimpleResponse has no known field named test`},
		{"wrong number of response (got 1; want 2)"},
		{`could not invoke method: target server does not expose service "invalid_service"`},
	}
	checkSuite(t, cfg, testsErrs)
}

func checkSuite(t *testing.T, cfg *Config, testsErrs [][]string) {
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("new suite failed: %v", err)
	}
	tests := s.Tests()
	if len(tests) != len(testsErrs) {
		panic("len(tests) != len(testsErrs)")
	}
	ctx := context.Background()
	for i, test := range tests {
		test := test
		t.Run(test.Name(), func(t *testing.T) {
			l := &mockLogger{}
			test.Run(ctx, l)
			if !cmp.Equal(l.errs, testsErrs[i]) {
				t.Errorf("unexpected errors (got %v; want %v)", l.errs, testsErrs[i])
			}
		})
	}
}

type mockLogger struct {
	errs []string
}

func (l *mockLogger) Error(args ...interface{}) {
	l.errs = append(l.errs, fmt.Sprint(args...))
}

func (l *mockLogger) Errorf(format string, args ...interface{}) {
	l.errs = append(l.errs, fmt.Sprintf(format, args...))
}

type testServer struct{}

func newGRPCTestServer(reflect bool) (net.Listener, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()
	grpctesting.RegisterTestServiceServer(s, testServer{})
	if reflect {
		reflection.Register(s)
	}
	go func() {
		s.Serve(l)
	}()
	return l, nil
}

func (testServer) EmptyCall(ctx context.Context, in *empty.Empty) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (testServer) SimpleCall(ctx context.Context, in *grpctesting.SimpleRequest) (*grpctesting.SimpleResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if vs := md["hello"]; len(vs) > 0 {
		grpc.SetHeader(ctx, metadata.Pairs("hello", vs[0]))
	}
	return &grpctesting.SimpleResponse{
		Message: fmt.Sprintf("Hello %s!", in.Username),
	}, nil
}

var _ grpctesting.TestServiceServer = (*testServer)(nil)
