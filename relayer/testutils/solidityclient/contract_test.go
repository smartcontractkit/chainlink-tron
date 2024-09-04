package solidityclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateEnergy(t *testing.T) {
	response := `{
  "result": {
    "result": true
  },
  "energy_required": 1082
}`
	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, response, nil))

	a := assert.New(t)
	r := require.New(t)

	estimation, err := solidityclient.EstimateEnergy(&EnergyEstimateRequest{})
	r.Nil(err, "EstimateEnergy failed", "error:", err)
	a.Equal(true, estimation.Result.Result)
	a.Equal(int64(1082), estimation.EnergyRequired)
}

func TestEstimateEnergyFail(t *testing.T) {
	solidityclient := NewTronSolidityClient("http://endpoint", NewMockSolidityClient(500, "", fmt.Errorf("request failed")))
	r := require.New(t)
	est, err := solidityclient.EstimateEnergy(&EnergyEstimateRequest{})
	r.Nil(est)
	r.NotNil(err)
}

func TestContractGet(t *testing.T) {
	contractjson := `{
  "bytecode": "608060405234801561001057600080fd5b50d3801561001d57600080fd5b50d2801561002a57600080fd5b50604080518082018252600b81526a2a32ba3432b92a37b5b2b760a91b6020808301918252835180850190945260048452631554d11560e21b90840152815191929160069161007c9160039190610236565b508151610090906004906020850190610236565b506005805460ff191660ff92909216919091179055506100cc9050336100b46100d1565b60ff16600a0a6402540be400026100db60201b60201c565b6102ce565b60055460ff165b90565b6001600160a01b038216610136576040805162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015290519081900360640190fd5b61014f816002546101d560201b61078c1790919060201c565b6002556001600160a01b0382166000908152602081815260409091205461017f91839061078c6101d5821b17901c565b6001600160a01b0383166000818152602081815260408083209490945583518581529351929391927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9281900390910190a35050565b60008282018381101561022f576040805162461bcd60e51b815260206004820152601b60248201527f536166654d6174683a206164646974696f6e206f766572666c6f770000000000604482015290519081900360640190fd5b9392505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061027757805160ff19168380011785556102a4565b828001600101855582156102a4579182015b828111156102a4578251825591602001919060010190610289565b506102b09291506102b4565b5090565b6100d891905b808211156102b057600081556001016102ba565b6108ad806102dd6000396000f3fe608060405234801561001057600080fd5b50d3801561001d57600080fd5b50d2801561002a57600080fd5b50600436106100b35760003560e01c806306fdde03146100b8578063095ea7b31461013557806318160ddd1461017557806323b872dd1461018f578063313ce567146101c557806339509351146101e357806370a082311461020f57806395d89b4114610235578063a457c2d71461023d578063a9059cbb14610269578063dd62ed3e14610295575b600080fd5b6100c06102c3565b6040805160208082528351818301528351919283929083019185019080838360005b838110156100fa5781810151838201526020016100e2565b50505050905090810190601f1680156101275780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6101616004803603604081101561014b57600080fd5b506001600160a01b038135169060200135610359565b604080519115158252519081900360200190f35b61017d61036f565b60408051918252519081900360200190f35b610161600480360360608110156101a557600080fd5b506001600160a01b03813581169160208101359091169060400135610375565b6101cd6103cc565b6040805160ff9092168252519081900360200190f35b610161600480360360408110156101f957600080fd5b506001600160a01b0381351690602001356103d5565b61017d6004803603602081101561022557600080fd5b50356001600160a01b0316610411565b6100c061042c565b6101616004803603604081101561025357600080fd5b506001600160a01b03813516906020013561048d565b6101616004803603604081101561027f57600080fd5b506001600160a01b0381351690602001356104c9565b61017d600480360360408110156102ab57600080fd5b506001600160a01b03813581169160200135166104d6565b60038054604080516020601f600260001961010060018816150201909516949094049384018190048102820181019092528281526060939092909183018282801561034f5780601f106103245761010080835404028352916020019161034f565b820191906000526020600020905b81548152906001019060200180831161033257829003601f168201915b5050505050905090565b6000610366338484610501565b50600192915050565b60025490565b60006103828484846105ed565b6001600160a01b0384166000908152600160209081526040808320338085529252909120546103c29186916103bd908663ffffffff61072f16565b610501565b5060019392505050565b60055460ff1690565b3360008181526001602090815260408083206001600160a01b038716845290915281205490916103669185906103bd908663ffffffff61078c16565b6001600160a01b031660009081526020819052604090205490565b60048054604080516020601f600260001961010060018816150201909516949094049384018190048102820181019092528281526060939092909183018282801561034f5780601f106103245761010080835404028352916020019161034f565b3360008181526001602090815260408083206001600160a01b038716845290915281205490916103669185906103bd908663ffffffff61072f16565b60006103663384846105ed565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205490565b6001600160a01b0383166105465760405162461bcd60e51b81526004018080602001828103825260248152602001806108566024913960400191505060405180910390fd5b6001600160a01b03821661058b5760405162461bcd60e51b815260040180806020018281038252602281526020018061080f6022913960400191505060405180910390fd5b6001600160a01b03808416600081815260016020908152604080832094871680845294825291829020859055815185815291517f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9259281900390910190a3505050565b6001600160a01b0383166106325760405162461bcd60e51b81526004018080602001828103825260258152602001806108316025913960400191505060405180910390fd5b6001600160a01b0382166106775760405162461bcd60e51b81526004018080602001828103825260238152602001806107ec6023913960400191505060405180910390fd5b6001600160a01b0383166000908152602081905260409020546106a0908263ffffffff61072f16565b6001600160a01b0380851660009081526020819052604080822093909355908416815220546106d5908263ffffffff61078c16565b6001600160a01b038084166000818152602081815260409182902094909455805185815290519193928716927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef92918290030190a3505050565b600082821115610786576040805162461bcd60e51b815260206004820152601e60248201527f536166654d6174683a207375627472616374696f6e206f766572666c6f770000604482015290519081900360640190fd5b50900390565b6000828201838110156107e4576040805162461bcd60e51b815260206004820152601b60248201527a536166654d6174683a206164646974696f6e206f766572666c6f7760281b604482015290519081900360640190fd5b939250505056fe45524332303a207472616e7366657220746f20746865207a65726f206164647265737345524332303a20617070726f766520746f20746865207a65726f206164647265737345524332303a207472616e736665722066726f6d20746865207a65726f206164647265737345524332303a20617070726f76652066726f6d20746865207a65726f2061646472657373a26474726f6e58205ad8bd992125d73612e695872f74ea2d6951c0410b7633d79c611eec48000cf864736f6c63430005120031",
  "name": "Token",
  "origin_address": "TSNEe5Tf4rnc9zPMNXfaTF5fZfHDDH8oyW",
  "abi": {
    "entrys": [
      {
        "stateMutability": "Nonpayable",
        "type": "Constructor"
      },
      {
        "inputs": [
          {
            "indexed": true,
            "name": "owner",
            "type": "address"
          },
          {
            "indexed": true,
            "name": "spender",
            "type": "address"
          },
          {
            "name": "value",
            "type": "uint256"
          }
        ],
        "name": "Approval",
        "type": "Event"
      },
      {
        "inputs": [
          {
            "indexed": true,
            "name": "from",
            "type": "address"
          },
          {
            "indexed": true,
            "name": "to",
            "type": "address"
          },
          {
            "name": "value",
            "type": "uint256"
          }
        ],
        "name": "Transfer",
        "type": "Event"
      },
      {
        "outputs": [
          {
            "type": "uint256"
          }
        ],
        "constant": true,
        "inputs": [
          {
            "name": "owner",
            "type": "address"
          },
          {
            "name": "spender",
            "type": "address"
          }
        ],
        "name": "allowance",
        "stateMutability": "View",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "bool"
          }
        ],
        "inputs": [
          {
            "name": "spender",
            "type": "address"
          },
          {
            "name": "value",
            "type": "uint256"
          }
        ],
        "name": "approve",
        "stateMutability": "Nonpayable",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "uint256"
          }
        ],
        "constant": true,
        "inputs": [
          {
            "name": "account",
            "type": "address"
          }
        ],
        "name": "balanceOf",
        "stateMutability": "View",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "uint8"
          }
        ],
        "constant": true,
        "name": "decimals",
        "stateMutability": "View",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "bool"
          }
        ],
        "inputs": [
          {
            "name": "spender",
            "type": "address"
          },
          {
            "name": "subtractedValue",
            "type": "uint256"
          }
        ],
        "name": "decreaseAllowance",
        "stateMutability": "Nonpayable",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "bool"
          }
        ],
        "inputs": [
          {
            "name": "spender",
            "type": "address"
          },
          {
            "name": "addedValue",
            "type": "uint256"
          }
        ],
        "name": "increaseAllowance",
        "stateMutability": "Nonpayable",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "string"
          }
        ],
        "constant": true,
        "name": "name",
        "stateMutability": "View",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "string"
          }
        ],
        "constant": true,
        "name": "symbol",
        "stateMutability": "View",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "uint256"
          }
        ],
        "constant": true,
        "name": "totalSupply",
        "stateMutability": "View",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "bool"
          }
        ],
        "inputs": [
          {
            "name": "recipient",
            "type": "address"
          },
          {
            "name": "amount",
            "type": "uint256"
          }
        ],
        "name": "transfer",
        "stateMutability": "Nonpayable",
        "type": "Function"
      },
      {
        "outputs": [
          {
            "type": "bool"
          }
        ],
        "inputs": [
          {
            "name": "sender",
            "type": "address"
          },
          {
            "name": "recipient",
            "type": "address"
          },
          {
            "name": "amount",
            "type": "uint256"
          }
        ],
        "name": "transferFrom",
        "stateMutability": "Nonpayable",
        "type": "Function"
      }
    ]
  },
  "origin_energy_limit": 10000000,
  "contract_address": "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs",
  "code_hash": "5933c5f6804befa730c18e6bf1b14393d9e062b466c2a772f81a6bfa932a8d46"
}`
	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, contractjson, nil))

	a := assert.New(t)
	r := require.New(t)

	contract, err := solidityclient.GetContract("someaddress")
	r.Nil(err, "GetContract failed", "error:", err)

	a.Equal("TSNEe5Tf4rnc9zPMNXfaTF5fZfHDDH8oyW", contract.OriginAddress)
	a.Equal("TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs", contract.ContractAddress)
	a.Equal("608060405234801561001057600080fd5b50d3801561001d57600080fd5b50d2801561002a57600080fd5b50604080518082018252600b81526a2a32ba3432b92a37b5b2b760a91b6020808301918252835180850190945260048452631554d11560e21b90840152815191929160069161007c9160039190610236565b508151610090906004906020850190610236565b506005805460ff191660ff92909216919091179055506100cc9050336100b46100d1565b60ff16600a0a6402540be400026100db60201b60201c565b6102ce565b60055460ff165b90565b6001600160a01b038216610136576040805162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015290519081900360640190fd5b61014f816002546101d560201b61078c1790919060201c565b6002556001600160a01b0382166000908152602081815260409091205461017f91839061078c6101d5821b17901c565b6001600160a01b0383166000818152602081815260408083209490945583518581529351929391927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9281900390910190a35050565b60008282018381101561022f576040805162461bcd60e51b815260206004820152601b60248201527f536166654d6174683a206164646974696f6e206f766572666c6f770000000000604482015290519081900360640190fd5b9392505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061027757805160ff19168380011785556102a4565b828001600101855582156102a4579182015b828111156102a4578251825591602001919060010190610289565b506102b09291506102b4565b5090565b6100d891905b808211156102b057600081556001016102ba565b6108ad806102dd6000396000f3fe608060405234801561001057600080fd5b50d3801561001d57600080fd5b50d2801561002a57600080fd5b50600436106100b35760003560e01c806306fdde03146100b8578063095ea7b31461013557806318160ddd1461017557806323b872dd1461018f578063313ce567146101c557806339509351146101e357806370a082311461020f57806395d89b4114610235578063a457c2d71461023d578063a9059cbb14610269578063dd62ed3e14610295575b600080fd5b6100c06102c3565b6040805160208082528351818301528351919283929083019185019080838360005b838110156100fa5781810151838201526020016100e2565b50505050905090810190601f1680156101275780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6101616004803603604081101561014b57600080fd5b506001600160a01b038135169060200135610359565b604080519115158252519081900360200190f35b61017d61036f565b60408051918252519081900360200190f35b610161600480360360608110156101a557600080fd5b506001600160a01b03813581169160208101359091169060400135610375565b6101cd6103cc565b6040805160ff9092168252519081900360200190f35b610161600480360360408110156101f957600080fd5b506001600160a01b0381351690602001356103d5565b61017d6004803603602081101561022557600080fd5b50356001600160a01b0316610411565b6100c061042c565b6101616004803603604081101561025357600080fd5b506001600160a01b03813516906020013561048d565b6101616004803603604081101561027f57600080fd5b506001600160a01b0381351690602001356104c9565b61017d600480360360408110156102ab57600080fd5b506001600160a01b03813581169160200135166104d6565b60038054604080516020601f600260001961010060018816150201909516949094049384018190048102820181019092528281526060939092909183018282801561034f5780601f106103245761010080835404028352916020019161034f565b820191906000526020600020905b81548152906001019060200180831161033257829003601f168201915b5050505050905090565b6000610366338484610501565b50600192915050565b60025490565b60006103828484846105ed565b6001600160a01b0384166000908152600160209081526040808320338085529252909120546103c29186916103bd908663ffffffff61072f16565b610501565b5060019392505050565b60055460ff1690565b3360008181526001602090815260408083206001600160a01b038716845290915281205490916103669185906103bd908663ffffffff61078c16565b6001600160a01b031660009081526020819052604090205490565b60048054604080516020601f600260001961010060018816150201909516949094049384018190048102820181019092528281526060939092909183018282801561034f5780601f106103245761010080835404028352916020019161034f565b3360008181526001602090815260408083206001600160a01b038716845290915281205490916103669185906103bd908663ffffffff61072f16565b60006103663384846105ed565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205490565b6001600160a01b0383166105465760405162461bcd60e51b81526004018080602001828103825260248152602001806108566024913960400191505060405180910390fd5b6001600160a01b03821661058b5760405162461bcd60e51b815260040180806020018281038252602281526020018061080f6022913960400191505060405180910390fd5b6001600160a01b03808416600081815260016020908152604080832094871680845294825291829020859055815185815291517f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9259281900390910190a3505050565b6001600160a01b0383166106325760405162461bcd60e51b81526004018080602001828103825260258152602001806108316025913960400191505060405180910390fd5b6001600160a01b0382166106775760405162461bcd60e51b81526004018080602001828103825260238152602001806107ec6023913960400191505060405180910390fd5b6001600160a01b0383166000908152602081905260409020546106a0908263ffffffff61072f16565b6001600160a01b0380851660009081526020819052604080822093909355908416815220546106d5908263ffffffff61078c16565b6001600160a01b038084166000818152602081815260409182902094909455805185815290519193928716927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef92918290030190a3505050565b600082821115610786576040805162461bcd60e51b815260206004820152601e60248201527f536166654d6174683a207375627472616374696f6e206f766572666c6f770000604482015290519081900360640190fd5b50900390565b6000828201838110156107e4576040805162461bcd60e51b815260206004820152601b60248201527a536166654d6174683a206164646974696f6e206f766572666c6f7760281b604482015290519081900360640190fd5b939250505056fe45524332303a207472616e7366657220746f20746865207a65726f206164647265737345524332303a20617070726f766520746f20746865207a65726f206164647265737345524332303a207472616e736665722066726f6d20746865207a65726f206164647265737345524332303a20617070726f76652066726f6d20746865207a65726f2061646472657373a26474726f6e58205ad8bd992125d73612e695872f74ea2d6951c0410b7633d79c611eec48000cf864736f6c63430005120031", contract.Bytecode)
	a.Equal(int64(0), contract.CallValue)
	a.Equal(int64(0), contract.ConsumeUserResourcePercent)
	a.Equal("Token", contract.Name)
	a.Equal(int64(10000000), contract.OriginEnergyLimit)
	a.Equal("5933c5f6804befa730c18e6bf1b14393d9e062b466c2a772f81a6bfa932a8d46", contract.CodeHash)

	a.Equal("Nonpayable", contract.ABI.Entrys[0].StateMutability)
	a.Equal("Constructor", contract.ABI.Entrys[0].Type)

	a.Equal("Approval", contract.ABI.Entrys[1].Name)
	a.Equal("Event", contract.ABI.Entrys[1].Type)
	a.Equal(true, contract.ABI.Entrys[1].Inputs[0].Indexed)
	a.Equal("owner", contract.ABI.Entrys[1].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[1].Inputs[0].Type)
	a.Equal(true, contract.ABI.Entrys[1].Inputs[1].Indexed)
	a.Equal("spender", contract.ABI.Entrys[1].Inputs[1].Name)
	a.Equal("address", contract.ABI.Entrys[1].Inputs[1].Type)
	a.Equal("value", contract.ABI.Entrys[1].Inputs[2].Name)
	a.Equal("uint256", contract.ABI.Entrys[1].Inputs[2].Type)

	a.Equal("Transfer", contract.ABI.Entrys[2].Name)
	a.Equal("Event", contract.ABI.Entrys[2].Type)
	a.Equal(true, contract.ABI.Entrys[2].Inputs[0].Indexed)
	a.Equal("from", contract.ABI.Entrys[2].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[2].Inputs[0].Type)
	a.Equal(true, contract.ABI.Entrys[2].Inputs[1].Indexed)
	a.Equal("to", contract.ABI.Entrys[2].Inputs[1].Name)
	a.Equal("address", contract.ABI.Entrys[2].Inputs[1].Type)
	a.Equal("value", contract.ABI.Entrys[2].Inputs[2].Name)
	a.Equal("uint256", contract.ABI.Entrys[2].Inputs[2].Type)

	a.Equal("allowance", contract.ABI.Entrys[3].Name)
	a.Equal("View", contract.ABI.Entrys[3].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[3].Type)
	a.Equal("owner", contract.ABI.Entrys[3].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[3].Inputs[0].Type)
	a.Equal("spender", contract.ABI.Entrys[3].Inputs[1].Name)
	a.Equal("address", contract.ABI.Entrys[3].Inputs[1].Type)
	a.Equal("uint256", contract.ABI.Entrys[3].Outputs[0].Type)

	a.Equal("approve", contract.ABI.Entrys[4].Name)
	a.Equal("Nonpayable", contract.ABI.Entrys[4].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[4].Type)
	a.Equal("spender", contract.ABI.Entrys[4].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[4].Inputs[0].Type)
	a.Equal("value", contract.ABI.Entrys[4].Inputs[1].Name)
	a.Equal("uint256", contract.ABI.Entrys[4].Inputs[1].Type)
	a.Equal("bool", contract.ABI.Entrys[4].Outputs[0].Type)

	a.Equal("balanceOf", contract.ABI.Entrys[5].Name)
	a.Equal("View", contract.ABI.Entrys[5].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[5].Type)
	a.Equal("account", contract.ABI.Entrys[5].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[5].Inputs[0].Type)
	a.Equal("uint256", contract.ABI.Entrys[5].Outputs[0].Type)

	a.Equal("decimals", contract.ABI.Entrys[6].Name)
	a.Equal("View", contract.ABI.Entrys[6].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[6].Type)
	a.Equal(true, contract.ABI.Entrys[6].Constant)
	a.Equal("uint8", contract.ABI.Entrys[6].Outputs[0].Type)

	a.Equal("decreaseAllowance", contract.ABI.Entrys[7].Name)
	a.Equal("Nonpayable", contract.ABI.Entrys[7].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[7].Type)
	a.Equal("spender", contract.ABI.Entrys[7].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[7].Inputs[0].Type)
	a.Equal("subtractedValue", contract.ABI.Entrys[7].Inputs[1].Name)
	a.Equal("uint256", contract.ABI.Entrys[7].Inputs[1].Type)
	a.Equal("bool", contract.ABI.Entrys[7].Outputs[0].Type)

	a.Equal("increaseAllowance", contract.ABI.Entrys[8].Name)
	a.Equal("Nonpayable", contract.ABI.Entrys[8].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[8].Type)
	a.Equal("spender", contract.ABI.Entrys[8].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[8].Inputs[0].Type)
	a.Equal("addedValue", contract.ABI.Entrys[8].Inputs[1].Name)
	a.Equal("uint256", contract.ABI.Entrys[8].Inputs[1].Type)
	a.Equal("bool", contract.ABI.Entrys[8].Outputs[0].Type)

	a.Equal("name", contract.ABI.Entrys[9].Name)
	a.Equal("View", contract.ABI.Entrys[9].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[9].Type)
	a.Equal(true, contract.ABI.Entrys[9].Constant)
	a.Equal("string", contract.ABI.Entrys[9].Outputs[0].Type)

	a.Equal("symbol", contract.ABI.Entrys[10].Name)
	a.Equal("View", contract.ABI.Entrys[10].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[10].Type)
	a.Equal(true, contract.ABI.Entrys[10].Constant)
	a.Equal("string", contract.ABI.Entrys[10].Outputs[0].Type)

	a.Equal("totalSupply", contract.ABI.Entrys[11].Name)
	a.Equal("View", contract.ABI.Entrys[11].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[11].Type)
	a.Equal(true, contract.ABI.Entrys[11].Constant)
	a.Equal("uint256", contract.ABI.Entrys[11].Outputs[0].Type)

	a.Equal("transfer", contract.ABI.Entrys[12].Name)
	a.Equal("Nonpayable", contract.ABI.Entrys[12].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[12].Type)
	a.Equal("recipient", contract.ABI.Entrys[12].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[12].Inputs[0].Type)
	a.Equal("amount", contract.ABI.Entrys[12].Inputs[1].Name)
	a.Equal("uint256", contract.ABI.Entrys[12].Inputs[1].Type)
	a.Equal("bool", contract.ABI.Entrys[12].Outputs[0].Type)

	a.Equal("transferFrom", contract.ABI.Entrys[13].Name)
	a.Equal("Nonpayable", contract.ABI.Entrys[13].StateMutability)
	a.Equal("Function", contract.ABI.Entrys[13].Type)
	a.Equal("sender", contract.ABI.Entrys[13].Inputs[0].Name)
	a.Equal("address", contract.ABI.Entrys[13].Inputs[0].Type)
	a.Equal("recipient", contract.ABI.Entrys[13].Inputs[1].Name)
	a.Equal("address", contract.ABI.Entrys[13].Inputs[1].Type)
	a.Equal("amount", contract.ABI.Entrys[13].Inputs[2].Name)
	a.Equal("uint256", contract.ABI.Entrys[13].Inputs[2].Type)
	a.Equal("bool", contract.ABI.Entrys[13].Outputs[0].Type)

}

