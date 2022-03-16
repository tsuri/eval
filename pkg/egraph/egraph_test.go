package egraph

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSmoke(t *testing.T) {
	b := ImportEvaluationGraph()
	assert.Equal(t, b, true)
}
