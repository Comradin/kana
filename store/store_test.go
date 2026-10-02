package store

import (
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestActiveScriptsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.ActiveScripts()
	if err != nil || got != nil {
		t.Fatalf("unset ActiveScripts = %v, %v; want nil, nil", got, err)
	}
	if err := st.SaveActiveScripts([]string{"hiragana", "katakana"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err = st.ActiveScripts()
	if err != nil || len(got) != 2 || got[0] != "hiragana" || got[1] != "katakana" {
		t.Fatalf("ActiveScripts = %v, %v", got, err)
	}
}

func TestKatakanaOfferedRoundTrip(t *testing.T) {
	st := openTemp(t)
	if got, err := st.KatakanaOffered(); err != nil || got {
		t.Fatalf("unset KatakanaOffered = %v, %v; want false, nil", got, err)
	}
	if err := st.SaveKatakanaOffered(true); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got, err := st.KatakanaOffered(); err != nil || !got {
		t.Fatalf("KatakanaOffered = %v, %v; want true", got, err)
	}
}
