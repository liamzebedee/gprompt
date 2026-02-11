package registry

import (
	"p2p/debug"
	"p2p/pipeline"
)

type Method struct {
	Name       string
	Params     []string
	Body       string
	IsPipeline bool
	Pipeline   *pipeline.Pipeline
}

type Registry struct {
	methods map[string]*Method
}

func New() *Registry {
	return &Registry{methods: make(map[string]*Method)}
}

func (r *Registry) Register(name string, params []string, body string) {
	m := &Method{Name: name, Params: params, Body: body}

	if pipeline.IsPipeline(body) {
		p, err := pipeline.Parse(body)
		if err != nil {
			debug.Log("pipeline parse error for %q: %v", name, err)
		} else {
			m.IsPipeline = true
			m.Pipeline = p
			debug.Log("registered pipeline method %q with %d steps", name, len(p.Steps))
		}
	}

	r.methods[name] = m
}

func (r *Registry) Get(name string) *Method {
	return r.methods[name]
}
