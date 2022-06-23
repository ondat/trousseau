package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRoundrobin(t *testing.T) {
	t.Parallel()

	expectedResults := [][]string{
		{"a", "b", "c", "d"},
		{"b", "c", "d", "a"},
		{"c", "d", "a", "b"},
		{"d", "a", "b", "c"},
	}

	rr := NewRoundrobin([]string{"a", "b", "c", "d"})

	for i := 0; i < 3; i++ {
		for _, expected := range expectedResults {
			assert.Equal(t, expected, rr.Next(), "Roundrobin failed")
		}
	}
}

func TestFastest(t *testing.T) {
	testcases := map[string]struct {
		producer func(chan<- Metric)
		expeted  []string
	}{
		"Reverse": {
			producer: func(m chan<- Metric) {
				m <- Metric{
					Provider:    "a",
					ReponseTime: 3 * time.Second,
				}
				m <- Metric{
					Provider:    "b",
					ReponseTime: 2 * time.Second,
				}
				m <- Metric{
					Provider:    "c",
					ReponseTime: 1 * time.Second,
				}

				time.Sleep(time.Millisecond)
			},
			expeted: []string{"c", "b", "a"},
		},
		"Peek in average": {
			producer: func(m chan<- Metric) {
				m <- Metric{
					Provider:    "a",
					ReponseTime: 2 * time.Second,
				}
				m <- Metric{
					Provider:    "a",
					ReponseTime: 1 * time.Second,
				}
				m <- Metric{
					Provider:    "b",
					ReponseTime: 2 * time.Second,
				}
				m <- Metric{
					Provider:    "b",
					ReponseTime: 2 * time.Minute,
				}
				m <- Metric{
					Provider:    "c",
					ReponseTime: 1 * time.Second,
				}

				time.Sleep(time.Millisecond)
			},
			expeted: []string{"c", "a", "b"},
		},
		"Reset after max age": {
			producer: func(m chan<- Metric) {
				m <- Metric{
					Provider:    "b",
					ReponseTime: time.Minute,
				}

				time.Sleep(fastestAverageMaxAge + 1)

				m <- Metric{
					Provider:    "a",
					ReponseTime: 3 * time.Second,
				}
				m <- Metric{
					Provider:    "c",
					ReponseTime: 1 * time.Second,
				}

				time.Sleep(time.Millisecond)
			},
			expeted: []string{"b", "c", "a"},
		},
	}

	for name, tc := range testcases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			fastestAverageMaxAge = time.Second

			t.Parallel()

			for i := 0; i < 3; i++ {
				fastest := NewFastest([]string{"a", "b", "c"})

				tc.producer(fastest.C())

				assert.Equal(t, tc.expeted, fastest.Fastest(), "Fastest failed")
			}
		})
	}
}
