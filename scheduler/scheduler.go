package scheduler

import (
	"errors"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"
)

// Errors.
var (
	ErrInvalidCronExpr = errors.New("invalid cron expression")
	ErrJobAlreadyExist = errors.New("job already exist")
	ErrJobNotFound     = errors.New("job not found")
)

// Scheduler schedules jobs to run them at a given interval.
type Scheduler struct {
	jobs    map[string]*job
	running bool

	mu sync.Mutex
}

// New returns a scheduler.
func New() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*job),
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.running = true
	for _, j := range s.jobs {
		j.schedule()
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	for _, j := range s.jobs {
		j.cancel()
	}
	s.running = false
}

// Add adds a job in the scheduler. The job runs every interval duration
// when the scheduler is started or already running.
func (s *Scheduler) Add(name string, j Job, interval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[name]; ok {
		return ErrJobAlreadyExist
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

// AddCron adds the job in the scheduler. The job will be run depending of the cron expression
// when the scheduler is started or already running.
func (s *Scheduler) AddCron(name string, j Job, expr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[name]; ok {
		return ErrJobAlreadyExist
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

// Remove removes the job from the scheduler.
func (s *Scheduler) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, ok := s.jobs[name]
	if !ok {
		return ErrJobNotFound
	}

	j.cancel()
	delete(s.jobs, name)
	return nil
}
