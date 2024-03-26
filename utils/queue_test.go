package utils

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

type DeployRequest struct {
	Name      string
	Partition string
	Operation string
}

// go test -bench=. -v -test.benchmem

func index5(x int) string {
	return fmt.Sprintf("r-%05d", x)
}
func makeDR(x int) DeployRequest {
	return DeployRequest{Name: index5(x)}
}

func makeDRWithOps(x int, op int) DeployRequest {
	return DeployRequest{Name: index5(x), Operation: fmt.Sprintf("%d", op%2)}
}

func Test_DeployQueue_Add(t *testing.T) {
	dq := NewDeployQueue()
	for i := 0; i < 100; i++ {
		dq.Add(makeDR(i))
	}
	if dq.Len() != 100 {
		t.Errorf("DeployQueue.Len() should be 100")
	}
}

func Test_DeployQueue_Get(t *testing.T) {
	dq := NewDeployQueue()
	rs := make(chan DeployRequest, 20)
	for i := 0; i < 10; i++ {
		dq.Add(makeDR(i))
	}
	go func() {
		for {
			select {
			case <-time.After(time.Duration(10 * time.Millisecond)):
				return
			case rs <- dq.Get().(DeployRequest):
			}

		}
	}()

	<-time.After(time.Duration(20 * time.Millisecond))
	if len(rs) != 10 {
		t.Errorf("gotten r should be 10")
	}
}

func Test_DeployQueue_Insert(t *testing.T) {
	dq := NewDeployQueue()
	for i := 0; i < 10; i++ {
		dq.Insert(makeDR(i))
	}
	if dq.Get().(DeployRequest).Name != index5(9) {
		t.Errorf("gotten r should be %s", index5(9))
	}
}

func Test_DeployQueue_Dump(t *testing.T) {
	dq := NewDeployQueue()
	for i := 0; i < 10; i++ {
		dq.Add(makeDR(i))
	}
	dumps := dq.Dumps()
	if len(dumps) != 10 {
		t.Errorf("dumped r should be len of 10")
	}
}

func Test_DeployQueue_Cocurrency(t *testing.T) {
	dq := NewDeployQueue()
	stopCh := make(chan struct{})
	total := []string{}
	lock := sync.Mutex{}

	go func() {
		for i := 0; i < 200; i++ {
			<-time.After(time.Duration(rand.Intn(5)) * time.Microsecond * 100)
			dq.Add(DeployRequest{Name: index5(i)})
		}
	}()
	go func() {
		for i := 200; i < 400; i++ {
			<-time.After(time.Duration(rand.Intn(5)) * time.Microsecond * 100)
			dq.Add(DeployRequest{Name: index5(i)})
		}
	}()

	go func() {
		for i := 400; i < 600; i++ {
			<-time.After(time.Duration(rand.Intn(5)) * time.Microsecond * 100)
			dq.Add(DeployRequest{Name: index5(i)})
		}
	}()

	go func() {
		r := make(chan DeployRequest, 1)
		for {
			select {
			case <-stopCh:
				return
			case r <- dq.Get().(DeployRequest):
				lock.Lock()
				n := "a" + (<-r).Name
				total = append(total, n)
				lock.Unlock()
			}
		}
	}()
	go func() {
		r := make(chan DeployRequest, 1)
		for {
			select {
			case <-stopCh:
				return
			case r <- dq.Get().(DeployRequest):
				lock.Lock()
				n := "b" + (<-r).Name
				total = append(total, n)
				lock.Unlock()
			}
		}
	}()
	go func() {
		r := make(chan DeployRequest, 1)
		for {
			select {
			case <-stopCh:
				return
			case r <- dq.Get().(DeployRequest):
				lock.Lock()
				n := "c" + (<-r).Name
				total = append(total, n)
				lock.Unlock()
			}
		}
	}()

	<-time.After(time.Duration(time.Second))
	close(stopCh)
	if len(total) != 600 {
		t.Errorf("total items should be 600")
	}
}

func Test_DeployQueue_Filter(t *testing.T) {
	dq := NewDeployQueue()
	compare := func(a, b interface{}) bool {
		x := a.(DeployRequest)
		y := b.(DeployRequest)
		return x.Name == y.Name && x.Partition == y.Partition
	}
	stop := func(a, b interface{}) bool {
		return false
	}

	// empty deployqueue
	fs := dq.Filter(makeDR(0), compare, stop)
	if len(fs) != 0 {
		t.Errorf("there should no item be filtered.")
	}

	// deployqueue with 1 item
	dq.Add(makeDR(0))
	fs = dq.Filter(makeDR(0), compare, stop)
	if len(fs) != 1 {
		t.Errorf("filtered fs should be len of 1")
	}
	if dq.Len() != 0 {
		t.Errorf("queue should be empty now.")
	}

	// deployqueue with 10 items, successfully filtered.
	for i := 0; i < 10; i++ {
		dq.Add(makeDR(i))
	}
	fs = dq.Filter(makeDR(0), compare, stop)
	if len(fs) != 1 {
		t.Errorf("filtered fs should be len of 1")
	}
	r := dq.Get()
	if !compare(r, makeDR(1)) {
		t.Errorf("get should work and return dr1")
	}
}

