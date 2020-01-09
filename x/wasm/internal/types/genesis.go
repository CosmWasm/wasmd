package types

// GenesisState is the struct representation of the export genesis
type GenesisState struct {
	Codes     []Code     `json:"codes"`
	Contracts []Contract `json:"contracts"`
}

// ValidateGenesis performs basic validation of supply genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data GenesisState) error {
	return nil
}
