// +build linux, darwin

package ppool

import (
	"bytes"
	"errors"
	"os/exec"
	"time"
)

type Process struct {
	path    string
	args    []string
	env     []string
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
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

func (p *Process) Stop() error {
	p.stopped = true
	p.pp.deleteProcess(p.Pid())
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

	//fmt.Println("starting")
	p.cmd = &exec.Cmd{
		Path:   p.path,
		Args:   p.args,
		Env:    p.env,
		Stdout: p.stdout,
		Stderr: p.stderr,
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
					//fmt.Println("no more backoffs")
					return
				}
				//fmt.Printf("sleeping for %f seconds\n", float64(d)/float64(time.Second))

				time.Sleep(d)
				_ = p.start()
			}
		}
	}()

	return nil
}

func (p *Process) StdoutOutput() []byte {
	return p.stdout.Bytes()
}

func (p *Process) StderrOutput() []byte {
	return p.stderr.Bytes()
}