func TestContractDeploy(t *testing.T) {
	jsonresponse := `{
  "visible": true,
  "txID": "3a1890716d68306f11fba8b44752e0d9c1920cd42d880150a02d2ff93886f2fb",
  "contract_address": "4112b841b538dc8377c1032e7af5db60e9d9249148",
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
    "ref_block_bytes": "b78b",
    "ref_block_hash": "b1b3ecd0fc85b076",
    "expiration": 1724323359000,
    "timestamp": 1724323299478
  },
  "raw_data_hex": "0a02b78b2208b1b3ecd0fc85b0764098d2b9cd97325ad703081e12d2030a30747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e437265617465536d617274436f6e7472616374129d030a1541608f8da72479edc7dd921e4c30bb7e7cddbe722e1283030a1541608f8da72479edc7dd921e4c30bb7e7cddbe722e1a5c0a2b1a03736574220e12036b65791a0775696e743235362210120576616c75651a0775696e74323536300240030a2d10011a03676574220e12036b65791a0775696e743235362a10120576616c75651a0775696e743235363002400222fd01608060405234801561001057600080fd5b5060de8061001f6000396000f30060806040526004361060485763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631ab06ee58114604d5780639507d39a146067575b600080fd5b348015605857600080fd5b506065600435602435608e565b005b348015607257600080fd5b50607c60043560a0565b60408051918252519081900360200190f35b60009182526020829052604090912055565b600090815260208190526040902054905600a165627a7a72305820fdfe832221d60dd582b4526afa20518b98c2e1cb0054653053a844cf265b250400293a0c536f6d65436f6e7472616374709681b6cd9732"
}`
	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	deployRequestJson := []byte(`{
  "abi": "[{\"constant\":false,\"inputs\":[{\"name\":\"key\",\"type\":\"uint256\"},{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"set\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"key\",\"type\":\"uint256\"}],\"name\":\"get\",\"outputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]",
  "bytecode": "608060405234801561001057600080fd5b5060de8061001f6000396000f30060806040526004361060485763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631ab06ee58114604d5780639507d39a146067575b600080fd5b348015605857600080fd5b506065600435602435608e565b005b348015607257600080fd5b50607c60043560a0565b60408051918252519081900360200190f35b60009182526020829052604090912055565b600090815260208190526040902054905600a165627a7a72305820fdfe832221d60dd582b4526afa20518b98c2e1cb0054653053a844cf265b25040029",
  "owner_address": "TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh",
  "name": "SomeContract",
  "visible": true
}`)
	deployRequest := DeployContractRequest{}
	err := json.Unmarshal(deployRequestJson, &deployRequest)
	r.Nil(err, "marshalling deploy contract request failed:", err)

	transaction, err := solidityclient.DeployContract(&deployRequest)
	r.Nil(err, "DeployContract failed", "error:", err)

	a.Equal("0a02b78b2208b1b3ecd0fc85b0764098d2b9cd97325ad703081e12d2030a30747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e437265617465536d617274436f6e7472616374129d030a1541608f8da72479edc7dd921e4c30bb7e7cddbe722e1283030a1541608f8da72479edc7dd921e4c30bb7e7cddbe722e1a5c0a2b1a03736574220e12036b65791a0775696e743235362210120576616c75651a0775696e74323536300240030a2d10011a03676574220e12036b65791a0775696e743235362a10120576616c75651a0775696e743235363002400222fd01608060405234801561001057600080fd5b5060de8061001f6000396000f30060806040526004361060485763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631ab06ee58114604d5780639507d39a146067575b600080fd5b348015605857600080fd5b506065600435602435608e565b005b348015607257600080fd5b50607c60043560a0565b60408051918252519081900360200190f35b60009182526020829052604090912055565b600090815260208190526040902054905600a165627a7a72305820fdfe832221d60dd582b4526afa20518b98c2e1cb0054653053a844cf265b250400293a0c536f6d65436f6e7472616374709681b6cd9732", transaction.RawDataHex)
	a.Equal("b78b", transaction.RawData.RefBlockBytes)
	a.Equal("b1b3ecd0fc85b076", transaction.RawData.RefBlockHash)
	a.Equal(int64(1724323359000), transaction.RawData.Expiration)
	a.Equal(int64(1724323299478), transaction.RawData.Timestamp)
	a.Equal("CreateSmartContract", transaction.RawData.Contract[0].Type)
	a.Equal(deployRequest.OwnerAddress, transaction.RawData.Contract[0].Parameter.Value.OwnerAddress)
	a.Equal("608060405234801561001057600080fd5b5060de8061001f6000396000f30060806040526004361060485763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416631ab06ee58114604d5780639507d39a146067575b600080fd5b348015605857600080fd5b506065600435602435608e565b005b348015607257600080fd5b50607c60043560a0565b60408051918252519081900360200190f35b60009182526020829052604090912055565b600090815260208190526040902054905600a165627a7a72305820fdfe832221d60dd582b4526afa20518b98c2e1cb0054653053a844cf265b25040029", transaction.RawData.Contract[0].Parameter.Value.NewContract.Bytecode)
	a.Equal("SomeContract", transaction.RawData.Contract[0].Parameter.Value.NewContract.Name)
	a.Equal("TJmmqjb1DK9TTZbQXzRQ2AuA94z4gKAPFh", transaction.RawData.Contract[0].Parameter.Value.NewContract.OriginAddress)
}

