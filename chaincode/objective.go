package main

import (
	"fmt"
	"strings"

	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"gopkg.in/go-playground/validator.v9"
)

// Set is a method of the receiver Objective. It checks the validity of inputObjective and uses its fields to set the Objective.
// Returns the objectiveKey and the dataManagerKey associated to test dataSample
func (objective *Objective) Set(stub shim.ChaincodeStubInterface, inp inputObjective) (objectiveKey string, dataManagerKey string, err error) {
	// checking validity of submitted fields
	validate := validator.New()
	if err = validate.Struct(inp); err != nil {
		err = fmt.Errorf("invalid objective inputs %s", err.Error())
		return
	}
	dataManagerKey = strings.Split(inp.TestDataset, ":")[0]
	dataSampleKeys := strings.Split(strings.Replace(strings.Split(inp.TestDataset, ":")[1], " ", "", -1), ",")
	testOnly, _, err := checkSameDataManager(stub, dataManagerKey, dataSampleKeys)
	if err != nil {
		err = fmt.Errorf("invalid test dataSample %s", err.Error())
		return
	} else if !testOnly {
		err = fmt.Errorf("test dataSample are not tagged as testOnly dataSample")
		return
	}
	objective.TestDataset = &Dataset{
		DataManagerKey: dataManagerKey,
		DataSampleKeys: dataSampleKeys,
	}
	objective.Name = inp.Name
	objective.DescriptionStorageAddress = inp.DescriptionStorageAddress
	objective.Metrics = &HashDressName{
		Name:           inp.MetricsName,
		Hash:           inp.MetricsHash,
		StorageAddress: inp.MetricsStorageAddress,
	}
	owner, err := getTxCreator(stub)
	if err != nil {
		return
	}
	objective.Owner = owner
	objective.Permissions = inp.Permissions
	objectiveKey = inp.DescriptionHash
	return
}

// -------------------------------------------------------------------------------------------
// Smart contract related to objectivess
// -------------------------------------------------------------------------------------------

// registerObjective stores a new objective in the ledger.
// If the key exists, it will override the value with the new one
func registerObjective(stub shim.ChaincodeStubInterface, args []string) (resp map[string]string, err error) {
	expectedArgs := getFieldNames(&inputObjective{})
	if nbArgs := len(expectedArgs); nbArgs != len(args) {
		err = fmt.Errorf("incorrect arguments, expecting %d args: %s", nbArgs, strings.Join(expectedArgs, ", "))
		return
	}

	// convert input strings args to input struct inputObjective
	inpc := inputObjective{}
	stringToInputStruct(args, &inpc)
	// check validity of input args and convert it to Objective
	objective := Objective{}
	objectiveKey, dataManagerKey, err := objective.Set(stub, inpc)
	if err != nil {
		return
	}
	// check objective is not already in ledger
	if elementBytes, _ := stub.GetState(objectiveKey); elementBytes != nil {
		err = fmt.Errorf("objective with this description already exists - %s", string(elementBytes))
		return
	}
	// submit to ledger
	objectiveBytes, _ := json.Marshal(objective)
	if err = stub.PutState(objectiveKey, objectiveBytes); err != nil {
		err = fmt.Errorf("failed to submit to ledger the objective with key %s, error is %s", objectiveKey, err.Error())
		return
	}
	// create composite key
	if err = createCompositeKey(stub, "objective~owner~key", []string{"objective", objective.Owner, objectiveKey}); err != nil {
		return
	}
	// add objective to dataManager
	err = addObjectiveDataManager(stub, dataManagerKey, objectiveKey)
	return map[string]string{"key": objectiveKey}, err
}

// queryObjective returns a objective of the ledger given its key
func queryObjective(stub shim.ChaincodeStubInterface, args []string) (out outputObjective, err error) {
	if len(args) != 1 || len(args[0]) != 64 {
		err = fmt.Errorf("incorrect arguments, expecting key, received: %s", args[0])
		return
	}
	key := args[0]
	var objective Objective
	if err = getElementStruct(stub, key, &objective); err != nil {
		return
	}
	out.Fill(key, objective)
	return
}

// queryObjectives returns all objectives of the ledger
func queryObjectives(stub shim.ChaincodeStubInterface, args []string) (outObjectives []outputObjective, err error) {
	if len(args) != 0 {
		err = fmt.Errorf("incorrect number of arguments, expecting nothing")
		return
	}
	var indexName = "objective~owner~key"
	elementsKeys, err := getKeysFromComposite(stub, indexName, []string{"objective"})
	if err != nil {
		err = fmt.Errorf("issue getting keys from composite key %s - %s", indexName, err.Error())
		return
	}
	for _, key := range elementsKeys {
		var objective Objective
		if err = getElementStruct(stub, key, &objective); err != nil {
			return
		}
		var out outputObjective
		out.Fill(key, objective)
		outObjectives = append(outObjectives, out)
	}
	return
}

// -------------------------------------------------------------------------------------------
// Utils for objectivess
// -------------------------------------------------------------------------------------------

// addObjectiveDataManager associates a objective to a dataManager, more precisely, it adds the objective key to the dataManager
func addObjectiveDataManager(stub shim.ChaincodeStubInterface, dataManagerKey string, objectiveKey string) error {
	dataManager := DataManager{}
	if err := getElementStruct(stub, dataManagerKey, &dataManager); err != nil {
		return nil
	}
	if dataManager.ObjectiveKey != "" {
		return fmt.Errorf("dataManager is already associated with a objective")
	}
	dataManager.ObjectiveKey = objectiveKey
	dataManagerBytes, _ := json.Marshal(dataManager)
	return stub.PutState(dataManagerKey, dataManagerBytes)
}
