package httpexpect

import (
	"github.com/xeipuuv/gojsonreference"
	"github.com/xeipuuv/gojsonschema"
)

type jsonGoLoader struct {
	source interface{}
}

func newJSONGoLoader(source interface{}) *jsonGoLoader {
	return &jsonGoLoader{source: source}
}

func (l *jsonGoLoader) JsonSource() interface{} {
	return l.source
}

func (l *jsonGoLoader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference("#")
}

func (l *jsonGoLoader) LoaderFactory() gojsonschema.JSONLoaderFactory {
	return &gojsonschema.DefaultJSONLoaderFactory{}
}

func (l *jsonGoLoader) LoadJSON() (interface{}, error) {
	return l.source, nil
}
