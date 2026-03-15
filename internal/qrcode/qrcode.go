package qrcode

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	stdraw "image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"strings"

	goqr "github.com/skip2/go-qrcode"
	xdraw "golang.org/x/image/draw"
)

// ── Options ───────────────────────────────────────────────────────────────────

type Options struct {
	Size          int
	EC            goqr.RecoveryLevel
	Logo          image.Image
	LogoSize      int     // percent of QR size, e.g. 25
	LogoRadius    float64 // corner radius in px for logo clip
	QuietZone     bool
	LogoPad       bool
	ModulePattern string // square | rounded | dot | diamond
	FinderPattern string // square | rounded | circle | diamond
	ColorFg       color.Color
	ColorBg       color.Color
}

func DefaultOptions() Options {
	return Options{
		Size:          512,
		EC:            goqr.High,
		LogoSize:      25,
		QuietZone:     true,
		LogoPad:       false,
		ModulePattern: "square",
		FinderPattern: "square",
		ColorFg:       color.Black,
		ColorBg:       color.White,
	}
}

// ── PNG generation ────────────────────────────────────────────────────────────

// GenerateQRWithLogo generates a QR code PNG with all options applied.
func GenerateQRWithLogo(text string, opts Options) (image.Image, error) {
	qr, err := goqr.New(text, opts.EC)
	if err != nil {
		return nil, fmt.Errorf("qrcode: %w", err)
	}
	qr.DisableBorder = true

	// Render at 4x size then scale down for anti-aliasing
	const ssScale = 4
	ssOpts := opts
	ssOpts.Size = opts.Size * ssScale
	ssOpts.LogoRadius = opts.LogoRadius * ssScale

	raw, err := renderQR(qr, ssOpts)
	if err != nil {
		return nil, err
	}

	// Scale down to target size using high-quality interpolation
	out := image.NewRGBA(image.Rect(0, 0, opts.Size, opts.Size))
	xdraw.CatmullRom.Scale(out, out.Bounds(), raw, raw.Bounds(), xdraw.Over, nil)

	return out, nil
}

// renderQR does the actual pixel rendering at whatever size opts.Size specifies.
func renderQR(qr *goqr.QRCode, opts Options) (*image.RGBA, error) {
	bitmap := qr.Bitmap()
	modules := len(bitmap)

	quietZone := 0
	if opts.QuietZone {
		quietZone = 4
	}

	total := modules + quietZone*2
	moduleSize := float64(opts.Size) / float64(total)
	offset := float64(quietZone) * moduleSize

	output := image.NewRGBA(image.Rect(0, 0, opts.Size, opts.Size))
	stdraw.Draw(output, output.Bounds(), &image.Uniform{opts.ColorBg}, image.Point{}, stdraw.Src)

	finderOrigins := []image.Point{
		{X: 0, Y: 0},
		{X: modules - 7, Y: 0},
		{X: 0, Y: modules - 7},
	}
	finderZones := finderModuleSet(modules)

	// Draw data modules
	for y, row := range bitmap {
		for x, dark := range row {
			if !dark || finderZones[image.Point{X: x, Y: y}] {
				continue
			}
			px := offset + float64(x)*moduleSize
			py := offset + float64(y)*moduleSize
			drawModule(output, px, py, moduleSize, opts.ModulePattern, opts.ColorFg)
		}
	}

	// Draw finder patterns as grouped units
	for _, o := range finderOrigins {
		fx := offset + float64(o.X)*moduleSize
		fy := offset + float64(o.Y)*moduleSize
		fw := moduleSize * 7
		drawFinderPattern(output, fx, fy, fw, opts.FinderPattern, opts.ColorFg, opts.ColorBg)
	}

	if opts.Logo != nil {
		if err := overlayLogo(output, opts); err != nil {
			return nil, fmt.Errorf("logo: %w", err)
		}
	}

	return output, nil
}

