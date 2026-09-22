package wsjtx

import (
	"bytes"
	"encoding/binary"
	"net"
	"reflect"
	"testing"
	"time"

	wsjtx "github.com/k0swe/wsjtx-go/v4"
)

// closeServerSocket closes the UDP socket by reaching into the library's
// unexported "conn" field, because wsjtx-go v4 exposes no Shutdown. A
// recover() there swallows any breakage, so an upgrade that renames or
// retypes the field would silently leave the port held. Fail loudly here
// instead.
func TestServerConnFieldStillMatchesShutdownAssumption(t *testing.T) {
	f, ok := reflect.TypeOf(wsjtx.Server{}).FieldByName("conn")
	if !ok {
		t.Fatal("wsjtx.Server has no \"conn\" field — Listener.closeServerSocket can no longer close the UDP socket")
	}
	if want := reflect.TypeOf((*net.UDPConn)(nil)); f.Type != want {
		t.Fatalf("wsjtx.Server.conn is %s, want %s — closeServerSocket casts to **net.UDPConn", f.Type, want)
	}
}

// =============================================================================
// Listener lifecycle tests — no real UDP / WSJT-X required
// =============================================================================
//
// Start() calls applog which requires initialization. These tests focus on
// the lifecycle state machine (NewListener, Stop, IsActive) which are safe
// without a running application.

func TestNewListener(t *testing.T) {
	l := NewListener()
	if l == nil {
		t.Fatal("NewListener returned nil")
	}
	if l.IsActive() {
		t.Error("new listener should not be active")
	}
}

func TestStopIdempotent(t *testing.T) {
	l := NewListener()
	// Double stop should not panic.
	l.Stop()
	l.Stop()
	if l.IsActive() {
		t.Error("listener should not be active after Stop")
	}
}

func TestStopBeforeStart(t *testing.T) {
	l := NewListener()
	// Stop before Start should be a no-op.
	l.Stop()
	l.Stop()
	if l.IsActive() {
		t.Error("Stop before Start should not activate the listener")
	}
}

func TestListenerMutexSafety(t *testing.T) {
	// Concurrent Stop calls should not race.
	l := NewListener()
	done := make(chan struct{})
	go func() {
		l.Stop()
		done <- struct{}{}
	}()
	go func() {
		l.Stop()
		done <- struct{}{}
	}()
	<-done
	<-done
	// If we get here without a race detector failure, the mutex is working.
}

func TestIsActiveInitialState(t *testing.T) {
	l := NewListener()
	if l.IsActive() {
		t.Error("IsActive should be false for new listener")
	}
	l.Stop()
	if l.IsActive() {
		t.Error("IsActive should be false after Stop on fresh listener")
	}
}

func TestCallbacksInitiallyNil(t *testing.T) {
	l := NewListener()
	if l.OnADIF != nil {
		t.Error("OnADIF callback should be nil on new listener")
	}
	if l.OnStatus != nil {
		t.Error("OnStatus callback should be nil on new listener")
	}
}

// =============================================================================
// Generation guard tests
// =============================================================================

func TestSnapshotOnADIF_RejectsStaleGeneration(t *testing.T) {
	l := NewListener()
	l.OnADIF = func(s string) {}

	// Simulate what Start() does: bump generation, snapshot it.
	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen := l.generation
	l.active = true
	l.stop = make(chan struct{}) // normally created by Start()
	l.mu.Unlock()

	// Current generation should return the callback.
	if cb := l.snapshotOnADIF(gen); cb == nil {
		t.Error("snapshotOnADIF returned nil for current generation")
	}

	// Simulate restart: deactivate and bump generation.
	l.mu.Lock()
	l.active = false
	l.generation++ // new generation starts, old gen is stale
	l.mu.Unlock()

	// Now the stale generation should be rejected.
	if cb := l.snapshotOnADIF(gen); cb != nil {
		t.Error("snapshotOnADIF returned non-nil for stale (stopped) generation")
	}
}

func TestSnapshotOnStatus_RejectsStaleGeneration(t *testing.T) {
	l := NewListener()
	l.OnStatus = func(call, grid string, freq uint64, mode, submode, report, txMessage string, transmitting bool) {}

	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen := l.generation
	l.active = true
	l.stop = make(chan struct{})
	l.mu.Unlock()

	if cb := l.snapshotOnStatus(gen); cb == nil {
		t.Error("snapshotOnStatus returned nil for current generation")
	}

	l.mu.Lock()
	l.active = false
	l.generation++
	l.mu.Unlock()

	if cb := l.snapshotOnStatus(gen); cb != nil {
		t.Error("snapshotOnStatus returned non-nil for stale (stopped) generation")
	}
}

