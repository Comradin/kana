package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/store"
	"kana/vocab"
)

// Learning-path line: script label, one segment per step, then the count.
const (
	pathLabelW float32 = 62
	pathCountW float32 = 30
	pathGap    float32 = 2
	pathSegMax float32 = 10
)

// pathSegW returns the segment width that fits the label, count, steps
// segments and their gaps into the grid width, capped at pathSegMax.
func pathSegW(steps int) float32 {
	avail := gridColumns().width() - pathLabelW - pathCountW - pathGap*float32(steps+1)
	return min(float32(math.Floor(float64(avail/float32(steps)))), pathSegMax)
}

// Status panel text: the panel's content is statsTextW wide (300 px minus
// the theme padding on both sides); the words line starts at
// statsWordsTextSize and shrinks down to statsMinTextSize before it is cut.
const (
	statsTextW         float32 = 292
	statsWordsTextSize float32 = 11
	statsMinTextSize   float32 = 9
)

// StatsSnapshot is a lock-free copy of the game state fields needed by the panel.
type StatsSnapshot struct {
	SessionStats  map[string]store.KanaStats
	SelectedRows  map[string]bool
	MissedKanas   []kanacore.Kana
	MissedWords   []vocab.Entry // missed words, in order
	Score         int
	ScoreLimit    int
	Missed        int
	ActiveScripts map[kanacore.Script]bool
	Levels        map[string]int  // learning level per character
	Flagged       map[string]bool // characters that slipped since they were last learned
	Paths         map[kanacore.Script]PathStatus

	LevelLabel string
	Tempo      float32 // shown speed-up, 1 / tempoFactor
	MissLimit  int

	WordCorrect int         // word hits this session
	Words       WordsStatus // the words line
}

// vowelColIndex returns the column index (0–4) for a kana based on its romaji vowel ending.
// Returns -1 if the mapping is unknown.
func vowelColIndex(romaji string) int {
	if len(romaji) == 0 {
		return -1
	}
	switch romaji[len(romaji)-1] {
	case 'a':
		return 0
	case 'i':
		return 1
	case 'u':
		return 2
	case 'e':
		return 3
	case 'o':
		return 4
	}
	// "n" (ん) maps to column 0
	if romaji == "n" {
		return 0
	}
	return -1
}

// extShortLabels are the extended katakana row labels, chosen to differ from
// the basic rows' labels and to fit rowLabelW (at most two characters).
var extShortLabels = map[string]string{
	"ext-t": "ti", "ext-f": "fa", "ext-w": "wi",
	"ext-sh": "je", "ext-v": "va", "ext-ts": "ts",
}

// rowShortLabel returns the short consonant label shown at the left of each row.
func rowShortLabel(rowID string) string {
	base := kanacore.BaseRowID(rowID)
	if strings.HasPrefix(base, "ext-") {
		if lbl, ok := extShortLabels[base]; ok {
			return lbl
		}
		return strings.TrimPrefix(base, "ext-")
	}
	switch base {
	case "vowels":
		return "–"
	case "n-only":
		return "ん"
	case "sy":
		return "sh"
	default:
		return base
	}
}

// StatsPanel shows the session, each active script's learning path and a
// side-by-side hiragana/katakana grid of every character's learning state.
type StatsPanel struct {
	widget.BaseWidget

	correctText, missedText, accuracyText *canvas.Text

	wordsText *canvas.Text // "Words: … introduced · … learned · JLPT N5", or a hint

	pathLines map[kanacore.Script]*fyne.Container
	pathSegs  map[kanacore.Script][]*canvas.Rectangle
	pathCount map[kanacore.Script]*canvas.Text
	pathNext  map[kanacore.Script]*canvas.Text

	cells      map[string]*kanaCell           // by character
	rowCells   map[string][]*kanaCell         // by row ID (hiragana or kata:), in column order
	rowLabels  map[string]*canvas.Text        // by hiragana or extended katakana row ID
	rowFillers map[string][]fyne.CanvasObject // spacer and empty slots of a base or extended row
	groupGrids map[kanacore.Group]*fyne.Container

	content *fyne.Container
}

