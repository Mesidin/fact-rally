package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Record is the best finish for one course and band.
type Record struct {
	Score int `json:"score"`
	Gates int `json:"gates"`
}

// Profile is one racer on this computer.
type Profile struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Races int               `json:"races"`
	Bests map[string]Record `json:"bests"`
}

// Standing is one Hall of Fame row. BestScore is that racer's best single race.
type Standing struct {
	ID        string
	Name      string
	Races     int
	BestScore int
	BestGates int
	BestKey   string
}

// Store is every racer, saved as JSON under the mathskill home directory.
// Active is a profile ID. A file saved before IDs existed may store a name;
// Load resolves that name to the matching ID.
type Store struct {
	Active   string    `json:"active"`
	Profiles []Profile `json:"profiles"`
	path     string
}

// Dir is MATHSKILL_HOME, or ~/.mathskill.
func Dir() string {
	if d := os.Getenv("MATHSKILL_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mathskill"
	}
	return filepath.Join(home, ".mathskill")
}

// Empty is a store that has not been saved yet.
func Empty() *Store {
	return &Store{path: filepath.Join(Dir(), "profiles.json")}
}

// Load reads profiles.json. A missing file is an empty store.
// A file that cannot be parsed is renamed to profiles.json.bak.
// Missing IDs are filled in. Active may be an ID or an older saved name.
func Load() (*Store, error) {
	path := filepath.Join(Dir(), "profiles.json")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Store{path: path}, nil
	}
	if err != nil {
		return &Store{path: path}, err
	}
	s := &Store{path: path}
	if err := json.Unmarshal(b, s); err != nil {
		_ = os.Rename(path, path+".bak")
		return &Store{path: path}, err
	}
	s.path = path
	s.Normalize()
	return s, nil
}

// Normalize fills missing IDs and resolves Active.
// An ID already in the file is kept. A name in Active selects that racer.
func (s *Store) Normalize() {
	if s == nil {
		return
	}
	s.ensureIDs()
	s.resolveActive()
}

// Save writes the store. The directory is created if needed.
func (s *Store) Save() error {
	if s.path == "" {
		s.path = filepath.Join(Dir(), "profiles.json")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(s.path, b, 0o644)
}

// Name is the stored spelling for an ID or a name. An unknown racer is empty.
func (s *Store) Name(idOrName string) string {
	i := s.index(idOrName)
	if i < 0 {
		return ""
	}
	return s.Profiles[i].Name
}

// Upsert adds a racer or selects one that already matches, ignoring case.
// The stored spelling is kept. The returned value is the racer's ID.
func (s *Store) Upsert(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if i := s.indexName(name); i >= 0 {
		s.Active = s.Profiles[i].ID
		return s.Active
	}
	runes := []rune(name)
	if len(runes) > 16 {
		name = string(runes[:16])
	}
	id := newID()
	s.Profiles = append(s.Profiles, Profile{ID: id, Name: name, Bests: map[string]Record{}})
	s.Active = id
	return id
}

// Select makes this racer active. idOrName may be an ID or a name.
func (s *Store) Select(idOrName string) bool {
	i := s.index(idOrName)
	if i < 0 {
		return false
	}
	s.Active = s.Profiles[i].ID
	return true
}

// Best is the saved record for a course key. A missing record is zero.
// name may be an ID or a racer name.
func (s *Store) Best(name, key string) Record {
	i := s.index(name)
	if i < 0 || s.Profiles[i].Bests == nil {
		return Record{}
	}
	return s.Profiles[i].Bests[key]
}

// SetBest replaces the record for a course key.
func (s *Store) SetBest(name, key string, rec Record) {
	if s.index(name) < 0 {
		s.Upsert(name)
	}
	i := s.index(name)
	if i < 0 {
		return
	}
	if s.Profiles[i].Bests == nil {
		s.Profiles[i].Bests = map[string]Record{}
	}
	s.Profiles[i].Bests[key] = rec
}

// AddRace counts one finished race, including a pull-off and a score of zero.
func (s *Store) AddRace(idOrName string) {
	i := s.index(idOrName)
	if i < 0 {
		return
	}
	s.Profiles[i].Races++
}

// HallOfFame ranks racers by their best single race, then the gates on that
// finish, then name.
func (s *Store) HallOfFame() []Standing {
	if s == nil || len(s.Profiles) == 0 {
		return nil
	}
	out := make([]Standing, len(s.Profiles))
	for i, p := range s.Profiles {
		score, gates, key := peak(p)
		out[i] = Standing{
			ID:        p.ID,
			Name:      p.Name,
			Races:     p.Races,
			BestScore: score,
			BestGates: gates,
			BestKey:   key,
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BestScore != out[j].BestScore {
			return out[i].BestScore > out[j].BestScore
		}
		if out[i].BestGates != out[j].BestGates {
			return out[i].BestGates > out[j].BestGates
		}
		ni := strings.ToLower(out[i].Name)
		nj := strings.ToLower(out[j].Name)
		if ni != nj {
			return ni < nj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func peak(p Profile) (score, gates int, key string) {
	keys := make([]string, 0, len(p.Bests))
	for k, rec := range p.Bests {
		if rec.Score > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		rec := p.Bests[k]
		if rec.Score > score || (rec.Score == score && rec.Gates > gates) {
			score, gates, key = rec.Score, rec.Gates, k
		}
	}
	return score, gates, key
}

func (s *Store) ensureIDs() {
	seen := make(map[string]bool, len(s.Profiles))
	for i := range s.Profiles {
		p := &s.Profiles[i]
		if p.Bests == nil {
			p.Bests = map[string]Record{}
		}
		if p.ID == "" || seen[p.ID] {
			p.ID = uniqueID(seen)
		}
		seen[p.ID] = true
	}
}

func (s *Store) resolveActive() {
	if s.indexID(s.Active) >= 0 {
		return
	}
	if i := s.indexName(s.Active); i >= 0 {
		s.Active = s.Profiles[i].ID
		return
	}
	if s.Active != "" && len(s.Profiles) > 0 {
		s.Active = s.Profiles[0].ID
	}
}

func (s *Store) index(idOrName string) int {
	if i := s.indexID(idOrName); i >= 0 {
		return i
	}
	return s.indexName(idOrName)
}

func (s *Store) indexID(id string) int {
	if id == "" {
		return -1
	}
	for i, p := range s.Profiles {
		if p.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) indexName(name string) int {
	if name == "" {
		return -1
	}
	for i, p := range s.Profiles {
		if strings.EqualFold(p.Name, name) {
			return i
		}
	}
	return -1
}

func uniqueID(seen map[string]bool) string {
	for {
		id := newID()
		if !seen[id] {
			return id
		}
	}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}
