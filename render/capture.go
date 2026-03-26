package render

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"swarmsim/logger"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// gifMu protects Renderer fields written by the background GIF encoding goroutine
// (GIFEncoding, GIFEncodedFile) from concurrent access on the main/draw thread.
var gifMu sync.Mutex

const (
	gifFrameSkip = 4   // capture every 4th frame (25% of frames)
	gifMaxFrames = 300 // max GIF frames (~10s at 30fps effective capture)
	gifDelay     = 5   // centiseconds per frame (50ms -> ~20fps playback)
)

// CaptureScreenshot saves the current screen as a PNG file.
// Returns the filename on success, empty string on error.
func CaptureScreenshot(screen *ebiten.Image) string {
	w := screen.Bounds().Dx()
	h := screen.Bounds().Dy()

	pix := make([]byte, w*h*4)
	screen.ReadPixels(pix)

	img := &image.RGBA{
		Pix:    pix,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}

	fname := fmt.Sprintf("swarmsim_%s.png", time.Now().Format("20060102_150405"))
	f, err := os.Create(fname)
	if err != nil {
		logger.Error("SCREENSHOT", "Error creating file: %v", err)
		return ""
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		logger.Error("SCREENSHOT", "Error encoding PNG: %v", err)
		return ""
	}

	logger.Info("SCREENSHOT", "Saved: %s (%dx%d)", fname, w, h)
	return fname
}

// StartRecording initializes GIF recording state on the renderer.
func StartRecording(r *Renderer) {
	if r.GIFEncoding {
		return // still encoding previous GIF
	}
	r.Recording = true
	r.RecRawFrames = make([]*image.RGBA, 0, gifMaxFrames)
	r.RecFrameCount = 0
	r.RecSkipCounter = 0
	logger.Info("GIF", "Recording STARTED")
}

// StopRecording stops capturing and encodes all frames in a background goroutine.
// The renderer shows an "Encoding GIF..." overlay until done.
func StopRecording(r *Renderer) {
	r.Recording = false

	if len(r.RecRawFrames) == 0 {
		logger.Warn("GIF", "No frames captured")
		return
	}

	// Move frames out of renderer and start background encoding
	frames := r.RecRawFrames
	r.RecRawFrames = nil
	r.GIFEncoding = true

	go func() {
		fname := encodeGIF(frames)
		// Signal completion back to renderer (checked in Draw)
		gifMu.Lock()
		r.GIFEncodedFile = fname
		r.GIFEncoding = false
		gifMu.Unlock()
	}()
}

// encodeGIF quantizes raw RGBA frames and writes the GIF file.
func encodeGIF(frames []*image.RGBA) string {
	logger.Info("GIF", "Encoding %d frames...", len(frames))

	palettedFrames := make([]*image.Paletted, len(frames))
	delays := make([]int, len(frames))

	for i, raw := range frames {
		bounds := raw.Bounds()
		p := image.NewPaletted(bounds, palette.Plan9)
		draw.FloydSteinberg.Draw(p, bounds, raw, image.Point{})
		palettedFrames[i] = p
		delays[i] = gifDelay
	}

	fname := fmt.Sprintf("swarmsim_%s.gif", time.Now().Format("20060102_150405"))
	f, err := os.Create(fname)
	if err != nil {
		logger.Error("GIF", "Error creating file: %v", err)
		return ""
	}
	defer f.Close()

	anim := &gif.GIF{
		Image: palettedFrames,
		Delay: delays,
	}

	if err := gif.EncodeAll(f, anim); err != nil {
		logger.Error("GIF", "Error encoding GIF: %v", err)
		return ""
	}

	logger.Info("GIF", "Saved: %s (%d frames)", fname, len(frames))
	return fname
}

// CaptureGIFFrame captures a raw RGBA frame for GIF recording (every 4th call).
// Only reads pixels and downscales — no dithering during recording.
// Returns true if max frames reached and recording should auto-stop.
func CaptureGIFFrame(screen *ebiten.Image, r *Renderer) bool {
	r.RecSkipCounter++
	if r.RecSkipCounter < gifFrameSkip {
		return false
	}
	r.RecSkipCounter = 0

	srcW := screen.Bounds().Dx()
	srcH := screen.Bounds().Dy()

	pix := make([]byte, srcW*srcH*4)
	screen.ReadPixels(pix)

	// Stop before exceeding max frames (prevents 301+ frame overruns)
	if r.RecFrameCount >= gifMaxFrames {
		return true
	}

	// Downscale to 50% (fast box averaging, no dithering)
	scaled := halfScale(pix, srcW, srcH)

	r.RecRawFrames = append(r.RecRawFrames, scaled)
	r.RecFrameCount++

	return r.RecFrameCount >= gifMaxFrames
}

