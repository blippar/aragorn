package scheduler

import (
	"errors"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"
)

// Errors.
var (
	ErrJobAlreadyExist = errors.New("job already exist")
	ErrInvalidCronExpr = errors.New("invalid cron expression")
)

type Scheduler struct {
	jobs    map[string]*job
	running bool

	mu sync.Mutex
}

func (s *Scheduler) Start() {
	s.running = true
	for _, j := range s.jobs {
		j.schedule()
	}
}

func (s *Scheduler) Stop() {
	for _, j := range s.jobs {
		j.cancel()
	}
	s.running = false
}

func (s *Scheduler) Add(name string, j Job, interval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[name]; ok {
		return ErrJobAlreadyExist
	}

	if s.jobs == nil {
		s.jobs = make(map[string]*job)
	}

	job := &job{
		job:      j,
		interval: interval,
	}
	s.jobs[name] = job

	if s.running {
		s.jobs[name].schedule()
	}

	return nil
}

func (s *Scheduler) AddCron(name string, j Job, expr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[name]; ok {
		return ErrJobAlreadyExist
	}

	if s.jobs == nil {
		s.jobs = make(map[string]*job)
	}

	cronExpr, err := cronexpr.Parse(expr)
	if err != nil {
		return ErrInvalidCronExpr
	}

	job := &job{
		job:      j,
		cronExpr: cronExpr,
	}
	s.jobs[name] = job

	if s.running {
		s.jobs[name].schedule()
	}

	return nil
}

func (s *Scheduler) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, ok := s.jobs[name]
	if !ok {
		return
	}

	j.cancel()
	delete(s.jobs, name)
}
