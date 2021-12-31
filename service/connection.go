package service

import (
	"sync"

	"xmqtt/utils/log"
)

// State indicates the state of connectivity.
// It can be the state of a ClientConn or SubConn.
type State int

func (s State) String() string {
	switch s {
	case Idle:
		return "IDLE"
	case Connecting:
		return "CONNECTING"
	case Ready:
		return "READY"
	case TransientFailure:
		return "TRANSIENT_FAILURE"
	case Shutdown:
		return "SHUTDOWN"
	default:
		log.Errorf("unknown connectivity state: %d", s)
		return "Invalid-State"
	}

}

const (
	// Idle indicates the ClientConn is idle.
	Idle State = iota
	// Connecting indicates the ClienConn is connecting.
	Connecting
	// Ready indicates the ClientConn is ready for work.
	Ready
	// TransientFailure indicates the ClientConn has seen a failure but expects to recover.
	TransientFailure
	// Shutdown indicates the ClientConn has started shutting down.
	Shutdown
)

// connectivityStateManager keeps the connectivity.State of ClientConn.
// This struct will eventually be exported so the balancers can access it.
type connectivityStateManager struct {
	mu         sync.Mutex
	state      State
	notifyChan chan struct{}
}

// updateState updates the connectivity.State of ClientConn.
// If there's a change it notifies goroutines waiting on state change to
// happen.
func (csm *connectivityStateManager) updateState(state State) {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	if csm.state == Shutdown {
		return
	}
	if csm.state == state {
		return
	}
	csm.state = state
	if csm.notifyChan != nil {
		// There are other goroutines waiting on this channel.
		close(csm.notifyChan)
		csm.notifyChan = nil
	}
}

func (csm *connectivityStateManager) getState() State {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	return csm.state
}

func (csm *connectivityStateManager) getNotifyChan() <-chan struct{} {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	if csm.notifyChan == nil {
		csm.notifyChan = make(chan struct{})
	}
	return csm.notifyChan
}