// halfScale reduces an RGBA pixel buffer to 50% using 2x2 box averaging.
func halfScale(src []byte, srcW, srcH int) *image.RGBA {
	dstW := srcW / 2
	dstH := srcH / 2
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			sx := x * 2
			sy := y * 2

			// 4 source pixels
			i00 := (sy*srcW + sx) * 4
			i10 := (sy*srcW + sx + 1) * 4
			i01 := ((sy+1)*srcW + sx) * 4
			i11 := ((sy+1)*srcW + sx + 1) * 4

			r := (uint16(src[i00]) + uint16(src[i10]) + uint16(src[i01]) + uint16(src[i11])) / 4
			g := (uint16(src[i00+1]) + uint16(src[i10+1]) + uint16(src[i01+1]) + uint16(src[i11+1])) / 4
			b := (uint16(src[i00+2]) + uint16(src[i10+2]) + uint16(src[i01+2]) + uint16(src[i11+2])) / 4
			a := (uint16(src[i00+3]) + uint16(src[i10+3]) + uint16(src[i01+3]) + uint16(src[i11+3])) / 4

			di := (y*dstW + x) * 4
			dst.Pix[di] = uint8(r)
			dst.Pix[di+1] = uint8(g)
			dst.Pix[di+2] = uint8(b)
			dst.Pix[di+3] = uint8(a)
		}
	}
	return dst
}

// DrawCaptureOverlay renders screenshot/GIF overlay text and the REC indicator.
func DrawCaptureOverlay(screen *ebiten.Image, r *Renderer) {
	sw := screen.Bounds().Dx()

	// Overlay snackbar (screenshot saved / GIF saved)
	if r.OverlayTimer > 0 {
		r.OverlayTimer--
		alpha := 255
		if r.OverlayTimer < 20 {
			alpha = r.OverlayTimer * 255 / 20
		}
		text := r.OverlayText
		charW := 6
		textW := len(text) * charW
		barW := textW + 30
		barH := 28
		x := sw/2 - barW/2
		y := 55

		// Solid dark background with accent border
		bgAlpha := uint8(alpha * 220 / 255)
		borderAlpha := uint8(alpha * 255 / 255)
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(barW), float32(barH),
			color.RGBA{15, 20, 40, bgAlpha}, false)
		vector.DrawFilledRect(screen, float32(x), float32(y+barH-3), float32(barW), 3,
			color.RGBA{80, 200, 120, borderAlpha}, false)

		// Centered white text
		textX := x + (barW-textW)/2
		textY := y + (barH-12)/2
		printColoredAt(screen, text, textX, textY, color.RGBA{240, 245, 255, uint8(alpha)})
	}

	// "Encoding GIF..." overlay with spinner
	gifMu.Lock()
	encoding := r.GIFEncoding
	encodedFile := r.GIFEncodedFile
	if !encoding && encodedFile != "" {
		r.GIFEncodedFile = ""
	}
	gifMu.Unlock()

	if encoding {
		spinChars := []string{"|", "/", "-", "\\"}
		spin := spinChars[(r.RecBlinkTick/8)%4]
		r.RecBlinkTick++
		text := fmt.Sprintf("GIF wird erstellt... %s  (Bitte warten)", spin)
		textW := len(text) * 6
		x := sw/2 - textW/2
		y := 5
		ebitenutil.DrawRect(screen, float64(x-5), float64(y-2), float64(textW+10), 22,
			color.RGBA{0, 0, 80, 220})
		printColoredAt(screen, text, x, y+2, color.RGBA{120, 180, 255, 255})
	}

	// Check if background encoding finished (using mutex-protected snapshot above)
	if !encoding && encodedFile != "" {
		r.OverlayText = "GIF gespeichert: " + encodedFile
		r.OverlayTimer = 90
	}

	// REC indicator (blinking red dot + frame counter)
	if r.Recording {
		r.RecBlinkTick++
		// Always show frame count, blink the red dot
		frameInfo := fmt.Sprintf("REC %d/%d", r.RecFrameCount, gifMaxFrames)
		recX := sw - len(frameInfo)*6 - 20
		recY := 5
		// Red background
		ebitenutil.DrawRect(screen, float64(recX-4), float64(recY-2), float64(len(frameInfo)*6+12), 18,
			color.RGBA{150, 0, 0, 200})
		// Blinking dot
		if (r.RecBlinkTick/15)%2 == 0 {
			printColoredAt(screen, "*", recX, recY, color.RGBA{255, 60, 60, 255})
		}
		printColoredAt(screen, frameInfo, recX+8, recY, color.RGBA{255, 120, 120, 255})
		// Progress bar under REC indicator
		barW := float32(len(frameInfo)*6 + 8)
		progress := float32(r.RecFrameCount) / float32(gifMaxFrames)
		ebitenutil.DrawRect(screen, float64(recX-2), float64(recY+16), float64(barW), 3,
			color.RGBA{60, 0, 0, 200})
		ebitenutil.DrawRect(screen, float64(recX-2), float64(recY+16), float64(barW*progress), 3,
			color.RGBA{255, 60, 60, 255})
	} else {
		r.RecBlinkTick = 0
	}
}
