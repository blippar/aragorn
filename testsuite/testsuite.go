package testsuite

import "context"

type Suite interface {
	Tests() []Test
}

type Test interface {
	Name() string
	Description() string
	Run(context.Context, Logger)
}

type Logger interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// A Header represents the key-value pairs.
type Header map[string]string

func MergeHeaders(hs ...Header) Header {
	res := make(Header)
	for _, h := range hs {
		res.Copy(h)
	}
	return res
}

func (h Header) Copy(src Header) {
	for k, v := range src {
		h[k] = v
	}
}

func (h Header) Slice() []string {
	res := make([]string, 0, len(h))
	for k, v := range h {
		res = append(res, k+":"+v)
	}
	return res
}

type MD map[string]interface{}

type mdKey struct{}

func NewMD() MD {
	return MD{}
}

func NewMDContext(ctx context.Context, md MD) context.Context {
	return context.WithValue(ctx, mdKey{}, md)
}

func MDFromContext(ctx context.Context) (md MD, ok bool) {
	md, ok = ctx.Value(mdKey{}).(MD)
	return
}