func TestTriggerSmartContract(t *testing.T) {
	jsonresponse := `{
  "result": {
    "result": true
  },
  "transaction": {
    "visible": true,
    "txID": "c067951834db369d4e94080f4ebd8f54885e76e8cfd1545e6afbcf63d52842a2",
    "raw_data": {
      "contract": [
        {
          "parameter": {
            "value": {
              "data": "a9059cbb00000000000000000000004115208ef33a926919ed270e2fa61367b2da3753da0000000000000000000000000000000000000000000000000000000000000032",
              "owner_address": "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
              "contract_address": "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs"
            },
            "type_url": "type.googleapis.com/protocol.TriggerSmartContract"
          },
          "type": "TriggerSmartContract"
        }
      ],
      "ref_block_bytes": "26da",
      "ref_block_hash": "6e5a2311d5f54849",
      "expiration": 1724409708000,
      "fee_limit": 1000000000,
      "timestamp": 1724409648766
    },
    "raw_data_hex": "0a0226da22086e5a2311d5f5484940e0fbcff697325aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a1541fd49eda0f23ff7ec1d03b52c3a45991c24cd440e12154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e02244a9059cbb00000000000000000000004115208ef33a926919ed270e2fa61367b2da3753da000000000000000000000000000000000000000000000000000000000000003270feacccf6973290018094ebdc03"
  }
}`

	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	triggerResponse, err := solidityclient.TriggerSmartContract(&TriggerSmartContractRequest{OwnerAddress: "owneraddress"})
	r.Nil(err, "get transaction info by id failed:", err)

	a.True(triggerResponse.Result.Result)
	a.True(triggerResponse.Transaction.Visible)
	a.Equal("c067951834db369d4e94080f4ebd8f54885e76e8cfd1545e6afbcf63d52842a2", triggerResponse.Transaction.TxID)
	r.Equal(1, len(triggerResponse.Transaction.RawData.Contract))
	a.Equal("a9059cbb00000000000000000000004115208ef33a926919ed270e2fa61367b2da3753da0000000000000000000000000000000000000000000000000000000000000032", triggerResponse.Transaction.RawData.Contract[0].Parameter.Value.Data)
	a.Equal("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", triggerResponse.Transaction.RawData.Contract[0].Parameter.Value.OwnerAddress)
	a.Equal("TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs", triggerResponse.Transaction.RawData.Contract[0].Parameter.Value.ContractAddress)
	a.Equal("type.googleapis.com/protocol.TriggerSmartContract", triggerResponse.Transaction.RawData.Contract[0].Parameter.TypeUrl)
	a.Equal("TriggerSmartContract", triggerResponse.Transaction.RawData.Contract[0].Type)
	a.Equal("26da", triggerResponse.Transaction.RawData.RefBlockBytes)
	a.Equal("6e5a2311d5f54849", triggerResponse.Transaction.RawData.RefBlockHash)
	a.Equal(int64(1724409708000), triggerResponse.Transaction.RawData.Expiration)
	a.Equal(int64(1000000000), triggerResponse.Transaction.RawData.FeeLimit)
	a.Equal(int64(1724409648766), triggerResponse.Transaction.RawData.Timestamp)
	a.Equal("0a0226da22086e5a2311d5f5484940e0fbcff697325aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a1541fd49eda0f23ff7ec1d03b52c3a45991c24cd440e12154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e02244a9059cbb00000000000000000000004115208ef33a926919ed270e2fa61367b2da3753da000000000000000000000000000000000000000000000000000000000000003270feacccf6973290018094ebdc03", triggerResponse.Transaction.RawDataHex)
}

