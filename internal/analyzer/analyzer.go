package analyzer

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type Token struct {
	Term     string `json:"term"`
	Position int    `json:"position"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Type     string `json:"type"`
}
type Pipeline struct {
	Name      string              `json:"name"`
	Version   uint64              `json:"version"`
	Lowercase bool                `json:"lowercase"`
	Stopwords []string            `json:"stopwords"`
	Synonyms  map[string][]string `json:"synonyms"`
}

func (p Pipeline) Validate() error {
	if strings.TrimSpace(p.Name) == "" || p.Version == 0 {
		return errors.New("analyzer name and version are required")
	}
	for source, targets := range p.Synonyms {
		if strings.TrimSpace(source) == "" || len(targets) == 0 {
			return errors.New("synonym source and targets cannot be empty")
		}
	}
	return nil
}

func (p Pipeline) Analyze(text string) []Token {
	stops := make(map[string]struct{}, len(p.Stopwords))
	for _, s := range p.Stopwords {
		stops[normalize(s, p.Lowercase)] = struct{}{}
	}
	var tokens []Token
	start := -1
	pos := 0
	runes := []rune(text)
	flush := func(end int) {
		if start < 0 {
			return
		}
		term := normalize(string(runes[start:end]), p.Lowercase)
		if _, skip := stops[term]; !skip && term != "" {
			tokens = append(tokens, Token{Term: term, Position: pos, Start: start, End: end, Type: "word"})
			pos++
			for _, syn := range p.Synonyms[term] {
				tokens = append(tokens, Token{Term: normalize(syn, p.Lowercase), Position: pos - 1, Start: start, End: end, Type: "synonym"})
			}
		}
		start = -1
	}
	for i, r := range runes {
		if isBoundary(r) {
			flush(i)
			if isCJK(r) || isEmoji(r) {
				term := normalize(string(r), p.Lowercase)
				if _, skip := stops[term]; !skip {
					tokens = append(tokens, Token{Term: term, Position: pos, Start: i, End: i + 1, Type: "word"})
					pos++
				}
			}
		} else if start < 0 {
			start = i
		}
	}
	flush(len(runes))
	return tokens
}

func normalize(value string, lower bool) string {
	value = strings.TrimSpace(value)
	if lower {
		return strings.ToLower(value)
	}
	return value
}
func isBoundary(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) || isCJK(r) || isEmoji(r)
}
func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}
func isEmoji(r rune) bool { return r >= 0x1F300 && r <= 0x1FAFF }

type Registry struct {
	mu      sync.RWMutex
	entries map[string]map[uint64]Pipeline
	current map[string]uint64
}

func NewRegistry() *Registry {
	return &Registry{entries: map[string]map[uint64]Pipeline{}, current: map[string]uint64{}}
}
func (r *Registry) Put(p Pipeline) error {
	if err := p.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	versions := r.entries[p.Name]
	if versions == nil {
		versions = map[uint64]Pipeline{}
		r.entries[p.Name] = versions
	}
	if _, ok := versions[p.Version]; ok {
		return errors.New("analyzer version already exists")
	}
	versions[p.Version] = clone(p)
	if p.Version > r.current[p.Name] {
		r.current[p.Name] = p.Version
	}
	return nil
}
func (r *Registry) Get(name string, version uint64) (Pipeline, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if version == 0 {
		version = r.current[name]
	}
	p, ok := r.entries[name][version]
	return clone(p), ok
}
func (r *Registry) List() []Pipeline {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Pipeline
	for name, v := range r.current {
		out = append(out, clone(r.entries[name][v]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func clone(p Pipeline) Pipeline {
	p.Stopwords = append([]string(nil), p.Stopwords...)
	p.Synonyms = cloneSynonyms(p.Synonyms)
	return p
}
func cloneSynonyms(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}
