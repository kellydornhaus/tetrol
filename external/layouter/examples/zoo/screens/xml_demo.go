package screens

import (
	"image/color"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	cmp "github.com/kellydornhaus/layouter/examples/zoo/internal/components"
	"github.com/kellydornhaus/layouter/layout"
	"github.com/kellydornhaus/layouter/xmlui"
)

type xmlScreen struct {
	name       string
	root       layout.Component
	scroll     *cmp.Scroll
	preview    *cmp.ScrollPreview
	result     xmlui.Result
	refs       xmlRefs
	tick       int
	ctx        *layout.Context
	reg        *xmlui.Registry
	fsys       fs.FS
	xmlPath    string
	cssPaths   []string
	xmlMTime   time.Time
	cssMTime   map[string]time.Time
	afterBuild func(*xmlui.Result) error
}

func (s *xmlScreen) Name() string           { return s.name }
func (s *xmlScreen) Root() layout.Component { return s.root }

func newDemoRegistry() *xmlui.Registry {
	reg := xmlui.NewRegistry()
	// Custom BG element: <BG color="#RRGGBB">...</BG>
	reg.Register("BG", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) {
		col := parseHexColor(n.Attrs["color"], color.RGBA{20, 20, 20, 255})
		children, err := l.BuildChildren(n)
		if err != nil {
			return nil, err
		}
		var child layout.Component
		if len(children) == 1 {
			child = children[0]
		} else {
			child = layout.NewVStack(children...)
		}
		panel := layout.NewPanelComponent(child)
		panel.SetBackgroundColor(toRGBA(col.R, col.G, col.B, col.A))
		panel.SetFillWidth(true)
		return panel, nil
	})
	// Particle field element: <Particles color rate width height>
	reg.Register("Particles", func(l *xmlui.Loader, n *xmlui.Node) (layout.Component, error) {
		field := cmp.NewParticleField()
		if c := strings.TrimSpace(n.Attrs["color"]); c != "" {
			col := parseHexColor(c, color.RGBA{120, 200, 255, 255})
			field.SetParticleColor(toRGBA(col.R, col.G, col.B, col.A))
		}
		if v := strings.TrimSpace(n.Attrs["rate"]); v != "" {
			rate := parseDpAttr(v)
			if rate > 0 {
				field.SetEmissionRate(rate)
			}
		}
		if v := strings.TrimSpace(n.Attrs["gravity"]); v != "" {
			field.SetGravity(parseDpAttr(v))
		}
		if v := strings.TrimSpace(n.Attrs["radius"]); v != "" {
			field.SetParticleRadius(parseDpAttr(v))
		}
		if v := strings.TrimSpace(n.Attrs["max"]); v != "" {
			field.SetMaxParticles(parseInt(v, 320))
		}
		if w := strings.TrimSpace(n.Attrs["width"]); w != "" {
			field.SetPreferredWidth(parseDpAttr(w))
		}
		if h := strings.TrimSpace(n.Attrs["height"]); h != "" {
			field.SetPreferredHeight(parseDpAttr(h))
		}
		field.SetFillWidth(true)
		if pad := strings.TrimSpace(n.Attrs["padding"]); pad != "" {
			field.SetPadding(parseInsetsAttr(pad))
		}
		if bg := strings.TrimSpace(n.Attrs["background"]); bg != "" {
			c := parseHexColor(bg, color.RGBA{})
			field.SetBackgroundColor(toRGBA(c.R, c.G, c.B, c.A))
		} else if bg := strings.TrimSpace(n.Attrs["background-color"]); bg != "" {
			c := parseHexColor(bg, color.RGBA{})
			field.SetBackgroundColor(toRGBA(c.R, c.G, c.B, c.A))
		}
		borderColorAttr := strings.TrimSpace(n.Attrs["border-color"])
		borderWidthAttr := strings.TrimSpace(n.Attrs["border-width"])
		if borderColorAttr != "" || borderWidthAttr != "" {
			width := parseDpAttr(borderWidthAttr)
			if width <= 0 {
				width = 1
			}
			col := parseHexColor(borderColorAttr, color.RGBA{A: 0})
			if col.A > 0 {
				field.SetBorder(toRGBA(col.R, col.G, col.B, col.A), width)
			} else {
				field.ClearBorder()
			}
		}
		return field, nil
	})

	return reg
}

