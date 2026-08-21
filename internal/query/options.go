package query

type Options struct{ Labels map[string]string }

func NewOptions() *Options              { return &Options{} }
func (o *Options) AddLabel(k, v string) { o.Labels[k] = v }
