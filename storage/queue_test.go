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
	"container/heap"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cockroachdb/cockroach/proto"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/hlc"
	"github.com/cockroachdb/cockroach/util/leaktest"
	"github.com/cockroachdb/cockroach/util/stop"
)

// testQueueImpl implements queueImpl with a closure for shouldQueue.
type testQueueImpl struct {
	shouldQueueFn func(proto.Timestamp, *Replica) (bool, float64)
	processed     int32
	duration      time.Duration
	blocker       chan struct{} // timer() blocks on this if not nil
}

func (tq *testQueueImpl) needsLeaderLease() bool { return false }

func (tq *testQueueImpl) shouldQueue(now proto.Timestamp, r *Replica) (bool, float64) {
	return tq.shouldQueueFn(now, r)
}

func (tq *testQueueImpl) process(now proto.Timestamp, r *Replica) error {
	atomic.AddInt32(&tq.processed, 1)
	return nil
}

func (tq *testQueueImpl) timer() time.Duration {
	if tq.blocker != nil {
		<-tq.blocker
	}
	if tq.duration != 0 {
		return tq.duration
	}
	return 0
}

// TestQueuePriorityQueue verifies priority queue implementation.
func TestQueuePriorityQueue(t *testing.T) {
	defer leaktest.AfterTest(t)
	// Create a priority queue, put the items in it, and
	// establish the priority queue (heap) invariants.
	const count = 3
	expRanges := make([]*Replica, count+1)
	pq := make(priorityQueue, count)
	for i := 0; i < count; {
		pq[i] = &rangeItem{
			value:    &Replica{},
			priority: float64(i),
			index:    i,
		}
		expRanges[3-i] = pq[i].value
		i++
	}
	heap.Init(&pq)

	// Insert a new item and then modify its priority.
	priorityItem := &rangeItem{
		value:    &Replica{},
		priority: 1.0,
	}
	heap.Push(&pq, priorityItem)
	pq.update(priorityItem, 4.0)
	expRanges[0] = priorityItem.value

	// Take the items out; they should arrive in decreasing priority order.
	for i := 0; pq.Len() > 0; i++ {
		item := heap.Pop(&pq).(*rangeItem)
		if item.value != expRanges[i] {
			t.Errorf("%d: unexpected range with priority %f", i, item.priority)
		}
	}
}

// TestBaseQueueAddUpdateAndRemove verifies basic operation with base
// queue including adding ranges which both should and shouldn't be
// queued, updating an existing range, and removing a range.
func TestBaseQueueAddUpdateAndRemove(t *testing.T) {
	defer leaktest.AfterTest(t)
	r1 := &Replica{}
	if err := r1.setDesc(&proto.RangeDescriptor{RangeID: 1}); err != nil {
		t.Fatal(err)
	}
	r2 := &Replica{}
	if err := r2.setDesc(&proto.RangeDescriptor{RangeID: 2}); err != nil {
		t.Fatal(err)
	}
	shouldAddMap := map[*Replica]bool{
		r1: true,
		r2: true,
	}
	priorityMap := map[*Replica]float64{
		r1: 1.0,
		r2: 2.0,
	}
	testQueue := &testQueueImpl{
		shouldQueueFn: func(now proto.Timestamp, r *Replica) (shouldQueue bool, priority float64) {
			return shouldAddMap[r], priorityMap[r]
		},
	}
	bq := newBaseQueue("test", testQueue, 2)
	bq.MaybeAdd(r1, proto.ZeroTimestamp)
	bq.MaybeAdd(r2, proto.ZeroTimestamp)
	if bq.Length() != 2 {
		t.Fatalf("expected length 2; got %d", bq.Length())
	}
	if bq.pop() != r2 {
		t.Error("expected r2")
	}
	if bq.pop() != r1 {
		t.Error("expected r1")
	}
	if r := bq.pop(); r != nil {
		t.Errorf("expected empty queue; got %v", r)
	}

	// Add again, but this time r2 shouldn't add.
	shouldAddMap[r2] = false
	bq.MaybeAdd(r1, proto.ZeroTimestamp)
	bq.MaybeAdd(r2, proto.ZeroTimestamp)
	if bq.Length() != 1 {
		t.Errorf("expected length 1; got %d", bq.Length())
	}

	// Try adding same range twice.
	bq.MaybeAdd(r1, proto.ZeroTimestamp)
	if bq.Length() != 1 {
		t.Errorf("expected length 1; got %d", bq.Length())
	}

	// Re-add r2 and update priority of r1.
	shouldAddMap[r2] = true
	priorityMap[r1] = 3.0
	bq.MaybeAdd(r1, proto.ZeroTimestamp)
	bq.MaybeAdd(r2, proto.ZeroTimestamp)
	if bq.Length() != 2 {
		t.Fatalf("expected length 2; got %d", bq.Length())
	}
	if bq.pop() != r1 {
		t.Error("expected r1")
	}
	if bq.pop() != r2 {
		t.Error("expected r2")
	}
	if r := bq.pop(); r != nil {
		t.Errorf("expected empty queue; got %v", r)
	}

	// Set !shouldAdd for r2 and add it; this has effect of removing it.
	bq.MaybeAdd(r1, proto.ZeroTimestamp)
	bq.MaybeAdd(r2, proto.ZeroTimestamp)
	shouldAddMap[r2] = false
	bq.MaybeAdd(r2, proto.ZeroTimestamp)
	if bq.Length() != 1 {
		t.Fatalf("expected length 1; got %d", bq.Length())
	}
	if bq.pop() != r1 {
		t.Errorf("expected r1")
	}
}

