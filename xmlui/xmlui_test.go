package xmlui

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kellydornhaus/layouter/layout"
)

type fakeTextEngine struct{}

type fakeRenderer struct{}

type fakeSurface struct{ w, h int }

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func (fakeTextEngine) Measure(text string, style layout.TextStyle, maxWidthPx int) (int, int) {
	if style.SizeDp <= 0 {
		return 0, 0
	}
	charCount := len([]rune(text))
	if charCount == 0 {
		return 0, 0
	}
	charWidth := int(math.Round(style.SizeDp * 0.6))
	if charWidth < 1 {
		charWidth = 1
	}
	lineHeight := int(math.Round(style.SizeDp * 1.2))
	if lineHeight < 1 {
		lineHeight = 1
	}
	width := charWidth * charCount
	height := lineHeight
	if maxWidthPx > 0 && width > maxWidthPx {
		lines := (width + maxWidthPx - 1) / maxWidthPx
		width = maxWidthPx
		height = lineHeight * lines
	}
	return width, height
}

func (fakeTextEngine) Draw(dst layout.Surface, text string, rectPx layout.PxRect, style layout.TextStyle) {
}

func (fakeRenderer) NewSurface(w, h int) layout.Surface                                  { return &fakeSurface{w: w, h: h} }
func (fakeRenderer) DrawSurface(dst layout.Surface, src layout.Surface, x, y int)        {}
func (fakeRenderer) FillRect(dst layout.Surface, rect layout.PxRect, color layout.Color) {}
func (fakeRenderer) TintRect(dst layout.Surface, rect layout.PxRect, color layout.Color) {}
func (fakeRenderer) DrawImage(dst layout.Surface, img layout.Image, rect layout.PxRect)  {}

func (s *fakeSurface) SizePx() (int, int) { return s.w, s.h }
func (s *fakeSurface) Clear()             {}

type countingRenderer struct {
	news []*fakeSurface
}

func (r *countingRenderer) NewSurface(w, h int) layout.Surface {
	s := &fakeSurface{w: w, h: h}
	r.news = append(r.news, s)
	return s
}
func (countingRenderer) DrawSurface(dst layout.Surface, src layout.Surface, x, y int)        {}
func (countingRenderer) FillRect(dst layout.Surface, rect layout.PxRect, color layout.Color) {}
func (countingRenderer) TintRect(dst layout.Surface, rect layout.PxRect, color layout.Color) {}
func (countingRenderer) DrawImage(dst layout.Surface, img layout.Image, rect layout.PxRect)  {}

type fakeImage struct{ path string }

func (f *fakeImage) SizePx() (int, int) { return 1, 1 }

type ratioImage struct {
	path string
	w    int
	h    int
}

func (r *ratioImage) SizePx() (int, int) { return r.w, r.h }

type recordingImageLoader struct {
	paths []string
}

func (r *recordingImageLoader) LoadImage(path string) (layout.Image, error) {
	r.paths = append(r.paths, path)
	return &fakeImage{path: path}, nil
}

type ratioImageLoader struct {
	w     int
	h     int
	paths []string
}

func (r *ratioImageLoader) LoadImage(path string) (layout.Image, error) {
	r.paths = append(r.paths, path)
	return &ratioImage{path: path, w: r.w, h: r.h}, nil
}

func mustParseStylesheet(css string) *Stylesheet {
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		panic(err)
	}
	return ss
}

