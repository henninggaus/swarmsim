package swarm

import (
	"math/rand"
	"testing"
)

func TestNewStigmergyGrid(t *testing.T) {
	sg := NewStigmergyGrid(800, 800, 10)
	if sg.GridCols != 40 || sg.GridRows != 40 {
		t.Fatalf("expected 40x40, got %dx%d", sg.GridCols, sg.GridRows)
	}
}

func TestPlaceBlock(t *testing.T) {
	sg := NewStigmergyGrid(800, 800, 10)
	ok := PlaceBlock(sg, 100, 100, BlockWall, 0, 1)
	if !ok {
		t.Fatal("should place block")
	}
	if !sg.HasBlock(100, 100) {
		t.Fatal("should have block at 100,100")
	}
	if sg.TotalPlaced != 1 {
		t.Fatal("total placed should be 1")
	}

	// Same position should fail
	ok = PlaceBlock(sg, 100, 100, BlockWall, 0, 2)
	if ok {
		t.Fatal("should not place on occupied cell")
	}
}

func TestRemoveBlock(t *testing.T) {
	sg := NewStigmergyGrid(800, 800, 10)
	PlaceBlock(sg, 100, 100, BlockWall, 0, 1)
	ok := RemoveBlock(sg, 100, 100)
	if !ok {
		t.Fatal("should remove block")
	}
	if sg.HasBlock(100, 100) {
		t.Fatal("should not have block after removal")
	}
	if sg.TotalRemoved != 1 {
		t.Fatal("total removed should be 1")
	}
}

func TestMaxBlocks(t *testing.T) {
	sg := NewStigmergyGrid(800, 800, 10)
	sg.MaxBlocks = 3
	for i := 0; i < 5; i++ {
		PlaceBlock(sg, float64(i)*30, 100, BlockWall, 0, i)
	}
	if ActiveBlockCount(sg) > 3 {
		t.Fatalf("should cap at 3 blocks, got %d", ActiveBlockCount(sg))
	}
}

func TestBlocksNearBot(t *testing.T) {
	sg := NewStigmergyGrid(800, 800, 10)
	PlaceBlock(sg, 100, 100, BlockWall, 0, 1)
	PlaceBlock(sg, 120, 100, BlockWall, 0, 2)
	PlaceBlock(sg, 500, 500, BlockWall, 0, 3)

	count := BlocksNearBot(sg, 110, 100, 50)
	if count != 2 {
		t.Fatalf("expected 2 nearby blocks, got %d", count)
	}
}

func TestNearestBlockAngle(t *testing.T) {
	sg := NewStigmergyGrid(800, 800, 10)
	PlaceBlock(sg, 200, 100, BlockWall, 0, 1)

	_, found := NearestBlockAngle(sg, 100, 100, 200)
	if !found {
		t.Fatal("should find nearest block")
	}

	_, found = NearestBlockAngle(sg, 100, 100, 10)
	if found {
		t.Fatal("should not find block outside range")
	}
}

func TestMaterialSystem(t *testing.T) {
	sg := NewStigmergyGrid(800, 800, 10)
	sg.MaterialEnabled = true
	sg.BotMaterial = make([]int, 10)

	// No material = can't build
	ok := PlaceBlock(sg, 100, 100, BlockWall, 0, 1)
	if ok {
		t.Fatal("should not place without material")
	}

	// Give material
	sg.BotMaterial[0] = 3
	ok = PlaceBlock(sg, 100, 100, BlockWall, 0, 1)
	if !ok {
		t.Fatal("should place with material")
	}
	if sg.BotMaterial[0] != 2 {
		t.Fatalf("material should decrease, got %d", sg.BotMaterial[0])
	}
}

func TestInitStigmergy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitStigmergy(ss)
	if ss.Stigmergy == nil {
		t.Fatal("stigmergy should be initialized")
	}
	if len(ss.Stigmergy.MaterialSources) != 4 {
		t.Fatalf("expected 4 sources, got %d", len(ss.Stigmergy.MaterialSources))
	}
}

func TestBotPickupMaterial(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ss := NewSwarmState(rng, 10)
	InitStigmergy(ss)
	ss.Stigmergy.Grid.MaterialEnabled = true

	// Place bot near first source
	src := &ss.Stigmergy.MaterialSources[0]
	ss.Bots[0].X = src.X + 5
	ss.Bots[0].Y = src.Y + 5

	ok := BotPickupMaterial(ss, 0)
	if !ok {
		t.Fatal("should pick up material")
	}
	if ss.Stigmergy.Grid.BotMaterial[0] != 1 {
		t.Fatal("bot should have 1 material")
	}
}
