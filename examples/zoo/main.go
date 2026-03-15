package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	adp "github.com/kellydornhaus/layouter/adapters/ebiten"
	text "github.com/kellydornhaus/layouter/adapters/etxt"
	zoo "github.com/kellydornhaus/layouter/examples/zoo/screens"
	"github.com/kellydornhaus/layouter/layout"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomediumitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/gosmallcaps"
	"golang.org/x/image/font/gofont/gosmallcapsitalic"
	"golang.org/x/image/font/sfnt"
)

type windowPreset struct {
	width  int
	height int
}

type game struct {
	ctx     *layout.Context
	screens []zoo.Screen
	idx     int
	header  *layout.PanelComponent
	root    layout.Component
	// draw settings
	showFPSOverlay bool
	lockScreen     bool
	hideChrome     bool
	// fps overlay
	lastUpdTime int64
	lastDrwTime int64
	updSamples  []fpsSample
	drwSamples  []fpsSample
	currFPS     float64 // draw calls per second (avg over window)
	currTPS     float64 // update calls per second (avg over window)
	avgFTms     int     // average update frametime (ms) over last 1s
	maxFTms     int     // max update frametime (ms) over last 1s
	sizePresets []windowPreset
	sizeIdx     int
}

type fpsSample struct {
	t   int64
	fps float64
	dt  float64
}

type gameOptions struct {
	demo        string
	embed       bool
	showFPS     bool
	fpsProvided bool
	logLayout   bool
	logCSS      bool
	logSurfaces bool
}

func newGame(opts gameOptions) *game {
	// Build context with Ebiten renderer + etxt text engine
	rnd := adp.NewRenderer()
	scale := adp.ScaleProvider{}
	// Text engine via etxt
	txt := text.New(scale)
	registerFont := func(name string, data []byte) {
		if name == "" || len(data) == 0 {
			return
		}
		font, err := sfnt.Parse(data)
		if err != nil {
			log.Printf("zoo: failed to parse font %q: %v", name, err)
			return
		}
		txt.RegisterFont(name, font)
	}
	registerFont("medium", gomedium.TTF)
	registerFont("medium-italic", gomediumitalic.TTF)
	registerFont("bold", gobold.TTF)
	registerFont("bold-italic", gobolditalic.TTF)
	registerFont("mono", gomono.TTF)
	registerFont("mono-italic", gomonoitalic.TTF)
	registerFont("smallcaps", gosmallcaps.TTF)
	registerFont("smallcaps-italic", gosmallcapsitalic.TTF)
	ctx := layout.NewContext(scale, rnd, txt)
	ctx.Debug = layout.DebugOptions{
		LogLayoutDecisions:    opts.logLayout,
		LogCSSQueries:         opts.logCSS,
		LogSurfaceAllocations: opts.logSurfaces,
		LogSurfaceUsage:       opts.logSurfaces,
	}

	showFPS := true
	if opts.fpsProvided {
		showFPS = opts.showFPS
	}
	if opts.embed && !opts.fpsProvided {
		showFPS = false
	}

	g := &game{
		ctx:            ctx,
		showFPSOverlay: showFPS,
		lockScreen:     opts.embed,
		hideChrome:     opts.embed,
	}
	g.sizePresets = []windowPreset{
		{width: 1024, height: 768}, // desktop small (scaled UX-friendly 4:3)
		{width: 1180, height: 820}, // iPad landscape scaled down
		{width: 944, height: 656},  // iPad portrait scaled down
		{width: 342, height: 741},  // iPhone portrait scaled down
	}
	if w, h := ebiten.WindowSize(); w > 0 && h > 0 {
		if idx := g.findPresetIndex(w, h); idx >= 0 {
			g.sizeIdx = idx
		} else {
			g.sizePresets = append([]windowPreset{{width: w, height: h}}, g.sizePresets...)
			g.sizeIdx = 0
		}
	}
	g.screens = zoo.NewScreens(ctx)
	g.header = layout.NewLabel("", layout.TextStyle{SizeDp: 14, AlignH: layout.AlignStart, Color: layout.Color{R: 230, G: 230, B: 230, A: 255}})
	if idx, ok := g.findScreenIndex(opts.demo); ok {
		g.idx = idx
	}
	g.rebuildRoot()
	return g
}

