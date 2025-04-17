package fullnode

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
)

var deployContractResponse = `{
  "visible": true,
  "txID": "36cfdc59c96dd425b102489a9de1c70455be40b075a565a56f06643be695f0c3",
  "contract_address": "41306d7f39ffc367edb1dee2a9782847e1579795a0",
  "raw_data": {
    "contract": [
      {
        "parameter": {
          "value": {
            "owner_address": "TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh",
            "new_contract": {
              "bytecode": "608060405234801561001057600080fd5b5060de8061001f6000396000f30060806040526004361060485763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631ab06ee58114604d5780639507d39a146067575b600080fd5b348015605857600080fd5b506065600435602435608e565b005b348015607257600080fd5b50607c60043560a0565b60408051918252519081900360200190f35b60009182526020829052604090912055565b600090815260208190526040902054905600a165627a7a72305820fdfe832221d60dd582b4526afa20518b98c2e1cb0054653053a844cf265b25040029",
              "name": "SomeContract",
              "origin_address": "TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh",
              "abi": {
                "entrys": [
                  {
                    "inputs": [
                      {
                        "name": "key",
                        "type": "uint256"
                      },
                      {
                        "name": "value",
                        "type": "uint256"
                      }
                    ],
                    "name": "set",
                    "stateMutability": "Nonpayable",
                    "type": "Function"
                  },
                  {
                    "outputs": [
                      {
                        "name": "value",
                        "type": "uint256"
                      }
                    ],
                    "constant": true,
                    "inputs": [
                      {
                        "name": "key",
                        "type": "uint256"
                      }
                    ],
                    "name": "get",
                    "stateMutability": "View",
                    "type": "Function"
                  }
                ]
              }
            }
          },
          "type_url": "type.googleapis.com/protocol.CreateSmartContract"
        },
        "type": "CreateSmartContract"
      }
    ],
    "ref_block_bytes": "a1f8",
    "ref_block_hash": "135be5ca457bc5fd",
    "expiration": 1742180361000,
    "timestamp": 1742180302542
  },
  "raw_data_hex": "0a02a1f82208135be5ca457bc5fd40a8c6aa90da325ad703081e12d2030a30747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e437265617465536d617274436f6e7472616374129d030a1541608f8da72479edc7dd921e4c30bb7e7cddbe722e1283030a1541608f8da72479edc7dd921e4c30bb7e7cddbe722e1a5c0a2b1a03736574220e12036b65791a0775696e743235362210120576616c75651a0775696e74323536300240030a2d10011a03676574220e12036b65791a0775696e743235362a10120576616c75651a0775696e743235363002400222fd01608060405234801561001057600080fd5b5060de8061001f6000396000f30060806040526004361060485763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631ab06ee58114604d5780639507d39a146067575b600080fd5b348015605857600080fd5b506065600435602435608e565b005b348015607257600080fd5b50607c60043560a0565b60408051918252519081900360200190f35b60009182526020829052604090912055565b600090815260208190526040902054905600a165627a7a72305820fdfe832221d60dd582b4526afa20518b98c2e1cb0054653053a844cf265b250400293a0c536f6d65436f6e747261637470cefda690da32"
}`

func TestDeployContract(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, deployContractResponse)
	}))
	defer testServer.Close()

	fullnodeClient := NewClient(testServer.URL, httpClient)
	owner, err := address.StringToAddress("TVSTZkvVosqh4YHLwHmmNuqeyn967aE2iv")
	assert.NoError(t, err)
	res, err := fullnodeClient.DeployContract(owner, "test", "[]", "0x1234", 0, 0, 0, nil)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "41306d7f39ffc367edb1dee2a9782847e1579795a0", res.ContractAddress)
	assert.Equal(t, 1, len(res.RawData.Contract))
}
