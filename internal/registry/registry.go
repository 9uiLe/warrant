package registry

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	Raw map[string]any
}

func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		raw = map[string]any{}
	}
	return &Registry{Raw: raw}, nil
}

// Requirements は root["requirements"] を []any で返す（壊れていても落ちない）
func (r *Registry) Requirements() []any {
	v, ok := r.Raw["requirements"]
	if !ok || v == nil {
		return nil
	}
	reqs, ok := v.([]any)
	if !ok {
		return nil
	}
	return reqs
}