func buildXMLDemo(ctx *layout.Context) (Screen, error) {
	reg := newDemoRegistry()
	s := &xmlScreen{
		name:     "XML Demo",
		ctx:      ctx,
		reg:      reg,
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/demo.xml",
		cssPaths: []string{"screens/layouts/demo.css", "screens/layouts/base.css"},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func buildXMLAutoFontDemo(ctx *layout.Context) (Screen, error) {
	reg := xmlui.NewRegistry()
	s := &xmlScreen{
		name:     "Auto Font XML",
		ctx:      ctx,
		reg:      reg,
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/font_auto.xml",
		cssPaths: []string{"screens/layouts/font_auto.css"},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func buildXMLAssetsDemo(ctx *layout.Context) (Screen, error) {
	reg := newDemoRegistry()
	s := &xmlScreen{
		name:     "Assets XML",
		ctx:      ctx,
		reg:      reg,
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/assets_demo.xml",
		cssPaths: []string{"screens/layouts/assets_demo.css"},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func buildXMLRadiusDemo(ctx *layout.Context) (Screen, error) {
	reg := xmlui.NewRegistry()
	s := &xmlScreen{
		name:     "Rounded XML",
		ctx:      ctx,
		reg:      reg,
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/radius_demo.xml",
		cssPaths: []string{"screens/layouts/radius_demo.css"},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func buildXMLThemeDemo(ctx *layout.Context) (Screen, error) {
	reg := xmlui.NewRegistry()
	s := &xmlScreen{
		name:     "Theme XML",
		ctx:      ctx,
		reg:      reg,
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/theme_demo.xml",
		cssPaths: []string{"screens/layouts/theme_demo.css"},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func buildXMLFlowStackDemo(ctx *layout.Context) (Screen, error) {
	reg := xmlui.NewRegistry()
	s := &xmlScreen{
		name:     "FlowStack XML",
		ctx:      ctx,
		reg:      reg,
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/flow_stack.xml",
		cssPaths: []string{"screens/layouts/flow_stack.css", "screens/layouts/base.css"},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func buildXMLAlignSelfDemo(ctx *layout.Context) (Screen, error) {
	reg := xmlui.NewRegistry()
	s := &xmlScreen{
		name:     "align-self XML",
		ctx:      ctx,
		reg:      reg,
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/align_self.xml",
		cssPaths: []string{"screens/layouts/align_self.css", "screens/layouts/base.css"},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func parseDpAttr(v string) float64 {
	v = strings.TrimSpace(v)
	if strings.HasSuffix(v, "dp") {
		v = strings.TrimSuffix(v, "dp")
	}
	if v == "" {
		return 0
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return 0
}

func parseInsetsAttr(s string) layout.EdgeInsets {
	s = strings.TrimSpace(s)
	if s == "" {
		return layout.Insets(0)
	}
	parts := strings.Split(s, ",")
	vals := make([]float64, len(parts))
	for i := range parts {
		vals[i] = parseDpAttr(strings.TrimSpace(parts[i]))
	}
	switch len(vals) {
	case 0:
		return layout.Insets(0)
	case 1:
		return layout.Insets(vals[0])
	case 2:
		return layout.EdgeInsets{Top: vals[0], Bottom: vals[0], Left: vals[1], Right: vals[1]}
	default:
		return layout.EdgeInsets{Top: vals[0], Right: vals[1], Bottom: vals[2], Left: vals[3]}
	}
}

func parseInt(s string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return v
}

func parseHexColor(s string, def color.RGBA) color.RGBA {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	if strings.HasPrefix(s, "#") {
		s = s[1:]
	}
	var r, g, b, a uint8 = 0, 0, 0, 255
	if len(s) == 6 || len(s) == 8 {
		if v, err := strconv.ParseUint(s[0:2], 16, 8); err == nil {
			r = uint8(v)
		}
		if v, err := strconv.ParseUint(s[2:4], 16, 8); err == nil {
			g = uint8(v)
		}
		if v, err := strconv.ParseUint(s[4:6], 16, 8); err == nil {
			b = uint8(v)
		}
		if len(s) == 8 {
			if v, err := strconv.ParseUint(s[6:8], 16, 8); err == nil {
				a = uint8(v)
			}
		}
		return color.RGBA{r, g, b, a}
	}
	return def
}

func parseAlign(s string, def layout.TextAlign) layout.TextAlign {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "start", "left":
		return layout.AlignStart
	case "end", "right":
		return layout.AlignEnd
	case "center":
		return layout.AlignCenter
	default:
		return def
	}
}

type xmlRefs struct {
	Title *layout.PanelComponent `ui:"title"`
	Para  *layout.PanelComponent `ui:"para"`
	Row1  *layout.HStack         `ui:"row1"`
	Root  layout.Component       `ui:"themeRoot"`
}

func (s *xmlScreen) UpdateFrame() {
	if s.scroll != nil {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			s.scroll.ScrollBy(-dy * 32)
			if s.preview != nil {
				s.preview.Trigger(s.scroll.Offset())
			}
		} else if s.preview != nil {
			s.preview.Tick()
		}
	} else if s.preview != nil {
		s.preview.Tick()
	}

	s.tick++
	if s.tick%120 == 0 && s.refs.Row1 != nil {
		if s.refs.Row1.Justify == layout.JustifyStart {
			s.refs.Row1.Justify = layout.JustifySpaceBetween
		} else {
			s.refs.Row1.Justify = layout.JustifyStart
		}
		s.refs.Row1.SetDirty()
	}
	if s.refs.Root != nil && s.tick%180 == 0 {
		classes := s.result.Classes(s.refs.Root)
		hasLight := false
		hasDark := false
		for _, cls := range classes {
			if cls == "light" {
				hasLight = true
			}
			if cls == "dark" {
				hasDark = true
			}
		}
		if hasLight {
			s.result.RemoveClass(s.refs.Root, "light")
			s.result.AddClass(s.refs.Root, "dark")
		} else if hasDark {
			s.result.RemoveClass(s.refs.Root, "dark")
			s.result.AddClass(s.refs.Root, "light")
		} else {
			s.result.AddClass(s.refs.Root, "light")
		}
	}
	// Hot reload XML/CSS if modified
	_ = s.checkReload()
}

func (s *xmlScreen) checkReload() error {
	if s.fsys != nil {
		return nil
	}
	xmlInfo, xerr := os.Stat(s.xmlPath)
	xmlChanged := xerr == nil && xmlInfo.ModTime().After(s.xmlMTime)
	cssChanged := false
	for _, cssPath := range s.cssPaths {
		if info, err := os.Stat(cssPath); err == nil {
			if s.cssMTime == nil {
				s.cssMTime = make(map[string]time.Time)
			}
			if info.ModTime().After(s.cssMTime[cssPath]) {
				cssChanged = true
				break
			}
		}
	}
	if xmlChanged || cssChanged {
		if err := s.reload(); err != nil {
			log.Printf("XML/CSS reload failed: %v", err)
			return err
		}
	}
	return nil
}

func (s *xmlScreen) reload() error {
	r, err := buildLayout(s.ctx, s.reg, s.xmlPath, s.fsys)
	if err != nil {
		return err
	}
	if err := s.applyResult(r); err != nil {
		return err
	}
	s.recordModTimes()
	return nil
}

func (s *xmlScreen) applyResult(r xmlui.Result) error {
	s.result = r
	if s.afterBuild != nil {
		if err := s.afterBuild(&s.result); err != nil {
			return err
		}
	}
	var refs xmlRefs
	_ = xmlui.BindByID(&refs, r)
	s.scroll = cmp.NewScroll(r.Root)
	s.scroll.TailPadDp = 32
	s.preview = cmp.NewScrollPreview()
	body := cmp.NewOverlay(s.scroll, s.preview.Painter(s.scroll))
	rootPanel := layout.NewPanelContainer(body, layout.Insets(12))
	rootPanel.SetBackgroundColor(toRGBA(12, 12, 18, 255))
	rootPanel.SetFillWidth(true)
	s.root = rootPanel
	s.refs = refs
	return nil
}

func (s *xmlScreen) recordModTimes() {
	if s.fsys != nil {
		return
	}
	if info, err := os.Stat(s.xmlPath); err == nil {
		s.xmlMTime = info.ModTime()
	}
	if len(s.cssPaths) == 0 {
		return
	}
	if s.cssMTime == nil {
		s.cssMTime = make(map[string]time.Time)
	}
	for _, cssPath := range s.cssPaths {
		if info, err := os.Stat(cssPath); err == nil {
			s.cssMTime[cssPath] = info.ModTime()
		}
	}
}