func TestSnapshotOnADIF_RejectsWhenInactive(t *testing.T) {
	l := NewListener()
	l.OnADIF = func(s string) {}

	// Never activated — generation 0, not active.
	if cb := l.snapshotOnADIF(0); cb != nil {
		t.Error("snapshotOnADIF returned non-nil when listener is not active")
	}
}

func TestGenerationIncrements(t *testing.T) {
	l := NewListener()
	l.OnADIF = func(s string) {}

	// Simulate two starts.
	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen1 := l.generation
	l.mu.Unlock()

	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen2 := l.generation
	l.active = true
	l.stop = make(chan struct{})
	l.mu.Unlock()

	if gen2 != gen1+1 {
		t.Errorf("generation did not increment: gen1=%d gen2=%d", gen1, gen2)
	}

	// gen1 should now be stale.
	if cb := l.snapshotOnADIF(gen1); cb != nil {
		t.Error("snapshotOnADIF returned non-nil for gen1 after gen2 was created")
	}
	if cb := l.snapshotOnADIF(gen2); cb == nil {
		t.Error("snapshotOnADIF returned nil for current gen2")
	}
}

func TestConcurrentSnapshotAndStop(t *testing.T) {
	// Concurrent deactivation and snapshot should not race.
	l := NewListener()
	l.OnADIF = func(s string) {}

	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen := l.generation
	l.active = true
	l.stop = make(chan struct{})
	l.mu.Unlock()

	done := make(chan struct{}, 2)
	go func() {
		l.snapshotOnADIF(gen)
		done <- struct{}{}
	}()
	go func() {
		l.mu.Lock()
		l.active = false
		l.generation++
		l.mu.Unlock()
		done <- struct{}{}
	}()
	<-done
	<-done
	// Race detector will catch any issues.
}

// =============================================================================
// Shutdown deadlock regression tests
// =============================================================================

// TestStopCompletesWhileLoopWaitsForMutex reproduces the reported shutdown
// deadlock shape: the event loop blocks in snapshotOnADIF waiting for l.mu
// while shutdown wants to join it. Stop must wait OUTSIDE the mutex, so
// releasing the mutex lets both the snapshot and the join complete.
func TestStopCompletesWhileLoopWaitsForMutex(t *testing.T) {
	l := NewListener()
	l.OnADIF = func(s string) {}
	msgCh := make(chan interface{}, 4)
	errCh := make(chan error, 16)

	l.mu.Lock()
	l.active = true
	l.generation = 1
	stop := make(chan struct{})
	l.stop = stop
	l.loopWG.Add(1)
	go func() { defer l.loopWG.Done(); l.eventLoop(1, stop, msgCh, errCh) }()
	l.mu.Unlock()

	// Deliver a message while holding the mutex so the event loop blocks
	// inside snapshotOnADIF.
	l.mu.Lock()
	msgCh <- wsjtx.LoggedAdifMessage{Adif: "X"}
	time.Sleep(20 * time.Millisecond) // let the loop reach the snapshot

	stopDone := make(chan struct{})
	go func() { l.Stop(); close(stopDone) }()

	// Stop cannot finish while the loop's snapshot is blocked on the mutex;
	// it must be waiting outside the mutex, not holding it.
	select {
	case <-stopDone:
		t.Fatal("Stop finished while the event loop was still blocked on the mutex")
	case <-time.After(100 * time.Millisecond):
	}

	l.mu.Unlock()

	select {
	case <-stopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop deadlocked: it waited for the event loop while the loop needed the mutex")
	}
}

// TestStopDoesNotDeadlockWithPendingMessages is the shutdown-deadlock
// regression test: Stop used to hold l.mu while waiting for the event loop,
// and the loop takes l.mu in its callback snapshots. With the message queue
// kept busy, that interleaving was reachable in production. Stop must now
// always return promptly.
func TestStopDoesNotDeadlockWithPendingMessages(t *testing.T) {
	for i := 0; i < 200; i++ {
		l := NewListener()
		l.OnADIF = func(s string) {}
		msgCh := make(chan interface{}, 16)
		errCh := make(chan error, 16)

		l.mu.Lock()
		l.active = true
		l.generation = 1
		stop := make(chan struct{})
		l.stop = stop
		l.loopWG.Add(1)
		go func() { defer l.loopWG.Done(); l.eventLoop(1, stop, msgCh, errCh) }()
		l.mu.Unlock()

		// Keep the queue busy so the loop frequently tries to snapshot
		// while Stop runs — the exact interleaving that deadlocked shutdown.
		stopFlood := make(chan struct{})
		go func() {
			msg := wsjtx.LoggedAdifMessage{Adif: "X"}
			for {
				select {
				case msgCh <- msg:
				case <-stopFlood:
					return
				}
			}
		}()

		stopDone := make(chan struct{})
		go func() { l.Stop(); close(stopDone) }()
		select {
		case <-stopDone:
		case <-time.After(2 * time.Second):
			close(stopFlood)
			t.Fatalf("iteration %d: Stop deadlocked while messages were pending", i)
		}
		close(stopFlood)
	}
}