// TestBaseQueueAdd verifies that calling Add() directly overrides the
// ShouldQueue method.
func TestBaseQueueAdd(t *testing.T) {
	defer leaktest.AfterTest(t)
	r := &Replica{}
	if err := r.setDesc(&proto.RangeDescriptor{RangeID: 1}); err != nil {
		t.Fatal(err)
	}
	testQueue := &testQueueImpl{
		shouldQueueFn: func(now proto.Timestamp, r *Replica) (shouldQueue bool, priority float64) {
			return false, 0.0
		},
	}
	bq := newBaseQueue("test", testQueue, 1)
	bq.MaybeAdd(r, proto.ZeroTimestamp)
	if bq.Length() != 0 {
		t.Fatalf("expected length 0; got %d", bq.Length())
	}
	if err := bq.Add(r, 1.0); err != nil {
		t.Fatalf("expected Add to succeed: %s", err)
	}
	if bq.Length() != 1 {
		t.Fatalf("expected length 1; got %d", bq.Length())
	}
}

// TestBaseQueueProcess verifies that items from the queue are
// processed according to the timer function.
func TestBaseQueueProcess(t *testing.T) {
	defer leaktest.AfterTest(t)
	r1 := &Replica{}
	if err := r1.setDesc(&proto.RangeDescriptor{RangeID: 1}); err != nil {
		t.Fatal(err)
	}
	r2 := &Replica{}
	if err := r2.setDesc(&proto.RangeDescriptor{RangeID: 2}); err != nil {
		t.Fatal(err)
	}
	testQueue := &testQueueImpl{
		blocker: make(chan struct{}, 1),
		shouldQueueFn: func(now proto.Timestamp, r *Replica) (shouldQueue bool, priority float64) {
			shouldQueue = true
			priority = float64(r.Desc().RangeID)
			return
		},
	}
	bq := newBaseQueue("test", testQueue, 2)
	stopper := stop.NewStopper()
	mc := hlc.NewManualClock(0)
	clock := hlc.NewClock(mc.UnixNano)
	bq.Start(clock, stopper)
	defer stopper.Stop()

	bq.MaybeAdd(r1, proto.ZeroTimestamp)
	bq.MaybeAdd(r2, proto.ZeroTimestamp)
	if pc := atomic.LoadInt32(&testQueue.processed); pc != 0 {
		t.Errorf("expected no processed ranges; got %d", pc)
	}

	testQueue.blocker <- struct{}{}
	if err := util.IsTrueWithin(func() bool {
		return atomic.LoadInt32(&testQueue.processed) == 1
	}, 250*time.Millisecond); err != nil {
		t.Error(err)
	}

	testQueue.blocker <- struct{}{}
	if err := util.IsTrueWithin(func() bool {
		return atomic.LoadInt32(&testQueue.processed) >= 2
	}, 250*time.Millisecond); err != nil {
		t.Error(err)
	}
}

// TestBaseQueueAddRemove adds then removes a range; ensure range is
// not processed.
func TestBaseQueueAddRemove(t *testing.T) {
	defer leaktest.AfterTest(t)
	r := &Replica{}
	if err := r.setDesc(&proto.RangeDescriptor{RangeID: 1}); err != nil {
		t.Fatal(err)
	}
	testQueue := &testQueueImpl{
		blocker: make(chan struct{}, 1),
		shouldQueueFn: func(now proto.Timestamp, r *Replica) (shouldQueue bool, priority float64) {
			shouldQueue = true
			priority = 1.0
			return
		},
	}
	bq := newBaseQueue("test", testQueue, 2)
	stopper := stop.NewStopper()
	mc := hlc.NewManualClock(0)
	clock := hlc.NewClock(mc.UnixNano)
	bq.Start(clock, stopper)
	defer stopper.Stop()

	bq.MaybeAdd(r, proto.ZeroTimestamp)
	bq.MaybeRemove(r)

	// Wake the queue
	testQueue.blocker <- struct{}{}

	// Make sure the queue has actually run through a few times
	for i := 0; i < cap(bq.incoming)+1; i++ {
		bq.incoming <- struct{}{}
		testQueue.blocker <- struct{}{}
	}

	if pc := atomic.LoadInt32(&testQueue.processed); pc > 0 {
		t.Errorf("expected processed count of 0; got %d", pc)
	}
}
