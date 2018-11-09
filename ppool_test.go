package ppool_test

import (
	"fmt"
	"os"
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
			//-1,
		}),
	)

	proc, err := pp.Run(
		os.Args[0],
		[]string{"-v", "-test.run=TestHelperProcess", "--", "proc1"},
		[]string{"TEST_HELPER_PROCESS=1"},
		nil,
		/*
			ppool.Backoff{
				100 * time.Millisecond,
				200 * time.Millisecond,
				300 * time.Millisecond,
				-1, // terminate after 3 runs
			},
		*/
	)
	if err != nil {
		t.Error(err)
	}

	fmt.Println("sleep 400")
	time.Sleep(400 * time.Millisecond)
	fmt.Println("sleep 400 done")
	proc.Stop()

	pp.WaitAll()
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("TEST_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Printf("%v\n", os.Args)
	os.Exit(1)
}
