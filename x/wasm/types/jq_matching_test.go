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
	jsonInput := []byte(`{
		"people": [
			{"name": true, "age": 30},
			{"name": false, "age": 25}
		]
	}`)
	expression := "people[0].name"

	expected, err := MatchJMESPaths(jsonInput, []string{expression})
	if err != nil {
		t.Fatalf("first JMESPath search failed: %v", err)
	}

	// Repeat parsing multiple times to check for determinism
	for i := 0; i < 100000; i++ {
		result, err := MatchJMESPaths(jsonInput, []string{expression})
		if err != nil {
			t.Errorf("JMESPath search failed on iteration %d: %v", i, err)
			continue
		}
		if !reflect.DeepEqual(expected, result) {
			t.Errorf("Non-deterministic result on iteration %d.\nExpected: %#v\nGot: %#v", i, expected, result)
		}
	}
}
