package main

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/stretchr/testify/assert"
)

func TestAlgo(t *testing.T) {
	scc := new(SubstraChaincode)
	mockStub := shim.NewMockStub("substra", scc)

	// Add algo with invalid field
	inpAlgo := inputAlgo{
		DescriptionHash: "aaa",
	}
	args := inpAlgo.createSample()
	resp := mockStub.MockInvoke("42", args)
	if status := resp.Status; status != 500 {
		t.Errorf("when adding algo with invalid hash, status %d and message %s", status, resp.Message)
	}

	// Add algo with unexisting challenge
	inpAlgo = inputAlgo{}
	args = inpAlgo.createSample()
	resp = mockStub.MockInvoke("42", args)
	if status := resp.Status; status != 500 {
		t.Errorf("when adding algo with unexisting challenge, status %d and message %s", status, resp.Message)
	}

	// Properly add algo
	err, resp, tt := registerItem(*mockStub, "algo")
	if err != nil {
		t.Errorf(err.Error())
	}
	inpAlgo = tt.(inputAlgo)
	algoKey := string(resp.Payload)
	if algoKey != inpAlgo.Hash {
		t.Errorf("when adding algo, key does not corresponds to its hash - key: %s and hash %s", algoKey, inpAlgo.Hash)
	}

	// Query algo from key and check the consistency of returned arguments
	args = [][]byte{[]byte("query"), []byte(algoKey)}
	resp = mockStub.MockInvoke("42", args)
	if status := resp.Status; status != 200 {
		t.Errorf("when querying an algo with status %d and message %s", status, resp.Message)
	}
	algo := outputAlgo{}
	err = bytesToStruct(resp.Payload, &algo)
	assert.NoError(t, err, "when unmarshalling queried challenge")
	expectedAlgo := outputAlgo{
		Key:  algoKey,
		Name: inpAlgo.Name,
		Storage: algoStorage{
			Hash:    algoKey,
			Address: inpAlgo.StorageAddress,
		},
		Description: &HashDress{
			Hash:           inpAlgo.DescriptionHash,
			StorageAddress: inpAlgo.DescriptionStorageAddress,
		},
		Owner:        "bbd157aa8e85eb985aeedb79361cd45739c92494dce44d351fd2dbd6190e27f0",
		ChallengeKey: inpAlgo.ChallengeKey,
		Permissions:  inpAlgo.Permissions,
	}
	assert.Equal(t, expectedAlgo.Key, algo.Key)
	assert.Exactly(t, expectedAlgo, algo)

	// Query all algo and check consistency
	args = [][]byte{[]byte("queryAlgos")}
	resp = mockStub.MockInvoke("42", args)
	if status := resp.Status; status != 200 {
		t.Errorf("when querying algos - status %d and message %s", status, resp.Message)
	}
	var algos []outputAlgo
	err = json.Unmarshal(resp.Payload, &algos)
	assert.NoError(t, err, "while unmarshalling algos")
	assert.Len(t, algos, 1)
	assert.Exactly(t, expectedAlgo, algos[0], "return algo different from registered one")
}
