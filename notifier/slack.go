package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
)

const (
	redColor    = "#ff2a00"
	orangeColor = "#f99157"
)

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
type slackNotifier struct {
	webhook  string
	username string
	channel  string
}

// NewSlackNotifier returns a new SlackNotifier given a Slack webhook and a test suite name.
func NewSlackNotifier(webhook, username, channel string) Notifier {
	return &slackNotifier{
		webhook:  webhook,
		username: username,
		channel:  channel,
	}
}

// SuiteDone implements the Notifier interface.
func (sn *slackNotifier) Notify(r *Report) {
	errors := 0
	for _, tr := range r.tests {
		errors += len(tr.errs)
	}
	// Only send a Slack notification if something went wrong.
	if errors == 0 {
		return
	}

	test := "test"
	if errors > 1 {
		test = "tests"
	}
	notif := &notification{
		Username: sn.username,
		Channel:  sn.channel,
		Text:     fmt.Sprintf("*%s* - %d %s failed", r.name, errors, test),
	}

	for _, tr := range r.tests {
		if len(tr.errs) == 0 {
			continue
		}
		a := attachment{
			MrkdwnIn: []string{"fields"},
			Fallback: tr.name + " failed",
			Color:    redColor,
			Title:    fmt.Sprintf("Test %q failed (%v):", tr.name, tr.duration),
		}
		for _, err := range tr.errs {
			a.Fields = append(a.Fields, attachmentField{
				Value: fmt.Sprintf("```%v```", err),
			})
		}
		notif.Attachments = append(notif.Attachments, a)
	}
	sn.send(notif)
}

func (sn *slackNotifier) send(notif *notification) {
	data, err := json.Marshal(notif)
	if err != nil {
		log.Error("could not marshal slack notification", zap.Error(err))
		return
	}
	resp, err := http.Post(sn.webhook, "application/json", bytes.NewBuffer(data))
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
