package ppool_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/yupi/ppool"
)

func TestRun(t *testing.T) {
	pp := ppool.New()

	pp.Run(
		os.Args[0],
		[]string{"-test.run=TestHelperProcess", "--", "proc1"},
		[]string{"TEST_HELPER_PROCESS=1"}, ppool.Backoff{
			100 * time.Millisecond,
			200 * time.Millisecond,
			300 * time.Millisecond,
			-1, // terminate after 3 runs
		},
	)
	pp.WaitAll()
	t.Fail()
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("TEST_HELPER_PROCESS") != "1" {
		return
	}

	fmt.Println(os.Args[2])
	os.Exit(1)
}
