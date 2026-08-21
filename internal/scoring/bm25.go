package scoring

import "math"

type BM25 struct {
	K1 float64
	B  float64
}

func DefaultBM25() BM25 { return BM25{K1: 1.2, B: .75} }
func (b BM25) Score(termFrequency, documentLength, averageLength, documentFrequency, totalDocuments float64) float64 {
	if termFrequency <= 0 || totalDocuments <= 0 {
		return 0
	}
	if averageLength <= 0 {
		averageLength = 1
	}
	idf := math.Log(1 + (totalDocuments-documentFrequency+.5)/(documentFrequency+.5))
	norm := termFrequency + b.K1*(1-b.B+b.B*documentLength/averageLength)
	return idf * termFrequency * (b.K1 + 1) / norm
}

type Explanation struct {
	Description string        `json:"description"`
	Value       float64       `json:"value"`
	Details     []Explanation `json:"details,omitempty"`
}

func ExplainBM25(b BM25, tf, dl, avg, df, total float64) Explanation {
	score := b.Score(tf, dl, avg, df, total)
	return Explanation{Description: "BM25 score", Value: score, Details: []Explanation{{Description: "term frequency", Value: tf}, {Description: "document length", Value: dl}, {Description: "average document length", Value: avg}, {Description: "document frequency", Value: df}, {Description: "document count", Value: total}}}
}
