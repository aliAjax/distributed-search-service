package index

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/distributed-search-service/internal/analyzer"
	"example.com/distributed-search-service/internal/query"
	"example.com/distributed-search-service/internal/scoring"
	"example.com/distributed-search-service/internal/search/domain"
)

type indexedDoc struct {
	doc     domain.Document
	lengths map[string]int
	terms   map[string]map[string][]int
}
type Segment struct {
	ID         string
	Generation uint64
	CreatedAt  time.Time
	docs       map[string]indexedDoc
	postings   map[string]map[string]map[string][]int
}
type Hit struct {
	ID          string               `json:"id"`
	Score       float64              `json:"score"`
	Fields      map[string]any       `json:"fields"`
	Sort        []any                `json:"sort"`
	Explanation *scoring.Explanation `json:"explanation,omitempty"`
}
type Result struct {
	Hits       []Hit  `json:"hits"`
	Total      int    `json:"total"`
	TookMS     int64  `json:"took_ms"`
	Generation uint64 `json:"generation"`
	Partial    bool   `json:"partial"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type Engine struct {
	mu         sync.RWMutex
	segments   []*Segment
	mutable    map[string]domain.Document
	analyzer   analyzer.Pipeline
	generation uint64
	cursor     *query.CursorCodec
	maxMutable int
}

func NewEngine(p analyzer.Pipeline, cursorKey string, maxMutable int) *Engine {
	if maxMutable < 1 {
		maxMutable = 1000
	}
	return &Engine{mutable: map[string]domain.Document{}, analyzer: p, cursor: query.NewCursorCodec(cursorKey), maxMutable: maxMutable}
}
func (e *Engine) Upsert(doc domain.Document) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if previous, ok := e.mutable[doc.ID]; ok && doc.ExternalVersion <= previous.ExternalVersion {
		return errors.New("external version conflict")
	}
	e.mutable[doc.ID] = cloneDoc(doc)
	if len(e.mutable) >= e.maxMutable {
		return e.flushLocked()
	}
	return nil
}
func (e *Engine) Delete(id string, version uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	d, ok := e.mutable[id]
	if !ok {
		for i := len(e.segments) - 1; i >= 0; i-- {
			if found, yes := e.segments[i].docs[id]; yes {
				d = found.doc
				ok = true
				break
			}
		}
	}
	if !ok {
		return errors.New("document not found")
	}
	if version <= d.ExternalVersion {
		return errors.New("external version conflict")
	}
	d.ExternalVersion = version
	d.Deleted = true
	d.UpdatedAt = time.Now().UTC()
	e.mutable[id] = d
	return nil
}
func (e *Engine) Flush() error { e.mu.Lock(); defer e.mu.Unlock(); return e.flushLocked() }
func (e *Engine) flushLocked() error {
	if len(e.mutable) == 0 {
		return nil
	}
	e.generation++
	s := &Segment{ID: fmt.Sprintf("seg-%020d", e.generation), Generation: e.generation, CreatedAt: time.Now().UTC(), docs: map[string]indexedDoc{}, postings: map[string]map[string]map[string][]int{}}
	for id, doc := range e.mutable {
		item := e.indexDocument(doc)
		s.docs[id] = item
		for field, terms := range item.terms {
			if s.postings[field] == nil {
				s.postings[field] = map[string]map[string][]int{}
			}
			for term, pos := range terms {
				if s.postings[field][term] == nil {
					s.postings[field][term] = map[string][]int{}
				}
				s.postings[field][term][id] = append([]int(nil), pos...)
			}
		}
	}
	e.segments = append(e.segments, s)
	e.mutable = map[string]domain.Document{}
	return nil
}
func (e *Engine) indexDocument(doc domain.Document) indexedDoc {
	out := indexedDoc{doc: cloneDoc(doc), lengths: map[string]int{}, terms: map[string]map[string][]int{}}
	for field, value := range doc.Fields {
		text, ok := value.(string)
		if !ok {
			continue
		}
		tokens := e.analyzer.Analyze(text)
		out.lengths[field] = len(tokens)
		out.terms[field] = map[string][]int{}
		for _, token := range tokens {
			out.terms[field][token.Term] = append(out.terms[field][token.Term], token.Position)
		}
	}
	return out
}
func (e *Engine) Search(ctx context.Context, req query.Request) (Result, error) {
	started := time.Now()
	e.mu.RLock()
	segments := append([]*Segment(nil), e.segments...)
	mutable := make([]domain.Document, 0, len(e.mutable))
	for _, d := range e.mutable {
		mutable = append(mutable, cloneDoc(d))
	}
	generation := e.generation
	e.mu.RUnlock()
	candidates := map[string]indexedDoc{}
	for i := len(segments) - 1; i >= 0; i-- {
		for id, d := range segments[i].docs {
			if _, seen := candidates[id]; !seen {
				candidates[id] = d
			}
		}
	}
	for _, d := range mutable {
		candidates[d.ID] = e.indexDocument(d)
	}
	var hits []Hit
	for _, d := range candidates {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		default:
		}
		if d.doc.Deleted || d.doc.TenantID != req.TenantID || d.doc.CollectionID != req.CollectionID {
			continue
		}
		matched, score, explain := e.evaluate(req.Query, d, len(candidates))
		if !matched {
			continue
		}
		accepted := true
		for _, f := range req.Filters {
			ok, _, _ := e.evaluate(f, d, len(candidates))
			if !ok {
				accepted = false
				break
			}
		}
		if !accepted {
			continue
		}
		hit := Hit{ID: d.doc.ID, Score: score, Fields: cloneMap(d.doc.Fields), Sort: []any{score, d.doc.ID}}
		if req.Explain {
			hit.Explanation = &explain
		}
		hits = append(hits, hit)
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score > hits[j].Score
	})
	total := len(hits)
	if req.SearchAfter != "" {
		_, afterID, gen, err := e.cursor.Decode(req.SearchAfter, time.Now())
		if err != nil {
			return Result{}, err
		}
		if gen != generation {
			return Result{}, errors.New("cursor index generation is no longer available")
		}
		idx := 0
		for idx < len(hits) && hits[idx].ID != afterID {
			idx++
		}
		if idx < len(hits) {
			hits = hits[idx+1:]
		}
	}
	if len(hits) > req.Size {
		hits = hits[:req.Size]
	}
	result := Result{Hits: hits, Total: total, TookMS: time.Since(started).Milliseconds(), Generation: generation}
	if len(hits) == req.Size {
		result.NextCursor, _ = e.cursor.Encode(hits[len(hits)-1].Sort, hits[len(hits)-1].ID, generation, time.Now().Add(15*time.Minute))
	}
	return result, nil
}
func (e *Engine) evaluate(c query.Clause, d indexedDoc, total int) (bool, float64, scoring.Explanation) {
	switch c.Kind {
	case query.KindMatch, query.KindPrefix, query.KindFuzzy, query.KindPhrase:
		return e.textMatch(c, d, total)
	case query.KindRange:
		return rangeMatch(c, d.doc.Fields[c.Field])
	case query.KindBool:
		score := 0.0
		details := []scoring.Explanation{}
		for _, child := range c.Must {
			ok, s, x := e.evaluate(child, d, total)
			if !ok {
				return false, 0, x
			}
			score += s
			details = append(details, x)
		}
		for _, child := range c.MustNot {
			ok, _, x := e.evaluate(child, d, total)
			if ok {
				return false, 0, x
			}
		}
		should := len(c.Should) == 0
		for _, child := range c.Should {
			ok, s, x := e.evaluate(child, d, total)
			if ok {
				should = true
				score += s
				details = append(details, x)
			}
		}
		return should, score, scoring.Explanation{Description: "boolean query", Value: score, Details: details}
	}
	return false, 0, scoring.Explanation{Description: "unsupported", Value: 0}
}
func (e *Engine) textMatch(c query.Clause, d indexedDoc, total int) (bool, float64, scoring.Explanation) {
	tokens := e.analyzer.Analyze(c.Value)
	if len(tokens) == 0 {
		return false, 0, scoring.Explanation{}
	}
	terms := d.terms[c.Field]
	score := 0.0
	positions := [][]int{}
	for _, token := range tokens {
		var pos []int
		switch c.Kind {
		case query.KindMatch, query.KindPhrase:
			pos = terms[token.Term]
		case query.KindPrefix:
			for t, p := range terms {
				if strings.HasPrefix(t, token.Term) {
					pos = append(pos, p...)
				}
			}
		case query.KindFuzzy:
			for t, p := range terms {
				if distance(t, token.Term) <= 1 {
					pos = append(pos, p...)
				}
			}
		}
		if len(pos) == 0 {
			return false, 0, scoring.Explanation{Description: "term not found"}
		}
		positions = append(positions, pos)
		score += scoring.DefaultBM25().Score(float64(len(pos)), float64(d.lengths[c.Field]), 10, 1, float64(total))
	}
	if c.Kind == query.KindPhrase && !consecutive(positions) {
		return false, 0, scoring.Explanation{Description: "phrase positions do not match"}
	}
	if c.Boost > 0 {
		score *= c.Boost
	}
	return true, score, scoring.Explanation{Description: string(c.Kind) + " query", Value: score}
}
func rangeMatch(c query.Clause, value any) (bool, float64, scoring.Explanation) {
	v := fmt.Sprint(value)
	ok := true
	if c.From != nil {
		ok = ok && v >= fmt.Sprint(c.From)
	}
	if c.To != nil {
		ok = ok && v <= fmt.Sprint(c.To)
	}
	score := 0.0
	if ok {
		score = 1
	}
	return ok, score, scoring.Explanation{Description: "lexical range", Value: score}
}
func consecutive(groups [][]int) bool {
	if len(groups) < 2 {
		return true
	}
	for _, start := range groups[0] {
		wanted := start + 1
		ok := true
		for i := 1; i < len(groups); i++ {
			found := false
			for _, p := range groups[i] {
				if p == wanted {
					found = true
					break
				}
			}
			if !found {
				ok = false
				break
			}
			wanted++
		}
		if ok {
			return true
		}
	}
	return false
}
func distance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	row := make([]int, len(br)+1)
	for j := range row {
		row[j] = j
	}
	for i, x := range ar {
		prev := row[0]
		row[0] = i + 1
		for j, y := range br {
			old := row[j+1]
			cost := 1
			if x == y {
				cost = 0
			}
			row[j+1] = min(row[j+1]+1, row[j]+1, prev+cost)
			prev = old
		}
	}
	return row[len(br)]
}
func min(v ...int) int {
	m := v[0]
	for _, n := range v[1:] {
		if n < m {
			m = n
		}
	}
	return m
}
func cloneDoc(d domain.Document) domain.Document { d.Fields = cloneMap(d.Fields); return d }
func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func (e *Engine) Stats() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	docs := len(e.mutable)
	for _, s := range e.segments {
		docs += len(s.docs)
	}
	return map[string]any{"segments": len(e.segments), "mutable_documents": len(e.mutable), "physical_documents": docs, "generation": e.generation}
}
func (e *Engine) Segments() []domain.Segment {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]domain.Segment, 0, len(e.segments))
	for _, s := range e.segments {
		out = append(out, domain.Segment{ID: s.ID, Generation: s.Generation, State: domain.SegmentReady, DocumentCount: len(s.docs), CreatedAt: s.CreatedAt})
	}
	return out
}

var _ = strconv.Itoa
