package screens

import (
	"fmt"

	"github.com/kellydornhaus/layouter/layout"
	"github.com/kellydornhaus/layouter/xmlui"
)

type scoreboardDemo struct {
	*xmlScreen
	states   []scoreboardState
	stateIdx int
	tick     int

	root     layout.Component
	homeCard *layout.PanelComponent
	awayCard *layout.PanelComponent
}

type scoreboardState struct {
	Venue        string
	HomeTeam     string
	HomeRecord   string
	HomeScore    int
	AwayTeam     string
	AwayRecord   string
	AwayScore    int
	Quarter      string
	Clock        string
	DownDistance string
	Play         string
	Possession   string // "home" or "away"
}

func buildXMLScoreboardDemo(ctx *layout.Context) (Screen, error) {
	demo := &scoreboardDemo{
		states:   demoScoreboardStates(),
		stateIdx: 0,
	}
	demo.xmlScreen = &xmlScreen{
		name:     "Scoreboard XML",
		ctx:      ctx,
		reg:      xmlui.NewRegistry(),
		fsys:     layoutFS(),
		xmlPath:  "screens/layouts/scoreboard_demo.xml",
		cssPaths: []string{"screens/layouts/scoreboard_demo.css", "screens/layouts/base.css"},
	}
	demo.xmlScreen.afterBuild = demo.afterBuild
	if err := demo.reload(); err != nil {
		return nil, err
	}
	return demo, nil
}

func (s *scoreboardDemo) afterBuild(res *xmlui.Result) error {
	s.root = res.ByID["score-root"]
	if s.root == nil {
		return fmt.Errorf("scoreboard demo: missing root panel")
	}
	if comp := res.ByID["home-card"]; comp != nil {
		if card, ok := comp.(*layout.PanelComponent); ok {
			s.homeCard = card
		}
	}
	if comp := res.ByID["away-card"]; comp != nil {
		if card, ok := comp.(*layout.PanelComponent); ok {
			s.awayCard = card
		}
	}
	s.applyState(s.states[0])
	return nil
}

func (s *scoreboardDemo) applyState(st scoreboardState) {
	if s.root == nil {
		return
	}
	layout.SetDialogVariable(s.root, "Venue", st.Venue)
	layout.SetDialogVariable(s.root, "HomeTeam", st.HomeTeam)
	layout.SetDialogVariable(s.root, "HomeRecord", st.HomeRecord)
	layout.SetDialogVariable(s.root, "HomeScore", fmt.Sprintf("%d", st.HomeScore))
	layout.SetDialogVariable(s.root, "AwayTeam", st.AwayTeam)
	layout.SetDialogVariable(s.root, "AwayRecord", st.AwayRecord)
	layout.SetDialogVariable(s.root, "AwayScore", fmt.Sprintf("%d", st.AwayScore))
	layout.SetDialogVariable(s.root, "Quarter", st.Quarter)
	layout.SetDialogVariable(s.root, "Clock", st.Clock)
	layout.SetDialogVariable(s.root, "DownDistance", st.DownDistance)
	layout.SetDialogVariable(s.root, "Play", st.Play)

	if s.homeCard != nil {
		s.result.RemoveClass(s.homeCard, "possession")
	}
	if s.awayCard != nil {
		s.result.RemoveClass(s.awayCard, "possession")
	}
	switch st.Possession {
	case "home":
		if s.homeCard != nil {
			s.result.AddClass(s.homeCard, "possession")
		}
	case "away":
		if s.awayCard != nil {
			s.result.AddClass(s.awayCard, "possession")
		}
	}
}

func (s *scoreboardDemo) UpdateFrame() {
	s.xmlScreen.UpdateFrame()
	if len(s.states) == 0 {
		return
	}
	s.tick++
	if s.tick%180 == 0 { // every ~3 seconds at 60 FPS
		s.stateIdx = (s.stateIdx + 1) % len(s.states)
		s.applyState(s.states[s.stateIdx])
	}
}

func demoScoreboardStates() []scoreboardState {
	return []scoreboardState{
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    0,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    0,
			Quarter:      "1",
			Clock:        "15:00",
			DownDistance: "Kickoff",
			Play:         "Cyclones kickoff sails to the back of the end zone for a touchback.",
			Possession:   "home",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    0,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    0,
			Quarter:      "1",
			Clock:        "12:38",
			DownDistance: "3rd & 6 at ORI 29",
			Play:         "QB Vega steps up and threads a strike to TE Miles for 12 yards and a first down.",
			Possession:   "home",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    3,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    0,
			Quarter:      "1",
			Clock:        "08:21",
			DownDistance: "4th & Goal at CED 6",
			Play:         "Owls settle for 24-yard FG by K Ramirez. Orion leads 3-0.",
			Possession:   "away",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    3,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    0,
			Quarter:      "1",
			Clock:        "05:02",
			DownDistance: "2nd & 3 at CED 44",
			Play:         "Cyclones RB Sutton bursts through the middle for 18 yards into Owls territory.",
			Possession:   "away",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    3,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    7,
			Quarter:      "2",
			Clock:        "13:44",
			DownDistance: "1st & Goal at ORI 4",
			Play:         "Cyclones fake the jet sweep, WR Carter wide open in the flat for the 4-yard TD.",
			Possession:   "home",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    10,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    7,
			Quarter:      "2",
			Clock:        "07:26",
			DownDistance: "2nd & 5 at CED 38",
			Play:         "Owls RB Kyren Jones breaks a tackle and accelerates 38 yards to the house!",
			Possession:   "home",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    10,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    13,
			Quarter:      "3",
			Clock:        "09:55",
			DownDistance: "3rd & Goal at ORI 2",
			Play:         "Cyclones capitalize on short field; QB Lin keeps it on the read option for the go-ahead score.",
			Possession:   "home",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    13,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    13,
			Quarter:      "4",
			Clock:        "11:48",
			DownDistance: "4th & 2 at CED 27",
			Play:         "Ramirez nails the 44-yard field goal to tie the game at 13 apiece.",
			Possession:   "away",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    13,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    16,
			Quarter:      "4",
			Clock:        "06:02",
			DownDistance: "4th & Goal at ORI 6",
			Play:         "Cyclones kicker Chen puts Cedar City back on top with a 24-yard FG.",
			Possession:   "home",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    20,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    16,
			Quarter:      "4",
			Clock:        "01:12",
			DownDistance: "1st & Goal at CED 5",
			Play:         "TRICK PLAY! WR Miles takes the reverse and lobs to TE Parker in the end zone. Owls lead 20-16.",
			Possession:   "away",
		},
		{
			Venue:        "Galaxy Field — Week 6",
			HomeTeam:     "Orion Owls",
			HomeRecord:   "3-2",
			HomeScore:    20,
			AwayTeam:     "Cedar City Cyclones",
			AwayRecord:   "4-1",
			AwayScore:    19,
			Quarter:      "4",
			Clock:        "00:03",
			DownDistance: "2nd & 10 at ORI 32",
			Play:         "Cyclones set up for the 49-yard attempt... GOOD! But time expires—Owls escape 20-19.",
			Possession:   "away",
		},
	}
}
