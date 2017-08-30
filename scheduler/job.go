package scheduler

import (
	"time"

	"github.com/gorhill/cronexpr"
)

type Job interface {
	Run()
}

type job struct {
	job   Job
	timer *time.Timer

	interval time.Duration
	cronExpr *cronexpr.Expression
}

func (j *job) schedule() {
	d := j.timeUntilNextRun()
	j.timer = time.AfterFunc(d, func() {
		j.job.Run()
		j.schedule()
	})
}

func (j *job) cancel() {
	if j.timer != nil {
		j.timer.Stop()
	}
}

func (j *job) timeUntilNextRun() time.Duration {
	if j.interval > 0 {
		return j.interval
	}
	return time.Until(j.cronExpr.Next(time.Now()))
}
