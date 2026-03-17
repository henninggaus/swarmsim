package swarm

import "testing"

func TestReplayBufferRecordAndGet(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{
		Bots: make([]SwarmBot, 3),
		Tick: 42,
	}
	ss.Bots[0].X = 10
	ss.Bots[1].X = 20
	ss.Bots[2].X = 30
	for i := range ss.Bots {
		ss.Bots[i].CarryingPkg = -1
	}

	buf.Record(ss)
	if buf.Count != 1 {
		t.Fatalf("expected count 1, got %d", buf.Count)
	}

	snap := buf.Get(0)
	if snap == nil {
		t.Fatal("expected snapshot")
	}
	if snap.Tick != 42 {
		t.Errorf("expected tick 42, got %d", snap.Tick)
	}
	if snap.BotData[1].X != 20 {
		t.Errorf("expected bot 1 X=20, got %f", snap.BotData[1].X)
	}
}

func TestReplayBufferRingOverwrite(t *testing.T) {
	buf := NewReplayBuffer(5)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1

	for i := 0; i < 8; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	if buf.Count != 5 {
		t.Fatalf("expected count capped at 5, got %d", buf.Count)
	}

	oldest := buf.Get(0)
	if oldest.Tick != 3 {
		t.Errorf("oldest should be tick 3, got %d", oldest.Tick)
	}
	newest := buf.Newest()
	if newest.Tick != 7 {
		t.Errorf("newest should be tick 7, got %d", newest.Tick)
	}
}

func TestReplayBufferOldest(t *testing.T) {
	buf := NewReplayBuffer(10)
	if buf.Oldest() != nil {
		t.Error("empty buffer Oldest should be nil")
	}
	ss := &SwarmState{Bots: make([]SwarmBot, 1), Tick: 5}
	ss.Bots[0].CarryingPkg = -1
	buf.Record(ss)
	if buf.Oldest().Tick != 5 {
		t.Errorf("expected tick 5, got %d", buf.Oldest().Tick)
	}
}

func TestReplayPlayerNewAtNewest(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 10; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	if rp.Cursor != 9 {
		t.Errorf("expected cursor at 9 (newest), got %d", rp.Cursor)
	}
	snap := rp.Current()
	if snap.Tick != 9 {
		t.Errorf("expected tick 9, got %d", snap.Tick)
	}
}

func TestReplayPlayerStepForwardBackward(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 5; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	rp.SeekStart()
	if rp.Cursor != 0 {
		t.Fatal("SeekStart should set cursor to 0")
	}

	rp.StepForward()
	if rp.Cursor != 1 {
		t.Errorf("expected cursor 1 after step forward, got %d", rp.Cursor)
	}

	rp.StepBackward()
	if rp.Cursor != 0 {
		t.Errorf("expected cursor 0 after step backward, got %d", rp.Cursor)
	}

	// At start, step backward without loop should fail
	ok := rp.StepBackward()
	if ok {
		t.Error("step backward at start should return false")
	}
}

func TestReplayPlayerSeekEnd(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 5; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	rp.SeekStart()
	rp.SeekEnd()
	if rp.Cursor != 4 {
		t.Errorf("SeekEnd: expected cursor 4, got %d", rp.Cursor)
	}
}

func TestReplayPlayerLoopMode(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 3; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	rp.LoopMode = true

	// At end (cursor=2), step forward should loop to 0
	ok := rp.StepForward()
	if !ok || rp.Cursor != 0 {
		t.Errorf("loop forward: expected cursor 0, got %d (ok=%v)", rp.Cursor, ok)
	}

	// At start (cursor=0), step backward should loop to end
	ok = rp.StepBackward()
	if !ok || rp.Cursor != 2 {
		t.Errorf("loop backward: expected cursor 2, got %d (ok=%v)", rp.Cursor, ok)
	}
}

func TestReplayPlayerSeekPercent(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 10; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	rp.SeekPercent(0.0)
	if rp.Cursor != 0 {
		t.Errorf("0%%: expected cursor 0, got %d", rp.Cursor)
	}
	rp.SeekPercent(1.0)
	if rp.Cursor != 9 {
		t.Errorf("100%%: expected cursor 9, got %d", rp.Cursor)
	}
	rp.SeekPercent(0.5)
	if rp.Cursor != 4 {
		t.Errorf("50%%: expected cursor 4, got %d", rp.Cursor)
	}
}

func TestReplayPlayerProgress(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 10; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	rp.SeekStart()
	if rp.Progress() != 0 {
		t.Errorf("expected progress 0 at start, got %f", rp.Progress())
	}
	rp.SeekEnd()
	if rp.Progress() != 1.0 {
		t.Errorf("expected progress 1.0 at end, got %f", rp.Progress())
	}
}

func TestReplayPlayerTickPlayback(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 20; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	rp.SeekStart()
	rp.Playing = true
	rp.Speed = 3

	rp.TickPlayback()
	if rp.Cursor != 3 {
		t.Errorf("after 3x speed tick: expected cursor 3, got %d", rp.Cursor)
	}

	// Rewind
	rp.Speed = -2
	rp.TickPlayback()
	if rp.Cursor != 1 {
		t.Errorf("after -2x rewind tick: expected cursor 1, got %d", rp.Cursor)
	}
}

func TestReplayPlayerTickPlaybackStopsAtEnd(t *testing.T) {
	buf := NewReplayBuffer(100)
	ss := &SwarmState{Bots: make([]SwarmBot, 1)}
	ss.Bots[0].CarryingPkg = -1
	for i := 0; i < 3; i++ {
		ss.Tick = i
		buf.Record(ss)
	}

	rp := NewReplayPlayer(buf)
	rp.SeekTo(1)
	rp.Playing = true
	rp.Speed = 5 // will overshoot

	rp.TickPlayback()
	if rp.Playing {
		t.Error("playback should stop at end")
	}
}

func TestReplayPlayerNilBuffer(t *testing.T) {
	rp := NewReplayPlayer(nil)
	if rp.Current() != nil {
		t.Error("nil buffer Current should be nil")
	}
	if rp.FramesTotal() != 0 {
		t.Error("nil buffer FramesTotal should be 0")
	}
	rp.SeekTo(5)   // should not panic
	rp.SeekPercent(0.5) // should not panic
	rp.StepForward()
	rp.StepBackward()
	rp.TickPlayback()
}

func TestReplayPlayerTogglePlay(t *testing.T) {
	rp := NewReplayPlayer(nil)
	if rp.Playing {
		t.Error("should start paused")
	}
	rp.TogglePlay()
	if !rp.Playing {
		t.Error("should be playing after toggle")
	}
	rp.TogglePlay()
	if rp.Playing {
		t.Error("should be paused after second toggle")
	}
}
