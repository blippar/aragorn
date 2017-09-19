package httpexpect

import (
	"bytes"
	"fmt"
	"io/ioutil"
)

func (s *Suite) runTest(t *test) error {
	t.Request.httpReq.Body = ioutil.NopCloser(bytes.NewReader(t.Request.httpBody))
	resp, err := s.client.Do(t.Request.httpReq)
	if err != nil {
		return fmt.Errorf("could not do HTTP request: %v", err)
	}
	// NOTE: not closing the body since NewResponse is taking care of that.

	r, err := NewResponse(s.notifier, resp)
	if err != nil {
		return err
	}

	r.StatusCode(t.Expect.StatusCode)
	r.ContainsHeaders(t.Expect.Headers)
	if t.Expect.Document != nil {
		if t.Expect.jsonDocument != nil {
			r.MatchJSONDocument(t.Expect.jsonDocument)
		} else {
			r.MatchRawDocument(t.Expect.rawDocument)
		}
	} else if t.Expect.JSONSchema != nil {
		r.MatchJSONSchema(t.Expect.jsonSchema)
		if t.Expect.JSONValues != nil {
			r.ContainsJSONValues(t.Expect.jsonValues)
		}
	}
	return nil
}
