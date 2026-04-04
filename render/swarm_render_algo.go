package render

import (
	"image"
	"image/color"
	"math"
	"swarmsim/domain/swarm"
	"swarmsim/locale"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// fitnessLandscapeHashKey computes a simple hash from the fitness function type
// and (for Gaussian) the shared peak parameters.
func fitnessLandscapeHashKey(sa *swarm.SwarmAlgorithmState) uint64 {
	h := uint64(sa.FitnessFunc) * 1000003
	h ^= uint64(len(sa.FitPeakX))
	for i := range sa.FitPeakX {
		h ^= math.Float64bits(sa.FitPeakX[i]) * 31
		h ^= math.Float64bits(sa.FitPeakY[i]) * 37
		h ^= math.Float64bits(sa.FitPeakH[i]) * 41
		h ^= math.Float64bits(sa.FitPeakS[i]) * 43
	}
	return h
}

// landscapeColor maps a normalized fitness value (0-1) to a color using a
// blue → cyan → green → yellow → red gradient.
func landscapeColor(t float64, alpha uint8) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	var r, g, b float64
	switch {
	case t < 0.25:
		s := t / 0.25
		r, g, b = 0, s, 1 // blue → cyan
	case t < 0.5:
		s := (t - 0.25) / 0.25
		r, g, b = 0, 1, 1-s // cyan → green
	case t < 0.75:
		s := (t - 0.5) / 0.25
		r, g, b = s, 1, 0 // green → yellow
	default:
		s := (t - 0.75) / 0.25
		r, g, b = 1, 1-s, 0 // yellow → red
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), alpha}
}

// contourSegment represents a single line segment of a contour line,
// in arena pixel coordinates.
type contourSegment struct {
	x0, y0, x1, y1 float32
	level          int // contour level index (0=lowest, numLevels-1=highest)
}

// buildFitnessLandscape generates a heatmap image from the shared Gaussian
// fitness landscape. Computed at 1/4 resolution for performance and cached
// until peaks change. Also computes iso-fitness contour lines via marching
// squares.
func (r *Renderer) buildFitnessLandscape(sa *swarm.SwarmAlgorithmState, arenaW, arenaH int) *ebiten.Image {
	const step = 4 // sample every 4 pixels
	imgW := arenaW / step
	imgH := arenaH / step

	// Find min/max fitness for normalization
	minF, maxF := math.MaxFloat64, -math.MaxFloat64
	values := make([]float64, imgW*imgH)
	for iy := 0; iy < imgH; iy++ {
		wy := float64(iy*step) + float64(step)/2
		for ix := 0; ix < imgW; ix++ {
			wx := float64(ix*step) + float64(step)/2
			f := swarm.EvaluateFitnessLandscape(sa, wx, wy)
			values[iy*imgW+ix] = f
			if f < minF {
				minF = f
			}
			if f > maxF {
				maxF = f
			}
		}
	}

	// Build RGBA image
	rgba := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	rangeF := maxF - minF
	if rangeF < 1e-9 {
		rangeF = 1
	}
	for iy := 0; iy < imgH; iy++ {
		for ix := 0; ix < imgW; ix++ {
			t := (values[iy*imgW+ix] - minF) / rangeF
			c := landscapeColor(t, 100)
			off := (iy*imgW + ix) * 4
			rgba.Pix[off+0] = c.R
			rgba.Pix[off+1] = c.G
			rgba.Pix[off+2] = c.B
			rgba.Pix[off+3] = c.A
		}
	}

	// Compute contour lines via marching squares
	const numContourLevels = 8
	r.psoContourSegs = r.psoContourSegs[:0]
	r.psoContourW = imgW
	r.psoContourH = imgH
	for li := 1; li <= numContourLevels; li++ {
		frac := float64(li) / float64(numContourLevels+1) // e.g. 1/9, 2/9, ..., 8/9
		level := minF + frac*rangeF
		marchingSquaresContour(&r.psoContourSegs, values, imgW, imgH, level, li-1, float32(step))
	}

	img := ebiten.NewImageFromImage(rgba)
	return img
}

