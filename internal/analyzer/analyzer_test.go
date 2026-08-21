package analyzer

import "testing"

func TestPipelineUnicodeSynonymAndStopword(t *testing.T) {
	p := Pipeline{Name: "test", Version: 1, Lowercase: true, Stopwords: []string{"the"}, Synonyms: map[string][]string{"golang": {"go"}}}
	tokens := p.Analyze("The Golang 中文")
	if len(tokens) != 4 {
		t.Fatalf("tokens=%v", tokens)
	}
	if tokens[0].Term != "golang" || tokens[1].Term != "go" {
		t.Fatalf("synonym tokens=%v", tokens)
	}
	if tokens[2].Term != "中" || tokens[3].Term != "文" {
		t.Fatalf("CJK tokens=%v", tokens)
	}
}
