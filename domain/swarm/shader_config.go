package swarm

// ShaderType identifies which visual shader effect is active.
type ShaderType int

const (
	ShaderNone       ShaderType = iota
	ShaderHeatmap               // bot density heatmap overlay
	ShaderPheromone             // pheromone trail glow
	ShaderEnergy                // energy level radial gradient per bot
	ShaderTeamAura              // team-colored aura around bots
	ShaderWaveform              // message broadcast wave effect
	ShaderNightVision           // green-tint night mode
	ShaderTypeCount
)

// ShaderConfig holds parameters for GPU shader effects.
type ShaderConfig struct {
	Active      ShaderType
	Intensity   float64 // 0.0-1.0 global intensity
	Speed       float64 // animation speed multiplier (default 1.0)
	ColorShift  float64 // hue rotation 0-360
	BlendMode   int     // 0=additive, 1=multiply, 2=overlay
	Params      [8]float64 // shader-specific parameters
}

// DefaultShaderConfig returns sensible defaults.
func DefaultShaderConfig() ShaderConfig {
	return ShaderConfig{
		Active:    ShaderNone,
		Intensity: 0.8,
		Speed:     1.0,
	}
}

// ShaderName returns the display name for a shader type.
func ShaderName(s ShaderType) string {
	switch s {
	case ShaderNone:
		return "Keiner"
	case ShaderHeatmap:
		return "Heatmap"
	case ShaderPheromone:
		return "Pheromon-Glow"
	case ShaderEnergy:
		return "Energie-Gradient"
	case ShaderTeamAura:
		return "Team-Aura"
	case ShaderWaveform:
		return "Wellen-Effekt"
	case ShaderNightVision:
		return "Nachtsicht"
	default:
		return "Unbekannt"
	}
}

// ShaderParamName returns the name of a shader-specific parameter.
func ShaderParamName(s ShaderType, idx int) string {
	if idx < 0 || idx > 7 {
		return ""
	}
	switch s {
	case ShaderHeatmap:
		names := [8]string{"Radius", "Schwelle", "Max-Dichte", "", "", "", "", ""}
		return names[idx]
	case ShaderPheromone:
		names := [8]string{"Glow-Radius", "Abklingrate", "Helligkeit", "", "", "", "", ""}
		return names[idx]
	case ShaderEnergy:
		names := [8]string{"Min-Radius", "Max-Radius", "Pulsgeschw.", "", "", "", "", ""}
		return names[idx]
	case ShaderTeamAura:
		names := [8]string{"Aura-Radius", "Transparenz", "", "", "", "", "", ""}
		return names[idx]
	case ShaderWaveform:
		names := [8]string{"Wellenlaenge", "Amplitude", "Daempfung", "", "", "", "", ""}
		return names[idx]
	case ShaderNightVision:
		names := [8]string{"Gruen-Ton", "Rauschen", "Scanline", "", "", "", "", ""}
		return names[idx]
	default:
		return ""
	}
}

// NextShader cycles to the next shader type.
func NextShader(current ShaderType) ShaderType {
	next := current + 1
	if next >= ShaderTypeCount {
		next = ShaderNone
	}
	return next
}

// KageHeatmapSource returns the Kage shader source for the heatmap effect.
// Kage is Ebiten's shader language (subset of Go).
func KageHeatmapSource() string {
	return `package main

var Intensity float
var Time float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	src := imageSrc0At(texCoord)
	heat := src.r * Intensity
	r := clamp(heat * 2.0, 0.0, 1.0)
	g := clamp(heat * 1.5 - 0.5, 0.0, 1.0)
	b := clamp(heat - 1.0, 0.0, 1.0)
	return vec4(r, g, b, src.a * Intensity)
}
`
}

// KageGlowSource returns the Kage shader source for a glow/bloom effect.
func KageGlowSource() string {
	return `package main

var Intensity float
var Radius float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	src := imageSrc0At(texCoord)
	glow := vec4(0)
	steps := 8
	for i := 0; i < steps; i++ {
		for j := 0; j < steps; j++ {
			fi := float(i) - float(steps)/2.0
			fj := float(j) - float(steps)/2.0
			offset := vec2(fi, fj) * Radius / float(steps)
			glow += imageSrc0At(texCoord + offset)
		}
	}
	glow /= float(steps * steps)
	return src + glow * Intensity * 0.5
}
`
}

// KageNightVisionSource returns a night-vision post-processing shader.
func KageNightVisionSource() string {
	return `package main

var Intensity float
var Time float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	src := imageSrc0At(texCoord)
	lum := src.r*0.299 + src.g*0.587 + src.b*0.114
	green := vec4(lum*0.2, lum*Intensity, lum*0.2, src.a)
	noise := fract(sin(texCoord.x*43758.5453+texCoord.y*12345.6789+Time) * 43758.5453)
	green.g += noise * 0.05
	scanline := 0.95 + 0.05*sin(texCoord.y*500.0)
	return green * scanline
}
`
}