// =============================================================================
// Reader shutdown with saturated queues
// =============================================================================
//
// ListenToWsjtx sends to msgCh/errCh with BLOCKING sends. Closing the UDP
// socket only unblocks its next ReadFromUDP — a reader already waiting to
// send into a full channel can never observe the close. Shutdown must keep
// draining both channels until the producer exits.

// wsjtxPacket builds a valid WSJT-X packet (magic + schema + type + payload)
// in the wire format wsjtx-go expects.
func wsjtxPacket(msgType uint32, payload func(put32 func(uint32), putStr func(string))) []byte {
	var b bytes.Buffer
	put32 := func(v uint32) { _ = binary.Write(&b, binary.BigEndian, v) }
	putStr := func(s string) {
		put32(uint32(len(s)))
		b.WriteString(s)
	}
	put32(0xadbccbda) // magic
	put32(2)          // schema qDataStream5_2
	put32(msgType)
	payload(put32, putStr)
	return b.Bytes()
}

// heartbeatPacket is a valid HeartbeatMessage packet.
func heartbeatPacket() []byte {
	return wsjtxPacket(0, func(put32 func(uint32), putStr func(string)) {
		putStr("TST")
		put32(2)
		putStr("2.7")
		putStr("2.7")
	})
}

// loggedADIFPacket is a valid LoggedAdifMessage packet, which exercises the
// OnADIF callback path in the event loop.
func loggedADIFPacket() []byte {
	return wsjtxPacket(12, func(put32 func(uint32), putStr func(string)) {
		putStr("TST")
		putStr("<ADIF_VER:5>3.1.0 <EOH>")
	})
}

// TestDrainChannelsUnblocksProducer verifies the drainer keeps consuming
// until both channels close, so a producer blocked on a full channel can
// finish its send and exit.
func TestDrainChannelsUnblocksProducer(t *testing.T) {
	msgCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)

	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		defer close(msgCh)
		defer close(errCh)
		for i := 0; i < 3; i++ {
			msgCh <- wsjtx.HeartbeatMessage{}
		}
	}()

	// Let the producer fill the buffer and block on the second send.
	time.Sleep(20 * time.Millisecond)

	done := make(chan struct{})
	go func() { drainChannels(msgCh, errCh); close(done) }()

	select {
	case <-producerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("producer stayed blocked on a full channel")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drainChannels did not return after the channels closed")
	}
}

// TestFinishStopUnblocksReaderStuckOnFullChannel deterministically
// reproduces the shutdown deadlock: a real ListenToWsjtx reader is flooded
// until it blocks sending into a full msgCh (no event loop running), and
// finishStop must still complete — draining the channels lets the blocked
// send finish, the reader then observes the closed socket, and exits.
func TestFinishStopUnblocksReaderStuckOnFullChannel(t *testing.T) {
	srv, err := wsjtx.MakeServerGiven(net.ParseIP("127.0.0.1"), 0)
	if err != nil {
		t.Fatalf("MakeServerGiven: %v", err)
	}
	msgCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)

	l := NewListener()
	l.readerWG.Add(1)
	go func() {
		defer l.readerWG.Done()
		srv.ListenToWsjtx(msgCh, errCh)
	}()

	addr := srv.ServingAddr.(*net.UDPAddr)
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Flood the reader until it blocks on the second channel send.
	pkt := heartbeatPacket()
	for i := 0; i < 500; i++ {
		if _, err := conn.Write(pkt); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	time.Sleep(100 * time.Millisecond)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() { l.finishStop(&srv, stop, msgCh, errCh); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("finishStop deadlocked: reader stuck on a full channel was never drained")
	}
}

// TestStopCompletesWithBackedUpTraffic is the end-to-end shutdown test:
// a delayed OnADIF handler backs the message queue up until the real UDP
// reader blocks on a full channel, and Stop must still return promptly.
func TestStopCompletesWithBackedUpTraffic(t *testing.T) {
	for i := 0; i < 5; i++ {
		l := NewListener()
		// Delayed handler — the event loop drains slowly, so traffic
		// backs up until the reader blocks on a full channel.
		l.OnADIF = func(string) { time.Sleep(time.Millisecond) }
		if err := l.Start("127.0.0.1", 0); err != nil {
			t.Fatalf("start: %v", err)
		}

		l.mu.Lock()
		addr := l.server.ServingAddr.(*net.UDPAddr)
		l.mu.Unlock()
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}

		pkt := loggedADIFPacket()
		for j := 0; j < 400; j++ {
			conn.Write(pkt)
		}

		done := make(chan struct{})
		go func() { l.Stop(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			conn.Close()
			t.Fatalf("iteration %d: Stop hung with backed-up UDP traffic", i)
		}
		conn.Close()
	}
}
