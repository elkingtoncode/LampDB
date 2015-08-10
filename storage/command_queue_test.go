// Copyright 2014 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Spencer Kimball (spencer.kimball@gmail.com)

package storage

import (
	"sync"
	"testing"
	"time"

	"github.com/cockroachdb/cockroach/proto"
	"github.com/cockroachdb/cockroach/util/leaktest"
)

// waitForCmd launches a goroutine to wait on the supplied
// WaitGroup. A channel is returned which signals the completion of
// the wait.
func waitForCmd(wg *sync.WaitGroup) <-chan struct{} {
	cmdDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(cmdDone)
	}()
	return cmdDone
}

// testCmdDone waits for the cmdDone channel to be closed for at most
// the specified wait duration. Returns true if the command finished in
// the allotted time, false otherwise.
func testCmdDone(cmdDone <-chan struct{}, wait time.Duration) bool {
	select {
	case <-cmdDone:
		return true
	case <-time.After(wait):
		return false
	}
}

func TestCommandQueue(t *testing.T) {
	defer leaktest.AfterTest(t)
	cq := NewCommandQueue()
	wg := sync.WaitGroup{}

	// Try a command with no overlapping already-running commands.
	cq.GetWait(proto.Key("a"), nil, false, &wg)
	wg.Wait()
	cq.GetWait(proto.Key("a"), proto.Key("b"), false, &wg)
	wg.Wait()

	// Add a command and verify wait group is returned.
	wk := cq.Add(proto.Key("a"), nil, false)
	cq.GetWait(proto.Key("a"), nil, false, &wg)
	cmdDone := waitForCmd(&wg)
	if testCmdDone(cmdDone, 1*time.Millisecond) {
		t.Fatal("command should not finish with command outstanding")
	}
	cq.Remove(wk)
	if !testCmdDone(cmdDone, 5*time.Millisecond) {
		t.Fatal("command should finish with no commands outstanding")
	}
}

func TestCommandQueueNoWaitOnReadOnly(t *testing.T) {
	defer leaktest.AfterTest(t)
	cq := NewCommandQueue()
	wg := sync.WaitGroup{}
	// Add a read-only command.
	wk := cq.Add(proto.Key("a"), nil, true)
	// Verify no wait on another read-only command.
	cq.GetWait(proto.Key("a"), nil, true, &wg)
	wg.Wait()
	// Verify wait with a read-write command.
	cq.GetWait(proto.Key("a"), nil, false, &wg)
	cmdDone := waitForCmd(&wg)
	if testCmdDone(cmdDone, 1*time.Millisecond) {
		t.Fatal("command should not finish with command outstanding")
	}
	cq.Remove(wk)
	if !testCmdDone(cmdDone, 5*time.Millisecond) {
		t.Fatal("command should finish with no commands outstanding")
	}
}

func TestCommandQueueMultipleExecutingCommands(t *testing.T) {
	defer leaktest.AfterTest(t)
	cq := NewCommandQueue()
	wg := sync.WaitGroup{}

	// Add multiple commands and add a command which overlaps them all.
	wk1 := cq.Add(proto.Key("a"), nil, false)
	wk2 := cq.Add(proto.Key("b"), proto.Key("c"), false)
	wk3 := cq.Add(proto.Key("0"), proto.Key("d"), false)
	cq.GetWait(proto.Key("a"), proto.Key("cc"), false, &wg)
	cmdDone := waitForCmd(&wg)
	cq.Remove(wk1)
	if testCmdDone(cmdDone, 1*time.Millisecond) {
		t.Fatal("command should not finish with two commands outstanding")
	}
	cq.Remove(wk2)
	if testCmdDone(cmdDone, 1*time.Millisecond) {
		t.Fatal("command should not finish with one command outstanding")
	}
	cq.Remove(wk3)
	if !testCmdDone(cmdDone, 5*time.Millisecond) {
		t.Fatal("command should finish with no commands outstanding")
	}
}

func TestCommandQueueMultiplePendingCommands(t *testing.T) {
	defer leaktest.AfterTest(t)
	cq := NewCommandQueue()
	wg1 := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}
	wg3 := sync.WaitGroup{}

	// Add a command which will overlap all commands.
	wk := cq.Add(proto.Key("a"), proto.Key("d"), false)
	cq.GetWait(proto.Key("a"), nil, false, &wg1)
	cq.GetWait(proto.Key("b"), nil, false, &wg2)
	cq.GetWait(proto.Key("c"), nil, false, &wg3)
	cmdDone1 := waitForCmd(&wg1)
	cmdDone2 := waitForCmd(&wg2)
	cmdDone3 := waitForCmd(&wg3)

	if testCmdDone(cmdDone1, 1*time.Millisecond) ||
		testCmdDone(cmdDone2, 1*time.Millisecond) ||
		testCmdDone(cmdDone3, 1*time.Millisecond) {
		t.Fatal("no commands should finish with command outstanding")
	}
	cq.Remove(wk)
	if !testCmdDone(cmdDone1, 5*time.Millisecond) ||
		!testCmdDone(cmdDone2, 5*time.Millisecond) ||
		!testCmdDone(cmdDone3, 5*time.Millisecond) {
		t.Fatal("commands should finish with no commands outstanding")
	}
}

func TestCommandQueueClear(t *testing.T) {
	defer leaktest.AfterTest(t)
	cq := NewCommandQueue()
	wg1 := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}

	// Add multiple commands and commands which access each.
	cq.Add(proto.Key("a"), nil, false)
	cq.Add(proto.Key("b"), nil, false)
	cq.GetWait(proto.Key("a"), nil, false, &wg1)
	cq.GetWait(proto.Key("b"), nil, false, &wg2)
	cmdDone1 := waitForCmd(&wg1)
	cmdDone2 := waitForCmd(&wg2)

	// Clear the queue and verify both commands are signaled.
	cq.Clear()

	if !testCmdDone(cmdDone1, 100*time.Millisecond) ||
		!testCmdDone(cmdDone2, 100*time.Millisecond) {
		t.Fatal("commands should finish when clearing queue")
	}
}
