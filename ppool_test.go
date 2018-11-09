// +build linux, darwin

package ppool_test

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/yupi/ppool"
)

func TestRun(t *testing.T) {
	pp := ppool.New(
		ppool.WithDefaultBackoff(ppool.Backoff{
			100 * time.Millisecond,
			200 * time.Millisecond,
			500 * time.Millisecond,
			-1,
		}),
	)

	proc1, err := pp.Run(
		os.Args[0],
		[]string{"-v", "-test.run=TestHelperProcess", "--", "p1", "1"},
		[]string{"TEST_HELPER_PROCESS=1"},
		nil,
	)
	if err != nil {
		t.Error(err)
	}

	proc2, err := pp.Run(
		os.Args[0],
		[]string{"-v", "-test.run=TestHelperProcess", "--", "p2", "0", "1"},
		[]string{"TEST_HELPER_PROCESS=1"},
		nil,
	)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(50 * time.Millisecond)
	pp.WaitAll()

	output := string(proc1.StdoutOutput())
	if output != "p1\np1\np1\np1\n" {
		t.Error(output)
	}

	output = string(proc2.StdoutOutput())
	if output != "p2\nsleep 1\n" {
		t.Error(output)
	}
}

func TestProcessStop(t *testing.T) {
	pp := ppool.New(
		ppool.WithDefaultBackoff(ppool.Backoff{
			100 * time.Millisecond,
			-1,
		}),
	)

	proc, err := pp.Run(
		os.Args[0],
		[]string{"-v", "-test.run=TestHelperProcess", "--", "p1", "1"},
		[]string{"TEST_HELPER_PROCESS=1"},
		ppool.Backoff{
			100 * time.Millisecond,
			200 * time.Millisecond,
			500 * time.Millisecond,
		},
	)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(400 * time.Millisecond)
	proc.Stop()

	// Process should be started 3 times
	output := string(proc.StdoutOutput())
	if output != "p1\np1\np1\n" {
		t.Error(output)
	}
}

func TestSigTermRelay(t *testing.T) {
	pp := ppool.New(
		ppool.WithSigTermRelay(),
		ppool.WithDefaultBackoff(ppool.Backoff{
			100 * time.Millisecond,
			200 * time.Millisecond,
			500 * time.Millisecond,
			-1,
		}),
	)

	_, err := pp.Run(
		os.Args[0],
		[]string{"-v", "-test.run=TestHelperProcess", "--", "p1", "1", "100000"},
		[]string{"TEST_HELPER_PROCESS=1"},
		nil,
	)
	if err != nil {
		t.Error(err)
	}

	_, err = pp.Run(
		os.Args[0],
		[]string{"-v", "-test.run=TestHelperProcess", "--", "p2", "0", "100000"},
		[]string{"TEST_HELPER_PROCESS=1"},
		nil,
	)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(50 * time.Millisecond)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	// Will wait for 100000 seconds if SIGTERM won't be delivered
	pp.WaitAll()
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("TEST_HELPER_PROCESS") != "1" {
		return
	}

	// Pertly copied from https://golang.org/src/os/exec/exec_test.go
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) < 2 {
		os.Exit(0)
	}

	name, exitCode, args := args[0], args[1], args[2:]
	fmt.Println(name)

	if len(args) >= 1 {
		wait, _ := strconv.Atoi(args[0])
		d := time.Duration(wait) * time.Second
		fmt.Printf("sleep %d\n", d/time.Second)
		time.Sleep(d)
	}

	code, _ := strconv.Atoi(exitCode)
	os.Exit(code)
}
