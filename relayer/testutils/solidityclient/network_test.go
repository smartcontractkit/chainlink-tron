package solidityclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnergyPrices(t *testing.T) {
	jsonresponse := `{
  "prices": "0:100,1575871200000:10,1606537680000:40,1614238080000:140,1635739080000:280,1681895880000:420"
}`
	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	eprices, err := solidityclient.GetEnergyPrices()
	r.Nil(err, "get energy prices failed:", err)
	a.Equal("0:100,1575871200000:10,1606537680000:40,1614238080000:140,1635739080000:280,1681895880000:420", eprices.Prices)
}

func TestGetNowBlock(t *testing.T) {
	jsonresponse := `{
  "blockID": "0000000002cc15576e3142f7d82e5dddaa21d6c19c798bdbda2085dbb24e7fb1",
  "block_header": {
    "raw_data": {
      "number": 46929239,
      "txTrieRoot": "a68786f79e81cd50754c111440e5e2701abe7a7c157efd3de5e699b9e52aa0fd",
      "witness_address": "41d0668c49826f2ca13e3848b5eb3414fb059b7cc0",
      "parentHash": "0000000002cc1556fbed6c734d1bff1154cec46dd572ad0d4a3142a373a1a829",
      "version": 30,
      "timestamp": 1724396007000
    },
    "witness_signature": "31c7a2ca838d47d29ccd008adbca0aabd1bcfea4e649402b529b067126bb9d6777ee6391f16180968cdb57017885f21a5af2b7da4f350cff82d219dd7dd81ee901"
  },
  "transactions": [
    {
      "ret": [
        {
          "contractRet": "SUCCESS"
        }
      ],
      "signature": [
        "e921a2fdbcc51449a331c5cd5b7586dbbf168cfee1d614516c704675cdcc7765334f2b5e33fcd22bd0f2ed143e5c19c8bac54066bf1acc748fb925382484bd4f01"
      ],
      "txID": "ab1f6628abb23a09a1bd1fe3f632cb917e7f6c1a438527cc2a32f63d92458a10",
      "raw_data": {
        "contract": [
          {
            "parameter": {
              "value": {
                "amount": 1000,
                "owner_address": "41382b835dd735c2d634646138ff8ef333d9ab0502",
                "to_address": "4137a9511e294f1757678f366ad3b6102d35ebd4dc"
              },
              "type_url": "type.googleapis.com/protocol.TransferContract"
            },
            "type": "TransferContract"
          }
        ],
        "ref_block_bytes": "1542",
        "ref_block_hash": "9d620d0dc8f981be",
        "expiration": 1724396064000,
        "timestamp": 1724396005011
      },
      "raw_data_hex": "0a02154222089d620d0dc8f981be40809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077093cd8bf09732"
    },
    {
      "ret": [
        {
          "contractRet": "SUCCESS"
        }
      ],
      "signature": [
        "ad89e79778b21e78a32aed29f162e55f98393e4518de463a2888a4608171c62c1e0cf0e00572c3a04e7f3fabfceb09c30aaa56a2e1c13f50404c1f590989090c00"
      ],
      "txID": "599b66a8d658974e57f906ed1953d66b8723403fe48198fa2c575668f6b9de5a",
      "raw_data": {
        "contract": [
          {
            "parameter": {
              "value": {
                "amount": 1000,
                "owner_address": "41382b835dd735c2d634646138ff8ef333d9ab0502",
                "to_address": "4137a9511e294f1757678f366ad3b6102d35ebd4dc"
              },
              "type_url": "type.googleapis.com/protocol.TransferContract"
            },
            "type": "TransferContract"
          }
        ],
        "ref_block_bytes": "1542",
        "ref_block_hash": "9d620d0dc8f981be",
        "expiration": 1724396064000,
        "timestamp": 1724396005010
      },
      "raw_data_hex": "0a02154222089d620d0dc8f981be40809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077092cd8bf09732"
    },
    {
      "ret": [
        {
          "contractRet": "SUCCESS"
        }
      ],
      "signature": [
        "4a7737a2ba6de1c864a4daa6aaa3666ac496b8c3ebaf2da54a4e8a13ca8b995f5d825647ed5aa7ab302742b7210b03a8ab56ff6ea4f1423ef808ec7913af0e1600"
      ],
      "txID": "a3e7da80300fc5b5220cf88d726c1d60c998073bc659d77bc419ede7443d95bc",
      "raw_data": {
        "contract": [
          {
            "parameter": {
              "value": {
                "amount": 1000,
                "owner_address": "41382b835dd735c2d634646138ff8ef333d9ab0502",
                "to_address": "4137a9511e294f1757678f366ad3b6102d35ebd4dc"
              },
              "type_url": "type.googleapis.com/protocol.TransferContract"
            },
            "type": "TransferContract"
          }
        ],
        "ref_block_bytes": "1544",
        "ref_block_hash": "59381cdb45f11478",
        "expiration": 1724396064000,
        "timestamp": 1724396005251
      },
      "raw_data_hex": "0a021544220859381cdb45f1147840809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077083cf8bf09732"
    },
    {
      "ret": [
        {
          "contractRet": "SUCCESS"
        }
      ],
      "signature": [
        "f5468fa750386595535d8e6fa9be5f509a5daf8208de8de9a15734d8e8af9ab86d6a04eaa48966647f2873b9bad2934bd798c8834e7155b74c44eb3bb667e9d21b"
      ],
      "txID": "e8fa01d91adde847d38112d022c441cc352c1636585182a2c1bfa5905e09891e",
      "raw_data": {
        "contract": [
          {
            "parameter": {
              "value": {
                "data": "a9059cbb00000000000000000000000083e99c951715215d63b36d8d126a2b81ad1b2c2d0000000000000000000000000000000000000000000000000000000011e1a300",
                "owner_address": "417edc1887cbb12c5a3eee51ec1a4882fd1d48e400",
                "contract_address": "4142a1e39aefa49290f2b3f9ed688d7cecf86cd6e0"
              },
              "type_url": "type.googleapis.com/protocol.TriggerSmartContract"
            },
            "type": "TriggerSmartContract"
          }
        ],
        "ref_block_bytes": "1543",
        "ref_block_hash": "53af20522d284657",
        "expiration": 1724396061000,
        "fee_limit": 1000000000,
        "timestamp": 1724396002996
      },
      "raw_data_hex": "0a021543220853af20522d28465740c8828ff097325aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a15417edc1887cbb12c5a3eee51ec1a4882fd1d48e40012154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e02244a9059cbb00000000000000000000000083e99c951715215d63b36d8d126a2b81ad1b2c2d0000000000000000000000000000000000000000000000000000000011e1a30070b4bd8bf0973290018094ebdc03"
    },
    {
      "ret": [
        {
          "contractRet": "SUCCESS"
        }
      ],
      "signature": [
        "15318100246b2a01f1f2dc641b1b50dc75e0e6f10e97349fecdba78827e003f228ba6825384251be5acbdb136c70586280d703ac29e3e611333f659b0ff86ba11b"
      ],
      "txID": "68bc1244da2dbcde50954c4b250d07a258eb02fbb228681d600dfac27b44d145",
      "raw_data": {
        "contract": [
          {
            "parameter": {
              "value": {
                "amount": 100000,
                "owner_address": "41bc649db213a41a11aac2d3186e832fa9fc84c030",
                "to_address": "414206ac4a5fe33602dc557b1352675d2ff1456222"
              },
              "type_url": "type.googleapis.com/protocol.TransferContract"
            },
            "type": "TransferContract"
          }
        ],
        "ref_block_bytes": "1556",
        "ref_block_hash": "fbed6c734d1bff11",
        "expiration": 1724396064000,
        "timestamp": 1724396004000
      },
      "raw_data_hex": "0a0215562208fbed6c734d1bff1140809a8ff097325a67080112630a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412320a1541bc649db213a41a11aac2d3186e832fa9fc84c0301215414206ac4a5fe33602dc557b1352675d2ff145622218a08d0670a0c58bf09732"
    }
  ]
}`
	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	block, err := solidityclient.GetNowBlock()
	r.Nil(err, "get now block failed:", err)
	a.Equal("0000000002cc15576e3142f7d82e5dddaa21d6c19c798bdbda2085dbb24e7fb1", block.BlockID)
	a.Equal(int64(1724396007000), block.BlockHeader.RawData.Timestamp)
	a.Equal("a68786f79e81cd50754c111440e5e2701abe7a7c157efd3de5e699b9e52aa0fd", block.BlockHeader.RawData.TxTrieRoot)
	a.Equal("0000000002cc1556fbed6c734d1bff1154cec46dd572ad0d4a3142a373a1a829", block.BlockHeader.RawData.ParentHash)
	a.Equal(int64(46929239), block.BlockHeader.RawData.Number)
	a.Equal("41d0668c49826f2ca13e3848b5eb3414fb059b7cc0", block.BlockHeader.RawData.WitnessAddress)
	a.Equal(int32(30), block.BlockHeader.RawData.Version)
	a.Equal("31c7a2ca838d47d29ccd008adbca0aabd1bcfea4e649402b529b067126bb9d6777ee6391f16180968cdb57017885f21a5af2b7da4f350cff82d219dd7dd81ee901", block.BlockHeader.WitnessSignature)
	a.Equal(5, len(block.Transactions))

	a.Equal("SUCCESS", block.Transactions[0].Ret[0].ContractRet)
	a.Equal("ab1f6628abb23a09a1bd1fe3f632cb917e7f6c1a438527cc2a32f63d92458a10",
		block.Transactions[0].TxID)
	a.Equal("e921a2fdbcc51449a331c5cd5b7586dbbf168cfee1d614516c704675cdcc7765334f2b5e33fcd22bd0f2ed143e5c19c8bac54066bf1acc748fb925382484bd4f01",
		block.Transactions[0].Signature[0])
	a.Equal("TransferContract", block.Transactions[0].RawData.Contract[0].Type)
	a.Equal("1542", block.Transactions[0].RawData.RefBlockBytes)
	a.Equal("9d620d0dc8f981be", block.Transactions[0].RawData.RefBlockHash)
	a.Equal(int64(1724396064000), block.Transactions[0].RawData.Expiration)
	a.Equal(int64(1724396005011), block.Transactions[0].RawData.Timestamp)
	a.Equal("0a02154222089d620d0dc8f981be40809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077093cd8bf09732", block.Transactions[0].RawDataHex)

	a.Equal("SUCCESS", block.Transactions[1].Ret[0].ContractRet)
	a.Equal("599b66a8d658974e57f906ed1953d66b8723403fe48198fa2c575668f6b9de5a",
		block.Transactions[1].TxID)
	a.Equal("ad89e79778b21e78a32aed29f162e55f98393e4518de463a2888a4608171c62c1e0cf0e00572c3a04e7f3fabfceb09c30aaa56a2e1c13f50404c1f590989090c00",
		block.Transactions[1].Signature[0])
	a.Equal("TransferContract", block.Transactions[1].RawData.Contract[0].Type)
	a.Equal("1542", block.Transactions[1].RawData.RefBlockBytes)
	a.Equal("9d620d0dc8f981be", block.Transactions[1].RawData.RefBlockHash)
	a.Equal(int64(1724396064000), block.Transactions[1].RawData.Expiration)
	a.Equal(int64(1724396005010), block.Transactions[1].RawData.Timestamp)
	a.Equal("0a02154222089d620d0dc8f981be40809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077092cd8bf09732", block.Transactions[1].RawDataHex)

	a.Equal("SUCCESS", block.Transactions[2].Ret[0].ContractRet)
	a.Equal("a3e7da80300fc5b5220cf88d726c1d60c998073bc659d77bc419ede7443d95bc", block.Transactions[2].TxID)
	a.Equal("4a7737a2ba6de1c864a4daa6aaa3666ac496b8c3ebaf2da54a4e8a13ca8b995f5d825647ed5aa7ab302742b7210b03a8ab56ff6ea4f1423ef808ec7913af0e1600", block.Transactions[2].Signature[0])
	a.Equal("TransferContract", block.Transactions[2].RawData.Contract[0].Type)
	a.Equal("1544", block.Transactions[2].RawData.RefBlockBytes)
	a.Equal("59381cdb45f11478", block.Transactions[2].RawData.RefBlockHash)
	a.Equal(int64(1724396064000), block.Transactions[2].RawData.Expiration)
	a.Equal(int64(1724396005251), block.Transactions[2].RawData.Timestamp)
	a.Equal("0a021544220859381cdb45f1147840809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077083cf8bf09732", block.Transactions[2].RawDataHex)

	a.Equal("SUCCESS", block.Transactions[3].Ret[0].ContractRet)
	a.Equal("e8fa01d91adde847d38112d022c441cc352c1636585182a2c1bfa5905e09891e", block.Transactions[3].TxID)
	a.Equal("f5468fa750386595535d8e6fa9be5f509a5daf8208de8de9a15734d8e8af9ab86d6a04eaa48966647f2873b9bad2934bd798c8834e7155b74c44eb3bb667e9d21b", block.Transactions[3].Signature[0])
	a.Equal("TriggerSmartContract", block.Transactions[3].RawData.Contract[0].Type)
	a.Equal("1543", block.Transactions[3].RawData.RefBlockBytes)
	a.Equal("53af20522d284657", block.Transactions[3].RawData.RefBlockHash)
	a.Equal(int64(1724396061000), block.Transactions[3].RawData.Expiration)
	a.Equal(int64(1000000000), block.Transactions[3].RawData.FeeLimit)
	a.Equal(int64(1724396002996), block.Transactions[3].RawData.Timestamp)
	a.Equal("0a021543220853af20522d28465740c8828ff097325aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a15417edc1887cbb12c5a3eee51ec1a4882fd1d48e40012154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e02244a9059cbb00000000000000000000000083e99c951715215d63b36d8d126a2b81ad1b2c2d0000000000000000000000000000000000000000000000000000000011e1a30070b4bd8bf0973290018094ebdc03", block.Transactions[3].RawDataHex)

	a.Equal("SUCCESS", block.Transactions[4].Ret[0].ContractRet)
	a.Equal("68bc1244da2dbcde50954c4b250d07a258eb02fbb228681d600dfac27b44d145", block.Transactions[4].TxID)
	a.Equal("15318100246b2a01f1f2dc641b1b50dc75e0e6f10e97349fecdba78827e003f228ba6825384251be5acbdb136c70586280d703ac29e3e611333f659b0ff86ba11b", block.Transactions[4].Signature[0])
	a.Equal("TransferContract", block.Transactions[4].RawData.Contract[0].Type)
	a.Equal("1556", block.Transactions[4].RawData.RefBlockBytes)
	a.Equal("fbed6c734d1bff11", block.Transactions[4].RawData.RefBlockHash)
	a.Equal(int64(1724396064000), block.Transactions[4].RawData.Expiration)
	a.Equal(int64(1724396004000), block.Transactions[4].RawData.Timestamp)
	a.Equal("0a0215562208fbed6c734d1bff1140809a8ff097325a67080112630a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412320a1541bc649db213a41a11aac2d3186e832fa9fc84c0301215414206ac4a5fe33602dc557b1352675d2ff145622218a08d0670a0c58bf09732", block.Transactions[4].RawDataHex)

}

