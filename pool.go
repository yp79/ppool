package ppool

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type Process struct {
	path    string
	args    []string
	env     []string
	backoff *Backoff
	pp      *ProcessPool
	cmd     *exec.Cmd
	stopped bool
}

func (p *Process) Pid() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return -1
}

func (p *Process) kill() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return errors.New("no process")
}

type ProcessPool struct {
	mu             sync.Mutex
	processes      map[int]*Process
	defaultBackoff Backoff
	wg             sync.WaitGroup
}

type opt func(*ProcessPool)

func WithDefaultBackoff(b Backoff) opt {
	return func(pp *ProcessPool) {
		pp.defaultBackoff = b
	}
}

func New(opts ...opt) *ProcessPool {
	p := &ProcessPool{
		processes: make(map[int]*Process),
	}
	for _, o := range opts {
		o(p)
	}

	return p
}

func (pp *ProcessPool) Run(path string, args []string, env []string, backoff Backoff) (*Process, error) {
	if backoff == nil && pp.defaultBackoff != nil {
		backoff = make(Backoff, len(pp.defaultBackoff))
		copy(backoff, pp.defaultBackoff)
	}
	proc := &Process{
		path:    path,
		args:    args,
		env:     env,
		pp:      pp,
		backoff: &backoff,
	}
	if err := proc.start(); err != nil {
		return nil, err
	}

	return proc, nil
}

func (pp *ProcessPool) WaitAll() {
	pp.wg.Wait()
}

func (pp *ProcessPool) addProcess(p *Process) {
	pp.wg.Add(1)
	pp.mu.Lock()
	pp.processes[p.Pid()] = p
	pp.mu.Unlock()
}

func (pp *ProcessPool) deleteProcess(pid int) {
	pp.mu.Lock()
	delete(pp.processes, pid)
	pp.mu.Unlock()
}

func (p *Process) start() error {
	p.cmd = &exec.Cmd{
		Path: p.path,
		Args: p.args,
		Env:  p.env,
	}

	if err := p.cmd.Start(); err != nil {
		return err
	}

	p.pp.addProcess(p)

	go func() {
		defer p.pp.wg.Done()

		err := p.cmd.Wait()
		p.pp.deleteProcess(p.Pid())

		if err != nil {
			if !p.stopped && p.backoff != nil {
				d, stop := p.backoff.Duration()
				if stop {
					fmt.Println("no more backoffs")
					return
				}

				time.Sleep(d)
				if err := p.start(); err != nil {
					return
				}
			}
		}
	}()

	return nil
}

func (p *Process) Stop() error {
	p.stopped = true
	p.pp.deleteProcess(p.Pid())
	return p.kill()
}