func (g *game) Update() error {
	if !g.lockScreen {
		// Keyboard navigation between screens with arrows (wrap around)
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			g.idx = (g.idx + 1) % len(g.screens)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			g.idx = (g.idx - 1 + len(g.screens)) % len(g.screens)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.bumpWindowSize(1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.bumpWindowSize(-1)
		}
		// Toggle fullscreen
		if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			ebiten.SetFullscreen(!ebiten.IsFullscreen())
		}
	}
	g.screens[g.idx].UpdateFrame()

	if g.header != nil && !g.hideChrome {
		w, h := ebiten.WindowSize()
		title := fmt.Sprintf("Layout Zoo — %s  %dx%d", g.screens[g.idx].Name(), w, h)
		if !g.lockScreen {
			title = fmt.Sprintf("Layout Zoo — %s  [←/→ screen, ↑/↓ size, F fullscreen, drag to resize]  %dx%d", g.screens[g.idx].Name(), w, h)
		}
		if g.header.Text() != title {
			g.header.SetText(title)
		}
	}
	// if screen index changed this frame, ensure root composition matches
	// rebuild root every frame is cheap enough for the example, but we keep it conditional
	g.rebuildRoot()

	// Update (TPS) samples — store last 1s of updates
	now := time.Now().UnixNano()
	if g.lastUpdTime != 0 {
		dt := float64(now-g.lastUpdTime) / 1e9
		if dt > 0 {
			g.updSamples = append(g.updSamples, fpsSample{t: now, dt: dt})
		}
	}
	g.lastUpdTime = now
	// drop update samples older than 1s
	cutoff := now - 1_000_000_000
	i := 0
	for i < len(g.updSamples) && g.updSamples[i].t < cutoff {
		i++
	}
	if i > 0 {
		g.updSamples = g.updSamples[i:]
	}
	// compute TPS and frametime stats over windows
	if n := len(g.updSamples); n > 0 {
		// Avg/max update frametime (ms) over last 1s
		sumDt := 0.0
		maxDt := 0.0
		for _, s := range g.updSamples {
			sumDt += s.dt
			if s.dt > maxDt {
				maxDt = s.dt
			}
		}
		avgDt := (sumDt / float64(n)) * 1000.0
		g.avgFTms = int(avgDt + 0.5)
		g.maxFTms = int(maxDt*1000.0 + 0.5)

		// Average TPS over last 300ms
		tpsCutoff := now - 300_000_000
		start := 0
		for start < n && g.updSamples[start].t < tpsCutoff {
			start++
		}
		if start < n {
			firstT := g.updSamples[start].t
			lastT := g.updSamples[n-1].t
			span := float64(lastT-firstT) / 1e9
			frames := n - start
			if frames >= 2 && span > 0 {
				g.currTPS = float64(frames-1) / span
			}
		}
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	// Clear to dark background
	screen.Fill(color.RGBA{5, 5, 10, 255})
	// Draw (FPS) samples — store last 1s of draws
	now := time.Now().UnixNano()
	if g.lastDrwTime != 0 {
		dt := float64(now-g.lastDrwTime) / 1e9
		if dt > 0 {
			g.drwSamples = append(g.drwSamples, fpsSample{t: now, dt: dt})
		}
	}
	g.lastDrwTime = now
	// drop draw samples older than 1s
	cutoff := now - 1_000_000_000
	j := 0
	for j < len(g.drwSamples) && g.drwSamples[j].t < cutoff {
		j++
	}
	if j > 0 {
		g.drwSamples = g.drwSamples[j:]
	}
	// compute FPS over last 300ms
	if n := len(g.drwSamples); n > 0 {
		fpsCutoff := now - 300_000_000
		start := 0
		for start < n && g.drwSamples[start].t < fpsCutoff {
			start++
		}
		if start < n {
			firstT := g.drwSamples[start].t
			lastT := g.drwSamples[n-1].t
			span := float64(lastT-firstT) / 1e9
			frames := n - start
			if frames >= 2 && span > 0 {
				g.currFPS = float64(frames-1) / span
			}
		}
	}
	// refresh scale each frame (monitor DPI may change)
	g.ctx.Scale = adp.ScaleProvider{}.DeviceScaleFactor()
	canvas := adp.WrapCanvas(screen)
	layout.LayoutAndDraw(g.ctx, g.root, canvas)
	g.drawFPSOverlay(screen)
}