// drawFinderPattern draws a complete 7×7 finder pattern as a single unit.
func drawFinderPattern(dst *image.RGBA, px, py, size float64, pattern string, fg, bg color.Color) {
	m := size / 7 // single module size within finder

	switch pattern {
	case "rounded", "circle":
		r := size * 0.18
		// Outer ring (7×7)
		drawRoundedRectRing(dst, px, py, size, size, r, m, fg, bg)
		// Inner dot (3×3 centred)
		dotX := px + m*2
		dotY := py + m*2
		dotS := m * 3
		dotR := dotS * 0.3
		if pattern == "circle" {
			dotR = dotS * 0.5
		}
		drawRoundedRect(dst, dotX, dotY, dotS, dotS, dotR, fg)

	case "diamond":
		// Outer diamond ring
		drawDiamondRing(dst, px, py, size, m, fg, bg)
		// Inner diamond dot
		dotX := px + m*2
		dotY := py + m*2
		dotS := m * 3
		drawDiamond(dst, dotX, dotY, dotS, fg)

	default: // square
		// Outer border (7×7 filled, then clear inside 5×5)
		x0 := int(math.Round(px))
		y0 := int(math.Round(py))
		x1 := int(math.Round(px + size))
		y1 := int(math.Round(py + size))
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				dst.Set(x, y, fg)
			}
		}
		// Clear 5×5 interior
		ix0 := int(math.Round(px + m))
		iy0 := int(math.Round(py + m))
		ix1 := int(math.Round(px + size - m))
		iy1 := int(math.Round(py + size - m))
		for y := iy0; y < iy1; y++ {
			for x := ix0; x < ix1; x++ {
				dst.Set(x, y, bg)
			}
		}
		// Inner 3×3 dot
		dx0 := int(math.Round(px + m*2))
		dy0 := int(math.Round(py + m*2))
		dx1 := int(math.Round(px + size - m*2))
		dy1 := int(math.Round(py + size - m*2))
		for y := dy0; y < dy1; y++ {
			for x := dx0; x < dx1; x++ {
				dst.Set(x, y, fg)
			}
		}
	}
}

// drawRoundedRectRing draws a rounded rectangle outline of thickness `thickness`.
func drawRoundedRectRing(dst *image.RGBA, px, py, w, h, radius, thickness float64, fg, bg color.Color) {
	// Draw full rounded rect in fg
	drawRoundedRect(dst, px, py, w, h, radius, fg)
	// Punch out inner area in bg
	innerR := math.Max(0, radius-thickness)
	drawRoundedRect(dst, px+thickness, py+thickness, w-thickness*2, h-thickness*2, innerR, bg)
}

// drawDiamondRing draws a diamond outline.
func drawDiamondRing(dst *image.RGBA, px, py, size, thickness float64, fg, bg color.Color) {
	drawDiamond(dst, px, py, size, fg)
	drawDiamond(dst, px+thickness, py+thickness, size-thickness*2, bg)
	// Re-draw inner dot
	dotX := px + size/2 - (size/7)*1.5
	dotY := py + size/2 - (size/7)*1.5
	dotS := (size / 7) * 3
	drawDiamond(dst, dotX, dotY, dotS, fg)
}

// finderModuleSet returns coordinates of all modules belonging to the three
// finder patterns (top-left, top-right, bottom-left).
func finderModuleSet(modules int) map[image.Point]bool {
	set := make(map[image.Point]bool)
	origins := []image.Point{
		{X: 0, Y: 0},
		{X: modules - 7, Y: 0},
		{X: 0, Y: modules - 7},
	}
	for _, o := range origins {
		for dy := 0; dy < 7; dy++ {
			for dx := 0; dx < 7; dx++ {
				set[image.Point{X: o.X + dx, Y: o.Y + dy}] = true
			}
		}
	}
	return set
}

// drawModule draws a single module at pixel position (px, py).
func drawModule(dst *image.RGBA, px, py, size float64, pattern string, c color.Color) {
	switch pattern {
	case "dot":
		drawCircle(dst, px+size/2, py+size/2, size/2*0.85, c)
	case "circle":
		drawCircle(dst, px+size/2, py+size/2, size/2*0.92, c)
	case "rounded":
		drawRoundedRect(dst, px, py, size, size, size*0.3, c)
	case "diamond":
		drawDiamond(dst, px, py, size, c)
	default: // square
		x0 := int(math.Round(px))
		y0 := int(math.Round(py))
		x1 := int(math.Round(px + size))
		y1 := int(math.Round(py + size))
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				dst.Set(x, y, c)
			}
		}
	}
}

func drawCircle(dst *image.RGBA, cx, cy, r float64, c color.Color) {
	x0 := int(math.Floor(cx - r))
	y0 := int(math.Floor(cy - r))
	x1 := int(math.Ceil(cx + r))
	y1 := int(math.Ceil(cy + r))
	r2 := r * r
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= r2 {
				dst.Set(x, y, c)
			}
		}
	}
}

func drawRoundedRect(dst *image.RGBA, px, py, w, h, radius float64, c color.Color) {
	r := radius
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	x0 := int(math.Floor(px))
	y0 := int(math.Floor(py))
	x1 := int(math.Ceil(px + w))
	y1 := int(math.Ceil(py + h))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if inRoundedRect(float64(x)+0.5, float64(y)+0.5, px, py, w, h, r) {
				dst.Set(x, y, c)
			}
		}
	}
}