var (
	segOpenColor  = color.RGBA{R: 0xd8, G: 0xc6, B: 0xaa, A: 0xff}
	panelSubColor = color.RGBA{R: 0x6b, G: 0x54, B: 0x40, A: 0xff}
)

func smallText(s string, size float32) *canvas.Text {
	t := canvas.NewText(s, kanaTextColor)
	t.TextSize = size
	return t
}

func newStatsPanel() *StatsPanel {
	p := &StatsPanel{
		pathLines:  map[kanacore.Script]*fyne.Container{},
		pathSegs:   map[kanacore.Script][]*canvas.Rectangle{},
		pathCount:  map[kanacore.Script]*canvas.Text{},
		pathNext:   map[kanacore.Script]*canvas.Text{},
		cells:      map[string]*kanaCell{},
		rowCells:   map[string][]*kanaCell{},
		rowLabels:  map[string]*canvas.Text{},
		rowFillers: map[string][]fyne.CanvasObject{},
		groupGrids: map[kanacore.Group]*fyne.Container{},
	}

	// Session line: correct · missed · accuracy.
	tile := func(caption string) (*canvas.Text, fyne.CanvasObject) {
		num := smallText("0", 15)
		num.TextStyle = fyne.TextStyle{Bold: true}
		num.Alignment = fyne.TextAlignCenter
		sub := smallText(caption, 11)
		sub.Color = panelSubColor
		sub.Alignment = fyne.TextAlignCenter
		bg := canvas.NewRectangle(tileFaceColor)
		bg.CornerRadius = 4
		return num, container.NewStack(bg, container.NewVBox(num, sub))
	}
	var c1, c2, c3 fyne.CanvasObject
	p.correctText, c1 = tile("correct")
	p.missedText, c2 = tile("missed")
	p.accuracyText, c3 = tile("accuracy")
	session := container.NewGridWithColumns(3, c1, c2, c3)

	// One path line per script: name, one segment per step, count; "Next" below.
	p.wordsText = smallText("", statsWordsTextSize)
	p.wordsText.Color = panelSubColor
	top := container.NewVBox(session, p.wordsText)
	for _, info := range kanacore.Scripts() {
		steps := len(kanacore.ProgressionStepsFor(info.Script))
		widths := []float32{pathLabelW}
		objs := []fyne.CanvasObject{smallText(info.Label, 12)}
		segW := pathSegW(steps)
		for i := 0; i < steps; i++ {
			seg := canvas.NewRectangle(segOpenColor)
			seg.CornerRadius = 2
			p.pathSegs[info.Script] = append(p.pathSegs[info.Script], seg)
			widths = append(widths, segW)
			objs = append(objs, seg)
		}
		count := smallText(fmt.Sprintf("0/%d", steps), 11)
		count.Color = panelSubColor
		p.pathCount[info.Script] = count
		widths = append(widths, pathCountW)
		objs = append(objs, count)
		line := container.New(&fixedColumns{widths: widths, rowHeight: 14, gap: pathGap}, objs...)
		next := smallText("", 11)
		next.Color = panelSubColor
		p.pathNext[info.Script] = next
		box := container.NewVBox(line, next)
		box.Hide()
		p.pathLines[info.Script] = box
		top.Add(box)
	}

	// Grid header and one fixed-column grid per group.
	head := func(s string) *canvas.Text {
		t := smallText(s, 11)
		t.Alignment = fyne.TextAlignCenter
		t.Color = panelSubColor
		return t
	}
	half := 5*cellW + 4*gridGap
	header := container.New(&fixedColumns{widths: []float32{rowLabelW, half, halfGapW, half}, rowHeight: 14, gap: gridGap},
		smallText("", 11), head("ひらがな"), smallText("", 11), head("カタカナ"))
	// VBox spaces the header and each group grid by theme.Padding() (4px).
	grids := container.NewVBox(header)
	for _, g := range kanacore.GroupsFor(kanacore.ScriptHiragana) {
		var objs []fyne.CanvasObject
		for _, row := range kanacore.RowsInGroup(kanacore.ScriptHiragana, g.Group) {
			lbl := smallText(rowShortLabel(row.ID), 11)
			lbl.Alignment = fyne.TextAlignTrailing
			lbl.Color = panelSubColor
			p.rowLabels[row.ID] = lbl
			objs = append(objs, lbl)
			objs = append(objs, p.half(row)...)
			spacer := canvas.NewRectangle(color.Transparent)
			p.rowFillers[row.ID] = append(p.rowFillers[row.ID], spacer)
			objs = append(objs, spacer)
			kata, _ := kanacore.RowByID("kata:" + row.ID)
			objs = append(objs, p.half(kata)...)
		}
		grid := container.New(gridColumns(), objs...)
		p.groupGrids[g.Group] = grid
		grids.Add(grid)
	}

	// Extended katakana: no hiragana half; entries in order, not by vowel.
	var ext []fyne.CanvasObject
	for _, row := range kanacore.RowsInGroup(kanacore.ScriptKatakana, kanacore.GroupExtended) {
		lbl := smallText(rowShortLabel(row.ID), 11)
		lbl.Alignment = fyne.TextAlignTrailing
		lbl.Color = panelSubColor
		p.rowLabels[row.ID] = lbl
		ext = append(ext, lbl)
		filler := func() fyne.CanvasObject {
			f := canvas.NewRectangle(color.Transparent)
			p.rowFillers[row.ID] = append(p.rowFillers[row.ID], f)
			return f
		}
		for i := 0; i < 6; i++ { // empty hiragana half and spacer
			ext = append(ext, filler())
		}
		for i := 0; i < 5; i++ {
			if i < len(row.Entries) {
				cell := newKanaCell(row.Entries[i].Char)
				p.cells[row.Entries[i].Char] = cell
				p.rowCells[row.ID] = append(p.rowCells[row.ID], cell)
				ext = append(ext, cell)
			} else {
				ext = append(ext, filler())
			}
		}
	}
	extGrid := container.New(gridColumns(), ext...)
	p.groupGrids[kanacore.GroupExtended] = extGrid
	grids.Add(extGrid)

	legend := container.NewHBox()
	for _, item := range []struct {
		label string
		st    cellState
		miss  bool
	}{{"new", cellNew, false}, {"learning", cellLearning, false}, {"learned", cellMastered, false}, {"slipped", cellNew, true}} {
		sw := newKanaCell("")
		sw.set(item.st, item.miss)
		t := smallText(item.label, 11)
		t.Color = panelSubColor
		legend.Add(container.NewGridWrap(fyne.NewSize(12, 12), sw))
		legend.Add(t)
	}

	// 4px right padding keeps the scroll bar off the katakana "o" column;
	// the grid is 288px wide, so the padded content is 292px, exactly
	// matching the panel's 300-2*theme.Padding() width.
	padded := container.New(layout.NewCustomPaddedLayout(0, 0, 0, 4), grids)
	p.content = container.NewBorder(top, legend, nil, nil, container.NewVScroll(padded))
	p.ExtendBaseWidget(p)
	return p
}

