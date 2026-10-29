package quill

import "errors"

// Errors returned by the package, matched with errors.Is. A *GapError is matched with errors.As.
var (
	// ErrCorrupt reports damage below the durable marker: a sealed segment that does not verify,
	// or bytes of the newest segment below quill.tail.off that do not, or a marker whose sequence
	// is above the last valid frame. Open refuses to write; the user guide has the procedure.
	ErrCorrupt = errors.New("quill: corrupt log")
	// ErrFutureVersion reports a quill.format marker newer than this build knows.
	ErrFutureVersion = errors.New("quill: log format is newer than this build")
	// ErrPayloadTooLarge reports a payload above MaxPayloadBytes.
	ErrPayloadTooLarge = errors.New("quill: payload too large")
)
