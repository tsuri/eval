package pointer

import "fmt"

type Pointer []string

func (p Pointer) String() (str string) {
	for _, fragment := range p {
		str += "." + fragment
	}
	return
}

func (p Pointer) Head() *string {
	if len(p) == 0 {
		return nil
	}
	return &p[0]
}

func (p Pointer) Get(data any) (any, error) {
	switch ch := data.(type) {
	case map[string]any:
		if h := p.Head(); p != nil {
			if result, ok := ch[*h]; ok {
				return result, nil
			}
			return nil, fmt.Errorf("invalid element: %s", *h)
		}
		return nil, fmt.Errorf("cannot dereference nil pointer")
	default:
		return nil, fmt.Errorf("invalid pointer: %s", p.String())
	}
}
