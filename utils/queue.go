package utils

import "sync"

func (dq *DeployQueue) Len() int {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	return len(dq.Items)
}

func (dq *DeployQueue) Add(r interface{}) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	dq.Items = append(dq.Items, r)
	if len(dq.Items) == 1 {
		dq.found <- true
	}
}
func (dq *DeployQueue) Insert(r interface{}) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	dq.Items = append([]interface{}{r}, dq.Items...)
	if len(dq.Items) == 1 {
		dq.found <- true
	}
}

func (dq *DeployQueue) Dumps() []interface{} {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	rlt := []interface{}{}
	return append(rlt, dq.Items...)
}

func (dq *DeployQueue) Get() interface{} {
	<-dq.found
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	rlt := dq.Items[0]
	dq.Items = dq.Items[1:]
	if len(dq.Items) > 0 {
		dq.found <- true
	}
	return rlt
}

func (dq *DeployQueue) Filter(item interface{}, f func(a, b interface{}) bool) []interface{} {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	left := []interface{}{}
	rlt := []interface{}{}
	if len(dq.Items) == 0 {
		return rlt
	}

	<-dq.found
	for _, n := range dq.Items {
		if f(item, n) {
			rlt = append(rlt, n)
		} else {
			left = append(left, n)
		}
	}
	dq.Items = left
	if len(dq.Items) > 0 {
		dq.found <- true
	}

	return rlt
}

func NewDeployQueue() *DeployQueue {
	dq := &DeployQueue{
		mutex: sync.Mutex{},
		found: make(chan bool, 1),
		Items: []interface{}{},
	}
	return dq
}
