package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
)

const (
	redColor    = "#ff2a00"
	orangeColor = "#f99157"
)

var _ = Notifier(&SlackNotifier{})

type notification struct {
	Channel     string       `json:"channel"`
	IconEmoji   string       `json:"icon_emoji"`
	Username    string       `json:"username"`
	Text        string       `json:"text"`
	Attachments []attachment `json:"attachments"`
}

type attachment struct {
	Color    string            `json:"color"`
	Fallback string            `json:"fallback"`
	ImageURL string            `json:"image_url"`
	Text     string            `json:"text"`
	Title    string            `json:"title"`
	ThumbURL string            `json:"thumb_url"`
	MrkdwnIn []string          `json:"mrkdwn_in"`
	Fields   []attachmentField `json:"fields"`
}

type attachmentField struct {
	Short bool   `json:"short"`
	Title string `json:"title"`
	Value string `json:"value"`
}

// SlackNotifier is a reporter that stacks errors for later use.
// Stacked errors are printed on each report and removed from the stack.
type SlackNotifier struct {
	webhook   string
	suiteName string

	start      time.Time
	currentRes testResult
	results    []testResult
}

type testResult struct {
	name     string
	duration time.Duration
	err      error
	failures []error
}

func (t *testResult) isFailed() bool {
	return len(t.failures) > 0
}

func (t *testResult) hasError() bool {
	return t.err != nil
}

// NewSlackNotifier returns a new SlackNotifier given a Slack webhook and a test suite name.
func NewSlackNotifier(webhook, name string) *SlackNotifier {
	return &SlackNotifier{webhook: webhook, suiteName: name}
}

// BeforeTest implements the Notifier interface.
func (r *SlackNotifier) BeforeTest(name string) {
	r.start = time.Now()
	r.currentRes = testResult{name: name}
}

// Report implements the Notifier interface.
func (r *SlackNotifier) Report(err error) {
	r.currentRes.failures = append(r.currentRes.failures, err)
}

// Errorf implements the Notifier interface.
func (r *SlackNotifier) Errorf(format string, args ...interface{}) {
	r.Report(fmt.Errorf(format, args...))
}

// TestError implements the Notifier interface.
func (r *SlackNotifier) TestError(err error) {
	r.currentRes.err = err
}

// AfterTest implements the Notifier interface.
func (r *SlackNotifier) AfterTest() {
	r.currentRes.duration = time.Since(r.start)
	r.results = append(r.results, r.currentRes)
}

// SuiteDone implements the Notifier interface.
func (r *SlackNotifier) SuiteDone() {
	failures := 0
	errors := 0
	for _, r := range r.results {
		if r.isFailed() {
			failures++
		} else if r.hasError() {
			errors++
		}
	}

	// Only send a Slack notification if something went wrong.
	if failures == 0 && errors == 0 {
		return
	}

	test := "test"
	if failures+errors > 1 {
		test = "tests"
	}
	notif := &notification{
		// Username: "aragorn",
		// Channel:  "aragorn-test",
		Text: fmt.Sprintf("*%s* - %d %s failed", r.suiteName, failures+errors, test),
	}

	var attachments []attachment
	for _, r := range r.results {
		var a attachment
		if r.hasError() {
			a = attachment{
				MrkdwnIn: []string{"fields"},
				Fallback: r.name + " could not run",
				Color:    orangeColor,
				Title:    fmt.Sprintf("Test %q could not run:", r.name),
			}
			a.Fields = append(a.Fields, attachmentField{
				Value: fmt.Sprintf("```%s```", r.err),
			})
			attachments = append(attachments, a)
		} else if r.isFailed() {
			a = attachment{
				MrkdwnIn: []string{"fields"},
				Fallback: r.name + " failed",
				Color:    redColor,
				Title:    fmt.Sprintf("Test %q failed (%v):", r.name, r.duration),
			}
			for _, failure := range r.failures {
				a.Fields = append(a.Fields, attachmentField{
					Value: fmt.Sprintf("```%s```", failure),
				})
			}
			attachments = append(attachments, a)
		}
	}

	notif.Attachments = append(notif.Attachments, attachments...)

	sendSlackNotification(r.webhook, notif)
	r.results = nil
}

func sendSlackNotification(webhook string, notif *notification) {
	data, err := json.Marshal(notif)
	if err != nil {
		log.Error("could not marshal slack notification", zap.Error(err))
		return
	}
	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Error("could not send slack notification", zap.Error(err))
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("could not read slack notification response body", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("post slack notification failed", zap.ByteString("body", body))
	}
}
