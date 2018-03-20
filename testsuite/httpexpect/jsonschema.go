package httpexpect

import (
	"github.com/xeipuuv/gojsonreference"
	"github.com/xeipuuv/gojsonschema"
)

type jsonGoLoader struct {
	doc interface{}
}

func newJSONGoLoader(doc interface{}) *jsonGoLoader {
	return &jsonGoLoader{doc: doc}
}

func (l *jsonGoLoader) JsonSource() interface{} {
	return l.doc
}

func (l *jsonGoLoader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference("#")
}

func (l *jsonGoLoader) LoaderFactory() gojsonschema.JSONLoaderFactory {
	return JSONLoaderFactory{}
}

func (l *jsonGoLoader) LoadJSON() (interface{}, error) {
	return l.doc, nil
}

type JSONLoaderFactory struct{}

func (JSONLoaderFactory) New(source string) gojsonschema.JSONLoader {
	return gojsonschema.NewReferenceLoader(source)
}
