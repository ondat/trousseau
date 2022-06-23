package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundrobin(t *testing.T) {
	expectedResults := [][]string{
		{"a", "b", "c", "d"},
		{"b", "c", "d", "a"},
		{"c", "d", "a", "b"},
		{"d", "a", "b", "c"},
	}

	rr := NewRoundrobin([]string{"a", "b", "c", "d"})

	for i := 0; i < 3; i++ {
		for _, expected := range expectedResults {
			actual := rr.Next()

			assert.Equal(t, expected, actual, "Roundrobin failed")
		}
	}
}
