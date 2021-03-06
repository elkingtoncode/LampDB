// Copyright 2015 The Cockroach Authors.
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
// Author: Ben Darnell

package storage_test

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/cockroachdb/cockroach/gossip"
	"github.com/cockroachdb/cockroach/keys"
	"github.com/cockroachdb/cockroach/multiraft"
	"github.com/cockroachdb/cockroach/proto"
	"github.com/cockroachdb/cockroach/storage"
	"github.com/cockroachdb/cockroach/storage/engine"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/hlc"
	"github.com/cockroachdb/cockroach/util/leaktest"
	"github.com/cockroachdb/cockroach/util/log"
	"github.com/cockroachdb/cockroach/util/stop"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
)

// mustGetInteger decodes an int64 value from the bytes field of the receiver
// and panics if the bytes field is not 0 or 8 bytes in length.
func mustGetInteger(v *proto.Value) int64 {
	i, err := v.GetInteger()
	if err != nil {
		panic(err)
	}
	return i
}

// TestStoreRecoverFromEngine verifies that the store recovers all ranges and their contents
// after being stopped and recreated.
func TestStoreRecoverFromEngine(t *testing.T) {
	defer leaktest.AfterTest(t)
	rangeID := proto.RangeID(1)
	splitKey := proto.Key("m")
	key1 := proto.Key("a")
	key2 := proto.Key("z")

	manual := hlc.NewManualClock(0)
	clock := hlc.NewClock(manual.UnixNano)
	eng := engine.NewInMem(proto.Attributes{}, 1<<20)
	var rangeID2 proto.RangeID

	get := func(store *storage.Store, rangeID proto.RangeID, key proto.Key) int64 {
		args := getArgs(key, rangeID, store.StoreID())
		resp, err := store.ExecuteCmd(context.Background(), &args)
		if err != nil {
			t.Fatal(err)
		}
		return mustGetInteger(resp.(*proto.GetResponse).Value)
	}
	validate := func(store *storage.Store) {
		if val := get(store, rangeID, key1); val != 13 {
			t.Errorf("key %q: expected 13 but got %v", key1, val)
		}
		if val := get(store, rangeID2, key2); val != 28 {
			t.Errorf("key %q: expected 28 but got %v", key2, val)
		}
	}

	// First, populate the store with data across two ranges. Each range contains commands
	// that both predate and postdate the split.
	func() {
		store, stopper := createTestStoreWithEngine(t, eng, clock, true, nil)
		defer stopper.Stop()

		increment := func(rangeID proto.RangeID, key proto.Key, value int64) (*proto.IncrementResponse, error) {
			args := incrementArgs(key, value, rangeID, store.StoreID())
			resp, err := store.ExecuteCmd(context.Background(), &args)
			return resp.(*proto.IncrementResponse), err
		}

		if _, err := increment(rangeID, key1, 2); err != nil {
			t.Fatal(err)
		}
		if _, err := increment(rangeID, key2, 5); err != nil {
			t.Fatal(err)
		}
		splitArgs := adminSplitArgs(proto.KeyMin, splitKey, rangeID, store.StoreID())
		if _, err := store.ExecuteCmd(context.Background(), &splitArgs); err != nil {
			t.Fatal(err)
		}
		rangeID2 = store.LookupReplica(key2, nil).Desc().RangeID
		if rangeID2 == rangeID {
			t.Errorf("got same range id after split")
		}
		if _, err := increment(rangeID, key1, 11); err != nil {
			t.Fatal(err)
		}
		if _, err := increment(rangeID2, key2, 23); err != nil {
			t.Fatal(err)
		}
		validate(store)
	}()

	// Now create a new store with the same engine and make sure the expected data is present.
	// We must use the same clock because a newly-created manual clock will be behind the one
	// we wrote with and so will see stale MVCC data.
	store, stopper := createTestStoreWithEngine(t, eng, clock, false, nil)
	defer stopper.Stop()

	// Raft processing is initialized lazily; issue a no-op write request on each key to
	// ensure that is has been started.
	incArgs := incrementArgs(key1, 0, rangeID, store.StoreID())
	if _, err := store.ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}
	incArgs = incrementArgs(key2, 0, rangeID2, store.StoreID())
	if _, err := store.ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	validate(store)
}

