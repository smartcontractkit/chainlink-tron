package txm

import (
	"fmt"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
)

type TxState int

const (
	NotFound TxState = iota
	AwaitingBroadcast
	Broadcasted
	Confirmed
	Finalized
	Errored
	FatallyErrored
)

func (s TxState) String() string {
	switch s {
	case NotFound:
		return "NotFound"
	case Errored:
		return "Errored"
	case AwaitingBroadcast:
		return "AwaitingBroadcast"
	case Broadcasted:
		return "Broadcasted"
	case Confirmed:
		return "Confirmed"
	case FatallyErrored:
		return "FatallyErrored"
	default:
		return fmt.Sprintf("TxState(%d)", s)
	}
}

var stateTransitions = map[TxState][]TxState{
	NotFound:          {AwaitingBroadcast, Errored, FatallyErrored},
	AwaitingBroadcast: {Broadcasted, Errored},
	Broadcasted:       {Confirmed, Errored},
	Errored:           {FatallyErrored},
}

func (s TxState) CanTransitionTo(t TxState) bool {
	allowedTransitions, exists := stateTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if t == allowed {
			return true
		}
	}

	return false
}

type pendingTx struct {
	tx    *TronTx
	id    string
	state TxState

	// Set only when the tx is broadcasted
	coreTx      *common.Transaction
	retentionTs time.Time
}

type InMemoryTxStore struct {
	queuedTxs             map[string]pendingTx
	broadcastedTxs        map[string]pendingTx
	confirmedOrErroredTxs map[string]pendingTx

	lock sync.RWMutex
}

func NewInMemoryTxStore() *InMemoryTxStore {
	return &InMemoryTxStore{
		queuedTxs:             make(map[string]pendingTx), // pending transactions awaiting broadcast
		broadcastedTxs:        make(map[string]pendingTx), // transactions that have been broadcasted awaiting confirmation
		confirmedOrErroredTxs: make(map[string]pendingTx), // transactions that have been confirmed or errored
	}
}

// Should be called for any read-only operations on the tx store
func (c *InMemoryTxStore) withReadLock(fn func() error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return fn()
}

// Should be called for any write operations on the tx store
func (c *InMemoryTxStore) withWriteLock(fn func() (string, error)) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return fn()
}

// Deletes a tx by id, returns an error if the tx is not found
func (s *InMemoryTxStore) _unsafeDeleteTx(id string) error {
	_, state, ok := s._unsafeGetTx(id)
	if !ok {
		return fmt.Errorf("tx not found: %s", id)
	}

	switch state {
	case NotFound:
		return fmt.Errorf("tx not found: %s", id)
	case AwaitingBroadcast:
		delete(s.queuedTxs, id)
	case Broadcasted:
		delete(s.broadcastedTxs, id)
	case Errored:
		delete(s.confirmedOrErroredTxs, id)
	case FatallyErrored:
		delete(s.confirmedOrErroredTxs, id)
	}

	return nil
}

// Gets a tx by id, returns the tx, the state, and a bool indicating if the tx was found
func (s *InMemoryTxStore) _unsafeGetTx(id string) (pendingTx, TxState, bool) {
	tx, ok := s.queuedTxs[id]
	if ok {
		return tx, tx.state, true
	}
	tx, ok = s.broadcastedTxs[id]
	if ok {
		return tx, tx.state, true
	}
	tx, ok = s.confirmedOrErroredTxs[id]
	if ok {
		return tx, tx.state, true
	}

	return pendingTx{}, NotFound, false
}

// Gets a tx by id, returns the tx, the state, and a bool indicating if the tx was found
func (s *InMemoryTxStore) GetTx(id string) (pendingTx, TxState, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s._unsafeGetTx(id)
}

// OnBroadcasted handles the logic for when a tx is broadcasted
func (s *InMemoryTxStore) OnBroadcasted(tx pendingTx, coreTx *common.Transaction) error {
	tx, txState, ok := s.GetTx(tx.id)
	if !ok {
		return fmt.Errorf("tx not found: %s", tx.id)
	}

	_, err := s.withWriteLock(func() (string, error) {
		if !txState.CanTransitionTo(Broadcasted) {
			return "", fmt.Errorf("Invalid State Transition: %s -> %s (tx: %s)", txState, Broadcasted, tx.id)
		}

		tx.state = Broadcasted
		tx.coreTx = coreTx
		s._unsafeDeleteTx(tx.id)
		s.broadcastedTxs[tx.id] = tx

		return "", nil
	})

	return err
}

// OnConfirmed handles the logic for when a tx is confirmed
func (s *InMemoryTxStore) OnConfirmed(tx pendingTx) error {
	tx, txState, ok := s.GetTx(tx.id)
	if !ok {
		return fmt.Errorf("tx not found: %s", tx.id)
	}

	_, err := s.withWriteLock(func() (string, error) {
		if !txState.CanTransitionTo(Confirmed) {
			return "", fmt.Errorf("Invalid State Transition: %s -> %s (tx: %s)", txState, Confirmed, tx.id)
		}

		tx.state = Confirmed
		s._unsafeDeleteTx(tx.id)
		s.confirmedOrErroredTxs[tx.id] = tx

		return "", nil
	})

	return err
}

// OnErrored handles the logic for when a tx is errored
func (s *InMemoryTxStore) OnErrored(tx pendingTx) error {
	tx, txState, ok := s.GetTx(tx.id)
	if !ok {
		return fmt.Errorf("tx not found: %s", tx.id)
	}

	_, err := s.withWriteLock(func() (string, error) {
		if !txState.CanTransitionTo(Errored) {
			return "", fmt.Errorf("Invalid State Transition: %s -> %s (tx: %s)", txState, Errored, tx.id)
		}

		tx.state = Errored
		s._unsafeDeleteTx(tx.id)
		tx.retentionTs = time.Now()
		s.confirmedOrErroredTxs[tx.id] = tx

		return "", nil
	})

	return err
}
