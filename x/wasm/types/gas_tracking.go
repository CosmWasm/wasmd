package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const gasTrackingKey = "__gt_key__"

type SessionRecord struct {
	ActualSDKGas      sdk.Gas
	OriginalSDKGas    sdk.Gas
	ActualVMGas       sdk.Gas
	OriginalVMGas     sdk.Gas
	ContractAddress   string
	ContractOperation uint64
	description       string
}

type VMRecord struct {
	OriginalVMGas sdk.Gas
	ActualVMGas   sdk.Gas
}

type activeSession struct {
	invokedGasMeter *ContractSDKGasMeter
	invokerGasMeter *ContractSDKGasMeter
	gasFilledIn     bool
	originalVMGas   sdk.Gas
	actualVMGas     sdk.Gas
}

type gasTracking struct {
	mainGasMeter   sdk.GasMeter
	activeSessions []*activeSession
	sessionRecords []*SessionRecord
}

func getGasTrackingData(ctx sdk.Context) (*gasTracking, error) {
	queryTracking, ok := ctx.Value(gasTrackingKey).(*gasTracking)
	if queryTracking == nil || !ok {
		return nil, fmt.Errorf("unable to read query tracking value")
	}

	return queryTracking, nil
}

func currentContractGasMeter(queryTracking *gasTracking) (*ContractSDKGasMeter, error) {
	if len(queryTracking.activeSessions) == 0 {
		return nil, fmt.Errorf("no active sessions")
	}

	lastSession := queryTracking.activeSessions[len(queryTracking.activeSessions)-1]

	if lastSession.invokedGasMeter == nil {
		return nil, fmt.Errorf("no contract meter in current session")
	}

	return lastSession.invokedGasMeter, nil
}

func createCompositeKey(record *SessionRecord) string {
	return record.ContractAddress + "." + fmt.Sprint(record.ContractOperation)
}

func consolidateSessions(queryTracking *gasTracking) {
	sessionRecords := queryTracking.sessionRecords

	recordSet := make(map[string]*SessionRecord)

	recordKeys := make([]string, 0)

	for _, sessionRecord := range sessionRecords {
		compositeKey := createCompositeKey(sessionRecord)
		existingRecord, ok := recordSet[compositeKey]
		if !ok {
			recordSet[compositeKey] = sessionRecord
			recordKeys = append(recordKeys, compositeKey)
		} else {
			recordSet[compositeKey] = &SessionRecord{
				ActualSDKGas:      existingRecord.ActualSDKGas + sessionRecord.ActualSDKGas,
				OriginalSDKGas:    existingRecord.OriginalSDKGas + sessionRecord.OriginalSDKGas,
				ActualVMGas:       existingRecord.ActualVMGas + sessionRecord.ActualVMGas,
				OriginalVMGas:     existingRecord.OriginalVMGas + sessionRecord.OriginalVMGas,
				ContractAddress:   sessionRecord.ContractAddress,
				ContractOperation: sessionRecord.ContractOperation,
			}
		}
	}

	queryTracking.sessionRecords = make([]*SessionRecord, len(recordKeys))

	for i, recordKey := range recordKeys {
		queryTracking.sessionRecords[i] = recordSet[recordKey]
	}
}

func doDestroyCurrentSession(ctx *sdk.Context, queryTracking *gasTracking) error {
	currentSession := queryTracking.activeSessions[len(queryTracking.activeSessions)-1]

	if currentSession.invokedGasMeter != nil {
		if !currentSession.gasFilledIn {
			return fmt.Errorf("vm gas is not recorded in query tracking")
		}

		queryTracking.mainGasMeter.ConsumeGas(currentSession.invokedGasMeter.GasConsumed(), "contract sub-query")

		queryTracking.sessionRecords = append(queryTracking.sessionRecords, &SessionRecord{
			ActualSDKGas:      currentSession.invokedGasMeter.GetActualGas(),
			OriginalSDKGas:    currentSession.invokedGasMeter.GetOriginalGas(),
			ContractAddress:   currentSession.invokedGasMeter.GetContractAddress(),
			ContractOperation: currentSession.invokedGasMeter.GetContractOperation(),
			OriginalVMGas:     currentSession.originalVMGas,
			ActualVMGas:       currentSession.actualVMGas,
			description:       "invoked",
		})
	}

	if currentSession.invokerGasMeter != nil {
		queryTracking.mainGasMeter.ConsumeGas(currentSession.invokerGasMeter.GasConsumed(), "query sdk gas consumption")

		queryTracking.sessionRecords = append(queryTracking.sessionRecords, &SessionRecord{
			ActualSDKGas:      currentSession.invokerGasMeter.GetActualGas(),
			OriginalSDKGas:    currentSession.invokerGasMeter.GetOriginalGas(),
			ContractAddress:   currentSession.invokerGasMeter.GetContractAddress(),
			ContractOperation: currentSession.invokerGasMeter.GetContractOperation(),
			description:       "invoker",
		})
	}

	queryTracking.activeSessions = queryTracking.activeSessions[:len(queryTracking.activeSessions)-1]

	// Revert to previous gas invokedGasMeter
	if len(queryTracking.activeSessions) != 0 {
		*ctx = ctx.WithGasMeter(queryTracking.activeSessions[len(queryTracking.activeSessions)-1].invokedGasMeter)
	} else {
		*ctx = ctx.WithGasMeter(queryTracking.mainGasMeter)
	}

	return nil
}

