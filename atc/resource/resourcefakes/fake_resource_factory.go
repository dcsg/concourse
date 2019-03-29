// Code generated by counterfeiter. DO NOT EDIT.
package resourcefakes

import (
	sync "sync"

	resource "github.com/concourse/concourse/atc/resource"
	worker "github.com/concourse/concourse/atc/worker"
)

type FakeResourceFactory struct {
	NewResourceForContainerStub        func(worker.Container) resource.Resource
	newResourceForContainerMutex       sync.RWMutex
	newResourceForContainerArgsForCall []struct {
		arg1 worker.Container
	}
	newResourceForContainerReturns struct {
		result1 resource.Resource
	}
	newResourceForContainerReturnsOnCall map[int]struct {
		result1 resource.Resource
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeResourceFactory) NewResourceForContainer(arg1 worker.Container) resource.Resource {
	fake.newResourceForContainerMutex.Lock()
	ret, specificReturn := fake.newResourceForContainerReturnsOnCall[len(fake.newResourceForContainerArgsForCall)]
	fake.newResourceForContainerArgsForCall = append(fake.newResourceForContainerArgsForCall, struct {
		arg1 worker.Container
	}{arg1})
	fake.recordInvocation("NewResourceForContainer", []interface{}{arg1})
	fake.newResourceForContainerMutex.Unlock()
	if fake.NewResourceForContainerStub != nil {
		return fake.NewResourceForContainerStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.newResourceForContainerReturns
	return fakeReturns.result1
}

func (fake *FakeResourceFactory) NewResourceForContainerCallCount() int {
	fake.newResourceForContainerMutex.RLock()
	defer fake.newResourceForContainerMutex.RUnlock()
	return len(fake.newResourceForContainerArgsForCall)
}

func (fake *FakeResourceFactory) NewResourceForContainerCalls(stub func(worker.Container) resource.Resource) {
	fake.newResourceForContainerMutex.Lock()
	defer fake.newResourceForContainerMutex.Unlock()
	fake.NewResourceForContainerStub = stub
}

func (fake *FakeResourceFactory) NewResourceForContainerArgsForCall(i int) worker.Container {
	fake.newResourceForContainerMutex.RLock()
	defer fake.newResourceForContainerMutex.RUnlock()
	argsForCall := fake.newResourceForContainerArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceFactory) NewResourceForContainerReturns(result1 resource.Resource) {
	fake.newResourceForContainerMutex.Lock()
	defer fake.newResourceForContainerMutex.Unlock()
	fake.NewResourceForContainerStub = nil
	fake.newResourceForContainerReturns = struct {
		result1 resource.Resource
	}{result1}
}

func (fake *FakeResourceFactory) NewResourceForContainerReturnsOnCall(i int, result1 resource.Resource) {
	fake.newResourceForContainerMutex.Lock()
	defer fake.newResourceForContainerMutex.Unlock()
	fake.NewResourceForContainerStub = nil
	if fake.newResourceForContainerReturnsOnCall == nil {
		fake.newResourceForContainerReturnsOnCall = make(map[int]struct {
			result1 resource.Resource
		})
	}
	fake.newResourceForContainerReturnsOnCall[i] = struct {
		result1 resource.Resource
	}{result1}
}

func (fake *FakeResourceFactory) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.newResourceForContainerMutex.RLock()
	defer fake.newResourceForContainerMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeResourceFactory) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ resource.ResourceFactory = new(FakeResourceFactory)