// marchingSquaresContour extracts iso-value contour line segments from a 2D
// scalar field using the marching squares algorithm. Results are appended to
// segs. Coordinates are scaled by pixelStep to convert from grid cells to
// arena pixels.
func marchingSquaresContour(segs *[]contourSegment, values []float64, w, h int, level float64, levelIdx int, pixelStep float32) {
	for iy := 0; iy < h-1; iy++ {
		for ix := 0; ix < w-1; ix++ {
			// Four corners: NW, NE, SE, SW
			nw := values[iy*w+ix]
			ne := values[iy*w+ix+1]
			se := values[(iy+1)*w+ix+1]
			sw := values[(iy+1)*w+ix]

			// 4-bit case index
			ci := 0
			if nw >= level {
				ci |= 1
			}
			if ne >= level {
				ci |= 2
			}
			if se >= level {
				ci |= 4
			}
			if sw >= level {
				ci |= 8
			}
			if ci == 0 || ci == 15 {
				continue
			}

			// Linear interpolation along an edge
			lerpT := func(a, b float64) float32 {
				d := b - a
				if d > -1e-10 && d < 1e-10 {
					return 0.5
				}
				t := (level - a) / d
				if t < 0 {
					t = 0
				} else if t > 1 {
					t = 1
				}
				return float32(t)
			}

			fx := float32(ix) * pixelStep
			fy := float32(iy) * pixelStep

			// Edge crossing points (in arena pixel coords)
			northX := fx + lerpT(nw, ne)*pixelStep
			northY := fy
			eastX := fx + pixelStep
			eastY := fy + lerpT(ne, se)*pixelStep
			southX := fx + lerpT(sw, se)*pixelStep
			southY := fy + pixelStep
			westX := fx
			westY := fy + lerpT(nw, sw)*pixelStep

			addSeg := func(ax, ay, bx, by float32) {
				*segs = append(*segs, contourSegment{ax, ay, bx, by, levelIdx})
			}

			switch ci {
			case 1, 14: // NW
				addSeg(northX, northY, westX, westY)
			case 2, 13: // NE
				addSeg(northX, northY, eastX, eastY)
			case 3, 12: // NW+NE → west-east
				addSeg(westX, westY, eastX, eastY)
			case 4, 11: // SE → east-south
				addSeg(eastX, eastY, southX, southY)
			case 5: // NW+SE (saddle) → two segments
				addSeg(northX, northY, westX, westY)
				addSeg(eastX, eastY, southX, southY)
			case 6, 9: // NE+SE or NW+SW → north-south
				addSeg(northX, northY, southX, southY)
			case 7, 8: // three corners or SW → west-south
				addSeg(westX, westY, southX, southY)
			case 10: // NE+SW (saddle) → two segments
				addSeg(northX, northY, eastX, eastY)
				addSeg(westX, westY, southX, southY)
			}
		}
	}
}

// drawFitnessLandscapeOverlay draws the shared Gaussian fitness landscape as a
// color heatmap with peak markers. Works for any algorithm using the shared
// fitness landscape (not just PSO).
func (r *Renderer) drawFitnessLandscapeOverlay(a *ebiten.Image, ss *swarm.SwarmState) {
	sa := ss.SwarmAlgo

	// Rebuild cached heatmap if peaks changed
	h := fitnessLandscapeHashKey(sa)
	if r.psoLandscapeImg == nil || r.psoLandscapeHash != h {
		arenaW := int(ss.ArenaW)
		arenaH := int(ss.ArenaH)
		r.psoLandscapeImg = r.buildFitnessLandscape(sa, arenaW, arenaH)
		r.psoLandscapeHash = h
	}

	// Draw scaled heatmap (4x upscale)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4, 4)
	a.DrawImage(r.psoLandscapeImg, op)

	// Draw contour lines on top of the heatmap
	if len(r.psoContourSegs) > 0 {
		for _, seg := range r.psoContourSegs {
			// Major contour levels (every other one) are brighter and thicker
			major := seg.level%2 == 1
			var alpha uint8
			var width float32
			if major {
				alpha = 160
				width = 1.5
			} else {
				alpha = 90
				width = 0.8
			}
			col := color.RGBA{255, 255, 255, alpha}
			vector.StrokeLine(a, seg.x0, seg.y0, seg.x1, seg.y1, width, col, false)
		}
	}

	// Draw peak center crosshairs (only for Gaussian peaks)
	if sa.FitnessFunc == swarm.FitGaussian {
		for p := range sa.FitPeakX {
			px, py := float32(sa.FitPeakX[p]), float32(sa.FitPeakY[p])
			arm := float32(8)
			crossCol := color.RGBA{255, 255, 255, 140}
			vector.StrokeLine(a, px-arm, py, px+arm, py, 1.5, crossCol, false)
			vector.StrokeLine(a, px, py-arm, px, py+arm, 1.5, crossCol, false)
		}
	}

	// Draw global best marker if PSO is active (PSO tracks global best position)
	if ss.PSO != nil && ss.PSOOn {
		st := ss.PSO
		vector.DrawFilledCircle(a, float32(st.GlobalX), float32(st.GlobalY), 8,
			color.RGBA{255, 255, 0, 200}, false)
		vector.StrokeCircle(a, float32(st.GlobalX), float32(st.GlobalY), 12,
			2, color.RGBA{255, 255, 0, 120}, false)
	}

	// Legend in top-left corner of arena
	algoName := swarm.SwarmAlgorithmName(sa.ActiveAlgo)
	fitName := swarm.FitnessLandscapeName(sa.FitnessFunc)
	legendY := 10
	printColoredAt(a, fitName+" ("+algoName+")", 10, legendY, ColorWhiteFaded)
	legendY += 14
	printColoredAt(a, "Low", 10, legendY, color.RGBA{0, 0, 255, 200})
	printColoredAt(a, " -> ", 28, legendY, color.RGBA{200, 200, 200, 180})
	printColoredAt(a, "High", 50, legendY, color.RGBA{255, 0, 0, 200})
	legendY += 14
	printColoredAt(a, locale.T("ui.change_function"), 10, legendY, color.RGBA{180, 180, 180, 160})
	if sa.DynamicLandscape {
		legendY += 14
		printColoredAt(a, locale.T("ui.dynamic"), 10, legendY, color.RGBA{255, 180, 0, 220})
	}
}