func IsGasTrackingInitialized(ctx sdk.Context) bool {
	_, err := getGasTrackingData(ctx)
	return err == nil
}

func InitializeGasTracking(ctx *sdk.Context, initialContractGasMeter *ContractSDKGasMeter) error {
	data := ctx.Value(gasTrackingKey)
	if data != nil {
		return fmt.Errorf("query gas tracking is already initialized")
	}

	queryTracking := gasTracking{
		mainGasMeter: ctx.GasMeter(),
		activeSessions: []*activeSession{
			{
				invokedGasMeter: initialContractGasMeter,
			},
		},
		sessionRecords: nil,
	}

	*ctx = ctx.WithValue(gasTrackingKey, &queryTracking)
	*ctx = ctx.WithGasMeter(initialContractGasMeter)
	return nil
}

func TerminateGasTracking(ctx *sdk.Context) ([]*SessionRecord, error) {
	queryTracking, err := getGasTrackingData(*ctx)
	if err != nil {
		return nil, err
	}

	if len(queryTracking.activeSessions) != 1 {
		if len(queryTracking.activeSessions) == 0 {
			return nil, fmt.Errorf("internal error: the initial contract gas invokedGasMeter not found")
		} else {
			return nil, fmt.Errorf("internal error: multiple active gas trackers in session")
		}
	}

	if err := doDestroyCurrentSession(ctx, queryTracking); err != nil {
		return nil, err
	}

	consolidateSessions(queryTracking)

	*ctx = ctx.WithValue(gasTrackingKey, nil)
	*ctx = ctx.WithGasMeter(queryTracking.mainGasMeter)

	return queryTracking.sessionRecords, nil
}

func AddVMRecord(ctx sdk.Context, vmRecord *VMRecord) error {
	queryTracking, err := getGasTrackingData(ctx)
	if err != nil {
		return err
	}

	if len(queryTracking.activeSessions) == 0 {
		return fmt.Errorf("internal error: no active sessions")
	}

	lastSession := queryTracking.activeSessions[len(queryTracking.activeSessions)-1]
	if lastSession.gasFilledIn {
		return fmt.Errorf("gas information already present for current session")
	}

	lastSession.gasFilledIn = true
	lastSession.originalVMGas = vmRecord.OriginalVMGas
	lastSession.actualVMGas = vmRecord.ActualVMGas

	return nil
}

func AssociateContractMeterWithCurrentSession(ctx *sdk.Context, contractGasMeter *ContractSDKGasMeter) error {
	queryTracking, err := getGasTrackingData(*ctx)
	if err != nil {
		return err
	}

	if len(queryTracking.activeSessions) == 0 {
		return fmt.Errorf("no current session found")
	}

	lastSession := queryTracking.activeSessions[len(queryTracking.activeSessions)-1]
	if lastSession.invokedGasMeter != nil {
		return fmt.Errorf("invokedGasMeter is associated already")
	}

	lastSession.invokedGasMeter = contractGasMeter

	*ctx = ctx.WithGasMeter(contractGasMeter)
	return nil
}

func CreateNewSession(ctx *sdk.Context, gasLimit uint64) error {
	queryTracking, err := getGasTrackingData(*ctx)
	if err != nil {
		return err
	}

	currentContractMeter, err := currentContractGasMeter(queryTracking)
	if err != nil {
		return err
	}

	invokerGasMeter := currentContractMeter.CloneWithNewLimit(gasLimit, "cloned for sdk")

	queryTracking.activeSessions = append(queryTracking.activeSessions, &activeSession{
		invokerGasMeter: invokerGasMeter,
		invokedGasMeter: nil,
		gasFilledIn:     false,
	})

	*ctx = ctx.WithGasMeter(invokerGasMeter)

	return nil
}

func DestroySession(ctx *sdk.Context) error {
	queryTracking, err := getGasTrackingData(*ctx)
	if err != nil {
		return err
	}

	return doDestroyCurrentSession(ctx, queryTracking)
}
