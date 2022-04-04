package egraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSmoke(t *testing.T) {
	b := ImportEvaluationGraph()
	assert.Equal(t, b, true)
}