func TestParseStylesheetFileWithImport(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.css")
	mainPath := filepath.Join(dir, "main.css")
	if err := os.WriteFile(basePath, []byte(".title { align-h: center; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainCSS := `@import "base.css";
.title { font-size: 18dp; }`
	if err := os.WriteFile(mainPath, []byte(mainCSS), 0o644); err != nil {
		t.Fatal(err)
	}
	ss, err := ParseStylesheetFile(mainPath)
	if err != nil {
		t.Fatalf("ParseStylesheetFile error: %v", err)
	}
	node := &Node{Name: "Panel", Attrs: map[string]string{"class": "title"}}
	applyStyles(node, ss)
	if got := node.Attrs["font-size"]; got != "18dp" {
		t.Fatalf("expected font-size 18dp from main.css, got %q", got)
	}
	if got := node.Attrs["align-h"]; got != "center" {
		t.Fatalf("expected align-h center from base.css, got %q", got)
	}
}

func TestLoadStylesheetHelpers(t *testing.T) {
	dir := t.TempDir()
	cssPath := filepath.Join(dir, "styles.css")
	if err := os.WriteFile(cssPath, []byte("Panel { padding-top: 6; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	if _, err := LoadStylesheet(ctx, cssPath, StylesheetOptions{}); err != nil {
		t.Fatalf("LoadStylesheet returned error: %v", err)
	}
	fsys := os.DirFS(dir)
	sheet, err := LoadStylesheetFS(ctx, fsys, "styles.css", StylesheetOptions{})
	if err != nil {
		t.Fatalf("LoadStylesheetFS returned error: %v", err)
	}
	if _, err := ParseStylesheetFS(fsys, "styles.css"); err != nil {
		t.Fatalf("ParseStylesheetFS returned error: %v", err)
	}
	n := &Node{Name: "Panel", Attrs: map[string]string{}}
	applyStyles(n, sheet)
	if pt := n.Attrs["padding-top"]; pt != "6" {
		t.Fatalf("expected padding-top 6 from stylesheet, got %q", pt)
	}
}

func TestImageLoaderFuncAdapter(t *testing.T) {
	called := false
	loader := ImageLoaderFunc(func(path string) (layout.Image, error) {
		called = true
		return nil, nil
	})
	if _, err := loader.LoadImage("foo.png"); err != nil {
		t.Fatalf("LoadImage returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected adapter to invoke underlying function")
	}
}

func TestBuildFileXmlStylesheet(t *testing.T) {
	dir := t.TempDir()
	cssPath := filepath.Join(dir, "styles.css")
	xmlPath := filepath.Join(dir, "layout.xml")
	css := `.title { font-size: 26dp; }`
	xml := `<?xml-stylesheet href="styles.css" type="text/css"?>
<VStack><Panel id="title" class="title" text="Hello"/></VStack>`
	if err := os.WriteFile(cssPath, []byte(css), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xmlPath, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := BuildFile(ctx, xmlPath, nil, Options{})
	if err != nil {
		t.Fatalf("BuildFile error: %v", err)
	}
	panelComp, ok := res.ByID["title"].(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("title component missing panel reference")
	}
	if got := panelComp.PanelRef().TextStyle().SizeDp; got != 26 {
		t.Fatalf("expected font size 26dp from stylesheet, got %.2f", got)
	}
}

func TestBuildFSXmlStylesheet(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shared.css"), []byte(".title { color: navy; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "styles"), 0o755); err != nil {
		t.Fatal(err)
	}
	themeCSS := `@import "../shared.css";
.title { font-size: 20dp; }`
	if err := os.WriteFile(filepath.Join(dir, "styles", "theme.css"), []byte(themeCSS), 0o644); err != nil {
		t.Fatal(err)
	}
	xml := `<?xml-stylesheet href="styles/theme.css"?>
<VStack><Panel id="title" class="title" text="Hello"/></VStack>`
	if err := os.WriteFile(filepath.Join(dir, "layout.xml"), []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &layout.Context{}
	res, err := BuildFS(ctx, os.DirFS(dir), "layout.xml", nil, Options{})
	if err != nil {
		t.Fatalf("BuildFS error: %v", err)
	}
	panelComp, ok := res.ByID["title"].(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("title component missing panel reference")
	}
	if got := panelComp.PanelRef().TextStyle().SizeDp; got != 20 {
		t.Fatalf("expected font size 20dp, got %.2f", got)
	}
	expectedColor := layout.Color{R: 0x00, G: 0x00, B: 0x80, A: 0xFF}
	if got := panelComp.PanelRef().TextStyle().Color; got != expectedColor {
		t.Fatalf("expected color %+v from shared.css, got %+v", expectedColor, got)
	}
}
func TestStylesheetParsingAndSpecificity(t *testing.T) {
	css := `
    Panel { font-size: 14dp; }
    .title { font-size: 18dp; }
    #hero { font-size: 20dp; }
    Panel.title { color: #FF0000; }
    `
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.Rules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(ss.Rules))
	}
}

func TestApplyStylesPrecedence(t *testing.T) {
	n := &Node{Name: "Panel", Attrs: map[string]string{"id": "hero", "class": "title", "style": "font-size: 19dp;"}}
	css := `Panel { font-size: 10dp; } .title { font-size: 18dp; } #hero { font-size: 20dp; }`
	ss, _ := ParseStylesheet(strings.NewReader(css))
	applyStyles(n, ss)
	if n.Attrs["font-size"] != "19dp" {
		t.Fatalf("inline style should win, got %q", n.Attrs["font-size"])
	}
}

func TestBuildWithCSS(t *testing.T) {
	xml := `<VStack class="page"><Panel id="title" class="title" text="Hello"/></VStack>`
	css := `.page { spacing: 12dp; } .title { font-size: 22dp; }`
	ss, _ := ParseStylesheet(strings.NewReader(css))
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	c := res.ByID["title"]
	if c == nil {
		t.Fatalf("missing title component")
	}
	provider, ok := c.(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("component does not expose panel")
	}
	if provider.PanelRef().TextStyle().SizeDp != 22 {
		t.Fatalf("expected size 22 from CSS, got %v", provider.PanelRef().TextStyle().SizeDp)
	}
}

func TestRootClassDescendantSelector(t *testing.T) {
	xml := `<VStack><Panel id="hud" text="Hello"/></VStack>`
	css := `.mode-hard Panel { font-size: 32dp; }`
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{
		Styles:      ss,
		RootClasses: []string{"mode-hard"},
	})
	if err != nil {
		t.Fatal(err)
	}
	comp := res.ByID["hud"]
	if comp == nil {
		t.Fatalf("missing hud component")
	}
	panel, ok := comp.(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("component does not expose panel")
	}
	if got := panel.PanelRef().TextStyle().SizeDp; got != 32 {
		t.Fatalf("expected font size 32dp, got %.2f", got)
	}
}

func TestMultiLevelDescendantSelector(t *testing.T) {
	xml := `<VStack class="mode-hard"><Panel class="status"><Panel id="detail"/></Panel></VStack>`
	css := `.mode-hard .status Panel { padding: 5dp; }`
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	comp := res.ByID["detail"]
	if comp == nil {
		t.Fatalf("missing detail component")
	}
	panel, ok := comp.(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("component does not expose panel")
	}
	expected := layout.Insets(5)
	if panel.PanelRef().Padding != expected {
		t.Fatalf("expected padding %+v, got %+v", expected, panel.PanelRef().Padding)
	}
}

func TestVisibilityAttributeApplied(t *testing.T) {
	xml := `<VStack><Panel id="secret" visibility="hidden" text="Hidden"/></VStack>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp := res.ByID["secret"]
	if comp == nil {
		t.Fatalf("missing component secret")
	}
	if layout.VisibilityOf(comp) != layout.VisibilityHidden {
		t.Fatalf("expected visibility hidden, got %s", layout.VisibilityOf(comp))
	}
}

func TestVisibilityCssApplied(t *testing.T) {
	xml := `<VStack><Panel id="collapsible" class="toggle" text="Gone"/></VStack>`
	css := `.toggle { visibility: collapse; }`
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	comp := res.ByID["collapsible"]
	if comp == nil {
		t.Fatalf("missing component collapsible")
	}
	if layout.VisibilityOf(comp) != layout.VisibilityCollapse {
		t.Fatalf("expected visibility collapse, got %s", layout.VisibilityOf(comp))
	}
}

func TestVisibilityTransitionAttributeApplied(t *testing.T) {
	xml := `<VStack><Panel id="secret" visibility-transition="size 2s 40%"/></VStack>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp := res.ByID["secret"]
	if comp == nil {
		t.Fatalf("missing component secret")
	}
	trans, ok := layout.VisibilityTransitionOf(comp)
	if !ok {
		t.Fatalf("expected transition to be configured")
	}
	if trans.Mode != layout.VisibilityTransitionSize {
		t.Fatalf("unexpected transition mode %v", trans.Mode)
	}
	if trans.Duration != 2*time.Second {
		t.Fatalf("expected duration 2s, got %v", trans.Duration)
	}
	if math.Abs(trans.Scale-0.4) > 1e-6 {
		t.Fatalf("expected scale 0.4, got %.3f", trans.Scale)
	}
}

func TestImageSrcAttributeLoadsImage(t *testing.T) {
	loader := &recordingImageLoader{}
	xml := `<Image id="icon" src="icon.png"/>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{ImageLoader: loader})
	if err != nil {
		t.Fatal(err)
	}
	imgComp, ok := res.ByID["icon"].(*layout.ImageComponent)
	if !ok {
		t.Fatalf("icon component missing image type")
	}
	if imgComp.Source == nil {
		t.Fatalf("expected image source to be set")
	}
	if src, ok := imgComp.Source.(*fakeImage); !ok || src.path != "icon.png" {
		t.Fatalf("expected fake image with path icon.png, got %#v", imgComp.Source)
	}
	if len(loader.paths) != 1 || loader.paths[0] != "icon.png" {
		t.Fatalf("unexpected loader paths %v", loader.paths)
	}
}

func TestImageHeightAutoAttribute(t *testing.T) {
	loader := &ratioImageLoader{w: 400, h: 200}
	xml := `<Image id="icon" src="icon.png" width="120" height="auto"/>`
	ctx := &layout.Context{Scale: 1}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{ImageLoader: loader})
	if err != nil {
		t.Fatal(err)
	}
	imgComp, ok := res.ByID["icon"].(*layout.ImageComponent)
	if !ok {
		t.Fatalf("icon component missing image type")
	}
	size := imgComp.Measure(ctx, layout.Constraints{})
	if !almostEqual(size.W, 120) {
		t.Fatalf("expected width 120dp, got %.2f", size.W)
	}
	if !almostEqual(size.H, 60) {
		t.Fatalf("expected height 60dp for height=auto, got %.2f", size.H)
	}
	if len(loader.paths) != 1 || loader.paths[0] != "icon.png" {
		t.Fatalf("unexpected loader paths %v", loader.paths)
	}
}

func TestImageWidthAutoAttribute(t *testing.T) {
	loader := &ratioImageLoader{w: 400, h: 200}
	xml := `<Image id="icon" src="icon.png" width="auto" height="90"/>`
	ctx := &layout.Context{Scale: 1}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{ImageLoader: loader})
	if err != nil {
		t.Fatal(err)
	}
	imgComp, ok := res.ByID["icon"].(*layout.ImageComponent)
	if !ok {
		t.Fatalf("icon component missing image type")
	}
	size := imgComp.Measure(ctx, layout.Constraints{})
	if !almostEqual(size.H, 90) {
		t.Fatalf("expected height 90dp, got %.2f", size.H)
	}
	if !almostEqual(size.W, 180) {
		t.Fatalf("expected width 180dp for width=auto, got %.2f", size.W)
	}
	if len(loader.paths) != 1 || loader.paths[0] != "icon.png" {
		t.Fatalf("unexpected loader paths %v", loader.paths)
	}
}

func TestImageSrcResolvedRelativeToFile(t *testing.T) {
	loader := &recordingImageLoader{}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ui.xml"), []byte(`<Image id="pic" src="art/pic.png"/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := BuildFile(ctx, filepath.Join(dir, "ui.xml"), nil, Options{ImageLoader: loader})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.ByID["pic"].(*layout.ImageComponent); !ok {
		t.Fatalf("missing image component")
	}
	expected := filepath.Clean(filepath.Join(dir, "art", "pic.png"))
	if len(loader.paths) != 1 || loader.paths[0] != expected {
		t.Fatalf("expected path %q, got %v", expected, loader.paths)
	}
}

func TestDynamicClassRestyle(t *testing.T) {
	xml := `<VStack id="root" class="baseline"><Panel id="title" text="Hello"/></VStack>`
	css := `.baseline Panel { font-size: 18dp; }
.night Panel { font-size: 32dp; }`
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	titleComp, ok := res.ByID["title"].(*layout.PanelComponent)
	if !ok {
		t.Fatalf("title component missing panel component type")
	}
	if classes := res.ClassesByID("root"); len(classes) != 1 || classes[0] != "baseline" {
		t.Fatalf("expected baseline class, got %v", classes)
	}
	if got := titleComp.PanelRef().TextStyle().SizeDp; !almostEqual(got, 18) {
		t.Fatalf("expected base font-size 18, got %.2f", got)
	}
	if res.HasClassByID("root", "night") {
		t.Fatalf("root should not start with night class")
	}
	if !res.AddClassByID("root", "night") {
		t.Fatalf("expected AddClassByID to report added class")
	}
	if res.AddClassByID("root", "night") {
		t.Fatalf("adding duplicate class should report false")
	}
	if !res.HasClassByID("root", "night") {
		t.Fatalf("expected night class to be present")
	}
	classes := res.ClassesByID("root")
	if len(classes) != 2 || classes[0] != "baseline" || classes[1] != "night" {
		t.Fatalf("unexpected classes %v", classes)
	}
	if got := titleComp.PanelRef().TextStyle().SizeDp; !almostEqual(got, 32) {
		t.Fatalf("expected font-size 32 after class, got %.2f", got)
	}
	if !res.RemoveClassByID("root", "night") {
		t.Fatalf("expected RemoveClassByID to report removal")
	}
	if res.HasClassByID("root", "night") {
		t.Fatalf("night class should be removed")
	}
	classes = res.ClassesByID("root")
	if len(classes) != 1 || classes[0] != "baseline" {
		t.Fatalf("expected baseline class after removal, got %v", classes)
	}
	if got := titleComp.PanelRef().TextStyle().SizeDp; !almostEqual(got, 18) {
		t.Fatalf("expected font-size 18 after removal, got %.2f", got)
	}
}

func TestDynamicClassVisibility(t *testing.T) {
	xml := `<VStack id="root"><Panel id="box" text="Hidden"/></VStack>`
	css := `.invisible #box { visibility: hidden; }`
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	comp := res.ByID["box"]
	if comp == nil {
		t.Fatalf("missing box component")
	}
	if layout.VisibilityOf(comp) != layout.VisibilityVisible {
		t.Fatalf("expected initial visibility visible")
	}
	if !res.AddClassByID("root", "invisible") {
		t.Fatalf("expected invisible class to be added")
	}
	if layout.VisibilityOf(comp) != layout.VisibilityHidden {
		t.Fatalf("expected hidden visibility after class, got %s", layout.VisibilityOf(comp))
	}
	if !res.RemoveClassByID("root", "invisible") {
		t.Fatalf("expected invisible class removal to succeed")
	}
	if layout.VisibilityOf(comp) != layout.VisibilityVisible {
		t.Fatalf("expected visibility visible after removal, got %s", layout.VisibilityOf(comp))
	}
}

func TestImageSrcFromCSS(t *testing.T) {
	xml := `<VStack id="root" class="baseline"><Image id="icon"/></VStack>`
	css := `.baseline #icon { src: url(day.png); }
.night #icon { src: url(night.png); }`
	loader := &recordingImageLoader{}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: mustParseStylesheet(css), ImageLoader: loader})
	if err != nil {
		t.Fatal(err)
	}
	icon, ok := res.ByID["icon"].(*layout.ImageComponent)
	if !ok {
		t.Fatalf("missing image component")
	}
	if src, ok := icon.Source.(*fakeImage); !ok || src.path != "day.png" {
		t.Fatalf("expected day image, got %#v", icon.Source)
	}
	if !res.AddClassByID("root", "night") {
		t.Fatalf("expected night class addition")
	}
	if src, ok := icon.Source.(*fakeImage); !ok || src.path != "night.png" {
		t.Fatalf("expected night image, got %#v", icon.Source)
	}
	if !res.RemoveClassByID("root", "night") {
		t.Fatalf("expected night class removal")
	}
	if src, ok := icon.Source.(*fakeImage); !ok || src.path != "day.png" {
		t.Fatalf("expected day image after removal, got %#v", icon.Source)
	}
	if got := loader.paths; len(got) != 2 || got[0] != "day.png" || got[1] != "night.png" {
		t.Fatalf("unexpected image load sequence %v", got)
	}
}

