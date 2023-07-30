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
