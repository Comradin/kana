package main

import (
	"io"
	"os"

	"fyne.io/fyne/v2"
	"kana/vocab"
)

// errFolder is why a dropped folder cannot be imported.
var errFolder = &vocab.ImportError{Reason: "This is a folder.", Hint: "Drop the word list files themselves."}

// dropImporter imports files dropped on the game window, one import preview
// after another, with the game paused as for the settings. Its fields are
// Fyne-thread-only.
type dropImporter struct {
	gs         *GameState
	statsPanel *StatsPanel
	gameCanvas *GameCanvas
	inputBar   *InputBar
	win        fyne.Window

	queue []fyne.URI // dropped files waiting for their preview
	busy  bool       // a preview or its error is showing
}

// drop queues dropped files and, unless a preview is showing already,
// pauses the game and shows the first. Must run on the Fyne thread.
func (d *dropImporter) drop(uris []fyne.URI) {
	if len(uris) == 0 {
		return
	}
	d.queue = append(d.queue, uris...)
	if d.busy {
		return
	}
	d.busy = true
	d.gs.SetImportOpen(true)
	d.gameCanvas.Refresh()
	d.next()
}

// next shows the next queued file's import preview, or why it cannot be
// imported, on top of any open dialog; after the last file it finishes. An
// open settings dialog shows and preselects an imported list. Must run on
// the Fyne thread.
func (d *dropImporter) next() {
	if len(d.queue) == 0 {
		d.finish()
		return
	}
	uri := d.queue[0]
	d.queue = d.queue[1:]
	data, err := readDropped(uri)
	if err != nil {
		showImportError(err, d.win, d.next)
		return
	}
	importChosen(d.gs, openSettings, uri.Name(), data, d.win, d.next)
}

// finish releases the import's pause and refreshes the status panel. A
// dialog that was open underneath is there again and goes on itself when
// it closes; with none, the game goes on. Must run on the Fyne thread.
func (d *dropImporter) finish() {
	d.busy = false
	d.gs.SetImportOpen(false)
	d.gameCanvas.Refresh()
	d.gs.mu.Lock()
	snap := d.gs.snapshot()
	d.gs.mu.Unlock()
	d.statsPanel.Update(snap)
	if openSettings == nil && !dialogShowing && !d.gs.IsOver() {
		showPendingDialogs(d.gs, d.statsPanel, d.inputBar, d.win)
	}
}

// readDropped reads a dropped file for import, at most one byte more than
// vocab.MaxImportSize so that Inspect rejects a larger one. A folder gives
// errFolder.
func readDropped(uri fyne.URI) ([]byte, error) {
	path := uri.Path()
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, errFolder
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return io.ReadAll(io.LimitReader(fh, vocab.MaxImportSize+1))
}
