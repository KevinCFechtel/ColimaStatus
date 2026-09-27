package colima

import (
	"context"
	"errors"
	"fmt"
)

// Kind classifies a failure so that the presentation layer can name it in the
// user's language. The package deliberately does not produce user-facing text
// itself: it would be English inside an otherwise localized menu, and the
// wording of a menu row is not a decision the domain should make.
type Kind int

const (
	// KindUnknown is a failure that does not fit any other kind.
	KindUnknown Kind = iota
	// KindUnavailable means no usable Colima executable was found.
	KindUnavailable
	// KindStatus means the profile status could not be read or interpreted.
	KindStatus
	// KindStart means starting the profile failed.
	KindStart
	// KindStop means stopping the profile failed.
	KindStop
	// KindTimeout means the command was still running when its deadline passed.
	KindTimeout
)

// Error carries the classification for the menu and the technical cause for
// the log. The cause is never dropped: it is what makes a bug report usable.
type Error struct {
	Kind  Kind
	Cause error
}

func (failure *Error) Error() string {
	if failure.Cause == nil {
		return failure.Kind.String()
	}
	return fmt.Sprintf("%s: %v", failure.Kind, failure.Cause)
}

func (failure *Error) Unwrap() error { return failure.Cause }

func (kind Kind) String() string {
	switch kind {
	case KindUnavailable:
		return "Colima is unavailable"
	case KindStatus:
		return "Colima status could not be queried"
	case KindStart:
		return "Colima could not be started"
	case KindStop:
		return "Colima could not be stopped"
	case KindTimeout:
		return "Colima did not respond in time"
	default:
		return "Colima reported an error"
	}
}

// KindOf reports the classification of err, defaulting to KindUnknown for an
// error that did not come from this package.
func KindOf(err error) Kind {
	var failure *Error
	if errors.As(err, &failure) {
		return failure.Kind
	}
	if err != nil {
		return KindUnknown
	}
	return KindUnknown
}

// Unavailable builds the error used when no Colima executable exists, so that
// the wiring in main does not have to construct one by hand.
func Unavailable(cause error) error {
	return &Error{Kind: KindUnavailable, Cause: cause}
}

// classify turns a command failure into a kind. A deadline that passed is
// reported as a timeout rather than as a failed action, because the two call
// for different advice: waiting longer versus looking at Colima itself.
func classify(ctx context.Context, fallback Kind, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &Error{Kind: KindTimeout, Cause: err}
	}
	return &Error{Kind: fallback, Cause: err}
}