func (g *game) Layout(_, _ int) (int, int) {
	panic("LayoutF should be used; requires Ebiten >= v2.5.0")
}

func (g *game) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	scale := adp.ScaleProvider{}.DeviceScaleFactor()
	if scale <= 0 {
		scale = 1
	}
	g.ctx.Scale = scale
	canvasWidth := math.Ceil(logicWinWidth * scale)
	canvasHeight := math.Ceil(logicWinHeight * scale)
	return canvasWidth, canvasHeight
}

func main() {
	var (
		headless  = flag.Bool("headless", false, "run without a window and capture all screens to PNG")
		outDir    = flag.String("out", "screenshots", "directory to write headless screenshots")
		width     = flag.Int("width", 1024, "canvas width in pixels for headless capture")
		height    = flag.Int("height", 768, "canvas height in pixels for headless capture")
		demo      = flag.String("demo", "", "screen name/slug/index to load")
		embed     = flag.Bool("embed", false, "lock to the selected screen and hide navigation chrome (for web embeds)")
		noFPS     = flag.Bool("no-fps", false, "disable the FPS overlay")
		logLayout = flag.Bool("log-layout", false, "enable verbose layout logging")
		logCSS    = flag.Bool("log-css", false, "enable verbose CSS attribute logging")
		logSurf   = flag.Bool("log-surfaces", false, "log surface allocations and cache usage")
	)
	flag.Parse()

	if *headless {
		if err := runHeadless(*outDir, *width, *height, *logLayout, *logCSS, *logSurf); err != nil {
			log.Fatal(err)
		}
		return
	}

	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(320, 240, 4096, 3072)
	ebiten.SetWindowTitle("Layouter — Component Zoo")
	// Run uncapped without vsync for demo purposes
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	ebiten.SetTPS(128)
	opts := gameOptions{
		demo:        strings.TrimSpace(*demo),
		embed:       *embed,
		showFPS:     !*noFPS,
		fpsProvided: true,
		logLayout:   *logLayout,
		logCSS:      *logCSS,
		logSurfaces: *logSurf,
	}
	if opts.embed {
		opts.showFPS = false
	}
	if err := ebiten.RunGame(newGame(opts)); err != nil {
		log.Fatal(err)
	}
}

func (g *game) rebuildRoot() {
	content := g.screens[g.idx].Root()
	if g.hideChrome {
		g.root = content
		return
	}
	headerBox := layout.NewPanelContainer(g.header, layout.Insets(8))
	headerBox.SetFillWidth(true)
	header := layout.NewPanelContainer(headerBox, layout.Insets(0))
	header.SetFillWidth(true)
	header.SetBackgroundColor(layout.Color{R: 12, G: 12, B: 20, A: 255})
	col := layout.NewVStack(header, content)
	col.Spacing = 8
	col.SetFillWidth(true)
	g.root = col
}

func (g *game) findPresetIndex(w, h int) int {
	for i, preset := range g.sizePresets {
		if preset.width == w && preset.height == h {
			return i
		}
	}
	return -1
}

func (g *game) findScreenIndex(target string) (int, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return 0, false
	}
	if idx, ok := parseScreenIndex(target, len(g.screens)); ok {
		return idx, true
	}
	needle := normalizeScreenKey(target)
	for i, s := range g.screens {
		if normalizeScreenKey(s.Name()) == needle {
			return i, true
		}
	}
	return 0, false
}

func parseScreenIndex(target string, max int) (int, bool) {
	num, err := strconv.Atoi(target)
	if err != nil {
		return 0, false
	}
	if num > 0 && num <= max {
		return num - 1, true
	}
	if num >= 0 && num < max {
		return num, true
	}
	return 0, false
}

func normalizeScreenKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	repl := strings.NewReplacer(
		"—", "-", "–", "-", "_", "-",
		"/", "-", "\\", "-", "&", "-and-",
		"+", "-", "@", "-", " ", "-",
		".", "-", ",", "-", ":", "-", ";", "-",
		"'", "", "\"", "", "(", "-", ")", "-",
		"[", "-", "]", "-", "{", "-", "}", "-",
		"|", "-", "!", "-", "?", "-", "#", "-",
		"%", "-", "~", "-", "`", "-", "^", "-",
		"*", "-", "=", "-",
	)
	name = repl.Replace(name)
	var b strings.Builder
	lastHyphen := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if lastHyphen {
			continue
		}
		b.WriteByte('-')
		lastHyphen = true
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return name
	}
	return slug
}

func (g *game) bumpWindowSize(delta int) {
	if len(g.sizePresets) == 0 {
		return
	}
	g.sizeIdx = (g.sizeIdx + delta + len(g.sizePresets)) % len(g.sizePresets)
	if ebiten.IsFullscreen() {
		ebiten.SetFullscreen(false)
	}
	preset := g.sizePresets[g.sizeIdx]
	ebiten.SetWindowSize(preset.width, preset.height)
}

func (g *game) drawFPSOverlay(screen *ebiten.Image) {
	if g == nil || !g.showFPSOverlay {
		return
	}
	if g.ctx == nil || g.ctx.Text == nil {
		return
	}
	// 5 lines: fps, resolution, device scale, avg frametime, max frametime (no labels, no decimals)
	sw, sh := screen.Size()
	fps := int(g.currFPS + 0.5)
	tps := int(g.currTPS + 0.5)
	scaleStr := fmt.Sprintf("@%dx", int(g.ctx.Scale+0.5))
	// Lines: FPS, TPS, resolution, scale, avg frametime (ms), max frametime (ms)
	txt := fmt.Sprintf("%d\n%d\n%dx%d\n%s\n%d\n%d", fps, tps, sw, sh, scaleStr, g.avgFTms, g.maxFTms)
	style := layout.TextStyle{SizeDp: 12, Color: layout.Color{R: 230, G: 230, B: 230, A: 255}, AlignH: layout.AlignEnd, AlignV: layout.AlignEnd}
	// Measure text in px
	w, h := g.ctx.Text.Measure(txt, style, 0)
	// Box with margin and padding in px
	margin := 8
	pad := 6
	bw := w + pad*2
	bh := h + pad*2
	x := sw - bw - margin
	y := sh - bh - margin
	// draw background box (semi-transparent)
	bg := ebiten.NewImage(bw, bh)
	bg.Fill(color.RGBA{0, 0, 0, 160})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(bg, op)
	// draw text aligned bottom-right within the box
	rect := layout.PxRect{X: x + pad, Y: y + pad, W: w, H: h}
	g.ctx.Text.Draw(adp.WrapCanvas(screen), txt, rect, style)
}

func runHeadless(outDir string, width, height int, logLayout bool, logCSS bool, logSurfaces bool) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("headless: width and height must be positive (got %d x %d)", width, height)
	}
	if outDir == "" {
		outDir = "screenshots"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("headless: create output dir: %w", err)
	}

	// Configure the window so it stays invisible/offscreen while capturing.
	ebiten.SetWindowTitle("Layouter — Component Zoo (headless)")
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetWindowDecorated(false)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowPosition(-width*2, -height*2)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	ebiten.SetTPS(60)

	base := newGame(gameOptions{
		showFPS:     false,
		fpsProvided: true,
		logLayout:   logLayout,
		logCSS:      logCSS,
		logSurfaces: logSurfaces,
	})

	headless := newHeadlessGame(base, outDir, width, height)

	if err := ebiten.RunGameWithOptions(headless, &ebiten.RunGameOptions{InitUnfocused: true}); err != nil {
		return err
	}
	if headless.lastErr != nil {
		return headless.lastErr
	}
	return nil
}

type headlessGame struct {
	base       *game
	outDir     string
	width      int
	height     int
	nextIndex  int
	pass       int
	coldFrame  []time.Duration
	warmFrame  []time.Duration
	coldLayout []time.Duration
	warmLayout []time.Duration
	coldRender []time.Duration
	warmRender []time.Duration
	coldHash   []string
	warmHash   []string
	lastFrame  time.Duration
	lastLayout time.Duration
	lastRender time.Duration
	lastErr    error
}