func TestBackgroundImageFromCSS(t *testing.T) {
	xml := `<VStack id="root" class="card"><Panel id="card"/></VStack>`
	css := `.card #card { background-image: url(card.png); }
.night #card { background-image: url(card-night.png); }`
	loader := &recordingImageLoader{}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: mustParseStylesheet(css), ImageLoader: loader})
	if err != nil {
		t.Fatal(err)
	}
	if !res.AddClassByID("root", "night") {
		t.Fatalf("expected night class addition")
	}
	if !res.RemoveClassByID("root", "night") {
		t.Fatalf("expected night class removal")
	}
	if got := loader.paths; len(got) != 2 || got[0] != "card.png" || got[1] != "card-night.png" {
		t.Fatalf("unexpected background image loads %v", got)
	}
}

func TestNoCacheFromXMLAndCSS(t *testing.T) {
	xml := `<VStack id="root">
		<Panel id="xml" text="hello" nocache="true"/>
		<Panel id="css" text="hello" class="nocache"/>
	</VStack>`
	css := `.nocache { nocache: true; }`

	r := &countingRenderer{}
	ctx := &layout.Context{Scale: 1.0, Renderer: r, Text: fakeTextEngine{}}

	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: mustParseStylesheet(css)})
	if err != nil {
		t.Fatal(err)
	}
	if res.Root == nil {
		t.Fatalf("missing root component")
	}

	res.Root.Layout(ctx, nil, layout.Rect{W: 120, H: 80})
	dst := &fakeSurface{w: 240, h: 160}
	res.Root.DrawTo(ctx, dst)
	res.Root.DrawTo(ctx, dst)

	if len(r.news) != 4 {
		t.Fatalf("expected new cache surface per draw for nocache panels, got %d", len(r.news))
	}
}

