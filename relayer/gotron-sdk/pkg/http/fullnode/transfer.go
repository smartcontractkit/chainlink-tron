package fullnode

import (
	"context"
	"errors"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
)

type CreateTransactionRequest struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       int64  `json:"amount"`
	Visible      bool   `json:"visible"`
}

func (tc *Client) Transfer(ctx context.Context, fromAddress address.Address, toAddress address.Address, amount int64) (*common.Transaction, error) {
	tx := common.Transaction{}
	err := tc.Post(ctx, "/createtransaction",
		&CreateTransactionRequest{
			OwnerAddress: fromAddress.String(),
			ToAddress:    toAddress.String(),
			Amount:       amount,
			Visible:      true,
		}, &tx)
	if err != nil {
		return nil, err
	}
	if tx.TxID == "" {
		return nil, errors.New("failed to create transaction")
	}
	return &tx, nil
}
