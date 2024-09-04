package jsonclient

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testcase struct {
	name          string
	jsonfile      string
	responsecode  int
	apicall       func(*assert.Assertions, *require.Assertions)
	apifuncparams []reflect.Value
	assertions    func(a *assert.Assertions, r *require.Assertions, data interface{})
}

var testcases = []testcase{
	{
		name: "GetBlockByNum",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "getblockbynum.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			block, err := client.GetBlockByNum(123)
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedGetBlockByNum, block)
		},
	},
	{
		name: "GetEnergyPrices",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "getenergyprices.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			eprices, err := client.GetEnergyPrices()
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedEnergyPrices, eprices)
		},
	},
	{
		name: "GetNowBlock",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "getnowblock.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			block, err := client.GetNowBlock()
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedGetNowBlock, block)
		},
	},
	{
		name: "GetTransactionInfoByID",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "gettransactioninfobyid.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			txinfo, err := client.GetTransactionInfoById("txhash")
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedGetTransactionInfoById, txinfo)
		},
	},
	{
		name: "EstimateEnergy",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "estimateenergy.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			energy, err := client.EstimateEnergy(&EnergyEstimateRequest{})
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedEstimateEnergy, energy)
		},
	},
	{
		name: "EstimateEnergyFail",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "emptyresponse.json"
			httpstatus := http.StatusInternalServerError
			client := setupJsonClient(jsonsource, httpstatus, r)
			energy, err := client.EstimateEnergy(&EnergyEstimateRequest{})
			r.Nil(energy)
			r.NotNil(err, "request did not fail: %v", err)
		},
	},
	{
		name: "ContractGet",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "getcontract.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			contract, err := client.GetContract("address")
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedContractGet, contract)
		},
	},
	{
		name: "ContractDeploy",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "deploycontract.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			contract, err := client.DeployContract(&DeployContractRequest{})
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedContractDeploy, contract)
		},
	},
	{
		name: "TriggerSmartContract",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "triggersmartcontract.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			contract, err := client.TriggerSmartContract(&TriggerSmartContractRequest{})
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedTriggerSmartContract, contract)
		},
	},
	{
		name: "TriggerConstantContract",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "triggerconstantcontract.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			contract, err := client.TriggerConstantContract(&TriggerConstantContractRequest{})
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedTriggerConstantContract, contract)
		},
	},
	{
		name: "BroadcastTransactionFailure",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			broadcasterr := "SIGERROR"
			broadcastMessage := "56616c6964617465207369676e6174757265206572726f723a206d69737320736967206f7220636f6e7472616374"

			jsonsource := "broadcasttransactionfailure.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			broadcastResponse, err := client.BroadcastTransaction(&Transaction{})
			r.Nil(broadcastResponse, "broadcast response should be nil")
			r.NotNil(err, "broadcast should have failed")
			errstr := fmt.Sprintf("broadcasting failed. Code: %s, Message: %s", broadcasterr, broadcastMessage)
			a.Equal(errstr, err.Error())
		},
	},
	{
		name: "GetAccount",
		apicall: func(a *assert.Assertions, r *require.Assertions) {
			jsonsource := "getaccount.json"
			httpstatus := http.StatusOK
			client := setupJsonClient(jsonsource, httpstatus, r)
			account, err := client.GetAccount("address")
			r.Nil(err, "request failed: %v", err)
			a.Equal(expectedGetAccount, account)

		},
	},
}

func setupJsonClient(jsonfile string, responsecode int, r *require.Assertions) *TronJsonClient {
	jsonresponse, err := readTestdata(jsonfile)
	r.Nil(err, "reading testdata failed: %v", err)
	mockclient := NewMockJsonClient(responsecode, jsonresponse, nil)
	return NewTronJsonClient("baseurl", mockclient)
}

func TestResponseUnmarshal(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc.apicall(a, r)

		})
	}
}
