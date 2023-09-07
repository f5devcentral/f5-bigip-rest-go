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
}

// go test -bench=. -v -test.benchmem

func index5(x int) string {
	return fmt.Sprintf("r-%05d", x)
}
func makeDR(x int) DeployRequest {
	return DeployRequest{Name: index5(x)}
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

	fs := dq.Filter(makeDR(0), compare)
	if len(fs) != 0 {
		t.Errorf("there should no item be filtered.")
	}

	dq.Add(makeDR(0))
	fs = dq.Filter(makeDR(0), compare)
	if len(fs) != 1 {
		t.Errorf("filtered fs should be len of 1")
	}
	if dq.Len() != 0 {
		t.Errorf("queue should be empty now.")
	}
	for i := 0; i < 10; i++ {
		dq.Add(makeDR(i))
	}
	fs = dq.Filter(makeDR(0), compare)
	if len(fs) != 1 {
		t.Errorf("filtered fs should be len of 1")
	}
	r := dq.Get()
	if !compare(r, makeDR(1)) {
		t.Errorf("get should work and return dr1")
	}
}

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
	dq := NewDeployQueue()
	for i := 0; i < b.N; i++ {
		dq.Add(makeDR(i))
	}

	compare := func(a, b interface{}) bool {
		x := a.(DeployRequest)
		y := b.(DeployRequest)
		return x.Name == y.Name && x.Partition == y.Partition
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs := dq.Filter(makeDR(i), compare)
		if len(fs) != 1 || dq.Len() != i {
			b.Errorf("filter runs error")
		}
	}
}