func TestBorderEdgeAttributes(t *testing.T) {
	xml := `<Panel id="box" border-top="2dp solid #ff0000" border-right-width="3dp" border-right-color="#00ff00"/>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	panelComp, ok := res.ByID["box"].(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("missing panel component")
	}
	panel := panelComp.PanelRef()
	if !almostEqual(panel.BorderTopWidth(), 2) {
		t.Fatalf("expected top width 2, got %.2f", panel.BorderTopWidth())
	}
	if panel.BorderTopColor() != (layout.Color{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}) {
		t.Fatalf("unexpected top color %+v", panel.BorderTopColor())
	}
	if !almostEqual(panel.BorderRightWidth(), 3) {
		t.Fatalf("expected right width 3, got %.2f", panel.BorderRightWidth())
	}
	if panel.BorderRightColor() != (layout.Color{R: 0x00, G: 0xFF, B: 0x00, A: 0xFF}) {
		t.Fatalf("unexpected right color %+v", panel.BorderRightColor())
	}
}

func TestBorderEdgeCss(t *testing.T) {
	xml := `<VStack id="root" class="state"><Panel id="box"/></VStack>`
	css := `.state #box { border-top-width: 2dp; border-top-color: #ff0000; }
.state #box { border-left: 3dp solid #0000ff; }
.state.alt #box { border-top-width: 0dp; border-left-color: #00ff00; }`
	loader := &recordingImageLoader{}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: mustParseStylesheet(css), ImageLoader: loader})
	if err != nil {
		t.Fatal(err)
	}
	panelComp, ok := res.ByID["box"].(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("missing panel component")
	}
	panel := panelComp.PanelRef()
	if !almostEqual(panel.BorderTopWidth(), 2) {
		t.Fatalf("expected top width 2, got %.2f", panel.BorderTopWidth())
	}
	if panel.BorderLeftColor() != (layout.Color{R: 0x00, G: 0x00, B: 0xFF, A: 0xFF}) {
		t.Fatalf("unexpected left color %+v", panel.BorderLeftColor())
	}
	if !res.AddClassByID("root", "alt") {
		t.Fatalf("expected class add")
	}
	if !almostEqual(panel.BorderTopWidth(), 0) {
		t.Fatalf("expected top width 0, got %.2f", panel.BorderTopWidth())
	}
	if panel.BorderLeftColor() != (layout.Color{R: 0x00, G: 0xFF, B: 0x00, A: 0xFF}) {
		t.Fatalf("unexpected left color after class %+v", panel.BorderLeftColor())
	}
}

func TestPaddingEdgeAttributes(t *testing.T) {
	xml := `<Panel id="box" padding="8" padding-top="12" padding-right="4" padding-bottom="2"/>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	panel, ok := res.ByID["box"].(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("missing panel component")
	}
	pad := panel.PanelRef().Padding
	if !almostEqual(pad.Top, 12) || !almostEqual(pad.Right, 4) || !almostEqual(pad.Bottom, 2) || !almostEqual(pad.Left, 8) {
		t.Fatalf("unexpected padding %+v", pad)
	}
}

func TestPaddingEdgeCss(t *testing.T) {
	xml := `<VStack id="root" class="state"><Panel id="box"/></VStack>`
	css := `.state #box { padding: 6dp; padding-top: 9dp; }
.state.alt #box { padding-right: 3dp; }`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: mustParseStylesheet(css)})
	if err != nil {
		t.Fatal(err)
	}
	panel, ok := res.ByID["box"].(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("missing panel component")
	}
	if pad := panel.PanelRef().Padding; !almostEqual(pad.Top, 9) || !almostEqual(pad.Right, 6) {
		t.Fatalf("unexpected padding %+v", pad)
	}
	res.AddClassByID("root", "alt")
	if pad := panel.PanelRef().Padding; !almostEqual(pad.Right, 3) {
		t.Fatalf("expected right padding 3, got %+v", pad)
	}
}

func TestComponentClassRestyle(t *testing.T) {
	xml := `<VStack><Panel id="box" text="Hi"/></VStack>`
	css := `.highlight { color: #FF0000; }`
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	panelComp, ok := res.ByID["box"].(*layout.PanelComponent)
	if !ok {
		t.Fatalf("box component missing panel")
	}
	if color := panelComp.PanelRef().TextStyle().Color; color != (layout.Color{}) {
		t.Fatalf("expected default color, got %+v", color)
	}
	if !res.AddClassByID("box", "highlight") {
		t.Fatalf("expected highlight class addition")
	}
	if color := panelComp.PanelRef().TextStyle().Color; color != (layout.Color{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}) {
		t.Fatalf("expected highlight color, got %+v", color)
	}
	if !res.RemoveClassByID("box", "highlight") {
		t.Fatalf("expected highlight class removal")
	}
	if color := panelComp.PanelRef().TextStyle().Color; color != (layout.Color{}) {
		t.Fatalf("expected color reset after removal, got %+v", color)
	}
}

