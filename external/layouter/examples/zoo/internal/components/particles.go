package components

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/kellydornhaus/layouter/layout"
)

// ParticleField renders a simple particle simulation inside a panel-backed component.
type ParticleField struct {
	*layout.Panel

	preferred layout.Size

	emissionRate float64 // particles per second
	gravity      float64 // dp/s^2 acting on Y velocity
	maxParticles int

	particleRadiusDp float64
	particleColor    layout.Color

	particles       []particle
	emitAccumulator float64
	lastUpdate      time.Time
	rng             *rand.Rand

	viewport layout.Size
}

type particle struct {
	x, y   float64
	vx, vy float64
	life   float64
	age    float64
	scale  float64
}

// NewParticleField constructs a particle simulation with sensible defaults.
func NewParticleField() *ParticleField {
	panel := layout.NewPanel()
	panel.SetPadding(layout.Insets(0))
	field := &ParticleField{
		Panel:            panel,
		preferred:        layout.Size{W: 260, H: 160},
		emissionRate:     180,
		gravity:          120,
		maxParticles:     320,
		particleRadiusDp: 2.5,
		particleColor:    layout.Color{R: 120, G: 200, B: 255, A: 255},
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	panel.SetBackgroundColor(layout.Color{R: 12, G: 16, B: 28, A: 255})
	panel.SetBorder(layout.Color{R: 40, G: 56, B: 92, A: 255}, 1.5)
	return field
}

// Measure reports the desired size for the field.
func (f *ParticleField) Measure(ctx *layout.Context, cs layout.Constraints) layout.Size {
	size := f.preferred
	if size.W < cs.Min.W {
		size.W = cs.Min.W
	}
	if size.H < cs.Min.H {
		size.H = cs.Min.H
	}
	if cs.Max.W > 0 && size.W > cs.Max.W {
		size.W = cs.Max.W
	}
	if cs.Max.H > 0 && size.H > cs.Max.H {
		size.H = cs.Max.H
	}
	size.W += f.Padding.Left + f.Padding.Right
	size.H += f.Padding.Top + f.Padding.Bottom
	return size
}

// Layout stores the allocated bounds for simulation purposes.
func (f *ParticleField) Layout(ctx *layout.Context, parent layout.Component, bounds layout.Rect) {
	f.SetFrame(parent, bounds)
	content := f.ContentBounds()
	f.viewport = layout.Size{W: content.W, H: content.H}
}

// DrawTo renders the particle simulation using panel decorations.
func (f *ParticleField) DrawTo(ctx *layout.Context, dst layout.Surface) {
	f.step(ctx)
	f.SetDirty()
	f.DrawPanel(ctx, dst, func(target layout.Surface) { f.renderParticles(ctx, target) })
	f.SetDirty()
}

// Render draws directly onto the target surface.
func (f *ParticleField) Render(ctx *layout.Context, dst layout.Surface) {
	f.step(ctx)
	f.renderParticles(ctx, dst)
	f.SetDirty()
}

// SetPreferredSize updates the preferred measured size (dp units).
func (f *ParticleField) SetPreferredSize(sz layout.Size) {
	if sz.W > 0 {
		f.preferred.W = sz.W
	}
	if sz.H > 0 {
		f.preferred.H = sz.H
	}
	f.SetDirty()
}

// SetPreferredWidth updates the preferred width in dp.
func (f *ParticleField) SetPreferredWidth(width float64) {
	if width <= 0 {
		return
	}
	f.preferred.W = width
	f.SetDirty()
}

// SetPreferredHeight updates the preferred height in dp.
func (f *ParticleField) SetPreferredHeight(height float64) {
	if height <= 0 {
		return
	}
	f.preferred.H = height
	f.SetDirty()
}

// SetEmissionRate adjusts the number of particles emitted per second.
func (f *ParticleField) SetEmissionRate(rate float64) {
	if rate <= 0 {
		f.emissionRate = 0
	} else {
		f.emissionRate = rate
	}
}

// SetGravity configures the downward acceleration (dp/s^2).
func (f *ParticleField) SetGravity(gravity float64) {
	f.gravity = gravity
}

// SetMaxParticles bounds the particle pool.
func (f *ParticleField) SetMaxParticles(max int) {
	if max <= 0 {
		max = 1
	}
	f.maxParticles = max
	if len(f.particles) > max {
		f.particles = f.particles[:max]
	}
}

// SetParticleRadius sets the particle radius in dp.
func (f *ParticleField) SetParticleRadius(radiusDp float64) {
	if radiusDp <= 0 {
		return
	}
	f.particleRadiusDp = radiusDp
}

// SetParticleColor changes the tint for newly spawned particles.
func (f *ParticleField) SetParticleColor(col layout.Color) {
	f.particleColor = col
}

func (f *ParticleField) renderParticles(ctx *layout.Context, dst layout.Surface) {
	if ctx == nil || ctx.Renderer == nil {
		return
	}

	scale := ctx.Scale
	if scale <= 0 {
		scale = 1
	}
	padLeftPx := int(math.Round(f.Padding.Left * scale))
	padTopPx := int(math.Round(f.Padding.Top * scale))

	for _, p := range f.particles {
		alpha := 1.0 - p.age/p.life
		if alpha <= 0 {
			continue
		}
		radiusPx := int(math.Round(f.particleRadiusDp * scale * clampFloat(p.scale, 0.6, 1.4)))
		if radiusPx < 1 {
			radiusPx = 1
		}
		xPx := padLeftPx + int(math.Round(p.x*scale))
		yPx := padTopPx + int(math.Round(p.y*scale))

		rect := layout.PxRect{
			X: xPx - radiusPx,
			Y: yPx - radiusPx,
			W: radiusPx * 2,
			H: radiusPx * 2,
		}
		rect = clampRect(rect, dst)
		if rect.W <= 0 || rect.H <= 0 {
			continue
		}
		col := layout.Color{
			R: f.particleColor.R,
			G: f.particleColor.G,
			B: f.particleColor.B,
			A: uint8(math.Round(float64(f.particleColor.A) * alpha)),
		}
		if col.A == 0 {
			continue
		}
		ctx.Renderer.FillRect(dst, rect, col)
	}

	// keep animating
	f.SetDirty()
}

func (f *ParticleField) step(ctx *layout.Context) {
	now := time.Now()
	if f.lastUpdate.IsZero() {
		f.lastUpdate = now.Add(-time.Second / 60)
	}
	dt := now.Sub(f.lastUpdate).Seconds()
	f.lastUpdate = now
	if dt > 0.05 {
		dt = 0.05
	}
	if dt <= 0 {
		return
	}

	if f.emissionRate > 0 && f.viewport.W > 0 && f.viewport.H > 0 {
		f.emitAccumulator += f.emissionRate * dt
		count := int(f.emitAccumulator)
		f.emitAccumulator -= float64(count)
		for i := 0; i < count && len(f.particles) < f.maxParticles; i++ {
			f.spawn()
		}
	}

	width := f.viewport.W
	height := f.viewport.H
	g := f.gravity * dt

	for i := 0; i < len(f.particles); {
		p := &f.particles[i]
		p.age += dt
		if p.age >= p.life {
			f.particles = append(f.particles[:i], f.particles[i+1:]...)
			continue
		}
		p.vy += g
		p.x += p.vx * dt
		p.y += p.vy * dt

		if p.x < 0 {
			p.x = 0
			p.vx *= -0.6
		} else if p.x > width {
			p.x = width
			p.vx *= -0.6
		}
		if p.y > height {
			p.y = height
			p.vy *= -0.5
			if math.Abs(p.vy) < 5 {
				p.life = p.age + 0.1
			}
		}
		i++
	}
}

func (f *ParticleField) spawn() {
	if f.viewport.W <= 0 || f.viewport.H <= 0 {
		return
	}
	spread := f.viewport.W * 0.4
	baseX := f.viewport.W / 2
	x := baseX + (f.rng.Float64()-0.5)*spread
	if x < 0 {
		x = 0
	}
	if x > f.viewport.W {
		x = f.viewport.W
	}
	speed := 80 + f.rng.Float64()*120
	p := particle{
		x:     x,
		y:     f.viewport.H,
		vx:    (f.rng.Float64() - 0.5) * 60,
		vy:    -speed,
		life:  1.2 + f.rng.Float64()*1.4,
		scale: 0.7 + f.rng.Float64()*0.6,
	}
	f.particles = append(f.particles, p)
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// ParseParticleOptions helps reuse panel attributes for particle fields.
func ParseParticleOptions(field *ParticleField, attrs map[string]string) {
	for key, val := range attrs {
		if val == "" {
			continue
		}
		switch strings.ToLower(key) {
		case "rate":
			field.SetEmissionRate(parseFloat(val, field.emissionRate))
		case "gravity":
			field.SetGravity(parseFloat(val, field.gravity))
		case "radius":
			field.SetParticleRadius(parseFloat(val, field.particleRadiusDp))
		case "max":
			field.SetMaxParticles(int(parseFloat(val, float64(field.maxParticles))))
		}
	}
}

func parseFloat(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "dp") {
		s = strings.TrimSuffix(s, "dp")
	}
	if s == "" {
		return def
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return def
}

func clampRect(rect layout.PxRect, dst layout.Surface) layout.PxRect {
	w, h := dst.SizePx()
	if rect.X < 0 {
		rect.W += rect.X
		rect.X = 0
	}
	if rect.Y < 0 {
		rect.H += rect.Y
		rect.Y = 0
	}
	if rect.X+rect.W > w {
		rect.W = w - rect.X
	}
	if rect.Y+rect.H > h {
		rect.H = h - rect.Y
	}
	if rect.W < 0 {
		rect.W = 0
	}
	if rect.H < 0 {
		rect.H = 0
	}
	return rect
}
