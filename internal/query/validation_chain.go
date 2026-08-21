package query

type Validator interface{ Validate(Request) error }
type ValidationChain struct{ Items map[string]Validator }

func (v *ValidationChain) Add(k string, item Validator) { v.Items[k] = item }
