package query

import "fmt"

type Cost struct {
	EstimatedDocs  int64  `json:"estimated_docs"`
	EstimatedTerms int    `json:"estimated_terms"`
	RequiresSort   bool   `json:"requires_sort"`
	Risk           string `json:"risk"`
}

func Estimate(c Clause) Cost {
	out := Cost{EstimatedDocs: 1000, EstimatedTerms: 1}
	switch c.Kind {
	case KindPhrase:
		out.EstimatedDocs = 300
		out.EstimatedTerms = 2
	case KindPrefix:
		out.EstimatedDocs = 500
		out.Risk = "dictionary_scan"
	case KindFuzzy:
		out.EstimatedDocs = 750
		out.Risk = "edit_distance"
	case KindBool:
		out.EstimatedDocs = 100
		out.EstimatedTerms = len(c.Must) + len(c.Should) + len(c.MustNot)
	}
	return out
}
func (c Cost) String() string {
	return fmt.Sprintf("docs=%d terms=%d risk=%s", c.EstimatedDocs, c.EstimatedTerms, c.Risk)
}
