package plugin

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

var (
	// ErrNoType is returned when no type is specified
	ErrNoType = errors.New("plugin: no type")
	// ErrNoPluginID is returned when no id is specified
	ErrNoPluginID = errors.New("plugin: no id")
)

// Type is the type of the plugin
type Type string

func (t Type) String() string { return string(t) }

const (
	// TestSuitePlugin implements a test suite
	TestSuitePlugin Type = "aragorn.testsuite.v1"
	// NotifierPlugin implements a notifier
	NotifierPlugin Type = "aragorn.notifier.v1"
)

// Registration contains information for registering a plugin
type Registration struct {
	// Type of the plugin
	Type Type
	// ID of the plugin
	ID string
	// Config specific to the plugin
	Config interface{}

	// InitFn is called when initializing a plugin.
	InitFn func(*InitContext) (interface{}, error)
}

// Init the registered plugin
func (r *Registration) Init(ic *InitContext) (interface{}, error) {
	return r.InitFn(ic)
}

// URI returns the full plugin URI
func (r *Registration) URI() string {
	return registrationURI(r.Type, r.ID)
}

func registrationURI(t Type, id string) string {
	return fmt.Sprintf("%s.%s", t, id)
}

// InitContext is used for plugin inititalization
type InitContext struct {
	Config interface{}
	Path   string
	Root   string
}

var register = struct {
	sync.RWMutex
	r map[string]*Registration
}{
	r: make(map[string]*Registration),
}

// Register allows plugins to register
func Register(r *Registration) {
	register.Lock()
	defer register.Unlock()
	if r.Type == "" {
		panic(ErrNoType)
	}
	if r.ID == "" {
		panic(ErrNoPluginID)
	}
	uri := r.URI()
	register.r[uri] = r
}

func NewContext(r *Registration, path string) *InitContext {
	cfgType := reflect.TypeOf(r.Config).Elem()
	return &InitContext{
		Config: reflect.New(cfgType).Interface(),
		Path:   path,
		Root:   filepath.Dir(path),
	}
}

func Get(t Type, id string) *Registration {
	register.RLock()
	defer register.RUnlock()
	uri := registrationURI(t, id)
	return register.r[uri]
}

func ForType(t Type) []*Registration {
	register.RLock()
	defer register.RUnlock()
	tStr := t.String()
	var rs []*Registration
	for k, r := range register.r {
		if strings.HasPrefix(k, tStr) {
			rs = append(rs, r)
		}
	}
	return rs
}
