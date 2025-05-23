package types

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJMESPathFilterAccept(t *testing.T) {
	specs := map[string]struct {
		src       []byte
		filter    string
		expResult bool
		expErr    error
	}{
		"happy": {
			src:       []byte(`{"msg": {"foo":"bar"}}`),
			filter:    "msg.foo == `\"bar\"`",
			expResult: true,
		},
		"happy with if else": {
			src: []byte(`{
				"valuea": 5,
				"valueb": 12
				}
			`),
			filter:    "valueb > `9`",
			expResult: true,
		},
		"unhappy with if else": {
			src: []byte(`{
				"valuea": 5,
				"valueb": 9
				}
			`),
			filter:    "valueb > `9`",
			expResult: false,
		},
		"should error, no boolean": {
			src: []byte(`{
				"valuea": 5,
				"valueb": 9
				}
			`),
			filter: "valueb",
			expErr: ErrInvalid.Wrap("JMESPath filter did not return a boolean : %!s(float64=9): invalid"),
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			exists, gotErr := MatchJMESPaths(spec.src, []string{spec.filter})

			if spec.expErr != nil {
				assert.ErrorIs(t, gotErr, spec.expErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expResult, exists)
		})
	}
}

// TDO(PR) add more tests to make sure result is deterministic

func TestJMESPathDeterminism(t *testing.T) {

	specs := map[string]struct {
		src    []byte
		filter string
	}{
		"array ordering": {
			src: []byte(`{
		"people": [
			{"name": true, "age": 30},
			{"name": false, "age": 25}
		]
	}`),
			filter: "people[0].name",
		},
		"field_parsing": {
			src: []byte(`{
		"people": [
			{"name": true, "name": false},
			{"name": false, "name": true}
		]
	}`),
			filter: "people[0].name",
		},
		"key parsing": {
			src: []byte(`{
		"people": [
			{"name": true, "age": true},
			{"name": false}
		]
	}`),
			filter: "people[*].age",
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			expected, err := MatchJMESPaths(spec.src, []string{spec.filter})
			// Repeat parsing multiple times to check for determinism
			for i := range 100000 {
				result, newErr := MatchJMESPaths(spec.src, []string{spec.filter})
				if !reflect.DeepEqual(expected, result) {
					t.Errorf("Non-deterministic result on iteration %d.\nExpected: %#v\nGot: %#v", i, expected, result)
				}
				if (err != nil && newErr != nil && !reflect.DeepEqual(err.Error(), newErr.Error())) || ((err == nil) != (newErr == nil)) {
					t.Errorf("Non-deterministic result on iteration %d.\nExpectedError: %#v,\nGotError: %#v", i, err, newErr)
				}
			}

		})
	}

}
