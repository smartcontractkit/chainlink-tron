package txm

import (
	"fmt"
	"sort"
	"sync"
	"time"

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

type InflightTx struct {
	Hash         string
	ExpirationMs int64
	Tx           *TronTx
}

// Errored or Finalized transactions
type FinishedTx struct {
	Hash        string
	Tx          *TronTx
	RetentionTs time.Time
}

// TxStore tracks broadcast & unconfirmed txs per account address per chain id
type TxStore struct {
	lock sync.RWMutex

	hashToId       map[string]string // map tx hash to idempotency key (tx.ID)
	pendingTxs     map[string]*TronTx
	unconfirmedTxs map[string]*InflightTx
	confirmedTxs   map[string]*InflightTx
	finishedTxs    map[string]*FinishedTx
}

func NewTxStore() *TxStore {
	return &TxStore{
		hashToId:       map[string]string{},
		pendingTxs:     map[string]*TronTx{},
		unconfirmedTxs: map[string]*InflightTx{},
		confirmedTxs:   map[string]*InflightTx{},
		finishedTxs:    map[string]*FinishedTx{},
	}
}

func (s *TxStore) OnPending(tx *TronTx, retry bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if retry {
		pt, txExists := s.unconfirmedTxs[tx.ID]
		_, hashExists := s.hashToId[pt.Hash]

		if !txExists || !hashExists {
			return fmt.Errorf("retry tx doesn't exist: %s", tx.ID)
		}

		delete(s.hashToId, pt.Hash)
		delete(s.unconfirmedTxs, tx.ID)
	}
	if tx.State != Pending {
		return fmt.Errorf("tx is not pending: %s", tx.ID)
	}
	if _, exists := s.pendingTxs[tx.ID]; exists {
		return fmt.Errorf("tx already exists: %s", tx.ID)
	}
	s.pendingTxs[tx.ID] = tx

	return nil
}

func (s *TxStore) OnBroadcasted(hash string, expirationMs int64, tx *TronTx) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, exists := s.hashToId[hash]; exists {
		return fmt.Errorf("hash already exists: %s", tx.ID)
	}

	_, exists := s.pendingTxs[tx.ID]
	if !exists {
		return fmt.Errorf("no such pending id: %s", tx.ID)
	}

	s.hashToId[hash] = tx.ID
	tx.State = Broadcasted

	s.unconfirmedTxs[tx.ID] = &InflightTx{
		Hash:         hash,
		ExpirationMs: expirationMs,
		Tx:           tx,
	}
	delete(s.pendingTxs, tx.ID)

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
		delete(s.unconfirmedTxs, id)
		s.finishedTxs[id] = &FinishedTx{
			Hash:        pt.Hash,
			Tx:          pt.Tx,
			RetentionTs: time.Now(),
		}

		pt.Tx.State = Errored
		return nil
	}

	// check if the tx is confirmed for sanity
	if pt, exists := s.confirmedTxs[id]; exists {
		delete(s.confirmedTxs, id)
		delete(s.hashToId, pt.Hash)

		s.finishedTxs[id] = &FinishedTx{
			Hash:        pt.Hash,
			Tx:          pt.Tx,
			RetentionTs: time.Now(),
		}

		pt.Tx.State = Errored
		return nil
	}

	return fmt.Errorf("no such unconfirmed or confirmed id: %s", id)
}

func (s *TxStore) OnFatalError(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var pt *InflightTx
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
		Hash:        pt.Hash,
		Tx:          pt.Tx,
		RetentionTs: time.Now(),
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
	delete(s.hashToId, pt.Hash)

	// mark it as pending again and re-broadcast
	pt.Tx.State = Pending
	s.pendingTxs[id] = pt.Tx
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
		Hash:        pt.Hash,
		Tx:          pt.Tx,
		RetentionTs: time.Now(),
	}
	return nil
}

func (s *TxStore) GetUnconfirmed() []*InflightTx {
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

func (s *TxStore) GetConfirmed() []*InflightTx {
	s.lock.RLock()
	defer s.lock.RUnlock()

	confirmed := maps.Values(s.confirmedTxs)

	sort.Slice(confirmed, func(i, j int) bool {
		a := confirmed[i]
		b := confirmed[j]
		return a.ExpirationMs < b.ExpirationMs
	})

	return confirmed
}

func (s *TxStore) GetFinished() []*FinishedTx {
	s.lock.RLock()
	defer s.lock.RUnlock()

	finished := maps.Values(s.finishedTxs)

	sort.Slice(finished, func(i, j int) bool {
		a := finished[i]
		b := finished[j]
		return a.RetentionTs.Before(b.RetentionTs)
	})

	return finished
}

func (s *TxStore) DeleteFinishedTxs(ids []string) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	deletedCount := 0
	for _, id := range ids {
		if ft, exists := s.finishedTxs[id]; exists {
			delete(s.finishedTxs, id)
			delete(s.hashToId, ft.Hash)
			deletedCount++
		}
	}
	return deletedCount
}

func (s *TxStore) Has(id string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, inP := s.pendingTxs[id]
	_, inUn := s.unconfirmedTxs[id]
	_, inCf := s.confirmedTxs[id]
	_, inF := s.finishedTxs[id]
	return inP || inUn || inCf || inF
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
	c.lock.RLock()
	store, ok := c.store[fromAddress]
	c.lock.RUnlock()

	if ok {
		return store
	}

	// upgrade to write lock if necessary
	c.lock.Lock()
	defer c.lock.Unlock()

	// double check to prevent race condition
	store, ok = c.store[fromAddress]
	if ok {
		return store
	}

	store = NewTxStore()
	c.store[fromAddress] = store

	return store
}

// GetStatus returns (state, exists)
func (s *TxStore) GetStatus(id string) (TxState, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if pt, ok := s.pendingTxs[id]; ok {
		return pt.State, true
	}
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

func (c *AccountStore) GetAllUnconfirmed() map[string][]*InflightTx {
	// use read lock for methods that read underlying data
	c.lock.RLock()
	defer c.lock.RUnlock()

	allUnconfirmed := map[string][]*InflightTx{}
	for fromAddressStr, store := range c.store {
		allUnconfirmed[fromAddressStr] = store.GetUnconfirmed()
	}
	return allUnconfirmed
}

func (c *AccountStore) GetAllConfirmed() map[string][]*InflightTx {
	// use read lock for methods that read underlying data
	c.lock.RLock()
	defer c.lock.RUnlock()

	allConfirmed := map[string][]*InflightTx{}
	for fromAddressStr, store := range c.store {
		allConfirmed[fromAddressStr] = store.GetConfirmed()
	}
	return allConfirmed
}

func (c *AccountStore) GetAllFinished() map[string][]*FinishedTx {
	// use read lock for methods that read underlying data
	c.lock.RLock()
	defer c.lock.RUnlock()

	allFinished := map[string][]*FinishedTx{}
	for fromAddressStr, store := range c.store {
		allFinished[fromAddressStr] = store.GetFinished()
	}
	return allFinished
}

func (c *AccountStore) DeleteAllFinishedTxs(accountTxIds map[string][]string) int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	totalDeleted := 0
	for acc, txIds := range accountTxIds {
		if store, exists := c.store[acc]; exists {
			deletedCount := store.DeleteFinishedTxs(txIds)
			totalDeleted += deletedCount
		}
	}
	return totalDeleted
}

func (c *AccountStore) GetStatusAll(id string) (TxState, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, store := range c.store {
		status, exists := store.GetStatus(id)
		if exists {
			return status, true
		}
	}
	return 0, false
}
