package ppool_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/yupi/ppool"
)

func TestBackoffSimple(t *testing.T) {
	type rv struct {
		d    time.Duration
		stop bool
	}
	tests := []struct {
		backoff *ppool.BackoffSimple
		tries   int
		result  []rv
	}{
		{&ppool.BackoffSimple{}, 4, []rv{{0, true}, {0, true}, {0, true}, {0, true}}},
		{&ppool.BackoffSimple{-1, 5}, 4, []rv{{0, true}, {0, true}, {0, true}, {0, true}}},
		{&ppool.BackoffSimple{1, 2, 3}, 4, []rv{{1, false}, {2, false}, {3, false}, {3, false}}},
		{&ppool.BackoffSimple{1, 2, -1}, 4, []rv{{1, false}, {2, false}, {0, true}, {0, true}}},
	}

	for _, test := range tests {
		r := make([]rv, 0, test.tries)
		for i := 0; i < test.tries; i++ {
			d, s := test.backoff.Duration()
			r = append(r, rv{d, s})
		}
		if !reflect.DeepEqual(r, test.result) {
			t.Fail()
		}
	}
}
