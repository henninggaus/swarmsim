package swarm

import (
	"math/rand"
	"testing"
)

func TestExportImportNeuro(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitNeuro(ss)

	// Export
	tg := ExportNeuro(ss)
	if tg == nil {
		t.Fatal("export should not be nil")
	}
	if len(tg.NeuroWeights) != 10 {
		t.Fatalf("expected 10 weight sets, got %d", len(tg.NeuroWeights))
	}
	if tg.Type != "neuro" {
		t.Fatal("type should be neuro")
	}

	// Create new sim and import
	ss2 := NewSwarmState(rng, 15) // different bot count
	InitNeuro(ss2)
	ok := ImportNeuro(ss2, tg)
	if !ok {
		t.Fatal("import should succeed")
	}

	// Bot 10 should get bot 0's weights (cyclic)
	for w := 0; w < NeuroWeights; w++ {
		if ss2.Bots[10].Brain.Weights[w] != tg.NeuroWeights[0][w] {
			t.Fatal("cyclic import should copy weights correctly")
		}
	}
}

func TestExportImportGP(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitGP(ss)

	tg := ExportGP(ss)
	if tg == nil {
		t.Fatal("export should not be nil")
	}
	if len(tg.GPPrograms) != 10 {
		t.Fatalf("expected 10 programs, got %d", len(tg.GPPrograms))
	}

	ss2 := NewSwarmState(rng, 10)
	InitGP(ss2)
	ok := ImportGP(ss2, tg)
	if !ok {
		t.Fatal("import should succeed")
	}
}

func TestExportImportParams(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	// Set some param values
	for i := range ss.Bots {
		ss.Bots[i].ParamValues[0] = float64(i * 10)
	}

	tg := ExportParams(ss)
	if len(tg.ParamValues) != 10 {
		t.Fatal("should export 10 param sets")
	}

	ss2 := NewSwarmState(rng, 10)
	ImportParams(ss2, tg)
	if ss2.Bots[5].ParamValues[0] != 50 {
		t.Fatalf("expected 50, got %f", ss2.Bots[5].ParamValues[0])
	}
}

func TestSerializeDeserializeTransfer(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	InitNeuro(ss)

	tg := ExportNeuro(ss)
	data, err := SerializeTransfer(tg)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("serialized data should not be empty")
	}

	tg2, err := DeserializeTransfer(data)
	if err != nil {
		t.Fatal(err)
	}
	if tg2.Type != "neuro" {
		t.Fatal("deserialized type should be neuro")
	}
	if len(tg2.NeuroWeights) != 5 {
		t.Fatalf("expected 5 weight sets, got %d", len(tg2.NeuroWeights))
	}
}

func TestExportBestNeuro(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 20)
	InitNeuro(ss)

	// Give some bots higher fitness
	for i := 0; i < 5; i++ {
		ss.Bots[i].Stats.TotalDeliveries = 10 + i
	}

	tg := ExportBestNeuro(ss, 20) // top 20% = 4 bots
	if len(tg.NeuroWeights) != 4 {
		t.Fatalf("expected 4 weight sets (20%%), got %d", len(tg.NeuroWeights))
	}
}

func TestTransferAdaptationScore(t *testing.T) {
	ts := &TransferState{
		LastImport: &TransferGenome{Fitness: 100},
	}
	score := TransferAdaptationScore(ts, 150)
	if score != 1.5 {
		t.Fatalf("expected 1.5, got %f", score)
	}
}

func TestImportFailsOnWrongType(t *testing.T) {
	tg := &TransferGenome{Type: "gp", GPPrograms: []string{"IF true THEN FWD"}}
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 5)
	ok := ImportNeuro(ss, tg)
	if ok {
		t.Fatal("should fail importing GP as neuro")
	}
}