// TestStoreRecoverWithErrors verifies that even commands that fail are marked as
// applied so they are not retried after recovery.
func TestStoreRecoverWithErrors(t *testing.T) {
	defer leaktest.AfterTest(t)
	defer func() { storage.TestingCommandFilter = nil }()
	manual := hlc.NewManualClock(0)
	clock := hlc.NewClock(manual.UnixNano)
	eng := engine.NewInMem(proto.Attributes{}, 1<<20)

	numIncrements := 0

	storage.TestingCommandFilter = func(args proto.Request) error {
		if _, ok := args.(*proto.IncrementRequest); ok && args.Header().Key.Equal(proto.Key("a")) {
			numIncrements++
		}
		return nil
	}

	func() {
		store, stopper := createTestStoreWithEngine(t, eng, clock, true, nil)
		defer stopper.Stop()

		// Write a bytes value so the increment will fail.
		putArgs := putArgs(proto.Key("a"), []byte("asdf"), 1, store.StoreID())
		if _, err := store.ExecuteCmd(context.Background(), &putArgs); err != nil {
			t.Fatal(err)
		}

		// Try and fail to increment the key. It is important for this test that the
		// failure be the last thing in the raft log when the store is stopped.
		incArgs := incrementArgs(proto.Key("a"), 42, 1, store.StoreID())
		if _, err := store.ExecuteCmd(context.Background(), &incArgs); err == nil {
			t.Fatal("did not get expected error")
		}
	}()

	if numIncrements != 1 {
		t.Fatalf("expected 1 increments; was %d", numIncrements)
	}

	// Recover from the engine.
	store, stopper := createTestStoreWithEngine(t, eng, clock, false, nil)
	defer stopper.Stop()

	// Issue a no-op write to lazily initialize raft on the range.
	incArgs := incrementArgs(proto.Key("b"), 0, 1, store.StoreID())
	if _, err := store.ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	// No additional increments were performed on key A during recovery.
	if numIncrements != 1 {
		t.Fatalf("expected 1 increments; was %d", numIncrements)
	}
}

