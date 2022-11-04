package types

import (
	"encoding/json"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// IsJSONObjectWithTopLevelKey returns if the given bytes are a valid JSON object
// with exactly one top-level key that is contained in the list of allowed keys.
func IsJSONObjectWithTopLevelKey(jsonBytes []byte, allowedKeys []string) (bool, error) {
	document := map[string]interface{}{}
	if err := json.Unmarshal(jsonBytes, &document); err != nil {
		return false, sdkerrors.Wrap(ErrNotAJSONObject, "failed to unmarshal JSON to map")
	}

	if len(document) == 0 {
		return false, sdkerrors.Wrap(ErrNoTopLevelKey, "JSON object has no top-level key")
	}

	if len(document) > 1 {
		return false, sdkerrors.Wrap(ErrMultipleTopLevelKeys, "JSON object has multiple top-level keys")
	}

	// Loop is executed exactly once
	for topLevelKey := range document {
		for _, allowedKey := range allowedKeys {
			if allowedKey == topLevelKey {
				return true, nil
			}
		}
		return false, nil
	}

	panic("Reached unreachable code. This is a bug.")
}
