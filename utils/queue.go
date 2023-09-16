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

// Filter is used to filter items from queue:
//
// "item" is the element to compare with, will be passed as the first argument to cmp and stop functions;
// "cmp" is a compare function to match elements, if cmp == nil, returns []interface{}{};
// "stop" is a function indicates filter to stop traversing, if stop == nil, Filter will traverse all items of DeployQueue.
func (dq *DeployQueue) Filter(item interface{}, cmp func(a, b interface{}) bool, stop func(a, b interface{}) bool) []interface{} {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	left := []interface{}{}
	rlt := []interface{}{}
	if len(dq.Items) == 0 {
		return rlt
	}
	if cmp == nil && stop == nil {
		return rlt
	}

	<-dq.found
	for i := 0; i < len(dq.Items); i++ {
		if cmp != nil && cmp(item, dq.Items[i]) {
			rlt = append(rlt, dq.Items[i])
		} else {
			left = append(left, dq.Items[i])
		}

		if i+1 >= len(dq.Items) {
			break
		}
		if stop != nil && stop(item, dq.Items[i+1]) {
			left = append(left, dq.Items[i+1:]...)
			break
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
