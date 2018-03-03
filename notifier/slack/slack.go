package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/plugin"
)

const (
	infoColor   = "good"
	warnColor   = "warning"
	dangerColor = "danger"
)

type notification struct {
	Channel     string       `json:"channel,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Username    string       `json:"username,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []attachment `json:"attachments,omitempty"`
}

type attachment struct {
	Color     string            `json:"color,omitempty"`
	Fallback  string            `json:"fallback,omitempty"`
	ImageURL  string            `json:"image_url,omitempty"`
	Text      string            `json:"text,omitempty"`
	Title     string            `json:"title,omitempty"`
	ThumbURL  string            `json:"thumb_url,omitempty"`
	MrkdwnIn  []string          `json:"mrkdwn_in,omitempty"`
	Fields    []attachmentField `json:"fields,omitempty"`
	Timestamp int64             `json:"ts,omitempty"`
}

type attachmentField struct {
	Short bool   `json:"short,omitempty"`
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
}

// Notifier is a slack notifier.
type Notifier struct {
	cfg *Config
}

type Config struct {
	Webhook  string `json:"webhook,omitempty"`
	Username string `json:"username,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Verbose  bool   `json:"verbose,omitempty"`
}

// New returns a new Notifier given a Slack webhook and a test suite name.
func New(webhook, username, channel string) *Notifier {
	cfg := &Config{
		Webhook:  webhook,
		Username: username,
		Channel:  channel,
	}
	return NewFromConfig(cfg)
}

func NewFromConfig(cfg *Config) *Notifier {
	return &Notifier{
		cfg: cfg,
	}
}

// Notify send a slack notification with the provided report.
func (sn *Notifier) Notify(r *notifier.Report) {
	errors := 0
	for _, tr := range r.TestReports {
		errors += len(tr.Errs)
	}
	extra := ""
	if errors == 0 {
		if !sn.cfg.Verbose {
			return
		}
		extra = "ok"
	} else if errors == 1 {
		extra = "1 test failed"
	} else {
		extra = fmt.Sprintf("%d tests failed", errors)
	}
	if r.Suite.FailFast() {
		extra += " (fail fast)"
	}
	notif := &notification{
		Username: sn.cfg.Username,
		Channel:  sn.cfg.Channel,
		Text:     fmt.Sprintf("*%s* - %s", r.Suite.Name(), extra),
	}
	for _, tr := range r.TestReports {
		color := dangerColor
		status := " failed"
		if len(tr.Errs) == 0 {
			if !sn.cfg.Verbose {
				continue
			}
			status = ""
			color = infoColor
		}
		title := fmt.Sprintf("Test %q%s", tr.Test.Name(), status)
		a := attachment{
			MrkdwnIn: []string{"fields"},
			Fallback: title,
			Color:    color,
			Title:    title,
			Fields: []attachmentField{
				{
					Title: "Description",
					Value: tr.Test.Description(),
				},
				{
					Title: "Duration",
					Value: tr.Duration.String(),
					Short: true,
				},
			},
			Timestamp: tr.Start.Unix(),
		}
		for _, err := range tr.Errs {
			a.Fields = append(a.Fields, attachmentField{
				Value: fmt.Sprintf("```%v```", err),
			})
		}
		notif.Attachments = append(notif.Attachments, a)
	}
	sn.send(notif)
}

func (sn *Notifier) send(notif *notification) {
	data, err := json.Marshal(notif)
	if err != nil {
		log.Error("could not marshal slack notification", zap.Error(err))
		return
	}
	resp, err := http.Post(sn.cfg.Webhook, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Error("could not send slack notification", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("could not read slack notification response body", zap.Error(err))
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Error("post slack notification failed", zap.ByteString("body", body))
	}
}

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.NotifierPlugin,
		ID:     "slack",
		Config: (*Config)(nil),
		InitFn: func(ctx *plugin.InitContext) (interface{}, error) {
			cfg := ctx.Config.(*Config)
			return NewFromConfig(cfg), nil
		},
	})
}
