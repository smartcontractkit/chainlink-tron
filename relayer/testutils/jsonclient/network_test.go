package jsonclient

var expectedEnergyPrices = &EnergyPrice{
	Prices: "0:100,1575871200000:10,1606537680000:40,1614238080000:140,1635739080000:280,1681895880000:420",
}

var expectedGetBlockByNum = &Block{
	BlockID: "00000000000000c86d2473411771f83db5e314c01bc8f8cf0dc2f8892be6fd7f",
	BlockHeader: BlockHeader{
		RawData: &BlockHeaderRaw{
			ParentHash:     "00000000000000c7d4d47132f21fd0b74e2f8bcb0c2e9130f7cab35b5d38af9f",
			Version:        9,
			Timestamp:      1575594618000,
			Number:         200,
			TxTrieRoot:     "0000000000000000000000000000000000000000000000000000000000000000",
			WitnessAddress: "41f16412b9a17ee9408646e2a21e16478f72ed1e95",
		},
		WitnessSignature: "97ecda5b130600d18304e02f7fd5ab9d115c5ec9c0e312c8c6fe83939771bb85505fafee598541dc902b1a7b8ca2735c83a12e640203ed4b8529d47ce4f413df00",
	},
}

var expectedGetNowBlock = &Block{
	Transactions: []BlockTransactions{
		{
			Ret: []Return{
				{
					ContractRet: "SUCCESS",
				},
			},
			Signature: []string{
				"e921a2fdbcc51449a331c5cd5b7586dbbf168cfee1d614516c704675cdcc7765334f2b5e33fcd22bd0f2ed143e5c19c8bac54066bf1acc748fb925382484bd4f01",
			},
			TxID: "ab1f6628abb23a09a1bd1fe3f632cb917e7f6c1a438527cc2a32f63d92458a10",
			RawData: RawData{
				RefBlockBytes: "1542",
				RefBlockHash:  "9d620d0dc8f981be",
				Expiration:    1724396064000,
				Timestamp:     1724396005011,
				Contract: []Contract{
					{
						Parameter: Parameter{
							Value: ParameterValue{
								OwnerAddress: "41382b835dd735c2d634646138ff8ef333d9ab0502",
								ToAddress:    "4137a9511e294f1757678f366ad3b6102d35ebd4dc",
								Amount:       1000,
							},
							TypeUrl: "type.googleapis.com/protocol.TransferContract",
						},
						Type: "TransferContract",
					},
				},
			},
			RawDataHex: "0a02154222089d620d0dc8f981be40809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077093cd8bf09732",
		},
		{
			Ret: []Return{
				{
					ContractRet: "SUCCESS",
				},
			},
			Signature: []string{
				"ad89e79778b21e78a32aed29f162e55f98393e4518de463a2888a4608171c62c1e0cf0e00572c3a04e7f3fabfceb09c30aaa56a2e1c13f50404c1f590989090c00",
			},
			TxID: "599b66a8d658974e57f906ed1953d66b8723403fe48198fa2c575668f6b9de5a",
			RawData: RawData{
				Timestamp: 1724396005010,
				Contract: []Contract{
					{
						Parameter: Parameter{
							Value: ParameterValue{
								Amount:       1000,
								OwnerAddress: "41382b835dd735c2d634646138ff8ef333d9ab0502",
								ToAddress:    "4137a9511e294f1757678f366ad3b6102d35ebd4dc",
							},
							TypeUrl: "type.googleapis.com/protocol.TransferContract",
						},
						Type: "TransferContract",
					},
				},
				RefBlockBytes: "1542",
				RefBlockHash:  "9d620d0dc8f981be",
				Expiration:    1724396064000,
			},
			RawDataHex: "0a02154222089d620d0dc8f981be40809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077092cd8bf09732",
		},
		{
			RawData: RawData{
				RefBlockHash: "59381cdb45f11478",
				Expiration:   1724396064000,
				Timestamp:    1724396005251,
				Contract: []Contract{
					{
						Parameter: Parameter{
							Value: ParameterValue{
								OwnerAddress: "41382b835dd735c2d634646138ff8ef333d9ab0502",
								ToAddress:    "4137a9511e294f1757678f366ad3b6102d35ebd4dc",
								Amount:       1000,
							},
							TypeUrl: "type.googleapis.com/protocol.TransferContract",
						},
						Type: "TransferContract",
					},
				},
				RefBlockBytes: "1544",
			},
			RawDataHex: "0a021544220859381cdb45f1147840809a8ff097325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541382b835dd735c2d634646138ff8ef333d9ab050212154137a9511e294f1757678f366ad3b6102d35ebd4dc18e8077083cf8bf09732",
			Ret: []Return{
				{
					ContractRet: "SUCCESS",
				},
			},
			Signature: []string{
				"4a7737a2ba6de1c864a4daa6aaa3666ac496b8c3ebaf2da54a4e8a13ca8b995f5d825647ed5aa7ab302742b7210b03a8ab56ff6ea4f1423ef808ec7913af0e1600",
			},
			TxID: "a3e7da80300fc5b5220cf88d726c1d60c998073bc659d77bc419ede7443d95bc",
		},
		{
			Signature: []string{
				"f5468fa750386595535d8e6fa9be5f509a5daf8208de8de9a15734d8e8af9ab86d6a04eaa48966647f2873b9bad2934bd798c8834e7155b74c44eb3bb667e9d21b",
			},
			TxID: "e8fa01d91adde847d38112d022c441cc352c1636585182a2c1bfa5905e09891e",
			RawData: RawData{
				Timestamp: 1724396002996,
				Contract: []Contract{
					{
						Parameter: Parameter{
							Value: ParameterValue{
								Data:            "a9059cbb00000000000000000000000083e99c951715215d63b36d8d126a2b81ad1b2c2d0000000000000000000000000000000000000000000000000000000011e1a300",
								OwnerAddress:    "417edc1887cbb12c5a3eee51ec1a4882fd1d48e400",
								ContractAddress: "4142a1e39aefa49290f2b3f9ed688d7cecf86cd6e0",
							},
							TypeUrl: "type.googleapis.com/protocol.TriggerSmartContract",
						},
						Type: "TriggerSmartContract",
					},
				},
				RefBlockBytes: "1543",
				RefBlockHash:  "53af20522d284657",
				Expiration:    1724396061000,
				FeeLimit:      1000000000,
			},
			RawDataHex: "0a021543220853af20522d28465740c8828ff097325aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a15417edc1887cbb12c5a3eee51ec1a4882fd1d48e40012154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e02244a9059cbb00000000000000000000000083e99c951715215d63b36d8d126a2b81ad1b2c2d0000000000000000000000000000000000000000000000000000000011e1a30070b4bd8bf0973290018094ebdc03",
			Ret: []Return{
				{
					ContractRet: "SUCCESS",
				},
			},
		},
		{
			Signature: []string{
				"15318100246b2a01f1f2dc641b1b50dc75e0e6f10e97349fecdba78827e003f228ba6825384251be5acbdb136c70586280d703ac29e3e611333f659b0ff86ba11b",
			},
			TxID: "68bc1244da2dbcde50954c4b250d07a258eb02fbb228681d600dfac27b44d145",
			RawData: RawData{
				Contract: []Contract{
					{
						Parameter: Parameter{
							Value: ParameterValue{
								OwnerAddress: "41bc649db213a41a11aac2d3186e832fa9fc84c030",
								ToAddress:    "414206ac4a5fe33602dc557b1352675d2ff1456222",
								Amount:       100000,
							},
							TypeUrl: "type.googleapis.com/protocol.TransferContract",
						},
						Type: "TransferContract",
					},
				},
				RefBlockBytes: "1556",
				RefBlockHash:  "fbed6c734d1bff11",
				Expiration:    1724396064000,
				Timestamp:     1724396004000,
			},
			RawDataHex: "0a0215562208fbed6c734d1bff1140809a8ff097325a67080112630a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412320a1541bc649db213a41a11aac2d3186e832fa9fc84c0301215414206ac4a5fe33602dc557b1352675d2ff145622218a08d0670a0c58bf09732",
			Ret: []Return{
				{
					ContractRet: "SUCCESS",
				},
			},
		},
	},
	BlockID: "0000000002cc15576e3142f7d82e5dddaa21d6c19c798bdbda2085dbb24e7fb1",
	BlockHeader: BlockHeader{
		WitnessSignature: "31c7a2ca838d47d29ccd008adbca0aabd1bcfea4e649402b529b067126bb9d6777ee6391f16180968cdb57017885f21a5af2b7da4f350cff82d219dd7dd81ee901",
		RawData: &BlockHeaderRaw{
			Number:         46929239,
			TxTrieRoot:     "a68786f79e81cd50754c111440e5e2701abe7a7c157efd3de5e699b9e52aa0fd",
			WitnessAddress: "41d0668c49826f2ca13e3848b5eb3414fb059b7cc0",
			ParentHash:     "0000000002cc1556fbed6c734d1bff1154cec46dd572ad0d4a3142a373a1a829",
			Version:        30,
			Timestamp:      1724396007000,
		},
	},
}