func TestPanelFontSizeAttribute(t *testing.T) {
	xml := `<Panel id="title" font-size="24dp" text="Hello"/>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	c := res.ByID["title"]
	if c == nil {
		t.Fatalf("missing title component")
	}
	panel, ok := c.(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("component does not expose panel")
	}
	if panel.PanelRef().TextStyle().SizeDp != 24 {
		t.Fatalf("expected font-size 24, got %.2f", panel.PanelRef().TextStyle().SizeDp)
	}
}

func TestPanelFontAutoSize(t *testing.T) {
	xml := `<Panel id="fit" font-size="12" font-autosize="true" text="Hello"/>`
	ctx := &layout.Context{Scale: 1, Text: fakeTextEngine{}}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	c := res.ByID["fit"]
	if c == nil {
		t.Fatalf("missing fit component")
	}
	panel, ok := c.(*layout.PanelComponent)
	if !ok {
		t.Fatalf("component does not expose panel component")
	}
	constraints := layout.Constraints{Max: layout.Size{W: 120, H: 40}}
	panel.Measure(ctx, constraints)
	panel.Layout(ctx, nil, layout.Rect{W: 120, H: 40})
	autoSize := panel.AutoTextSize()
	if autoSize <= 0 {
		t.Fatalf("expected auto text size > 0")
	}
	if autoSize < 30 || autoSize > 34 {
		t.Fatalf("expected auto text size around 33, got %.2f", autoSize)
	}
}

func TestPanelFontAutoSizeMax(t *testing.T) {
	xml := `<Panel id="fit" font-size="12" font-autosize="true" font-autosize-max="18" text="Hello"/>`
	ctx := &layout.Context{Scale: 1, Text: fakeTextEngine{}}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	c := res.ByID["fit"]
	if c == nil {
		t.Fatalf("missing fit component")
	}
	panel, ok := c.(*layout.PanelComponent)
	if !ok {
		t.Fatalf("component does not expose panel component")
	}
	constraints := layout.Constraints{Max: layout.Size{W: 300, H: 200}}
	panel.Measure(ctx, constraints)
	panel.Layout(ctx, nil, layout.Rect{W: 300, H: 200})
	autoSize := panel.AutoTextSize()
	if math.Abs(autoSize-18) > 0.1 {
		t.Fatalf("expected auto text size capped at 18, got %.2f", autoSize)
	}
}

func TestNamedColorAttributes(t *testing.T) {
	xml := `<Panel id="title" color="rebeccapurple" text="Hello"/>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	c := res.ByID["title"]
	if c == nil {
		t.Fatalf("missing title component")
	}
	panel, ok := c.(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("component does not expose panel")
	}
	expected := layout.Color{R: 0x66, G: 0x33, B: 0x99, A: 0xFF}
	if got := panel.PanelRef().TextStyle().Color; got != expected {
		t.Fatalf("expected color %+v, got %+v", expected, got)
	}
}

func TestNamedColorCSS(t *testing.T) {
	xml := `<VStack><Panel id="title" text="Hello"/></VStack>`
	css := `Panel { color: navy; }`
	ss, _ := ParseStylesheet(strings.NewReader(css))
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	c := res.ByID["title"]
	if c == nil {
		t.Fatalf("missing title component")
	}
	panel, ok := c.(interface{ PanelRef() *layout.Panel })
	if !ok {
		t.Fatalf("component does not expose panel")
	}
	expected := layout.Color{R: 0x00, G: 0x00, B: 0x80, A: 0xFF}
	if got := panel.PanelRef().TextStyle().Color; got != expected {
		t.Fatalf("expected color %+v, got %+v", expected, got)
	}
}

func TestPanelWidthHeightAttributes(t *testing.T) {
	xml := `<Panel id="box" width="120dp" height="48dp"/>`
	ctx := &layout.Context{Scale: 1.0, Renderer: &fakeRenderer{}, Text: fakeTextEngine{}}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	pc, ok := res.ByID["box"].(*layout.PanelComponent)
	if !ok {
		t.Fatalf("component missing panel component type")
	}
	size := pc.Measure(ctx, layout.Constraints{})
	if !almostEqual(size.W, 120) || !almostEqual(size.H, 48) {
		t.Fatalf("expected size 120x48, got %+v", size)
	}
}

func TestPanelCornerRadiusAttributes(t *testing.T) {
	xml := `<Panel id="card" corner-radius="10dp 20dp 30dp 40dp" corner-top-left-radius="12dp"/>`
	ctx := &layout.Context{Scale: 1.0, Renderer: &fakeRenderer{}, Text: fakeTextEngine{}}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	pc, ok := res.ByID["card"].(*layout.PanelComponent)
	if !ok {
		t.Fatalf("component missing panel component type")
	}
	radii := pc.PanelRef().CornerRadii()
	if !almostEqual(radii.TopLeft, 12) || !almostEqual(radii.TopRight, 20) || !almostEqual(radii.BottomRight, 30) || !almostEqual(radii.BottomLeft, 40) {
		t.Fatalf("unexpected corner radii %+v", radii)
	}
}

func TestBindByID(t *testing.T) {
	xml := `<VStack><Panel id="title" text="Hi"/></VStack>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var refs struct {
		Title *layout.PanelComponent `ui:"title"`
	}
	if err := BindByID(&refs, res); err != nil {
		t.Fatal(err)
	}
	if refs.Title == nil {
		t.Fatalf("binding failed for title")
	}
}

func TestTemplateInstantiation(t *testing.T) {
	xml := `
<VStack>
  <template id="card">
    <Panel id="tpl-panel" text="Hello"/>
  </template>
  <Panel id="main" text="Base"/>
</VStack>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := res.ByID["tpl-panel"]; exists {
		t.Fatalf("template contents should not be registered in result ByID")
	}
	tmpl, ok := res.Template("card")
	if !ok || tmpl == nil {
		t.Fatalf("missing template card")
	}
	inst, err := tmpl.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	if len(inst.Components) != 1 {
		t.Fatalf("expected 1 component from template, got %d", len(inst.Components))
	}
	panel, ok := inst.Components[0].(*layout.PanelComponent)
	if !ok {
		t.Fatalf("template root is not panel component")
	}
	if panel.PanelRef().Text() != "Hello" {
		t.Fatalf("expected template text Hello, got %q", panel.PanelRef().Text())
	}
	if inst.ByID["tpl-panel"] != panel {
		t.Fatalf("instance ByID missing tpl-panel mapping")
	}
	inst2, err := tmpl.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	if len(inst2.Components) != 1 {
		t.Fatalf("expected 1 component for second instance, got %d", len(inst2.Components))
	}
	if inst2.Components[0] == panel {
		t.Fatalf("expected fresh component per template instantiation")
	}
}