func TestGetBlockByNumRequest(t *testing.T) {
	jsonresponse := `{
  "blockID": "00000000000000c86d2473411771f83db5e314c01bc8f8cf0dc2f8892be6fd7f",
  "block_header": {
    "raw_data": {
      "number": 200,
      "txTrieRoot": "0000000000000000000000000000000000000000000000000000000000000000",
      "witness_address": "41f16412b9a17ee9408646e2a21e16478f72ed1e95",
      "parentHash": "00000000000000c7d4d47132f21fd0b74e2f8bcb0c2e9130f7cab35b5d38af9f",
      "version": 9,
      "timestamp": 1575594618000
    },
    "witness_signature": "97ecda5b130600d18304e02f7fd5ab9d115c5ec9c0e312c8c6fe83939771bb85505fafee598541dc902b1a7b8ca2735c83a12e640203ed4b8529d47ce4f413df00"
  }
}`
	code := http.StatusOK
	solidityclient := NewTronSolidityClient("baseurl", NewMockSolidityClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	block, err := solidityclient.GetBlockByNum(123)
	r.Nil(err, "get block by number failed:", err)

	a.Equal("00000000000000c86d2473411771f83db5e314c01bc8f8cf0dc2f8892be6fd7f", block.BlockID)
	a.Equal("97ecda5b130600d18304e02f7fd5ab9d115c5ec9c0e312c8c6fe83939771bb85505fafee598541dc902b1a7b8ca2735c83a12e640203ed4b8529d47ce4f413df00", block.BlockHeader.WitnessSignature)
	a.Equal(int64(200), block.BlockHeader.RawData.Number)
	a.Equal("0000000000000000000000000000000000000000000000000000000000000000", block.BlockHeader.RawData.TxTrieRoot)
	a.Equal("41f16412b9a17ee9408646e2a21e16478f72ed1e95", block.BlockHeader.RawData.WitnessAddress)
	a.Equal("00000000000000c7d4d47132f21fd0b74e2f8bcb0c2e9130f7cab35b5d38af9f", block.BlockHeader.RawData.ParentHash)
	a.Equal(int32(9), block.BlockHeader.RawData.Version)
	a.Equal(int64(1575594618000), block.BlockHeader.RawData.Timestamp)
}
