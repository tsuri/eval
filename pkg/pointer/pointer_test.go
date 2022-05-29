package pointer_test

import (
	"eval/pkg/pointer"
	"fmt"
	"testing"
)

func TestTop(t *testing.T) {
	m := make(map[string]any)
	m["topA"] = 10
	m["topB"] = 20

	p := pointer.Pointer{"topA"}
	o, _ := p.Get(m)
	fmt.Printf("%v\n", o)

	p = pointer.Pointer{"X"}
	o, err := p.Get(m)
	fmt.Printf("%v %v\n", o, err)

}
