// This file was generated by counterfeiter
package fakes

import "sync"

type CNIController struct {
	UpStub        func(namespacePath, handle, spec string) error
	upMutex       sync.RWMutex
	upArgsForCall []struct {
		namespacePath string
		handle        string
		spec          string
	}
	upReturns struct {
		result1 error
	}
	DownStub        func(namespacePath, handle, spec string) error
	downMutex       sync.RWMutex
	downArgsForCall []struct {
		namespacePath string
		handle        string
		spec          string
	}
	downReturns struct {
		result1 error
	}
}

func (fake *CNIController) Up(namespacePath string, handle string, spec string) error {
	fake.upMutex.Lock()
	fake.upArgsForCall = append(fake.upArgsForCall, struct {
		namespacePath string
		handle        string
		spec          string
	}{namespacePath, handle, spec})
	fake.upMutex.Unlock()
	if fake.UpStub != nil {
		return fake.UpStub(namespacePath, handle, spec)
	} else {
		return fake.upReturns.result1
	}
}

func (fake *CNIController) UpCallCount() int {
	fake.upMutex.RLock()
	defer fake.upMutex.RUnlock()
	return len(fake.upArgsForCall)
}

func (fake *CNIController) UpArgsForCall(i int) (string, string, string) {
	fake.upMutex.RLock()
	defer fake.upMutex.RUnlock()
	return fake.upArgsForCall[i].namespacePath, fake.upArgsForCall[i].handle, fake.upArgsForCall[i].spec
}

func (fake *CNIController) UpReturns(result1 error) {
	fake.UpStub = nil
	fake.upReturns = struct {
		result1 error
	}{result1}
}

func (fake *CNIController) Down(namespacePath string, handle string, spec string) error {
	fake.downMutex.Lock()
	fake.downArgsForCall = append(fake.downArgsForCall, struct {
		namespacePath string
		handle        string
		spec          string
	}{namespacePath, handle, spec})
	fake.downMutex.Unlock()
	if fake.DownStub != nil {
		return fake.DownStub(namespacePath, handle, spec)
	} else {
		return fake.downReturns.result1
	}
}

func (fake *CNIController) DownCallCount() int {
	fake.downMutex.RLock()
	defer fake.downMutex.RUnlock()
	return len(fake.downArgsForCall)
}

func (fake *CNIController) DownArgsForCall(i int) (string, string, string) {
	fake.downMutex.RLock()
	defer fake.downMutex.RUnlock()
	return fake.downArgsForCall[i].namespacePath, fake.downArgsForCall[i].handle, fake.downArgsForCall[i].spec
}

func (fake *CNIController) DownReturns(result1 error) {
	fake.DownStub = nil
	fake.downReturns = struct {
		result1 error
	}{result1}
}
