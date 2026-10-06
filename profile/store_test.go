package profile

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MATHSKILL_HOME", dir)
	s := Empty()
	id := s.Upsert("  Sam  ")
	if s.Name(id) != "Sam" || !isHexID(id) {
		t.Fatalf("upsert name %q id %q", s.Name(id), id)
	}
	if again := s.Upsert("sam"); again != id || len(s.Profiles) != 1 {
		t.Fatalf("fold %q profiles %d", again, len(s.Profiles))
	}
	if s.Profiles[0].Name != "Sam" {
		t.Fatalf("spelling %q", s.Profiles[0].Name)
	}
	s.SetBest("Sam", "mul:easy", Record{Score: 1200, Gates: 3})
	s.AddRace(id)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	s2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s2.Active != id {
		t.Fatalf("active %q", s2.Active)
	}
	if s2.Name(s2.Active) != "Sam" {
		t.Fatalf("active name %q", s2.Name(s2.Active))
	}
	rec := s2.Best("sam", "mul:easy")
	if rec.Score != 1200 || rec.Gates != 3 {
		t.Fatalf("record %+v", rec)
	}
	if s2.Best(id, "mul:easy") != rec {
		t.Fatal("id lookup")
	}
	if s2.Best("Sam", "add:hard").Score != 0 {
		t.Fatal("missing record")
	}
	if s2.Profiles[0].Races != 1 {
		t.Fatalf("races %d", s2.Profiles[0].Races)
	}
}

func TestLegacyNameActiveMigrates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MATHSKILL_HOME", dir)
	path := filepath.Join(dir, "profiles.json")
	raw := []byte(`{
  "active": "Sam",
  "profiles": [
    {"name": "Sam", "bests": {"add:easy": {"score": 400, "gates": 2}}},
    {"name": "Ada", "races": 3, "bests": {"mul:hard": {"score": 900, "gates": 1}}}
  ]
}
`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != string(raw) {
		t.Fatal("load rewrote the file")
	}
	if s.Name(s.Active) != "Sam" {
		t.Fatalf("active %q name %q", s.Active, s.Name(s.Active))
	}
	if s.Profiles[0].ID == "" || s.Profiles[1].ID == "" || s.Profiles[0].ID == s.Profiles[1].ID {
		t.Fatalf("ids %q %q", s.Profiles[0].ID, s.Profiles[1].ID)
	}
	if s.Best("Ada", "mul:hard").Score != 900 || s.Profiles[1].Races != 3 {
		t.Fatalf("ada %+v", s.Profiles[1])
	}
	hof := s.HallOfFame()
	if len(hof) != 2 || hof[0].Name != "Ada" || hof[1].Name != "Sam" {
		t.Fatalf("hall %+v", hof)
	}
	samID := s.Profiles[0].ID
	adaID := s.Profiles[1].ID
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	s2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s2.Active != samID || s2.Profiles[0].ID != samID || s2.Profiles[1].ID != adaID {
		t.Fatalf("ids did not stick: active %q %+v", s2.Active, s2.Profiles)
	}
}

func TestExistingIDIsKept(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MATHSKILL_HOME", dir)
	path := filepath.Join(dir, "profiles.json")
	raw := []byte(`{"active":"abc","profiles":[{"id":"abc","name":"Sam"}]}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Active != "abc" || s.Profiles[0].ID != "abc" {
		t.Fatalf("active %q id %q", s.Active, s.Profiles[0].ID)
	}
	if s.Profiles[0].Bests == nil {
		t.Fatal("nil bests")
	}
	if s.Best("Sam", "add:easy").Score != 0 {
		t.Fatal("phantom score")
	}
}

func TestHallOfFameSort(t *testing.T) {
	s := Empty()
	bo := s.Upsert("Bo")
	s.SetBest(bo, "div:medium", Record{Score: 900, Gates: 4})
	s.AddRace(bo)
	s.AddRace(bo)
	sam := s.Upsert("Sam")
	s.SetBest(sam, "add:easy", Record{Score: 100, Gates: 9})
	s.SetBest(sam, "mul:hard", Record{Score: 900, Gates: 6})
	ada := s.Upsert("Ada")
	s.AddRace(ada)
	ann := s.Upsert("Ann")
	if ann == "" {
		t.Fatal("ann")
	}
	hof := s.HallOfFame()
	want := []string{"Sam", "Bo", "Ada", "Ann"}
	if len(hof) != len(want) {
		t.Fatalf("len %d", len(hof))
	}
	for i, name := range want {
		if hof[i].Name != name {
			t.Fatalf("order %+v", hof)
		}
	}
	if hof[0].BestKey != "mul:hard" || hof[0].BestGates != 6 || hof[0].BestScore != 900 {
		t.Fatalf("sam peak %+v", hof[0])
	}
	if hof[1].BestKey != "div:medium" || hof[1].BestGates != 4 || hof[1].Races != 2 {
		t.Fatalf("bo peak %+v", hof[1])
	}
	if hof[2].Races != 1 || hof[2].BestScore != 0 || hof[2].BestKey != "" {
		t.Fatalf("ada %+v", hof[2])
	}
	if !s.Select("ada") || s.Name(s.Active) != "Ada" {
		t.Fatalf("select %q %q", s.Active, s.Name(s.Active))
	}
}

func TestPeakTieBreak(t *testing.T) {
	score, gates, key := peak(Profile{Bests: map[string]Record{
		"zzz:easy": {Score: 50, Gates: 1},
		"aaa:easy": {Score: 50, Gates: 1},
		"mul:hard": {Score: 50, Gates: 2},
		"add:easy": {Score: 10, Gates: 9},
	}})
	if score != 50 || gates != 2 || key != "mul:hard" {
		t.Fatalf("higher gates %d %d %s", score, gates, key)
	}
	_, _, key = peak(Profile{Bests: map[string]Record{
		"zzz:easy": {Score: 40, Gates: 3},
		"aaa:easy": {Score: 40, Gates: 3},
	}})
	if key != "aaa:easy" {
		t.Fatalf("tie key %s", key)
	}
}

func TestUnknownActiveFallsBack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MATHSKILL_HOME", dir)
	path := filepath.Join(dir, "profiles.json")
	raw := []byte(`{"active":"gone","profiles":[{"name":"Sam"},{"name":"Ada"}]}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Name(s.Active) != "Sam" {
		t.Fatalf("active %q", s.Name(s.Active))
	}
}

func TestCorruptFileIsRenamed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MATHSKILL_HOME", dir)
	path := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if len(s.Profiles) != 0 {
		t.Fatal("profiles kept")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("corrupt file still in place")
	}
}

func isHexID(s string) bool {
	if len(s) != 32 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