func TestTriggerConstantContract(t *testing.T) {
	jsonresponse := `{
  "result": {
    "result": true
  },
  "energy_used": 541,
  "constant_result": [
    "00000000000000000000000000000000000000000000000000000001663ea8d6"
  ],
  "transaction": {
    "ret": [
      {}
    ],
    "visible": true,
    "txID": "bf5d8b1917f8c6d31793ac0c0428b23a0a711143860fcf9be34b53801fd17454",
    "raw_data": {
      "contract": [
        {
          "parameter": {
            "value": {
              "data": "70a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c",
              "owner_address": "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
              "contract_address": "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs"
            },
            "type_url": "type.googleapis.com/protocol.TriggerSmartContract"
          },
          "type": "TriggerSmartContract"
        }
      ],
      "ref_block_bytes": "2adf",
      "ref_block_hash": "62a72c7aa9af9c65",
      "expiration": 1724412825000,
      "timestamp": 1724412768022
    },
    "raw_data_hex": "0a022adf220862a72c7aa9af9c6540a89b8ef897325a8e01081f1289010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412540a1541fd49eda0f23ff7ec1d03b52c3a45991c24cd440e12154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e0222470a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c7096de8af89732"
  }
}`

	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	triggerConstantC, err := solidityclient.TriggerConstantContract(&TriggerConstantContractRequest{})
	r.Nil(err, "trigger constant contract failed: %w", err)

	a.True(triggerConstantC.Result.Result)
	a.Equal(int64(541), triggerConstantC.EnergyUsed)
	a.Equal("00000000000000000000000000000000000000000000000000000001663ea8d6", triggerConstantC.ConstantResult[0])
	a.True(triggerConstantC.Transaction.Visible)
	a.Equal("bf5d8b1917f8c6d31793ac0c0428b23a0a711143860fcf9be34b53801fd17454", triggerConstantC.Transaction.TxID)
	r.Equal(1, len(triggerConstantC.Transaction.RawData.Contract))
	a.Equal("70a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c",
		triggerConstantC.Transaction.RawData.Contract[0].Parameter.Value.Data)
	a.Equal("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", triggerConstantC.Transaction.RawData.Contract[0].Parameter.Value.OwnerAddress)
	a.Equal("TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs", triggerConstantC.Transaction.RawData.Contract[0].Parameter.Value.ContractAddress)
	a.Equal("type.googleapis.com/protocol.TriggerSmartContract", triggerConstantC.Transaction.RawData.Contract[0].Parameter.TypeUrl)
	a.Equal("TriggerSmartContract", triggerConstantC.Transaction.RawData.Contract[0].Type)
	a.Equal("2adf", triggerConstantC.Transaction.RawData.RefBlockBytes)
	a.Equal("62a72c7aa9af9c65", triggerConstantC.Transaction.RawData.RefBlockHash)
	a.Equal(int64(1724412825000), triggerConstantC.Transaction.RawData.Expiration)
	a.Equal(int64(1724412768022), triggerConstantC.Transaction.RawData.Timestamp)
	a.Equal("0a022adf220862a72c7aa9af9c6540a89b8ef897325a8e01081f1289010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412540a1541fd49eda0f23ff7ec1d03b52c3a45991c24cd440e12154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e0222470a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c7096de8af89732",
		triggerConstantC.Transaction.RawDataHex)

}

func TestBroadcastTransactionFailure(t *testing.T) {
	broadcasterr := "SIGERROR"
	broadcastMessage := "56616c6964617465207369676e6174757265206572726f723a206d69737320736967206f7220636f6e7472616374"
	jsonresponse := `{
  "code": "` + broadcasterr + `",
  "txid": "77ddfa7093cc5f745c0d3a54abb89ef070f983343c05e0f89e5a52f3e5401299",
  "message": "` + broadcastMessage + `"
}`
	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	broadcastResponse, err := solidityclient.BroadcastTransaction(&Transaction{})
	r.Nil(broadcastResponse, "broadcast response should be nil")
	r.NotNil(err, "broadcast successful when it should fail: %w", err)
	errstr := fmt.Sprintf("broadcasting failed. Code: %s, Message: %s", broadcasterr, broadcastMessage)
	a.Equal(errstr, err.Error())

}