// half builds the five cell slots of one row in vowel-column order; slots
// without a kana (yōon i/e) are transparent fillers.
func (p *StatsPanel) half(row kanacore.KanaRow) []fyne.CanvasObject {
	slots := make([]fyne.CanvasObject, 5)
	for _, e := range row.Entries {
		if col := vowelColIndex(e.Romaji); col >= 0 && col < 5 {
			cell := newKanaCell(e.Char)
			p.cells[e.Char] = cell
			p.rowCells[row.ID] = append(p.rowCells[row.ID], cell)
			slots[col] = cell
		}
	}
	base := kanacore.BaseRowID(row.ID)
	for i, s := range slots {
		if s == nil {
			filler := canvas.NewRectangle(color.Transparent)
			p.rowFillers[base] = append(p.rowFillers[base], filler)
			slots[i] = filler
		}
	}
	return slots
}

func (p *StatsPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.content)
}

// MinSize fixes the panel at 300px wide; buildWindow wraps it in
// container.NewPadded, so subtracting the theme padding on both sides keeps
// the overall right column at exactly 300px.
func (p *StatsPanel) MinSize() fyne.Size {
	return fyne.NewSize(300-2*theme.Padding(), p.content.MinSize().Height)
}

// Update maps a snapshot onto the panel.
func (p *StatsPanel) Update(snap StatsSnapshot) {
	correct := snap.WordCorrect
	for _, st := range snap.SessionStats {
		correct += st.CorrectCount
	}
	p.correctText.Text = fmt.Sprint(correct)
	p.missedText.Text = fmt.Sprint(snap.Missed)
	if answered := correct + snap.Missed; answered > 0 {
		p.accuracyText.Text = fmt.Sprintf("%d %%", int(math.Round(100*float64(correct)/float64(answered))))
	} else {
		p.accuracyText.Text = "–"
	}
	p.wordsText.Text = formatWordsLine(snap.Words)
	p.wordsText.TextSize = statsWordsTextSize
	fitText(p.wordsText, statsTextW, statsMinTextSize)

	for _, info := range kanacore.Scripts() {
		box := p.pathLines[info.Script]
		ps, ok := snap.Paths[info.Script]
		if !ok {
			box.Hide()
			continue
		}
		box.Show()
		for i, seg := range p.pathSegs[info.Script] {
			seg.FillColor = segOpenColor
			if i < len(ps.Steps) {
				switch ps.Steps[i] {
				case StepMastered:
					seg.FillColor = cellMasteredColor
				case StepLearning:
					seg.FillColor = cellLearningColor
				}
			}
			seg.Refresh()
		}
		p.pathCount[info.Script].Text = fmt.Sprintf("%d/%d", ps.Unlocked, len(ps.Steps))
		if len(ps.Next) == 0 {
			p.pathNext[info.Script].Text = "All rows unlocked"
		} else {
			labels := make([]string, 0, len(ps.Next))
			for _, id := range ps.Next {
				if row, ok := kanacore.RowByID(id); ok {
					labels = append(labels, row.Label)
				}
			}
			p.pathNext[info.Script].Text = "Next: " + strings.Join(labels, " · ")
		}
	}

	for char, cell := range p.cells {
		cell.set(stateFor(snap.Levels[char]), snap.Flagged[char])
	}

	for _, g := range kanacore.GroupsFor(kanacore.ScriptHiragana) {
		groupVisible := false
		for _, row := range kanacore.RowsInGroup(kanacore.ScriptHiragana, g.Group) {
			hOn := snap.ActiveScripts[kanacore.ScriptHiragana] && snap.SelectedRows[row.ID]
			kOn := snap.ActiveScripts[kanacore.ScriptKatakana] && snap.SelectedRows["kata:"+row.ID]
			show := hOn || kOn
			groupVisible = groupVisible || show
			setVisible(p.rowLabels[row.ID], show)
			for _, f := range p.rowFillers[row.ID] {
				setVisible(f, show)
			}
			for _, c := range p.rowCells[row.ID] {
				setVisible(c, hOn)
			}
			for _, c := range p.rowCells["kata:"+row.ID] {
				setVisible(c, kOn)
			}
		}
		setVisible(p.groupGrids[g.Group], groupVisible)
	}
	extVisible := false
	for _, row := range kanacore.RowsInGroup(kanacore.ScriptKatakana, kanacore.GroupExtended) {
		on := snap.ActiveScripts[kanacore.ScriptKatakana] && snap.SelectedRows[row.ID]
		extVisible = extVisible || on
		setVisible(p.rowLabels[row.ID], on)
		for _, f := range p.rowFillers[row.ID] {
			setVisible(f, on)
		}
		for _, c := range p.rowCells[row.ID] {
			setVisible(c, on)
		}
	}
	setVisible(p.groupGrids[kanacore.GroupExtended], extVisible)

	p.content.Refresh()
}

// SetKanaFont shows the kana grid in the kana font kf.
func (p *StatsPanel) SetKanaFont(kf fontChoice) {
	for _, c := range p.cells {
		setKanaFont(c.text, kf)
		c.text.Refresh()
	}
}

func setVisible(o fyne.CanvasObject, on bool) {
	if on {
		o.Show()
	} else {
		o.Hide()
	}
}
