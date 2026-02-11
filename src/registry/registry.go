package registry

type Method struct {
	Name   string
	Params []string
	Body   string
}

type Registry struct {
	methods map[string]*Method
}

func New() *Registry {
	return &Registry{methods: make(map[string]*Method)}
}

func (r *Registry) Register(name string, params []string, body string) {
	r.methods[name] = &Method{Name: name, Params: params, Body: body}
}

func (r *Registry) Get(name string) *Method {
	return r.methods[name]
}
