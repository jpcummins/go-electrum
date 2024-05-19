package electrum

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// GetBalanceResp represents the response to GetBalance().
type GetBalanceResp struct {
	Result GetBalanceResult `json:"result"`
}

// GetBalanceResult represents the content of the result field in the response to GetBalance().
type GetBalanceResult struct {
	Confirmed   float64 `json:"confirmed"`
	Unconfirmed float64 `json:"unconfirmed"`
}

// GetBalance returns the confirmed and unconfirmed balance for a scripthash.
// https://electrumx.readthedocs.io/en/latest/protocol-methods.html#blockchain-scripthash-get-balance
func (s *Client) GetBalance(
	ctx context.Context,
	scripthash string,
) (GetBalanceResult, error) {
	var resp GetBalanceResp

	err := s.request(
		ctx,
		"blockchain.scripthash.get_balance",
		[]interface{}{scripthash},
		&resp,
	)
	if err != nil {
		return GetBalanceResult{}, err
	}

	return resp.Result, err
}

// GetMempoolResp represents the response to GetHistory() and GetMempool().
type GetMempoolResp struct {
	Result []*GetMempoolResult `json:"result"`
}

// GetMempoolResult represents the content of the result field in the response
// to GetHistory() and GetMempool().
type GetMempoolResult struct {
	Hash   string `json:"tx_hash"`
	Height int64  `json:"height"`
	Fee    uint32 `json:"fee,omitempty"`
}

type DetailedMempoolResult struct {
	*DetailedTransaction
	Height   int64  `json:"height"`
	Fee      uint32 `json:"fee,omitempty"`
	Incoming bool   `json:"incoming,omitempty"`
}

// GetHistory returns the confirmed and unconfirmed history for a scripthash.
func (s *Client) GetHistory(
	ctx context.Context,
	scripthash string,
) ([]*GetMempoolResult, error) {
	var resp GetMempoolResp

	err := s.request(
		ctx,
		"blockchain.scripthash.get_history",
		[]interface{}{scripthash},
		&resp,
	)
	if err != nil {
		return nil, err
	}

	return resp.Result, err
}

// Details history of a scripthash by adding Prevout and Incoming fields.
func (s *Client) DetailHistory(
	ctx context.Context,
	address string,
	history []*GetMempoolResult,
) ([]*DetailedMempoolResult, error) {
	var result []*DetailedMempoolResult

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(10)
	mtx := sync.Mutex{}
	for _, h := range history {
		func(h *GetMempoolResult) {
			eg.Go(func() error {
				tx, err := s.GetTransaction(ctx, h.Hash)
				if err != nil {
					return err
				}
				s.logger.Debugf("detailing tx: %s", tx.TxID)
				detailedTx, err := s.DetailTransaction(ctx, tx)
				if err != nil {
					return err
				}
				incoming := false
				findAddressFunc[Vout](
					address,
					detailedTx.Vout,
					func(elem Vout, index int) bool {
						incoming = true
						return false // break
					},
				)
				s.logger.Debugf(
					"DetailedTx: %s incoming: %v",
					detailedTx.TxID,
					incoming,
				)

				mtx.Lock()
				defer mtx.Unlock()
				result = append(result, &DetailedMempoolResult{
					DetailedTransaction: detailedTx,
					Height:              h.Height,
					Fee:                 h.Fee,
					Incoming:            incoming,
				})
				return nil
			})
		}(h)
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetMempool returns the unconfirmed transacations of a scripthash.
func (s *Client) GetMempool(
	ctx context.Context,
	scripthash string,
) ([]*GetMempoolResult, error) {
	var resp GetMempoolResp

	err := s.request(
		ctx,
		"blockchain.scripthash.get_mempool",
		[]interface{}{scripthash},
		&resp,
	)
	if err != nil {
		return nil, err
	}

	return resp.Result, err
}

// ListUnspentResp represents the response to ListUnspent()
type ListUnspentResp struct {
	Result []*ListUnspentResult `json:"result"`
}

// ListUnspentResult represents the content of the result field in the response to ListUnspent()
type ListUnspentResult struct {
	Height   uint64 `json:"height"`
	Position uint32 `json:"tx_pos"`
	Hash     string `json:"tx_hash"`
	Value    uint64 `json:"value"`
}

// ListUnspent returns an ordered list of UTXOs for a scripthash.
func (s *Client) ListUnspent(
	ctx context.Context,
	scripthash string,
) ([]*ListUnspentResult, error) {
	var resp ListUnspentResp

	err := s.request(
		ctx,
		"blockchain.scripthash.listunspent",
		[]interface{}{scripthash},
		&resp,
	)
	if err != nil {
		return nil, err
	}

	return resp.Result, err
}
