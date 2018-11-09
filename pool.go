// +build linux, darwin

package ppool

import (
	"bytes"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// ProcessPool stores information about running processes and pool configuration.
type ProcessPool struct {
	mu             sync.Mutex
	processes      map[int]*Process
	defaultBackoff Backoff
	wg             sync.WaitGroup
}

type opt func(*ProcessPool)

// New creates ProcessPool with configured options.
func New(opts ...opt) *ProcessPool {
	p := &ProcessPool{
		processes: make(map[int]*Process),
	}
	for _, o := range opts {
		o(p)
	}

	return p
}

// WithDefaultBackoff sets default backoff strategy to be used if one is not
// specified when running process.
func WithDefaultBackoff(b Backoff) opt {
	return func(pp *ProcessPool) {
		pp.defaultBackoff = b
	}
}

// WithSigTermRelay enables relaying of term signal to running processes.
func WithSigTermRelay() opt {
	return func(pp *ProcessPool) {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM)

		go func() {
			_ = <-c
			pp.KillAll()
		}()
	}
}

// Run starts a new process.
func (pp *ProcessPool) Run(path string, args []string, env []string, backoff Backoff) (*Process, error) {
	if backoff == nil && pp.defaultBackoff != nil {
		backoff = pp.defaultBackoff
	}
	proc := &Process{
		path:    path,
		args:    args,
		env:     env,
		pp:      pp,
		backoff: backoff,
		stdout:  &bytes.Buffer{},
		stderr:  &bytes.Buffer{},
	}
	if err := proc.start(); err != nil {
		return nil, err
	}

	return proc, nil
}

// WaitAll waits for all running processes to complete
func (pp *ProcessPool) WaitAll() {
	pp.wg.Wait()
}

// KillAll immidiately kills all running processes by sending kill signal
func (pp *ProcessPool) KillAll() {
	pp.mu.Lock()
	for _, p := range pp.processes {
		p.Stop()
	}
	pp.mu.Unlock()
}

func (pp *ProcessPool) addProcess(p *Process) {
	pp.wg.Add(1)
	pp.mu.Lock()
	pp.processes[p.Pid()] = p
	pp.mu.Unlock()
}

func (pp *ProcessPool) deleteProcess(pid int) {
	pp.wg.Done()
	pp.mu.Lock()
	delete(pp.processes, pid)
	pp.mu.Unlock()
}
