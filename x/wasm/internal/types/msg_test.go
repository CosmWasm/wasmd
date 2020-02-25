package types

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderRegexp(t *testing.T) {
	cases := []struct {
		example string
		valid   bool
	}{
		{"fedora/httpd:version1.0", true},
		{"cosmwasm-opt:0.6.3", true},
		{"cosmwasm-opt-:0.6.3", false},
		{"confio/js-builder-1:test", true},
		{"confio/.builder-1:manual", false},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			ok, err := regexp.MatchString(BuildTagRegexp, tc.example)
			assert.NoError(t, err)
			assert.Equal(t, tc.valid, ok)
		})

	}
}
