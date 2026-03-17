package swarm

import (
	"strings"
	"testing"
)

func TestDefaultShaderConfig(t *testing.T) {
	cfg := DefaultShaderConfig()
	if cfg.Active != ShaderNone {
		t.Error("default should be ShaderNone")
	}
	if cfg.Intensity != 0.8 {
		t.Errorf("expected intensity 0.8, got %f", cfg.Intensity)
	}
	if cfg.Speed != 1.0 {
		t.Errorf("expected speed 1.0, got %f", cfg.Speed)
	}
}

func TestShaderName(t *testing.T) {
	if ShaderName(ShaderNone) != "Keiner" {
		t.Error("wrong name for ShaderNone")
	}
	if ShaderName(ShaderHeatmap) != "Heatmap" {
		t.Error("wrong name for ShaderHeatmap")
	}
	if ShaderName(ShaderNightVision) != "Nachtsicht" {
		t.Error("wrong name for ShaderNightVision")
	}
	if ShaderName(ShaderType(99)) != "Unbekannt" {
		t.Error("unknown shader should be Unbekannt")
	}
}

func TestShaderParamName(t *testing.T) {
	if ShaderParamName(ShaderHeatmap, 0) != "Radius" {
		t.Error("wrong param name for heatmap[0]")
	}
	if ShaderParamName(ShaderHeatmap, -1) != "" {
		t.Error("negative index should return empty")
	}
	if ShaderParamName(ShaderHeatmap, 8) != "" {
		t.Error("out of bounds should return empty")
	}
	if ShaderParamName(ShaderNone, 0) != "" {
		t.Error("ShaderNone should have no params")
	}
}

func TestShaderParamNameAllTypes(t *testing.T) {
	types := []ShaderType{ShaderPheromone, ShaderEnergy, ShaderTeamAura, ShaderWaveform, ShaderNightVision}
	for _, st := range types {
		name := ShaderParamName(st, 0)
		if name == "" {
			t.Errorf("shader type %d should have param 0", st)
		}
	}
}

func TestNextShader(t *testing.T) {
	if NextShader(ShaderNone) != ShaderHeatmap {
		t.Error("ShaderNone should cycle to ShaderHeatmap")
	}
	if NextShader(ShaderNightVision) != ShaderNone {
		t.Error("last shader should cycle to ShaderNone")
	}
}

func TestNextShaderCycle(t *testing.T) {
	s := ShaderNone
	for i := 0; i < int(ShaderTypeCount); i++ {
		s = NextShader(s)
	}
	if s != ShaderNone {
		t.Error("full cycle should return to ShaderNone")
	}
}

func TestShaderTypeCount(t *testing.T) {
	if ShaderTypeCount != 7 {
		t.Errorf("expected 7 shader types, got %d", ShaderTypeCount)
	}
}

func TestKageHeatmapSource(t *testing.T) {
	src := KageHeatmapSource()
	if !strings.Contains(src, "package main") {
		t.Error("Kage source should contain 'package main'")
	}
	if !strings.Contains(src, "Fragment") {
		t.Error("Kage source should contain Fragment function")
	}
	if !strings.Contains(src, "Intensity") {
		t.Error("heatmap should use Intensity uniform")
	}
}

func TestKageGlowSource(t *testing.T) {
	src := KageGlowSource()
	if !strings.Contains(src, "Fragment") {
		t.Error("glow shader should have Fragment")
	}
	if !strings.Contains(src, "Radius") {
		t.Error("glow should use Radius")
	}
}

func TestKageNightVisionSource(t *testing.T) {
	src := KageNightVisionSource()
	if !strings.Contains(src, "Fragment") {
		t.Error("night vision should have Fragment")
	}
	if !strings.Contains(src, "scanline") {
		t.Error("night vision should have scanline effect")
	}
}

func TestAllShaderNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := ShaderType(0); i < ShaderTypeCount; i++ {
		name := ShaderName(i)
		if seen[name] {
			t.Errorf("duplicate shader name: %s", name)
		}
		seen[name] = true
	}
}