func Test_DeployQueue_Filter_nilfunc(t *testing.T) {
	dq := NewDeployQueue()
	dq.Add(makeDR(0))
	var fs []interface{}

	// cmp and stop are both nil
	fs = dq.Filter(makeDR(0), nil, nil)
	if len(fs) != 0 || dq.Len() != 1 {
		t.Errorf("there should no item be filtered.")
	}

	// cmp is nil
	fs = dq.Filter(makeDR(0), nil, func(a interface{}, b interface{}) bool { return false })
	if len(fs) != 0 || dq.Len() != 1 {
		t.Errorf("there should no item be filtered.")
	}

	// stop is nil
	fs = dq.Filter(makeDR(0), func(a interface{}, b interface{}) bool { return true }, nil)
	if len(fs) != 1 || dq.Len() != 0 {
		t.Errorf("there should 1 item be filtered.")
	}
}

func Test_DeployQueue_Filter_stop1(t *testing.T) {
	dq := NewDeployQueue()
	compare := func(a, b interface{}) bool {
		x := a.(DeployRequest)
		y := b.(DeployRequest)
		return x.Name == y.Name && x.Partition == y.Partition
	}
	stop1 := func(a, b interface{}) bool {
		x, y := a.(DeployRequest), b.(DeployRequest)
		return x.Partition != y.Partition ||
			x.Name != y.Name || x.Operation != y.Operation
	}

	ns := []int{0, 1, 2, 3, 4, 5, 0, 0, 0, 6, 7, 8, 9}
	os := []int{0, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1}
	for i := range ns {
		dq.Add(makeDRWithOps(ns[i], os[i]))
	}
	fs := dq.Filter(makeDR(0), compare, stop1)
	if len(fs) != 1 {
		t.Errorf("there should be 1 item been filtered.")
	}
	if dq.Len() != 12 {
		t.Errorf("queue should left with length of 12")
	}
}

func Test_DeployQueue_Filter_stop2(t *testing.T) {
	dq := NewDeployQueue()
	compare := func(a, b interface{}) bool {
		x := a.(DeployRequest)
		y := b.(DeployRequest)
		return x.Name == y.Name && x.Partition == y.Partition && x.Operation == y.Operation
	}
	stop2 := func(a, b interface{}) bool {
		x, y := a.(DeployRequest), b.(DeployRequest)
		b1 := x.Partition == y.Partition &&
			x.Name == y.Name

		if b1 {
			return x.Operation != y.Operation
		} else {
			return false
		}
	}
	ns := []int{0, 1, 2, 3, 4, 5, 0, 0, 0, 6, 7, 8, 9}
	os := []int{0, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1}
	for i := range ns {
		dq.Add(makeDRWithOps(ns[i], os[i]))
	}

	fs := dq.Filter(makeDRWithOps(0, 0), compare, stop2)
	if len(fs) != 2 {
		t.Errorf("there should be 2 item been filtered.")
	}
	if dq.Len() != 11 {
		t.Errorf("queue should left with length of 11")
	}
}

func Test_DeployQueue_Filter_stop3(t *testing.T) {
	dq := NewDeployQueue()
	compare := func(a, b interface{}) bool {
		x := a.(DeployRequest)
		y := b.(DeployRequest)
		return x.Name == y.Name && x.Partition == y.Partition && x.Operation == y.Operation
	}
	stop3 := func(a, b interface{}) bool { return false }

	ns := []int{0, 1, 2, 3, 4, 5, 0, 0, 0, 6, 7, 8, 9}
	os := []int{0, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1}
	for i := range ns {
		dq.Add(makeDRWithOps(ns[i], os[i]))
	}

	fs := dq.Filter(makeDRWithOps(0, 0), compare, stop3)
	if len(fs) != 3 {
		t.Errorf("there should be 3 item been filtered.")
	}
	if dq.Len() != 10 {
		t.Errorf("queue should left with length of 10")
	}
}

//go test -benchmem -run=^$ -bench "^.*$" github.com/f5devcentral/f5-bigip-rest-go/utils

func Benchmark_DeployQueue_Add(b *testing.B) {
	dq := NewDeployQueue()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dq.Add(DeployRequest{})
	}
}

func Benchmark_DeployQueue_Insert(b *testing.B) {
	dq := NewDeployQueue()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dq.Insert(DeployRequest{})
	}
}

func Benchmark_DeployQueue_Get(b *testing.B) {
	dq := NewDeployQueue()
	for i := 0; i < b.N; i++ {
		dq.Add(DeployRequest{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dq.Get()
	}
}

func Benchmark_DeployQueue_Filter(b *testing.B) {

	compare := func(a, b interface{}) bool {
		x := a.(DeployRequest)
		y := b.(DeployRequest)
		return x.Name == y.Name && x.Partition == y.Partition
	}

	dq := NewDeployQueue()
	for i := 0; i < b.N; i++ {
		dq.Add(makeDR(i))
	}

	fs := dq.Filter(makeDR(b.N-1), compare, func(a, b interface{}) bool { return false })
	if len(fs) != 1 || dq.Len() != b.N-1 {
		b.Errorf("b.N: %d, filter runs error: fs.len: %d, dq.len: %d", b.N, len(fs), dq.Len())
	}
}