func inRoundedRect(fx, fy, rx, ry, w, h, r float64) bool {
	if fx < rx || fx > rx+w || fy < ry || fy > ry+h {
		return false
	}
	// Find the nearest corner centre to this pixel
	cx := rx + r
	if fx > rx+w-r {
		cx = rx + w - r
	}
	cy := ry + r
	if fy > ry+h-r {
		cy = ry + h - r
	}
	// Only apply circle test if pixel is in a corner region
	inCornerX := fx < rx+r || fx > rx+w-r
	inCornerY := fy < ry+r || fy > ry+h-r
	if inCornerX && inCornerY {
		dx := fx - cx
		dy := fy - cy
		if dx*dx+dy*dy > r*r {
			return false
		}
	}
	return true
}

func drawDiamond(dst *image.RGBA, px, py, size float64, c color.Color) {
	cx := px + size/2
	cy := py + size/2
	half := size / 2 * 0.9
	x0 := int(math.Floor(px))
	y0 := int(math.Floor(py))
	x1 := int(math.Ceil(px + size))
	y1 := int(math.Ceil(py + size))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := math.Abs(float64(x)+0.5-cx) / half
			dy := math.Abs(float64(y)+0.5-cy) / half
			if dx+dy <= 1.0 {
				dst.Set(x, y, c)
			}
		}
	}
}

// ── Logo overlay ──────────────────────────────────────────────────────────────

func overlayLogo(dst *image.RGBA, opts Options) error {
	logoSize := opts.Size * opts.LogoSize / 100
	logoBounds := image.Rect(0, 0, logoSize, logoSize)

	resized := image.NewRGBA(logoBounds)
	xdraw.BiLinear.Scale(resized, logoBounds, opts.Logo, opts.Logo.Bounds(), xdraw.Over, nil)

	center := opts.Size / 2
	cx := center - logoSize/2
	cy := center - logoSize/2

	if opts.LogoPad {
		moduleSize := float64(opts.Size) / float64(opts.Size/8)
		pad := int(math.Round(moduleSize * 2))
		bgRect := image.Rect(cx-pad, cy-pad, cx+logoSize+pad, cy+logoSize+pad)
		stdraw.Draw(dst, bgRect, &image.Uniform{color.White}, image.Point{}, stdraw.Src)
	}

	offset := image.Point{X: cx, Y: cy}
	stdraw.Draw(dst, logoBounds.Add(offset), resized, image.Point{}, stdraw.Over)

	return nil
}

// ── SVG generation ────────────────────────────────────────────────────────────

func GenerateQRSVG(text string, opts Options) (string, error) {
	qr, err := goqr.New(text, opts.EC)
	if err != nil {
		return "", fmt.Errorf("qrcode: %w", err)
	}
	qr.DisableBorder = true

	bitmap := qr.Bitmap()
	modules := len(bitmap)

	quietZone := 0
	if opts.QuietZone {
		quietZone = 4
	}

	moduleSize := float64(opts.Size) / float64(modules+quietZone*2)
	offset := float64(quietZone) * moduleSize

	fgHex := colorToHex(opts.ColorFg)
	bgHex := colorToHex(opts.ColorBg)

	finderOrigins := []image.Point{
		{X: 0, Y: 0},
		{X: modules - 7, Y: 0},
		{X: 0, Y: modules - 7},
	}
	finderZones := finderModuleSet(modules)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 %d %d" width="%d" height="%d">`,
		opts.Size, opts.Size, opts.Size, opts.Size,
	))
	sb.WriteString(fmt.Sprintf(`<rect width="100%%" height="100%%" fill="%s"/>`, bgHex))

	// Data modules
	var path strings.Builder
	for y, row := range bitmap {
		for x, module := range row {
			if !module || finderZones[image.Point{X: x, Y: y}] {
				continue
			}
			sx := offset + float64(x)*moduleSize
			sy := offset + float64(y)*moduleSize
			sw := math.Ceil(moduleSize)
			path.WriteString(svgModulePath(sx, sy, sw, opts.ModulePattern))
		}
	}
	sb.WriteString(fmt.Sprintf(`<path d="%s" fill="%s"/>`, path.String(), fgHex))

	// Finder patterns as grouped units
	for _, o := range finderOrigins {
		fx := offset + float64(o.X)*moduleSize
		fy := offset + float64(o.Y)*moduleSize
		fw := moduleSize * 7
		sb.WriteString(svgFinderPattern(fx, fy, fw, moduleSize, opts.FinderPattern, fgHex, bgHex))
	}

	// Logo
	if opts.Logo != nil {
		logoSize := float64(opts.Size) * float64(opts.LogoSize) / 100.0
		padding := moduleSize * 2
		radius := opts.LogoRadius
		cx := float64(opts.Size)/2 - logoSize/2
		cy := float64(opts.Size)/2 - logoSize/2

		sb.WriteString(fmt.Sprintf(
			`<defs><clipPath id="logo-clip"><rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f"/></clipPath></defs>`,
			cx-padding, cy-padding, logoSize+padding*2, logoSize+padding*2, radius,
		))
		sb.WriteString(fmt.Sprintf(
			`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="white" rx="%.2f"/>`,
			cx-padding, cy-padding, logoSize+padding*2, logoSize+padding*2, radius,
		))
		logoB64, mimeType, err := encodeLogoToBase64(opts.Logo)
		if err == nil {
			sb.WriteString(fmt.Sprintf(
				`<image x="%.2f" y="%.2f" width="%.2f" height="%.2f" href="data:%s;base64,%s" clip-path="url(#logo-clip)"/>`,
				cx, cy, logoSize, logoSize, mimeType, logoB64,
			))
		}
	}

	sb.WriteString(`</svg>`)
	return sb.String(), nil
}

