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
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	gifFrameSkip = 3   // capture every 3rd frame
	gifMaxFrames = 200 // max GIF frames (10s at 60fps / 3)
	gifDelay     = 5   // centiseconds per frame (50ms → ~20fps playback)
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
		fmt.Println("[SCREENSHOT] Error creating file:", err)
		return ""
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		fmt.Println("[SCREENSHOT] Error encoding PNG:", err)
		return ""
	}

	fmt.Printf("[SCREENSHOT] Saved: %s (%dx%d)\n", fname, w, h)
	return fname
}

// StartRecording initializes GIF recording state on the renderer.
func StartRecording(r *Renderer) {
	r.Recording = true
	r.RecFrames = make([]*image.Paletted, 0, 200)
	r.RecFrameCount = 0
	r.RecSkipCounter = 0
	fmt.Println("[GIF] Recording STARTED")
}

// StopRecording encodes all captured frames into a GIF file.
// Returns the filename on success, empty string on error.
func StopRecording(r *Renderer) string {
	r.Recording = false

	if len(r.RecFrames) == 0 {
		fmt.Println("[GIF] No frames captured")
		return ""
	}

	fname := fmt.Sprintf("swarmsim_%s.gif", time.Now().Format("20060102_150405"))
	f, err := os.Create(fname)
	if err != nil {
		fmt.Println("[GIF] Error creating file:", err)
		r.RecFrames = nil
		return ""
	}
	defer f.Close()

	delays := make([]int, len(r.RecFrames))
	for i := range delays {
		delays[i] = gifDelay
	}

	anim := &gif.GIF{
		Image: r.RecFrames,
		Delay: delays,
	}

	if err := gif.EncodeAll(f, anim); err != nil {
		fmt.Println("[GIF] Error encoding GIF:", err)
		r.RecFrames = nil
		return ""
	}

	fmt.Printf("[GIF] Saved: %s (%d frames)\n", fname, len(r.RecFrames))
	r.RecFrames = nil
	return fname
}

// CaptureGIFFrame captures a frame for GIF recording (every 3rd call).
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

	// Downscale to 50%
	scaled := halfScale(pix, srcW, srcH)

	// Quantize to Plan9 palette
	bounds := scaled.Bounds()
	palettedImg := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(palettedImg, bounds, scaled, image.Point{})

	r.RecFrames = append(r.RecFrames, palettedImg)
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

	// Overlay text (screenshot saved / GIF saved)
	if r.OverlayTimer > 0 {
		r.OverlayTimer--
		alpha := 255
		if r.OverlayTimer < 20 {
			alpha = r.OverlayTimer * 255 / 20
		}
		text := r.OverlayText
		textW := len(text) * 6
		x := sw/2 - textW/2
		y := 5

		// Background for readability
		bgAlpha := uint8(alpha * 180 / 255)
		ebitenutil.DrawRect(screen, float64(x-5), float64(y-2), float64(textW+10), 18,
			color.RGBA{0, 0, 0, bgAlpha})

		// Text with fade
		op := &ebiten.DrawImageOptions{}
		img := cachedTextImage(text)
		op.GeoM.Translate(float64(x), float64(y+1))
		op.ColorScale.ScaleAlpha(float32(alpha) / 255.0)
		screen.DrawImage(img, op)
	}

	// REC indicator (blinking red dot + text)
	if r.Recording {
		r.RecBlinkTick++
		if (r.RecBlinkTick/15)%2 == 0 {
			recText := "* REC"
			recX := sw - len(recText)*6 - 15
			recY := 5
			// Red background
			ebitenutil.DrawRect(screen, float64(recX-4), float64(recY-2), float64(len(recText)*6+8), 18,
				color.RGBA{150, 0, 0, 200})
			printColoredAt(screen, recText, recX, recY, color.RGBA{255, 60, 60, 255})
		}
	} else {
		r.RecBlinkTick = 0
	}
}
