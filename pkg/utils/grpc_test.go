package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEndpoint(t *testing.T) {
	testcases := map[string]struct {
		endpoint     string
		wantProtocol string
		wantEndpoint string
		wantErr      error
	}{
		"OK": {
			endpoint:     "unix://foo.bar",
			wantProtocol: "unix",
			wantEndpoint: "foo.bar",
		},
		"Wrong protocol": {
			endpoint: "http://foo.bar",
			wantErr:  errors.New("invalid endpoint: http://foo.bar"),
		},
	}

	for name, tc := range testcases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for i := 0; i < 3; i++ {
				proto, address, err := ParseEndpoint(tc.endpoint)

				assert.Equal(t, err, tc.wantErr, "Invalid error")
				assert.Equal(t, tc.wantProtocol, proto, "Invalid protocol")
				assert.Equal(t, tc.wantEndpoint, address, "Invalid address")
			}
		})
	}
}