// svgFinderPattern returns SVG for a complete 7×7 finder pattern as a single unit.
func svgFinderPattern(px, py, size, m float64, pattern, fgHex, bgHex string) string {
	var sb strings.Builder

	switch pattern {
	case "rounded", "circle":
		outerR := size * 0.18
		innerR := math.Max(0, outerR-m)
		dotS := m * 3
		dotR := dotS * 0.3
		if pattern == "circle" {
			dotR = dotS * 0.5
		}
		dotX := px + m*2
		dotY := py + m*2

		// Outer rounded rect
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
			px, py, size, size, outerR, fgHex))
		// Inner cutout
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
			px+m, py+m, size-m*2, size-m*2, innerR, bgHex))
		// Centre dot
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
			dotX, dotY, dotS, dotS, dotR, fgHex))

	case "diamond":
		hx := px + size/2
		hy := py + size/2
		h := size / 2 * 0.95
		hi := h - m
		hd := m * 3 / 2 * 0.9

		// Outer diamond
		sb.WriteString(fmt.Sprintf(`<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="%s"/>`,
			hx, hy-h, hx+h, hy, hx, hy+h, hx-h, hy, fgHex))
		// Inner cutout
		sb.WriteString(fmt.Sprintf(`<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="%s"/>`,
			hx, hy-hi, hx+hi, hy, hx, hy+hi, hx-hi, hy, bgHex))
		// Centre dot
		sb.WriteString(fmt.Sprintf(`<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="%s"/>`,
			hx, hy-hd, hx+hd, hy, hx, hy+hd, hx-hd, hy, fgHex))

	default: // square
		// Outer filled rect
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			px, py, size, size, fgHex))
		// Inner cutout (5×5)
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			px+m, py+m, size-m*2, size-m*2, bgHex))
		// Centre dot (3×3)
		sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
			px+m*2, py+m*2, m*3, m*3, fgHex))
	}

	return sb.String()
}

// svgModulePath returns an SVG path fragment for a single module.
func svgModulePath(sx, sy, sw float64, pattern string) string {
	switch pattern {
	case "dot":
		r := sw / 2 * 0.85
		cx := sx + sw/2
		cy := sy + sw/2
		return fmt.Sprintf("M%.2f %.2fa%.2f,%.2f 0 1,0 %.2f,0a%.2f,%.2f 0 1,0 %.2f,0Z",
			cx-r, cy, r, r, r*2, r, r, -r*2)

	case "circle":
		r := sw / 2 * 0.92
		cx := sx + sw/2
		cy := sy + sw/2
		return fmt.Sprintf("M%.2f %.2fa%.2f,%.2f 0 1,0 %.2f,0a%.2f,%.2f 0 1,0 %.2f,0Z",
			cx-r, cy, r, r, r*2, r, r, -r*2)

	case "rounded":
		r := sw * 0.3
		// Standard rounded rect: start at top-left after corner, go clockwise
		return fmt.Sprintf(
			"M%.2f %.2f h%.2f a%.2f,%.2f 0 0,1 %.2f,%.2f v%.2f a%.2f,%.2f 0 0,1 %.2f,%.2f h%.2f a%.2f,%.2f 0 0,1 %.2f,%.2f v%.2f a%.2f,%.2f 0 0,1 %.2f,%.2f Z",
			sx+r, sy,
			sw-r*2, r, r, r, r,
			sw-r*2, r, r, -r, r,
			-(sw - r*2), r, r, -r, -r,
			-(sw - r*2), r, r, r, -r,
		)

	case "diamond":
		hx := sx + sw/2
		hy := sy + sw/2
		h := sw / 2 * 0.9
		return fmt.Sprintf("M%.2f %.2fL%.2f %.2fL%.2f %.2fL%.2f %.2fZ",
			hx, hy-h, hx+h, hy, hx, hy+h, hx-h, hy)

	default: // square
		return fmt.Sprintf("M%.0f %.0fh%.0fv%.0fh-%.0fZ", sx, sy, sw, sw, sw)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func colorToHex(c color.Color) string {
	if c == nil {
		return "#000000"
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func encodeLogoToBase64(img image.Image) (string, string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), "image/png", nil
}
