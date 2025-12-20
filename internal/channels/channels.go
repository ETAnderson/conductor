package channels

import "context"

type ProductRef struct {
	ProductKey string
	Hash       string
}

type ProductOutcome struct {
	ProductKey string
	Status     string // "ok" | "skipped" | "error"
	Message    string
}

type BuildResult struct {
	Channel  string
	Attempt  int
	OkCount  int
	ErrCount int
	Items    []ProductOutcome
}

type Channel interface {
	Name() string
	Build(ctx context.Context, tenantID uint64, products []ProductRef) (BuildResult, error)
}

type Registry struct {
	byName map[string]Channel
}

func NewRegistry(chans ...Channel) Registry {
	m := make(map[string]Channel, len(chans))
	for _, c := range chans {
		if c == nil {
			continue
		}
		m[c.Name()] = c
	}
	return Registry{byName: m}
}

func (r Registry) Get(name string) (Channel, bool) {
	if r.byName == nil {
		return nil, false
	}
	c, ok := r.byName[name]
	return c, ok
}
