// +build linux, darwin

package ppool

import (
	"bytes"
	"errors"
	"os/exec"
	"time"
)

// Process stores process related data.
type Process struct {
	path    string
	args    []string
	env     []string
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
	backoff Backoff
	pp      *ProcessPool
	cmd     *exec.Cmd
	stopped bool
}

// Pid returns pid of this process if started or -1 if it isn't running yet
func (p *Process) Pid() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return -1
}

// Stop sends kill signal to the process.
func (p *Process) Stop() error {
	p.stopped = true
	return p.kill()
}

func (p *Process) kill() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return errors.New("no process")
}

func (p *Process) start() error {
	if p.stopped {
		return nil
	}

	p.cmd = &exec.Cmd{
		Path:   p.path,
		Args:   p.args,
		Env:    p.env,
		Stdout: p.stdout,
		Stderr: p.stderr,
	}

	ready := make(chan error)
	go func() {
		err := p.cmd.Start()
		ready <- err
		if err != nil {
			return
		}

		p.pp.addProcess(p)
		defer p.pp.deleteProcess(p.Pid())

		if err := p.cmd.Wait(); err != nil {
			if !p.stopped && p.backoff != nil {
				d, stop := p.backoff.Duration()
				if stop {
					return
				}

				time.Sleep(d)
				_ = p.start()
			}
		}
	}()

	return <-ready
}

// StdoutOutput returns combined output of process' stdout
func (p *Process) StdoutOutput() []byte {
	return p.stdout.Bytes()
}

// StderrOutput returns combined output of process' stderr
func (p *Process) StderrOutput() []byte {
	return p.stderr.Bytes()
}