func newHeadlessGame(base *game, outDir string, width, height int) *headlessGame {
	if base == nil {
		base = newGame(gameOptions{showFPS: false, fpsProvided: true})
	}
	base.ctx.Scale = 1
	n := len(base.screens)
	return &headlessGame{
		base:       base,
		outDir:     outDir,
		width:      width,
		height:     height,
		coldFrame:  make([]time.Duration, n),
		warmFrame:  make([]time.Duration, n),
		coldLayout: make([]time.Duration, n),
		warmLayout: make([]time.Duration, n),
		coldRender: make([]time.Duration, n),
		warmRender: make([]time.Duration, n),
		coldHash:   make([]string, n),
		warmHash:   make([]string, n),
	}
}

func (h *headlessGame) Update() error {
	if h == nil {
		return nil
	}
	if h.lastErr != nil {
		return h.lastErr
	}
	if h.nextIndex >= len(h.base.screens) {
		return ebiten.Termination
	}
	h.base.idx = h.nextIndex
	screen := h.base.screens[h.base.idx]
	screen.UpdateFrame()

	title := fmt.Sprintf("Layout Zoo — %s  [headless capture]  %dx%d", screen.Name(), h.width, h.height)
	if h.base.header.Text() != title {
		h.base.header.SetText(title)
	}
	h.base.rebuildRoot()
	return nil
}

func (h *headlessGame) Draw(screen *ebiten.Image) {
	if h == nil || screen == nil {
		return
	}
	screen.Fill(color.RGBA{5, 5, 10, 255})
	h.base.ctx.Scale = 1
	layoutDur, renderDur := layoutAndDrawTimed(h.base.ctx, h.base.root, adp.WrapCanvas(screen))
	h.lastLayout = layoutDur
	h.lastRender = renderDur
	h.lastFrame = layoutDur + renderDur
}

func (h *headlessGame) Layout(logicWinWidth, logicWinHeight int) (int, int) {
	panic("LayoutF should be used by headlessGame")
}

func (h *headlessGame) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	return float64(h.width), float64(h.height)
}

func (h *headlessGame) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	if h == nil {
		return
	}
	if h.nextIndex >= len(h.base.screens) {
		return
	}
	if h.lastErr != nil {
		return
	}
	if offscreen == nil {
		h.lastErr = fmt.Errorf("headless: nil offscreen buffer")
		return
	}
	if screen != nil && offscreen != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM = geoM
		screen.DrawImage(offscreen, op)
	}
	label := "cold"
	if h.pass == 1 {
		label = "warm"
	}
	baseName := fmt.Sprintf("%02d-%s", h.nextIndex+1, sanitizeFileName(h.base.screens[h.nextIndex].Name()))
	path := filepath.Join(h.outDir, fmt.Sprintf("%s-%s.png", baseName, label))
	hash, err := writePNG(offscreen, path)
	if err != nil {
		h.lastErr = err
		return
	}
	frameMs := h.lastFrame.Seconds() * 1000
	layoutMs := h.lastLayout.Seconds() * 1000
	renderMs := h.lastRender.Seconds() * 1000
	if h.pass == 0 {
		h.coldFrame[h.nextIndex] = h.lastFrame
		h.coldLayout[h.nextIndex] = h.lastLayout
		h.coldRender[h.nextIndex] = h.lastRender
		h.coldHash[h.nextIndex] = hash
		log.Printf("headless: wrote %s (cold %.2fms layout=%.2f render=%.2f)", path, frameMs, layoutMs, renderMs)
		h.pass = 1
		return
	}

	h.warmFrame[h.nextIndex] = h.lastFrame
	h.warmLayout[h.nextIndex] = h.lastLayout
	h.warmRender[h.nextIndex] = h.lastRender
	h.warmHash[h.nextIndex] = hash
	match := h.coldHash[h.nextIndex] == h.warmHash[h.nextIndex]
	log.Printf("headless: wrote %s (warm %.2fms layout=%.2f render=%.2f, match=%t)", path, frameMs, layoutMs, renderMs, match)

	reportPath := filepath.Join(h.outDir, fmt.Sprintf("%s.txt", baseName))
	if err := writeTimingReport(reportPath,
		h.coldFrame[h.nextIndex], h.warmFrame[h.nextIndex],
		h.coldLayout[h.nextIndex], h.warmLayout[h.nextIndex],
		h.coldRender[h.nextIndex], h.warmRender[h.nextIndex],
		h.coldHash[h.nextIndex], h.warmHash[h.nextIndex],
	); err != nil {
		h.lastErr = err
		return
	}
	log.Printf("headless: wrote %s (summary)", reportPath)
	h.pass = 0
	h.nextIndex++
}

