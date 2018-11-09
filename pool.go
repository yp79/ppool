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
	wg             sync.WaitGroup
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

	return proc, nil
}

func (pp *ProcessPool) WaitAll() {
	pp.wg.Wait()
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
	p.pp.wg.Add(1)
	p.pp.mu.Lock()
	p.pp.processes[p.pid] = p
	p.pp.mu.Unlock()

	go func() {
		log.Print("waiting\n")
		defer p.pp.wg.Done()

		err := cmd.Wait()
		log.Print("process exited\n")

		p.pp.mu.Lock()
		delete(p.pp.processes, p.pid)
		p.pp.mu.Unlock()

		if err != nil {
			fmt.Printf("process exited with error, restarting\n")
			if !p.stopped && p.backoff != nil {
				d, stop := p.backoff.Duration()
				if stop {
					fmt.Println("no more backoffs")
					return
				}

				fmt.Printf("sleeping for %f seconds\n", float64(d)/float64(time.Second))
				time.Sleep(d)
				if err := p.start(path, args, env); err != nil {
					return
				}
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
