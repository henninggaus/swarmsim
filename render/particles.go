package render

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const maxParticles = 512

// Particle is a single visual particle.
type Particle struct {
	X, Y    float64
	VX, VY  float64
	Life    int
	MaxLife int
	Color   color.RGBA
	Size    float64
	Active  bool
}

// ParticleSystem manages a pool of particles.
type ParticleSystem struct {
	Particles [maxParticles]Particle
	nextSlot  int
	rng       *rand.Rand
}

// NewParticleSystem creates a new particle system.
func NewParticleSystem() *ParticleSystem {
	return &ParticleSystem{
		rng: rand.New(rand.NewSource(99)),
	}
}

// Emit spawns a burst of particles at a world position.
func (ps *ParticleSystem) Emit(x, y float64, count int, col color.RGBA, speed, size float64, life int) {
	for i := 0; i < count; i++ {
		p := &ps.Particles[ps.nextSlot]
		angle := ps.rng.Float64() * 2 * math.Pi
		spd := speed * (0.5 + ps.rng.Float64())
		p.X = x
		p.Y = y
		p.VX = math.Cos(angle) * spd
		p.VY = math.Sin(angle) * spd
		p.Life = life + ps.rng.Intn(life/2+1)
		p.MaxLife = p.Life
		p.Color = col
		p.Size = size * (0.5 + ps.rng.Float64())
		p.Active = true
		ps.nextSlot = (ps.nextSlot + 1) % maxParticles
	}
}

// Update moves all active particles forward one step.
func (ps *ParticleSystem) Update() {
	for i := range ps.Particles {
		p := &ps.Particles[i]
		if !p.Active {
			continue
		}
		p.X += p.VX
		p.Y += p.VY
		p.VY += 0.02 // slight gravity
		p.VX *= 0.98 // drag
		p.VY *= 0.98
		p.Life--
		if p.Life <= 0 {
			p.Active = false
		}
	}
}

// Draw renders all active particles to screen.
func (ps *ParticleSystem) Draw(screen *ebiten.Image, cam *Camera, sw, sh int) {
	for i := range ps.Particles {
		p := &ps.Particles[i]
		if !p.Active {
			continue
		}
		sx, sy := cam.WorldToScreen(p.X, p.Y, sw, sh)
		alpha := float64(p.Life) / float64(p.MaxLife)
		col := p.Color
		col.A = uint8(float64(col.A) * alpha)
		size := float32(p.Size * cam.Zoom * alpha)
		if size < 0.5 {
			size = 0.5
		}
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), size, col, false)
	}
}
