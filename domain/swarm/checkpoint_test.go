package swarm

import (
	"math/rand"
	"testing"
)

func TestNewCheckpointStore(t *testing.T) {
	store := NewCheckpointStore(5)
	if store.MaxSlots != 5 {
		t.Errorf("expected 5 slots, got %d", store.MaxSlots)
	}
	if len(store.Slots) != 5 {
		t.Errorf("expected 5 slots array, got %d", len(store.Slots))
	}
}

func TestNewCheckpointStoreMinSlots(t *testing.T) {
	store := NewCheckpointStore(0)
	if store.MaxSlots != 5 {
		t.Errorf("0 should default to 5, got %d", store.MaxSlots)
	}
}

func TestSaveAndLoadCheckpoint(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	ss.Tick = 1000
	ss.Bots[0].X = 42.5
	ss.Bots[0].Fitness = 99.9
	ss.DeliveryOn = true

	store := NewCheckpointStore(3)
	ok := SaveCheckpoint(ss, store, 0, "test save")
	if !ok {
		t.Fatal("save should succeed")
	}

	// Modify state
	ss.Tick = 2000
	ss.Bots[0].X = 0
	ss.Bots[0].Fitness = 0
	ss.DeliveryOn = false

	// Load
	ok = LoadCheckpoint(ss, store, 0)
	if !ok {
		t.Fatal("load should succeed")
	}
	if ss.Tick != 1000 {
		t.Errorf("expected tick 1000, got %d", ss.Tick)
	}
	if ss.Bots[0].X != 42.5 {
		t.Errorf("expected X=42.5, got %f", ss.Bots[0].X)
	}
	if ss.Bots[0].Fitness != 99.9 {
		t.Errorf("expected fitness 99.9, got %f", ss.Bots[0].Fitness)
	}
	if !ss.DeliveryOn {
		t.Error("DeliveryOn should be restored")
	}
}

func TestSaveCheckpointBadSlot(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	store := NewCheckpointStore(3)
	if SaveCheckpoint(ss, store, -1, "bad") {
		t.Error("negative slot should fail")
	}
	if SaveCheckpoint(ss, store, 5, "bad") {
		t.Error("out of bounds slot should fail")
	}
	if SaveCheckpoint(ss, nil, 0, "bad") {
		t.Error("nil store should fail")
	}
}

func TestLoadCheckpointBadSlot(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	store := NewCheckpointStore(3)
	if LoadCheckpoint(ss, store, 0) {
		t.Error("empty slot should fail")
	}
	if LoadCheckpoint(ss, nil, 0) {
		t.Error("nil store should fail")
	}
}

func TestCheckpointExists(t *testing.T) {
	store := NewCheckpointStore(3)
	if CheckpointExists(store, 0) {
		t.Error("empty slot should not exist")
	}
	if CheckpointExists(nil, 0) {
		t.Error("nil store should return false")
	}
	store.Slots[1] = &Checkpoint{Name: "test"}
	if !CheckpointExists(store, 1) {
		t.Error("slot 1 should exist")
	}
}

func TestCheckpointName(t *testing.T) {
	store := NewCheckpointStore(3)
	if CheckpointName(store, 0) != "" {
		t.Error("empty slot should return empty name")
	}
	store.Slots[0] = &Checkpoint{Name: "my save"}
	if CheckpointName(store, 0) != "my save" {
		t.Errorf("expected 'my save', got '%s'", CheckpointName(store, 0))
	}
}

func TestDeleteCheckpoint(t *testing.T) {
	store := NewCheckpointStore(3)
	store.Slots[0] = &Checkpoint{Name: "test"}
	if !DeleteCheckpoint(store, 0) {
		t.Error("delete should succeed")
	}
	if CheckpointExists(store, 0) {
		t.Error("slot should be empty after delete")
	}
}

func TestDeleteCheckpointBad(t *testing.T) {
	if DeleteCheckpoint(nil, 0) {
		t.Error("nil store should fail")
	}
}

func TestUsedSlots(t *testing.T) {
	if UsedSlots(nil) != 0 {
		t.Error("nil should return 0")
	}
	store := NewCheckpointStore(5)
	if UsedSlots(store) != 0 {
		t.Error("empty store should have 0 used")
	}
	store.Slots[0] = &Checkpoint{}
	store.Slots[3] = &Checkpoint{}
	if UsedSlots(store) != 2 {
		t.Errorf("expected 2 used, got %d", UsedSlots(store))
	}
}

func TestSerializeDeserialize(t *testing.T) {
	cp := &Checkpoint{
		Name:     "test",
		Tick:     500,
		BotCount: 3,
		BotData: []CheckpointBot{
			{X: 10, Y: 20, Fitness: 50},
			{X: 30, Y: 40, Fitness: 60},
			{X: 50, Y: 60, Fitness: 70},
		},
	}
	data, err := SerializeCheckpoint(cp)
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("serialized data should not be empty")
	}

	cp2, err := DeserializeCheckpoint(data)
	if err != nil {
		t.Fatalf("deserialize error: %v", err)
	}
	if cp2.Name != "test" {
		t.Error("name should be preserved")
	}
	if cp2.BotCount != 3 {
		t.Error("bot count should be preserved")
	}
	if cp2.BotData[0].X != 10 {
		t.Error("bot data should be preserved")
	}
}

func TestSerializeNil(t *testing.T) {
	data, err := SerializeCheckpoint(nil)
	if err != nil || data != nil {
		t.Error("nil should return nil,nil")
	}
}

func TestDeserializeEmpty(t *testing.T) {
	cp, err := DeserializeCheckpoint(nil)
	if err != nil || cp != nil {
		t.Error("empty should return nil,nil")
	}
}

func TestQuickSaveLoad(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.Tick = 777
	store := NewCheckpointStore(5)

	if !QuickSave(ss, store) {
		t.Fatal("quick save should succeed")
	}
	if !CheckpointExists(store, 0) {
		t.Error("slot 0 should exist after quick save")
	}

	ss.Tick = 999
	if !QuickLoad(ss, store) {
		t.Fatal("quick load should succeed")
	}
	if ss.Tick != 777 {
		t.Errorf("expected tick 777, got %d", ss.Tick)
	}
}

func TestRewindToCheckpoint(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ss.Tick = 500
	store := NewCheckpointStore(3)
	SaveCheckpoint(ss, store, 0, "rewind")
	ss.Tick = 1000

	ok := RewindToCheckpoint(ss, store, 0, 123)
	if !ok {
		t.Fatal("rewind should succeed")
	}
	if ss.Tick != 500 {
		t.Errorf("expected tick 500, got %d", ss.Tick)
	}
	if ss.Rng == nil {
		t.Error("rng should be reseeded")
	}
}

func TestSaveBrainWeights(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 3)
	ss.Bots[0].Brain = &NeuroBrain{}
	for w := range ss.Bots[0].Brain.Weights {
		ss.Bots[0].Brain.Weights[w] = float64(w)
	}

	store := NewCheckpointStore(3)
	SaveCheckpoint(ss, store, 0, "brain test")

	// Modify weights
	for w := range ss.Bots[0].Brain.Weights {
		ss.Bots[0].Brain.Weights[w] = 0
	}

	LoadCheckpoint(ss, store, 0)
	if ss.Bots[0].Brain.Weights[5] != 5.0 {
		t.Error("brain weights should be restored")
	}
}
