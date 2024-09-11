package api

type EnergyPrice struct {
	Prices string `json:"prices"` // All historical energy unit price information. Each unit price change is separated by a comma. Before the colon is the millisecond timestamp, and after the colon is the energy unit price in sun.
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

type Return struct {
	ContractRet string `json:"contractRet"`
}

type BlockTransactions struct {
	Ret        []Return `json:"ret"`
	TxID       string   `json:"txID"`
	Signature  []string `json:"signature"`
	RawData    RawData  `json:"raw_data"`
	RawDataHex string   `json:"raw_data_hex"`
}

type Block struct {
	BlockID      string              `json:"blockID"`
	Transactions []BlockTransactions `json:"transactions,omitempty"`
	BlockHeader  BlockHeader         `json:"block_header,omitempty"`
}

type GetBlockByNumRequest struct {
	Num int32 `json:"num"` // defined as int32 in https://developers.tron.network/reference/wallet-getblockbynum
}
