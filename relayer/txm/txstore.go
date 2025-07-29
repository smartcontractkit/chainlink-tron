package txm

import (
	"fmt"
	"sort"
	"sync"

	"golang.org/x/exp/maps"
)

type TxState int

const (
	Pending TxState = iota
	Errored
	FatallyErrored
	Broadcasted
	Confirmed
	Finalized
)

type PendingTx struct {
	Hash         string
	ExpirationMs int64
	Tx           *TronTx
}

// Errored or Finalized transactions
type FinishedTx struct {
	Hash string
	Tx   *TronTx
}

// TxStore tracks broadcast & unconfirmed txs per account address per chain id
type TxStore struct {
	lock sync.RWMutex

	hashToId       map[string]string // map tx hash to idempotency key (tx.ID)
	unconfirmedTxs map[string]*PendingTx
	confirmedTxs   map[string]*PendingTx
	finishedTxs    map[string]*FinishedTx
}

func NewTxStore() *TxStore {
	return &TxStore{
		hashToId:       map[string]string{},
		unconfirmedTxs: map[string]*PendingTx{},
		confirmedTxs:   map[string]*PendingTx{},
		finishedTxs:    map[string]*FinishedTx{},
	}
}

func (s *TxStore) OnPending(hash string, expirationMs int64, tx *TronTx) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, exists := s.hashToId[hash]; exists {
		return fmt.Errorf("hash already exists: %s", tx.ID)
	}

	s.hashToId[hash] = tx.ID
	tx.State = Pending

	s.unconfirmedTxs[tx.ID] = &PendingTx{
		Hash:         hash,
		ExpirationMs: expirationMs,
		Tx:           tx,
	}

	return nil
}

func (s *TxStore) OnBroadcasted(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	tx, exists := s.unconfirmedTxs[id]
	if !exists {
		return fmt.Errorf("no such unconfirmed id: %s", id)
	}
	if tx.Tx.State != Pending {
		return fmt.Errorf("tx is not pending: %s", id)
	}
	tx.Tx.State = Broadcasted

	return nil
}

func (s *TxStore) OnConfirmed(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	tx, exists := s.unconfirmedTxs[id]
	if !exists {
		return fmt.Errorf("no such unconfirmed id: %s", id)
	}
	if tx.Tx.State != Broadcasted {
		return fmt.Errorf("tx is not broadcasted, state: %d | id: %s", tx.Tx.State, id)
	}
	delete(s.unconfirmedTxs, id)

	tx.Tx.State = Confirmed
	s.confirmedTxs[id] = tx

	return nil
}

func (s *TxStore) OnErrored(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if pt, exists := s.unconfirmedTxs[id]; exists {
		delete(s.hashToId, pt.Hash)
		pt.Tx.State = Errored
		return nil
	}

	// check if the tx is confirmed for sanity
	if pt, exists := s.confirmedTxs[id]; exists {
		delete(s.confirmedTxs, id)
		delete(s.hashToId, pt.Hash)
		pt.Tx.State = Errored
		s.unconfirmedTxs[id] = pt
		return nil
	}

	return fmt.Errorf("no such unconfirmed or confirmed id: %s", id)
}

func (s *TxStore) OnFatalError(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var pt *PendingTx
	var exists bool
	if pt, exists = s.unconfirmedTxs[id]; exists {
		delete(s.unconfirmedTxs, id)
	} else if pt, exists = s.confirmedTxs[id]; exists {
		delete(s.confirmedTxs, id)
	} else {
		return fmt.Errorf("no such unconfirmed or confirmed id: %s", id)
	}

	pt.Tx.State = FatallyErrored
	s.finishedTxs[id] = &FinishedTx{
		Hash: pt.Hash,
		Tx:   pt.Tx,
	}
	return nil
}

// OnReorg moves a previously-confirmed tx back to unconfirmed if it's been
// dropped by a chain reorg.
func (s *TxStore) OnReorg(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	pt, exists := s.confirmedTxs[id]
	if !exists {
		return fmt.Errorf("no such confirmed id: %s", id)
	}
	// remove from confirmed
	delete(s.confirmedTxs, id)

	// mark it as broadcasted again and re-queue
	// TODO: Should we be rebroadcasting here?
	pt.Tx.State = Broadcasted
	s.unconfirmedTxs[id] = pt
	return nil
}

func (s *TxStore) OnFinalized(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	pt, exists := s.confirmedTxs[id]
	if !exists {
		return fmt.Errorf("no such confirmed id: %s", id)
	}
	delete(s.confirmedTxs, id)

	pt.Tx.State = Finalized
	s.finishedTxs[id] = &FinishedTx{
		Hash: pt.Hash,
		Tx:   pt.Tx,
	}
	return nil
}

func (s *TxStore) GetUnconfirmed() []*PendingTx {
	s.lock.RLock()
	defer s.lock.RUnlock()

	unconfirmed := maps.Values(s.unconfirmedTxs)

	sort.Slice(unconfirmed, func(i, j int) bool {
		a := unconfirmed[i]
		b := unconfirmed[j]
		return a.ExpirationMs < b.ExpirationMs
	})

	return unconfirmed
}

func (s *TxStore) Has(id string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, inUn := s.unconfirmedTxs[id]
	_, inCf := s.confirmedTxs[id]
	_, inF := s.finishedTxs[id]
	return inUn || inCf || inF
}

func (s *TxStore) InflightCount() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.unconfirmedTxs)
}

func (s *TxStore) FinishedCount() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.finishedTxs)
}

type AccountStore struct {
	store map[string]*TxStore // map account address to txstore
	lock  sync.RWMutex
}

func NewAccountStore() *AccountStore {
	return &AccountStore{
		store: map[string]*TxStore{},
	}
}

func (c *AccountStore) GetTxStore(fromAddress string) *TxStore {
	c.lock.Lock()
	defer c.lock.Unlock()
	store, ok := c.store[fromAddress]
	if !ok {
		store = NewTxStore()
		c.store[fromAddress] = store
	}
	return store
}

// GetStatus returns (state, exists)
func (s *TxStore) GetStatus(id string) (TxState, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if pt, ok := s.unconfirmedTxs[id]; ok {
		return pt.Tx.State, true
	}
	if pt, ok := s.confirmedTxs[id]; ok {
		return pt.Tx.State, true
	}
	if ft, ok := s.finishedTxs[id]; ok {
		return ft.Tx.State, true
	}
	return 0, false
}

func (c *AccountStore) GetTotalInflightCount() int {
	// use read lock for methods that read underlying data
	c.lock.RLock()
	defer c.lock.RUnlock()

	count := 0
	for _, store := range c.store {
		count += store.InflightCount()
	}

	return count
}

func (c *AccountStore) GetHashToIdMap() map[string]string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	hashToId := map[string]string{}
	for _, store := range c.store {
		for hash, id := range store.hashToId {
			hashToId[hash] = id
		}
	}
	return hashToId
}

func (c *AccountStore) GetTotalFinishedCount() int {
	// use read lock for methods that read underlying data
	c.lock.RLock()
	defer c.lock.RUnlock()

	count := 0
	for _, store := range c.store {
		count += store.FinishedCount()
	}

	return count
}

func (c *AccountStore) GetAllUnconfirmed() map[string][]*PendingTx {
	// use read lock for methods that read underlying data
	c.lock.RLock()
	defer c.lock.RUnlock()

	allUnconfirmed := map[string][]*PendingTx{}
	for fromAddressStr, store := range c.store {
		allUnconfirmed[fromAddressStr] = store.GetUnconfirmed()
	}
	return allUnconfirmed
}
