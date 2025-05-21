package types

import (
	"encoding/json"

	"github.com/jmespath/go-jmespath"
)

// The function returns true if the given maps are a valid JSON object
// and match all the given filters.

// Accept only payload messages which pass the given JMESPath filter.
func MatchJMESPaths(msg RawContractMessage, filters []string) (bool, error) {
	var msg_data any
	err := json.Unmarshal(msg, &msg_data)
	if err != nil {
		return false, ErrInvalid.Wrapf("Error unmarshaling message %s: %s", msg, err.Error())
	}
	for _, filter := range filters {

		result, err := jmespath.Search(filter, msg_data)
		if err != nil {
			// We are not logging the error because of the undeterministic nature of json unmarshalling within go
			return false, ErrInvalid.Wrapf("JMESPath filter %s applied on %s is invalid", filter, msg_data)
		}
		b, ok := result.(bool)
		if !ok {
			return false, ErrInvalid.Wrapf("JMESPath filter did not return a boolean : %s", result)
		}
		if !b {
			return false, nil
		}
	}
	return true, nil
}