func layoutAndDrawTimed(ctx *layout.Context, root layout.Component, canvas layout.Canvas) (time.Duration, time.Duration) {
	if ctx == nil || root == nil || canvas == nil || ctx.Renderer == nil {
		return 0, 0
	}
	ctx.BeginFrameLogAuto()
	defer ctx.EndFrameLog()
	w, h := canvas.SizePx()
	dpSize := layout.PxSize{W: w, H: h}.ToDp(ctx.Scale)
	prevSize := ctx.ViewportSize()
	prevHad := ctx.HasViewport()
	ctx.SetViewportSize(layout.Size{W: dpSize.W, H: dpSize.H})
	defer func() {
		if prevHad {
			ctx.SetViewportSize(prevSize)
		} else {
			ctx.SetViewportSize(layout.Size{})
		}
	}()

	layoutStart := time.Now()
	constraints := layout.Tight(dpSize)
	_ = root.Measure(ctx, constraints)
	root.Layout(ctx, nil, layout.Rect{X: 0, Y: 0, W: dpSize.W, H: dpSize.H})
	layoutDur := time.Since(layoutStart)

	renderStart := time.Now()
	root.DrawTo(ctx, canvas)
	renderDur := time.Since(renderStart)
	return layoutDur, renderDur
}

func sanitizeFileName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	prevDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_' || r == '/' || r == '\\':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "screen"
	}
	return slug
}

func writePNG(img *ebiten.Image, path string) (string, error) {
	if img == nil {
		return "", fmt.Errorf("headless: nil image")
	}
	w, h := img.Size()
	if w == 0 || h == 0 {
		return "", fmt.Errorf("headless: invalid image size %dx%d", w, h)
	}
	pixels := make([]byte, 4*w*h)
	img.ReadPixels(pixels)
	sum := sha256.Sum256(pixels)
	rgba := &image.RGBA{
		Pix:    pixels,
		Stride: 4 * w,
		Rect:   image.Rect(0, 0, w, h),
	}
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("headless: create %s: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, rgba); err != nil {
		return "", fmt.Errorf("headless: encode png %s: %w", path, err)
	}
	return fmt.Sprintf("%x", sum[:]), nil
}

func writeTimingReport(path string,
	coldFrame, warmFrame time.Duration,
	coldLayout, warmLayout time.Duration,
	coldRender, warmRender time.Duration,
	coldHash, warmHash string,
) error {
	match := coldHash == warmHash
	coldFPS := frameRate(coldFrame)
	warmFPS := frameRate(warmFrame)
	content := fmt.Sprintf(
		"cold_ms: %.3f (%.2ffps)\n"+
			"  layout_ms: %.3f\n"+
			"  render_ms: %.3f\n"+
			"warm_ms: %.3f (%.2ffps)\n"+
			"  layout_ms: %.3f\n"+
			"  render_ms: %.3f\n"+
			"match: %t\n"+
			"cold_sha256: %s\n"+
			"warm_sha256: %s\n",
		ms(coldFrame), coldFPS,
		ms(coldLayout),
		ms(coldRender),
		ms(warmFrame), warmFPS,
		ms(warmLayout),
		ms(warmRender),
		match,
		coldHash,
		warmHash,
	)
	return os.WriteFile(path, []byte(content), 0o644)
}

func ms(d time.Duration) float64 { return d.Seconds() * 1000 }

func frameRate(d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return 1 / d.Seconds()
}
