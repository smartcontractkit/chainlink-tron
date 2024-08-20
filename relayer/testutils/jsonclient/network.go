package jsonclient

import "fmt"

type EnergyPrice struct {
	Prices string `json:"prices"` // All historical energy unit price information. Each unit price change is separated by a comma. Before the colon is the millisecond timestamp, and after the colon is the energy unit price in sun.
}

func (tc *TronJsonClient) GetEnergyPrices() (*[]EnergyPrice, error) {
	energyPrices := []EnergyPrice{}
	getEnergyPricesEndpoint := "/wallet/getenergyprices"

	_, _, err := tc.get(tc.baseURL+getEnergyPricesEndpoint, &energyPrices)
	if err != nil {
		return nil, fmt.Errorf("get energy prices failed: %v", err)
	}
	return &energyPrices, nil
}

type BlockHeaderRaw struct {
	Timestamp        int64  `json:"timestamp,omitempty"`
	TxTrieRoot       string `json:"txTrieRoot,omitempty"`
	ParentHash       string `json:"parentHash,omitempty"`
	Number           int64  `json:"number,omitempty"`
	WitnessId        int64  `json:"witness_id,omitempty"`
	WitnessAddress   string `json:"witness_address,omitempty"`
	Version          int32  `json:"version,omitempty"`
	AccountStateRoot string `json:"accountStateRoot,omitempty"`
}

type BlockHeader struct {
	RawData          *BlockHeaderRaw `json:"raw_data,omitempty"`
	WitnessSignature string          `json:"witness_signature,omitempty"`
}

type Block struct {
	Transactions []*Transaction `json:"transactions,omitempty"`
	BlockHeader  *BlockHeader   `json:"block_header,omitempty"`
}

func (tc *TronJsonClient) GetNowBlock() (*Block, error) {
	block := Block{}
	getNowBlockEndpoint := "/wallet/getnowblock"

	_, _, err := tc.post(tc.baseURL+getNowBlockEndpoint, nil, &block)
	if err != nil {
		return nil, fmt.Errorf("get latest block failed: %v", err)
	}

	return &block, nil
}

type GetBlockByNumRequest struct {
	Num int32 `json:"num"` // defined as int32 in https://developers.tron.network/reference/wallet-getblockbynum
}

func (tc *TronJsonClient) GetBlockByNum(num int32) (*Block, error) {
	block := Block{}
	getBlockByNumEndpoint := "/wallet/getblockbynum"

	_, _, err := tc.post(tc.baseURL+getBlockByNumEndpoint,
		&GetBlockByNumRequest{
			Num: num,
		}, &block)
	if err != nil {
		return nil, fmt.Errorf("get block by num failed: %v", err)
	}

	return &block, nil
}
