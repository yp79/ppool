package ppool

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

type Process struct {
	pid     int
	backoff *Backoff
	pp      *ProcessPool
	cmd     *exec.Cmd
	stopped bool
}

type ProcessPool struct {
	mu             sync.Mutex
	processes      map[int]*Process
	defaultBackoff Backoff
}

type opt func(*ProcessPool)

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
	proc := &Process{pp: pp, backoff: &backoff}
	err := proc.start(path, args, env)
	if err != nil {
		return nil, err
	}

	pp.mu.Lock()
	pp.processes[proc.pid] = proc
	pp.mu.Unlock()

	return proc, nil
}

// Dumb implementation for tests
func (pp *ProcessPool) WaitAll() {
	for {
		time.Sleep(50 * time.Millisecond)
		if len(pp.processes) == 0 {
			return
		}
	}
}

func (p *Process) start(path string, args []string, env []string) error {
	cmd := &exec.Cmd{
		Path: path,
		Args: args,
		Env:  env,
	}
	p.cmd = cmd

	if err := cmd.Start(); err != nil {
		return err
	}

	p.pid = cmd.Process.Pid
	p.pp.mu.Lock()
	p.pp.processes[p.pid] = p
	p.pp.mu.Unlock()

	go func() {
		log.Print("waiting\n")
		err := cmd.Wait()
		log.Print("process exited\n")

		oldPid := p.pid
		if err != nil {
			fmt.Printf("process exited with error, restarting\n")
			if !p.stopped && p.backoff != nil {
				d, stop := p.backoff.Duration()
				if stop {
					fmt.Printf("no more backoffs")
					p.pp.mu.Lock()
					delete(p.pp.processes, oldPid)
					p.pp.mu.Unlock()
					return
				}

				fmt.Printf("sleeping for %d seconds\n", d*1000/time.Second)
				time.Sleep(d * 1000)
				if err := p.start(path, args, env); err != nil {
					return
				}

				p.pp.mu.Lock()
				delete(p.pp.processes, oldPid)
				p.pp.mu.Unlock()
			}
		}
	}()

	return nil
}

func (p *Process) Stop() error {
	p.stopped = true
	p.pp.mu.Lock()
	delete(p.pp.processes, p.pid)
	p.pp.mu.Unlock()
	return p.cmd.Process.Kill()
}