func TestTemplateMissingIDError(t *testing.T) {
	xml := `<VStack><template><Panel/></template></VStack>`
	ctx := &layout.Context{}
	_, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "template missing id") {
		t.Fatalf("expected missing id error, got %v", err)
	}
}

func TestTemplateDuplicateIDError(t *testing.T) {
	xml := `
<VStack>
  <template id="item"><Panel/></template>
  <template id="item"><Panel/></template>
</VStack>`
	ctx := &layout.Context{}
	_, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "duplicate template id") {
		t.Fatalf("expected duplicate template id error, got %v", err)
	}
}

func TestTemplateInstanceStylingWithClasses(t *testing.T) {
	xml := `<VStack><template id="card"><Panel id="tpl" text="hi"/></template></VStack>`
	css := `.highlight { font-size: 18dp; }`
	ss, err := ParseStylesheet(strings.NewReader(css))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: ss})
	if err != nil {
		t.Fatal(err)
	}
	tmpl, ok := res.Template("card")
	if !ok {
		t.Fatalf("template not registered")
	}
	inst, err := tmpl.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	comp, ok := inst.ByID["tpl"]
	if !ok {
		t.Fatalf("template instance missing tpl component")
	}
	panel, ok := comp.(*layout.PanelComponent)
	if !ok {
		t.Fatalf("tpl component not a panel")
	}
	if panel.PanelRef().TextStyle().SizeDp != 0 {
		t.Fatalf("expected default font size 0, got %.2f", panel.PanelRef().TextStyle().SizeDp)
	}
	if !inst.AddClass(panel, "highlight") {
		t.Fatalf("expected AddClass to return true")
	}
	if !almostEqual(panel.PanelRef().TextStyle().SizeDp, 18) {
		t.Fatalf("expected font size 18 after class, got %.2f", panel.PanelRef().TextStyle().SizeDp)
	}
}

