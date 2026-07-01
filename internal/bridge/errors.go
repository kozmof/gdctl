package bridge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
)

// ErrUnreachable is a sentinel that matches any failure to contact the bridge at
// the network layer (connection refused, dial timeout, DNS resolution failure).
// Callers can test for it with errors.Is to render a friendly "is Godot
// running?" message instead of a raw socket error.
var ErrUnreachable = errors.New("bridge unreachable")

// UnreachableError wraps the underlying transport cause while satisfying
// errors.Is(err, ErrUnreachable). It carries the address that was dialed so the
// message is actionable.
type UnreachableError struct {
	Addr string
	Err  error
}

func (e *UnreachableError) Error() string {
	return fmt.Sprintf("bridge unreachable at %s: %v", e.Addr, e.Err)
}

func (e *UnreachableError) Unwrap() error { return e.Err }

func (e *UnreachableError) Is(target error) bool { return target == ErrUnreachable }

// HTTPError reports a non-2xx response from the bridge that did not carry a
// structured BridgeError body. Callers can inspect StatusCode to distinguish,
// for example, an auth failure (401/403) from a server fault (5xx).
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("bridge returned HTTP %d: %s", e.StatusCode, e.Body)
}

// classifyTransportError converts a raw net/http transport failure into an
// *UnreachableError when it stems from an inability to reach the bridge. It
// leaves context cancellation/deadline errors untouched so a user abort is not
// mislabeled as an unreachable bridge, and passes through anything it does not
// recognize.
func classifyTransportError(addr string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) {
		return &UnreachableError{Addr: addr, Err: err}
	}
	return err
}