// TestReplicateRange verifies basic replication functionality by creating two stores
// and a range, replicating the range to the second store, and reading its data there.
func TestReplicateRange(t *testing.T) {
	defer leaktest.AfterTest(t)
	mtc := startMultiTestContext(t, 2)
	defer mtc.Stop()

	// Issue a command on the first node before replicating.
	incArgs := incrementArgs([]byte("a"), 5, 1, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	rng, err := mtc.stores[0].GetReplica(1)
	if err != nil {
		t.Fatal(err)
	}

	if err := rng.ChangeReplicas(proto.ADD_REPLICA,
		proto.Replica{
			NodeID:  mtc.stores[1].Ident.NodeID,
			StoreID: mtc.stores[1].Ident.StoreID,
		}); err != nil {
		t.Fatal(err)
	}
	// Verify no intent remains on range descriptor key.
	key := keys.RangeDescriptorKey(rng.Desc().StartKey)
	desc := proto.RangeDescriptor{}
	if ok, err := engine.MVCCGetProto(mtc.stores[0].Engine(), key, mtc.stores[0].Clock().Now(), true, nil, &desc); !ok || err != nil {
		t.Fatalf("fetching range descriptor yielded %t, %s", ok, err)
	}
	// Verify that in time, no intents remain on meta addressing
	// keys, and that range descriptor on the meta records is correct.
	util.SucceedsWithin(t, 1*time.Second, func() error {
		meta2 := keys.RangeMetaKey(proto.KeyMax)
		meta1 := keys.RangeMetaKey(meta2)
		for _, key := range []proto.Key{meta2, meta1} {
			metaDesc := proto.RangeDescriptor{}
			if ok, err := engine.MVCCGetProto(mtc.stores[0].Engine(), key, mtc.stores[0].Clock().Now(), true, nil, &metaDesc); !ok || err != nil {
				return util.Errorf("failed to resolve %s", key)
			}
			if !reflect.DeepEqual(metaDesc, desc) {
				return util.Errorf("descs not equal: %+v != %+v", metaDesc, desc)
			}
		}
		return nil
	})

	// Verify that the same data is available on the replica.
	util.SucceedsWithin(t, 1*time.Second, func() error {
		getArgs := getArgs([]byte("a"), 1, mtc.stores[1].StoreID())
		getArgs.ReadConsistency = proto.INCONSISTENT
		if reply, err := mtc.stores[1].ExecuteCmd(context.Background(), &getArgs); err != nil {
			return util.Errorf("failed to read data")
		} else if v := mustGetInteger(reply.(*proto.GetResponse).Value); v != 5 {
			return util.Errorf("failed to read correct data: %d", v)
		}
		return nil
	})
}

// TestRestoreReplicas ensures that consensus group membership is properly
// persisted to disk and restored when a node is stopped and restarted.
func TestRestoreReplicas(t *testing.T) {
	defer leaktest.AfterTest(t)
	mtc := startMultiTestContext(t, 2)
	defer mtc.Stop()

	firstRng, err := mtc.stores[0].GetReplica(1)
	if err != nil {
		t.Fatal(err)
	}

	// Perform an increment before replication to ensure that commands are not
	// repeated on restarts.
	incArgs := incrementArgs([]byte("a"), 23, 1, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	if err := firstRng.ChangeReplicas(proto.ADD_REPLICA,
		proto.Replica{
			NodeID:  mtc.stores[1].Ident.NodeID,
			StoreID: mtc.stores[1].Ident.StoreID,
		}); err != nil {
		t.Fatal(err)
	}

	// TODO(bdarnell): use the stopper.Quiesce() method. The problem
	//   right now is that raft / multiraft isn't creating a task for
	//   high-level work it's creating while snapshotting and catching
	//   up. Ideally we'll be able to capture that and then can just
	//   invoke mtc.stopper.Quiesce() here.

	// TODO(bdarnell): initial creation and replication needs to be atomic;
	// cutting off the process too soon currently results in a corrupted range.
	time.Sleep(500 * time.Millisecond)

	mtc.restart()

	// Send a command on each store. The original store (the leader still)
	// will succeed.
	incArgs = incrementArgs([]byte("a"), 5, 1, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}
	// The follower will return a not leader error, indicating the command
	// should be forwarded to the leader.
	incArgs = incrementArgs([]byte("a"), 11, 1, mtc.stores[1].StoreID())
	_, err = mtc.stores[1].ExecuteCmd(context.Background(), &incArgs)
	if _, ok := err.(*proto.NotLeaderError); !ok {
		t.Fatalf("expected not leader error; got %s", err)
	}
	incArgs.Replica.StoreID = mtc.stores[0].StoreID()
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	if err := util.IsTrueWithin(func() bool {
		getArgs := getArgs([]byte("a"), 1, mtc.stores[1].StoreID())
		getArgs.ReadConsistency = proto.INCONSISTENT
		reply, err := mtc.stores[1].ExecuteCmd(context.Background(), &getArgs)
		if err != nil {
			return false
		}
		return mustGetInteger(reply.(*proto.GetResponse).Value) == 39
	}, 1*time.Second); err != nil {
		t.Fatal(err)
	}

	// Both replicas have a complete list in Desc.Replicas
	for i, store := range mtc.stores {
		rng, err := store.GetReplica(1)
		if err != nil {
			t.Fatal(err)
		}
		rng.RLock()
		if len(rng.Desc().Replicas) != 2 {
			t.Fatalf("store %d: expected 2 replicas, found %d", i, len(rng.Desc().Replicas))
		}
		if rng.Desc().Replicas[0].NodeID != mtc.stores[0].Ident.NodeID {
			t.Errorf("store %d: expected replica[0].NodeID == %d, was %d",
				i, mtc.stores[0].Ident.NodeID, rng.Desc().Replicas[0].NodeID)
		}
		rng.RUnlock()
	}
}

func TestFailedReplicaChange(t *testing.T) {
	defer leaktest.AfterTest(t)
	defer func() { storage.TestingCommandFilter = nil }()

	mtc := startMultiTestContext(t, 2)
	defer mtc.Stop()

	var runFilter atomic.Value
	runFilter.Store(true)

	storage.TestingCommandFilter = func(args proto.Request) error {
		if runFilter.Load().(bool) {
			if et, ok := args.(*proto.EndTransactionRequest); ok && et.Commit {
				return util.Errorf("boom")
			}
			return nil
		}
		return nil
	}

	rng, err := mtc.stores[0].GetReplica(1)
	if err != nil {
		t.Fatal(err)
	}

	err = rng.ChangeReplicas(proto.ADD_REPLICA,
		proto.Replica{
			NodeID:  mtc.stores[1].Ident.NodeID,
			StoreID: mtc.stores[1].Ident.StoreID,
		})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("did not get expected error: %s", err)
	}

	// After the aborted transaction, r.Desc was not updated.
	// TODO(bdarnell): expose and inspect raft's internal state.
	if len(rng.Desc().Replicas) != 1 {
		t.Fatalf("expected 1 replica, found %d", len(rng.Desc().Replicas))
	}

	// The pending config change flag was cleared, so a subsequent attempt
	// can succeed.
	runFilter.Store(false)

	err = rng.ChangeReplicas(proto.ADD_REPLICA,
		proto.Replica{
			NodeID:  mtc.stores[1].Ident.NodeID,
			StoreID: mtc.stores[1].Ident.StoreID,
		})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the range to sync to both replicas (mainly so leaktest doesn't
	// complain about goroutines involved in the process).
	if err := util.IsTrueWithin(func() bool {
		for _, store := range mtc.stores {
			rang, err := store.GetReplica(1)
			if err != nil {
				return false
			}
			if len(rang.Desc().Replicas) == 1 {
				return false
			}
		}
		return true
	}, 1*time.Second); err != nil {
		t.Fatal(err)
	}
}

// We can truncate the old log entries and a new replica will be brought up from a snapshot.
func TestReplicateAfterTruncation(t *testing.T) {
	defer leaktest.AfterTest(t)
	mtc := startMultiTestContext(t, 2)
	defer mtc.Stop()

	rng, err := mtc.stores[0].GetReplica(1)
	if err != nil {
		t.Fatal(err)
	}

	// Issue a command on the first node before replicating.
	incArgs := incrementArgs([]byte("a"), 5, 1, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	// Get that command's log index.
	index, err := rng.LastIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Truncate the log at index+1 (log entries < N are removed, so this includes
	// the increment).
	truncArgs := truncateLogArgs(index+1, 1, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &truncArgs); err != nil {
		t.Fatal(err)
	}

	// Issue a second command post-truncation.
	incArgs = incrementArgs([]byte("a"), 11, 1, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}
	mvcc := rng.GetMVCCStats()

	// Now add the second replica.
	if err := rng.ChangeReplicas(proto.ADD_REPLICA,
		proto.Replica{
			NodeID:  mtc.stores[1].Ident.NodeID,
			StoreID: mtc.stores[1].Ident.StoreID,
		}); err != nil {
		t.Fatal(err)
	}

	// Once it catches up, the effects of both commands can be seen.
	if err := util.IsTrueWithin(func() bool {
		getArgs := getArgs([]byte("a"), 1, mtc.stores[1].StoreID())
		getArgs.ReadConsistency = proto.INCONSISTENT
		reply, err := mtc.stores[1].ExecuteCmd(context.Background(), &getArgs)
		if err != nil {
			return false
		}
		getResp := reply.(*proto.GetResponse)
		if log.V(1) {
			log.Infof("read value %d", mustGetInteger(getResp.Value))
		}
		return mustGetInteger(getResp.Value) == 16
	}, 1*time.Second); err != nil {
		t.Fatal(err)
	}

	rng2, err := mtc.stores[1].GetReplica(1)
	if err != nil {
		t.Fatal(err)
	}
	if mvcc2 := rng2.GetMVCCStats(); !reflect.DeepEqual(mvcc, mvcc2) {
		log.Errorf("expected stats on new range to equal old; %+v != %+v", mvcc2, mvcc)
	}

	// Send a third command to verify that the log states are synced up so the
	// new node can accept new commands.
	incArgs = incrementArgs([]byte("a"), 23, 1, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	if err := util.IsTrueWithin(func() bool {
		getArgs := getArgs([]byte("a"), 1, mtc.stores[1].StoreID())
		getArgs.ReadConsistency = proto.INCONSISTENT
		reply, err := mtc.stores[1].ExecuteCmd(context.Background(), &getArgs)
		if err != nil {
			return false
		}
		getResp := reply.(*proto.GetResponse)
		log.Infof("read value %d", mustGetInteger(getResp.Value))
		return mustGetInteger(getResp.Value) == 39
	}, 1*time.Second); err != nil {
		t.Fatal(err)
	}
}

// TestStoreRangeReplicate verifies that the replication queue will notice
// under-replicated ranges and replicate them.
func TestStoreRangeReplicate(t *testing.T) {
	defer leaktest.AfterTest(t)
	mtc := startMultiTestContext(t, 3)
	defer mtc.Stop()

	// Initialize the gossip network.
	var wg sync.WaitGroup
	wg.Add(len(mtc.stores))
	key := gossip.MakePrefixPattern(gossip.KeyCapacityPrefix)
	mtc.stores[0].Gossip().RegisterCallback(key, func(_ string, _ bool) { wg.Done() })
	for _, s := range mtc.stores {
		s.GossipCapacity()
	}
	wg.Wait()

	// Once we know our peers, trigger a scan.
	mtc.stores[0].ForceReplicationScan(t)

	// The range should become available on every node.
	if err := util.IsTrueWithin(func() bool {
		for _, s := range mtc.stores {
			r := s.LookupReplica(proto.Key("a"), proto.Key("b"))
			if r == nil {
				return false
			}
		}
		return true
	}, 1*time.Second); err != nil {
		t.Fatal(err)
	}
}

// TestProgressWithDownNode verifies that a surviving quorum can make progress
// with a downed node.
func TestProgressWithDownNode(t *testing.T) {
	defer leaktest.AfterTest(t)
	mtc := startMultiTestContext(t, 3)
	defer mtc.Stop()

	rangeID := proto.RangeID(1)
	mtc.replicateRange(rangeID, 0, 1, 2)

	incArgs := incrementArgs([]byte("a"), 5, rangeID, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	// Verify that the first increment propagates to all the engines.
	verify := func(expected []int64) {
		util.SucceedsWithin(t, time.Second, func() error {
			values := []int64{}
			for _, eng := range mtc.engines {
				val, _, err := engine.MVCCGet(eng, proto.Key("a"), mtc.clock.Now(), true, nil)
				if err != nil {
					return err
				}
				values = append(values, mustGetInteger(val))
			}
			if !reflect.DeepEqual(expected, values) {
				return util.Errorf("expected %v, got %v", expected, values)
			}
			return nil
		})
	}
	verify([]int64{5, 5, 5})

	// Stop one of the replicas and issue a new increment.
	mtc.stopStore(1)
	incArgs = incrementArgs([]byte("a"), 11, rangeID, mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}

	// The new increment can be seen on both live replicas.
	verify([]int64{16, 5, 16})

	// Once the downed node is restarted, it will catch up.
	mtc.restartStore(1)
	verify([]int64{16, 16, 16})
}

func TestReplicateAddAndRemove(t *testing.T) {
	defer leaktest.AfterTest(t)

	// Run the test twice, once adding the replacement before removing
	// the downed node, and once removing the downed node first.
	for _, addFirst := range []bool{true, false} {
		mtc := startMultiTestContext(t, 4)
		defer mtc.Stop()

		// Replicate the initial range to three of the four nodes.
		rangeID := proto.RangeID(1)
		mtc.replicateRange(rangeID, 0, 3, 1)

		incArgs := incrementArgs([]byte("a"), 5, rangeID, mtc.stores[0].StoreID())
		if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
			t.Fatal(err)
		}

		verify := func(expected []int64) {
			util.SucceedsWithin(t, time.Second, func() error {
				values := []int64{}
				for _, eng := range mtc.engines {
					val, _, err := engine.MVCCGet(eng, proto.Key("a"), mtc.clock.Now(), true, nil)
					if err != nil {
						return err
					}
					values = append(values, mustGetInteger(val))
				}
				if !reflect.DeepEqual(expected, values) {
					return util.Errorf("addFirst: %t, expected %v, got %v", addFirst, expected, values)
				}
				return nil
			})
		}

		// The first increment is visible on all three replicas.
		verify([]int64{5, 5, 0, 5})

		// Stop a store and replace it.
		mtc.stopStore(1)
		if addFirst {
			mtc.replicateRange(rangeID, 0, 2)
			mtc.unreplicateRange(rangeID, 0, 1)
		} else {
			mtc.unreplicateRange(rangeID, 0, 1)
			mtc.replicateRange(rangeID, 0, 2)
		}
		verify([]int64{5, 5, 5, 5})

		// Ensure that the rest of the group can make progress.
		incArgs = incrementArgs([]byte("a"), 11, rangeID, mtc.stores[0].StoreID())
		if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
			t.Fatal(err)
		}
		verify([]int64{16, 5, 16, 16})

		// Bring the downed store back up (required for a clean shutdown).
		mtc.restartStore(1)

		// Node 1 never sees the increment that was added while it was
		// down. Perform another increment on the live nodes to verify.
		incArgs = incrementArgs([]byte("a"), 23, rangeID, mtc.stores[0].StoreID())
		if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &incArgs); err != nil {
			t.Fatal(err)
		}
		verify([]int64{39, 5, 39, 39})

		// Wait out the leader lease and the unleased duration to make the range GC'able.
		mtc.manualClock.Increment(int64(storage.RangeGCQueueInactivityThreshold+storage.DefaultLeaderLeaseDuration) + 1)
		mtc.stores[1].ForceRangeGCScan(t)

		// The removed store no longer has any of the data from the range.
		verify([]int64{39, 0, 39, 39})

		desc := mtc.stores[0].LookupReplica(proto.KeyMin, nil).Desc()
		replicaIDsByStore := map[proto.StoreID]proto.ReplicaID{}
		for _, rep := range desc.Replicas {
			replicaIDsByStore[rep.StoreID] = rep.ReplicaID
		}
		expected := map[proto.StoreID]proto.ReplicaID{1: 1, 4: 2, 3: 4}
		if !reflect.DeepEqual(expected, replicaIDsByStore) {
			t.Fatalf("expected replica IDs to be %v but got %v", expected, replicaIDsByStore)
		}
	}
}

// TestRaftHeartbeats verifies that coalesced heartbeats are correctly
// suppressing elections in an idle cluster.
func TestRaftHeartbeats(t *testing.T) {
	defer leaktest.AfterTest(t)

	mtc := startMultiTestContext(t, 3)
	defer mtc.Stop()
	mtc.replicateRange(1, 0, 1, 2)

	// Capture the initial term and state.
	status := mtc.stores[0].RaftStatus(1)
	initialTerm := status.Term
	if status.SoftState.RaftState != raft.StateLeader {
		t.Errorf("expected node 0 to initially be leader but was %s", status.SoftState.RaftState)
	}

	// Wait for several ticks to elapse.
	time.Sleep(5 * mtc.makeContext(0).RaftTickInterval)
	status = mtc.stores[0].RaftStatus(1)
	if status.SoftState.RaftState != raft.StateLeader {
		t.Errorf("expected node 0 to be leader after sleeping but was %s", status.SoftState.RaftState)
	}
	if status.Term != initialTerm {
		t.Errorf("while sleeping, term changed from %d to %d", initialTerm, status.Term)
	}
}

// TestReplicateAfterSplit verifies that a new replica whose start key
// is not KeyMin replicating to a fresh store can apply snapshots correctly.
func TestReplicateAfterSplit(t *testing.T) {
	defer leaktest.AfterTest(t)
	mtc := startMultiTestContext(t, 2)
	defer mtc.Stop()

	rangeID := proto.RangeID(1)
	splitKey := proto.Key("m")
	key := proto.Key("z")

	store0 := mtc.stores[0]
	// Make the split
	splitArgs := adminSplitArgs(proto.KeyMin, splitKey, rangeID, store0.StoreID())
	if _, err := store0.ExecuteCmd(context.Background(), &splitArgs); err != nil {
		t.Fatal(err)
	}

	rangeID2 := store0.LookupReplica(key, nil).Desc().RangeID
	if rangeID2 == rangeID {
		t.Errorf("got same range id after split")
	}
	// Issue an increment for later check.
	incArgs := incrementArgs(key, 11, rangeID2, store0.StoreID())
	if _, err := store0.ExecuteCmd(context.Background(), &incArgs); err != nil {
		t.Fatal(err)
	}
	// Now add the second replica.
	mtc.replicateRange(rangeID2, 0, 1)

	if mtc.stores[1].LookupReplica(key, nil).GetMaxBytes() == 0 {
		t.Error("Range MaxBytes is not set after snapshot applied")
	}
	// Once it catches up, the effects of increment commands can be seen.
	if err := util.IsTrueWithin(func() bool {
		getArgs := getArgs(key, rangeID2, mtc.stores[1].StoreID())
		// Reading on non-leader replica should use inconsistent read
		getArgs.ReadConsistency = proto.INCONSISTENT
		reply, err := mtc.stores[1].ExecuteCmd(context.Background(), &getArgs)
		if err != nil {
			return false
		}
		getResp := reply.(*proto.GetResponse)
		if log.V(1) {
			log.Infof("read value %d", mustGetInteger(getResp.Value))
		}
		return mustGetInteger(getResp.Value) == 11
	}, 1*time.Second); err != nil {
		t.Fatal(err)
	}
}

// TestRangeDescriptorSnapshotRace calls Snapshot() repeatedly while
// transactions are performed on the range descriptor.
func TestRangeDescriptorSnapshotRace(t *testing.T) {
	defer leaktest.AfterTest(t)

	mtc := startMultiTestContext(t, 1)
	defer mtc.Stop()

	stopper := stop.NewStopper()
	defer stopper.Stop()
	// Call Snapshot() in a loop and ensure it never fails.
	stopper.RunWorker(func() {
		for {
			select {
			case <-stopper.ShouldStop():
				return
			default:
				rng := mtc.stores[0].LookupReplica(proto.KeyMin, nil)
				if rng == nil {
					t.Fatal("failed to look up min range")
				}
				_, err := rng.Snapshot()
				if err != nil {
					t.Fatalf("failed to snapshot min range: %s", err)
				}

				rng = mtc.stores[0].LookupReplica(proto.Key("Z"), nil)
				if rng == nil {
					t.Fatal("failed to look up max range")
				}
				_, err = rng.Snapshot()
				if err != nil {
					t.Fatalf("failed to snapshot max range: %s", err)
				}
			}
		}
	})

	// Split the range repeatedly, carving chunks off the end of the
	// initial range.  The bug that this test was designed to find
	// usually occurred within the first 5 iterations.
	for i := 20; i > 0; i-- {
		rng := mtc.stores[0].LookupReplica(proto.KeyMin, nil)
		if rng == nil {
			t.Fatal("failed to look up min range")
		}
		args := adminSplitArgs(proto.KeyMin, []byte(fmt.Sprintf("A%03d", i)), rng.Desc().RangeID,
			mtc.stores[0].StoreID())
		if _, err := rng.AdminSplit(args); err != nil {
			t.Fatal(err)
		}
	}

	// Split again, carving chunks off the beginning of the final range.
	for i := 0; i < 20; i++ {
		rng := mtc.stores[0].LookupReplica(proto.Key("Z"), nil)
		if rng == nil {
			t.Fatal("failed to look up max range")
		}
		args := adminSplitArgs(proto.KeyMin, []byte(fmt.Sprintf("B%03d", i)), rng.Desc().RangeID, mtc.stores[0].StoreID())
		if _, err := rng.AdminSplit(args); err != nil {
			t.Fatal(err)
		}
	}
}

// TestRaftAfterRemoveRange verifies that the MultiRaft state removes
// a remote node correctly after the Replica was removed from the Store.
func TestRaftAfterRemoveRange(t *testing.T) {
	defer leaktest.AfterTest(t)
	mtc := startMultiTestContext(t, 3)
	defer mtc.Stop()

	// Make the split.
	splitArgs := adminSplitArgs(proto.KeyMin, []byte("b"), proto.RangeID(1), mtc.stores[0].StoreID())
	if _, err := mtc.stores[0].ExecuteCmd(context.Background(), &splitArgs); err != nil {
		t.Fatal(err)
	}

	rangeID := proto.RangeID(2)
	mtc.replicateRange(rangeID, 0, 1, 2)

	mtc.unreplicateRange(rangeID, 0, 2)
	mtc.unreplicateRange(rangeID, 0, 1)

	// Wait for the removal to be processed.
	util.SucceedsWithin(t, time.Second, func() error {
		_, err := mtc.stores[1].GetReplica(rangeID)
		if _, ok := err.(*proto.RangeNotFoundError); ok {
			return nil
		} else if err != nil {
			return err
		}
		return util.Errorf("range still exists")
	})

	if err := mtc.transport.Send(&multiraft.RaftMessageRequest{
		GroupID: proto.RangeID(0),
		Message: raftpb.Message{
			From: uint64(mtc.stores[2].RaftNodeID()),
			To:   uint64(mtc.stores[1].RaftNodeID()),
			Type: raftpb.MsgHeartbeat,
		}}); err != nil {
		t.Fatal(err)
	}
	// Execute another replica change to ensure that MultiRaft has processed the heartbeat just sent.
	mtc.replicateRange(proto.RangeID(1), 0, 1)
}