func TestDialogVariablesFromXML(t *testing.T) {
	xml := `<VStack id="root"><Panel id="greeting" text="Hello, {{Name}}"/></VStack>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	root := res.ByID["root"]
	if root == nil {
		t.Fatalf("missing root component")
	}
	name := res.ByID["greeting"]
	if name == nil {
		t.Fatalf("missing greeting panel")
	}
	layout.SetDialogVariable(root, "Name", "Joseph")
	panel, ok := name.(*layout.PanelComponent)
	if !ok {
		t.Fatalf("greeting panel type mismatch")
	}
	if got := panel.PanelRef().Text(); got != "Hello, Joseph" {
		t.Fatalf("expected resolved text, got %q", got)
	}
}

func TestParseColorNamed(t *testing.T) {
	def := layout.Color{R: 1, G: 2, B: 3, A: 4}
	tests := map[string]layout.Color{
		"navy":                 {R: 0x00, G: 0x00, B: 0x80, A: 0xFF},
		"Grey":                 {R: 0x80, G: 0x80, B: 0x80, A: 0xFF},
		"transparent":          {},
		"rebeccapurple":        {R: 0x66, G: 0x33, B: 0x99, A: 0xFF},
		"#ff6347":              {R: 0xFF, G: 0x63, B: 0x47, A: 0xFF},
		"40E0D0":               {R: 0x40, G: 0xE0, B: 0xD0, A: 0xFF},
		"rgba(255, 0, 0, 0.1)": {R: 0xFF, G: 0x00, B: 0x00, A: 0x1A},
		"rgb(120, 60, 30)":     {R: 120, G: 60, B: 30, A: 0xFF},
		"rgb(50%, 25%, 0%)":    {R: 0x80, G: 0x40, B: 0x00, A: 0xFF},
		"rgb(255 0 0)":         {R: 0xFF, G: 0x00, B: 0x00, A: 0xFF},
		"rgb(10 20 30 / 0.5)":  {R: 10, G: 20, B: 30, A: 0x80},
		"rgba(25 50 75 / 50%)": {R: 25, G: 50, B: 75, A: 0x80},
		"rgba(200, 150, 100)":  {R: 200, G: 150, B: 100, A: 0xFF},
		"unknown-color":        def,
		" ":                    def,
		"":                     def,
		"magenta":              {R: 0xFF, G: 0x00, B: 0xFF, A: 0xFF},
		"darkslategrey":        {R: 0x2F, G: 0x4F, B: 0x4F, A: 0xFF},
	}
	for input, expected := range tests {
		got := ParseColor(input, def)
		if got != expected {
			t.Fatalf("ParseColor(%q) = %+v, expected %+v", input, got, expected)
		}
	}
}

func TestParseAspectRatio(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
		want  float64
	}{
		{"1", true, 1},
		{"4/3", true, 4.0 / 3.0},
		{"16:9", true, 16.0 / 9.0},
		{"2 : 1", true, 2},
		{"0", false, 0},
		{"", false, 0},
		{"foo", false, 0},
	}
	for _, tc := range tests {
		got, ok := parseAspectRatio(tc.input)
		if ok != tc.ok {
			t.Fatalf("parseAspectRatio(%q) ok=%v, want %v", tc.input, ok, tc.ok)
		}
		if ok && math.Abs(got-tc.want) > 1e-6 {
			t.Fatalf("parseAspectRatio(%q) = %.6f, want %.6f", tc.input, got, tc.want)
		}
	}
}

func TestParseLengthCalcExpression(t *testing.T) {
	l := &Loader{}
	length, ok := l.parseLength("calc(100vh - 150dp)/1.5")
	if !ok {
		t.Fatalf("parseLength calc failed")
	}
	ctx := &layout.Context{}
	ctx.SetViewportSize(layout.Size{W: 400, H: 800})
	got := length.ResolveWidth(ctx, 0)
	expected := (800 - 150) / 1.5
	if math.Abs(got-expected) > 1e-6 {
		t.Fatalf("calc length resolve = %.3f, want %.3f", got, expected)
	}
}

func TestParseLengthCalcWithFunctions(t *testing.T) {
	l := &Loader{}
	ctx := &layout.Context{}
	ctx.SetViewportSize(layout.Size{W: 800, H: 600})
	parent := 400.0

	length, ok := l.parseLength("calc(max(120dp, 50%))")
	if !ok {
		t.Fatalf("parseLength max failed")
	}
	got := length.ResolveWidth(ctx, parent)
	want := math.Max(120, parent*0.5)
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("calc max resolve = %.3f, want %.3f", got, want)
	}

	length, ok = l.parseLength("min(80vw, 250dp)")
	if !ok {
		t.Fatalf("parseLength min failed")
	}
	got = length.ResolveWidth(ctx, parent)
	want = math.Min(ctx.ViewportWidth()*0.8, 250)
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("min resolve = %.3f, want %.3f", got, want)
	}

	length, ok = l.parseLength("clamp(100dp, 60vw, 200dp)/2")
	if !ok {
		t.Fatalf("parseLength clamp failed")
	}
	got = length.ResolveWidth(ctx, parent)
	want = 100.0
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("clamp resolve = %.3f, want %.3f", got, want)
	}

	length, ok = l.parseLength("calc(max(30vw, clamp(50dp, min(60%, 40vh), 300dp)) - 12dp)/1.5")
	if !ok {
		t.Fatalf("parseLength nested functions failed")
	}
	got = length.ResolveWidth(ctx, parent)
	minCandidate := math.Min(parent*0.6, ctx.ViewportHeight()*0.4)
	clamped := math.Min(math.Max(minCandidate, 50), 300)
	nestedValue := math.Max(ctx.ViewportWidth()*0.3, clamped)
	want = (nestedValue - 12) / 1.5
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("nested resolve = %.3f, want %.3f", got, want)
	}
}

func TestParseLengthUnits(t *testing.T) {
	l := &Loader{}
	if length, ok := l.parseLength("15vw"); !ok || length.Unit != layout.LengthUnitViewportWidth || math.Abs(length.Value-0.15) > 1e-6 {
		t.Fatalf("expected 15vw => 0.15 viewport width, got %+v ok=%v", length, ok)
	}
	if length, ok := l.parseLength("0.5vh"); !ok || length.Unit != layout.LengthUnitViewportHeight || math.Abs(length.Value-0.5) > 1e-6 {
		t.Fatalf("expected 0.5vh => 0.5 viewport height, got %+v ok=%v", length, ok)
	}
	if length, ok := l.parseLength("75%"); !ok || length.Unit != layout.LengthUnitPercent || math.Abs(length.Value-0.75) > 1e-6 {
		t.Fatalf("expected 75%% => 0.75 percent, got %+v ok=%v", length, ok)
	}
	if length, ok := l.parseLength("32dp"); !ok || length.Unit != layout.LengthUnitDP || math.Abs(length.Value-32) > 1e-6 {
		t.Fatalf("expected 32dp => 32dp, got %+v ok=%v", length, ok)
	}
	if length, ok := l.parseLength("0.4"); !ok || length.Unit != layout.LengthUnitPercent || math.Abs(length.Value-0.4) > 1e-6 {
		t.Fatalf("expected bare 0.4 => percent, got %+v ok=%v", length, ok)
	}
	if length, ok := l.parseLength("30vmin"); !ok || length.Unit != layout.LengthUnitViewportMin || math.Abs(length.Value-0.3) > 1e-6 {
		t.Fatalf("expected 30vmin => 0.3 viewport min, got %+v ok=%v", length, ok)
	}
	if length, ok := l.parseLength("0.2vmax"); !ok || length.Unit != layout.LengthUnitViewportMax || math.Abs(length.Value-0.2) > 1e-6 {
		t.Fatalf("expected 0.2vmax => 0.2 viewport max, got %+v ok=%v", length, ok)
	}
}

func TestParseHelpers(t *testing.T) {
	if v := ParseFloat("18dp", 0); !almostEqual(v, 18) {
		t.Fatalf("ParseFloat mismatch, got %.2f", v)
	}
	if b := ParseBool("true", false); !b {
		t.Fatalf("ParseBool expected true")
	}
	if insets, ok := ParseInsets("4,8,12,16"); !ok || !almostEqual(insets.Top, 4) || !almostEqual(insets.Left, 16) {
		t.Fatalf("ParseInsets mismatch: %+v ok=%v", insets, ok)
	}
	if length, ok := ParseLength("25dp"); !ok || !length.Defined() {
		t.Fatalf("ParseLength dp failed: %+v ok=%v", length, ok)
	}
	if length, ok := ParseLength("50%"); !ok || length.Unit != layout.LengthUnitPercent {
		t.Fatalf("ParseLength percent failed: %+v ok=%v", length, ok)
	}
	col := ParseColor("rgba(34,68,102,0.5)", layout.Color{})
	if col.A == 0 {
		t.Fatalf("ParseColor expected non-zero alpha")
	}
}

func TestStyleResolverMatchesApplyStyles(t *testing.T) {
	css := `
Panel { background-color: #224466; tint-color: rgba(255, 64, 32, 0.25); padding-top: 8; width: 120dp; height: 48dp; }
Panel.tile { border-width: 2dp; border-color: #ff00ff; fill-width: true; max-width: 120dp; }
VBox Panel { font-size: 18dp; }
`
	ss := mustParseStylesheet(css)

	node := &Node{Name: "Panel", Attrs: map[string]string{"class": "tile"}}
	node.initClassSet()
	ancestor := &Node{Name: "VBox"}
	applyStylesWithAncestors(node, ss, []*Node{ancestor})

	resolver := NewStyleResolver(ss)
	style := resolver.Resolve("VBox Panel.tile", nil)

	expectedColor := ParseColor(node.Attrs["background-color"], layout.Color{})
	if got := style.Color("background-color", layout.Color{}); got != expectedColor {
		t.Fatalf("background color mismatch: got %+v expected %+v", got, expectedColor)
	}
	expectedTint := ParseColor(node.Attrs["tint-color"], layout.Color{})
	if got := style.Color("tint-color", layout.Color{}); got != expectedTint {
		t.Fatalf("tint-color mismatch: got %+v expected %+v", got, expectedTint)
	}
	if got := style.Float("border-width", 0); !almostEqual(got, 2) {
		t.Fatalf("expected border width 2, got %.2f", got)
	}
	expectedBorder := ParseColor(node.Attrs["border-color"], layout.Color{})
	if got := style.Color("border-color", layout.Color{}); got != expectedBorder {
		t.Fatalf("border color mismatch")
	}
	if got := style.String("font-size", ""); got != "18dp" {
		t.Fatalf("font-size expected 18dp, got %q", got)
	}
	if _, ok := style.Insets("padding"); ok {
		t.Fatalf("unexpected padding shorthand present")
	}
	if inset, ok := style.Insets("padding-top"); !ok || !almostEqual(inset.Top, 8) {
		t.Fatalf("expected padding-top 8, got %+v ok=%v", inset, ok)
	}
	if got := style.String("width", ""); got != "120dp" {
		t.Fatalf("expected width 120dp, got %q", got)
	}
	if length, ok := style.Length("height"); !ok || !length.Defined() {
		t.Fatalf("expected height length")
	}
	vals := style.Values()
	vals["border-width"] = "999"
	if style.String("border-width", "") == "999" {
		t.Fatalf("Values should return copy")
	}

	// Ensure ResolveOptions ancestors merge correctly.
	styleWithOpts := resolver.Resolve("Panel.tile", &ResolveOptions{Ancestors: []StyleSelector{{Tag: "VBox"}}})
	if got := styleWithOpts.String("font-size", ""); got != "18dp" {
		t.Fatalf("font-size expected via options, got %q", got)
	}
	if !styleWithOpts.Has("border-width") {
		t.Fatalf("expected border-width to exist")
	}
	if raw, _ := styleWithOpts.Raw("border-color"); raw == "" {
		t.Fatalf("expected raw border color value")
	}
	if !styleWithOpts.Bool("fill-width", false) {
		t.Fatalf("expected fill-width bool true")
	}
	if length, ok := styleWithOpts.Length("max-width"); !ok || !length.Defined() {
		t.Fatalf("expected max-width length")
	}
	if length, ok := styleWithOpts.Length("width"); !ok || !length.Defined() {
		t.Fatalf("expected width length")
	}
}

func TestApplyPanelAttributesTintColor(t *testing.T) {
	l := &Loader{}
	panelComp := layout.NewPanelComponent(nil)

	node := &Node{
		Name:  "Panel",
		Attrs: map[string]string{"tint-color": "rgba(10, 20, 30, 0.5)"},
	}
	l.applyPanelAttributes(node, panelComp)

	panel := panelComp.PanelRef()
	if panel == nil {
		t.Fatalf("panel reference is nil")
	}
	if !panel.HasTint() {
		t.Fatalf("expected HasTint after tint-color attribute applied")
	}
	expected := ParseColor(node.Attrs["tint-color"], layout.Color{})
	if got := panel.TintColor(); got != expected {
		t.Fatalf("tint color mismatch: got %+v expected %+v", got, expected)
	}

	nodeAlias := &Node{
		Name:  "Panel",
		Attrs: map[string]string{"tint": "#8040ff80"},
	}
	l.applyPanelAttributes(nodeAlias, panelComp)
	aliasExpected := ParseColor(nodeAlias.Attrs["tint"], layout.Color{})
	if got := panel.TintColor(); got != aliasExpected {
		t.Fatalf("tint alias color mismatch: got %+v expected %+v", got, aliasExpected)
	}
	if !panel.HasTint() {
		t.Fatalf("expected HasTint to remain true after alias application")
	}

	nodeClear := &Node{Name: "Panel", Attrs: map[string]string{}}
	l.applyPanelAttributes(nodeClear, panelComp)
	if panel.HasTint() {
		t.Fatalf("expected tint cleared when attribute omitted")
	}
}

func TestPositionAttributesFromXML(t *testing.T) {
	xml := `<Panel id="root">
		<Panel id="rel" position="relative" top="4" left="3"/>
		<Panel id="abs" position="absolute" left="12" top="8" z-index="5"/>
	</Panel>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	rel := res.ByID["rel"]
	if rel == nil {
		t.Fatalf("missing relative panel")
	}
	if mode := layout.PositionModeOf(rel); mode != layout.PositionRelative {
		t.Fatalf("expected relative position, got %v", mode)
	}
	offsets := layout.PositionOffsetsOf(rel)
	if !offsets.Top.Defined || !almostEqual(offsets.Top.Value, 4) {
		t.Fatalf("expected top offset 4, got %+v", offsets.Top)
	}
	if !offsets.Left.Defined || !almostEqual(offsets.Left.Value, 3) {
		t.Fatalf("expected left offset 3, got %+v", offsets.Left)
	}
	if z, ok := layout.ZIndexOf(rel); ok || z != 0 {
		t.Fatalf("expected no z-index for relative node, got %d (defined=%v)", z, ok)
	}
	abs := res.ByID["abs"]
	if abs == nil {
		t.Fatalf("missing absolute panel")
	}
	if mode := layout.PositionModeOf(abs); mode != layout.PositionAbsolute {
		t.Fatalf("expected absolute position, got %v", mode)
	}
	absOffsets := layout.PositionOffsetsOf(abs)
	if !absOffsets.Left.Defined || !almostEqual(absOffsets.Left.Value, 12) {
		t.Fatalf("expected absolute left 12, got %+v", absOffsets.Left)
	}
	if !absOffsets.Top.Defined || !almostEqual(absOffsets.Top.Value, 8) {
		t.Fatalf("expected absolute top 8, got %+v", absOffsets.Top)
	}
	if z, ok := layout.ZIndexOf(abs); !ok || z != 5 {
		t.Fatalf("expected z-index 5, got %d (defined=%v)", z, ok)
	}
}

func TestPositionAttributesFromCSS(t *testing.T) {
	css := `.rel { position: relative; top: 6; left: 2; }
	.abs { position: absolute; left: 14; top: 11; z-index: 7; }`
	xml := `<Panel id="root">
		<Panel id="rel" class="rel"/>
		<Panel id="abs" class="abs"/>
	</Panel>`
	ctx := &layout.Context{}
	res, err := Build(ctx, bytes.NewReader([]byte(xml)), nil, Options{Styles: mustParseStylesheet(css)})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	rel := res.ByID["rel"]
	if rel == nil {
		t.Fatalf("missing relative panel")
	}
	if mode := layout.PositionModeOf(rel); mode != layout.PositionRelative {
		t.Fatalf("expected relative position, got %v", mode)
	}
	relOffsets := layout.PositionOffsetsOf(rel)
	if !relOffsets.Top.Defined || !almostEqual(relOffsets.Top.Value, 6) {
		t.Fatalf("expected top offset 6 from CSS, got %+v", relOffsets.Top)
	}
	if !relOffsets.Left.Defined || !almostEqual(relOffsets.Left.Value, 2) {
		t.Fatalf("expected left offset 2 from CSS, got %+v", relOffsets.Left)
	}
	abs := res.ByID["abs"]
	if abs == nil {
		t.Fatalf("missing absolute panel")
	}
	if mode := layout.PositionModeOf(abs); mode != layout.PositionAbsolute {
		t.Fatalf("expected absolute position, got %v", mode)
	}
	absOffsets := layout.PositionOffsetsOf(abs)
	if !absOffsets.Left.Defined || !almostEqual(absOffsets.Left.Value, 14) {
		t.Fatalf("expected absolute left 14 from CSS, got %+v", absOffsets.Left)
	}
	if !absOffsets.Top.Defined || !almostEqual(absOffsets.Top.Value, 11) {
		t.Fatalf("expected absolute top 11 from CSS, got %+v", absOffsets.Top)
	}
	if z, ok := layout.ZIndexOf(abs); !ok || z != 7 {
		t.Fatalf("expected z-index 7 from CSS, got %d (defined=%v)", z, ok)
	}
}
