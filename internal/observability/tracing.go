package observability

import (
	"context"
	"time"
)

type Span struct {
	Name       string
	Started    time.Time
	Attributes map[string]string
	Ended      time.Time
}
type spanKey struct{}

func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	span := &Span{Name: name, Started: time.Now(), Attributes: map[string]string{}}
	return context.WithValue(ctx, spanKey{}, span), span
}
func EndSpan(span *Span) {
	if span != nil {
		span.Ended = time.Now()
	}
}
func SpanFromContext(ctx context.Context) *Span { span, _ := ctx.Value(spanKey{}).(*Span); return span }
