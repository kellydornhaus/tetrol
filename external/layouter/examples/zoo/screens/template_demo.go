package screens

import (
	"fmt"

	"github.com/kellydornhaus/layouter/layout"
	"github.com/kellydornhaus/layouter/xmlui"
)

type templateShowcase struct {
	*xmlScreen
	cards        []*templateCard
	highlightIdx int
	pulseTick    int
	root         layout.Component
}

type templateCard struct {
	panel    *layout.PanelComponent
	instance xmlui.TemplateInstance
}

func buildXMLTemplateDemo(ctx *layout.Context) (Screen, error) {
	ts := &templateShowcase{
		highlightIdx: -1,
	}
	ts.xmlScreen = &xmlScreen{
		name:     "Templates XML",
		ctx:      ctx,
		reg:      xmlui.NewRegistry(),
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/template_demo.xml",
		cssPaths: []string{"screens/layouts/template_demo.css", "screens/layouts/base.css"},
	}
	ts.xmlScreen.afterBuild = ts.populateRoster
	if err := ts.reload(); err != nil {
		return nil, err
	}
	return ts, nil
}

func (t *templateShowcase) populateRoster(res *xmlui.Result) error {
	gridComp, ok := res.ByID["crew-grid"]
	if !ok {
		return fmt.Errorf("templates demo: missing FlowStack with id crew-grid")
	}
	grid, ok := gridComp.(*layout.FlowStack)
	if !ok {
		return fmt.Errorf("templates demo: crew-grid is not a FlowStack")
	}
	grid.Clear()

	tmpl, ok := res.Template("crew-card")
	if !ok || tmpl == nil {
		return fmt.Errorf("templates demo: missing template crew-card")
	}

	t.cards = t.cards[:0]
	t.highlightIdx = -1
	t.root = res.ByID["template-root"]

	type crewMember struct {
		Name        string
		Role        string
		Shift       string
		Status      string
		CardClass   string
		StatusClass string
	}

	roster := []crewMember{
		{
			Name:        "Commander Vega",
			Role:        "Navigation Lead",
			Shift:       "Sol 42 · 06:00 – 14:00",
			Status:      "ON SHIFT",
			CardClass:   "card-dawn",
			StatusClass: "status-on",
		},
		{
			Name:        "Aki Tanaka",
			Role:        "Habitat Systems | Thermal & air",
			Shift:       "Sol 42 · 14:00 – 22:00",
			Status:      "PRE-FLIGHT CHECKS",
			CardClass:   "card-day",
			StatusClass: "status-prep",
		},
		{
			Name:        "Dr. Imani Rhodes",
			Role:        "Medical & Life Sciences",
			Shift:       "Sol 42 · Standby rotation",
			Status:      "ON CALL",
			CardClass:   "card-night",
			StatusClass: "status-standby",
		},
		{
			Name:        "Milo Ortiz",
			Role:        "Fabrication Deck | Robotics tech",
			Shift:       "Sol 42 · 22:00 – 06:00",
			Status:      "REST CYCLE",
			CardClass:   "card-night",
			StatusClass: "status-rest",
		},
	}

	if t.root != nil {
		layout.SetDialogVariable(t.root, "WatchName", "Sol 42 Night Watch")
		layout.SetDialogVariable(t.root, "RosterSummary", fmt.Sprintf("%d specialists on duty • data supplied via dialog variables", len(roster)))
	}

	for _, member := range roster {
		inst, err := tmpl.Instantiate()
		if err != nil {
			return err
		}
		if len(inst.Components) == 0 {
			continue
		}
		root := inst.Components[0]
		grid.Add(root)

		panel, ok := root.(*layout.PanelComponent)
		if !ok {
			continue
		}

		layout.SetDialogVariable(panel, "Name", member.Name)
		layout.SetDialogVariable(panel, "Role", member.Role)
		layout.SetDialogVariable(panel, "Shift", member.Shift)
		layout.SetDialogVariable(panel, "Status", member.Status)

		if status, ok := inst.ByID["card-status"].(*layout.PanelComponent); ok {
			if member.StatusClass != "" {
				inst.AddClass(status, member.StatusClass)
			}
		}
		if member.CardClass != "" {
			inst.AddClass(panel, member.CardClass)
		}

		t.cards = append(t.cards, &templateCard{
			panel:    panel,
			instance: inst,
		})
	}

	return nil
}

func (t *templateShowcase) Name() string { return t.xmlScreen.Name() }

func (t *templateShowcase) Root() layout.Component { return t.xmlScreen.Root() }

func (t *templateShowcase) UpdateFrame() {
	t.xmlScreen.UpdateFrame()
	if len(t.cards) == 0 {
		return
	}
	t.pulseTick++
	if t.pulseTick%240 == 0 {
		next := t.highlightIdx + 1
		if next >= len(t.cards) {
			next = 0
		}
		if t.highlightIdx >= 0 {
			card := t.cards[t.highlightIdx]
			card.instance.RemoveClass(card.panel, "card-pulse")
		}
		card := t.cards[next]
		card.instance.AddClass(card.panel, "card-pulse")
		t.highlightIdx = next
	}
}
