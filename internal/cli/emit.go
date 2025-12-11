package cli

import "github.com/vburojevic/xcw/internal/output"

// EmitNDJSON writes via a shared emitter, falling back to nil.
type EmitNDJSON struct {
	emitter *output.Emitter
}

func NewEmitNDJSON(w interface{}) *EmitNDJSON {
	switch v := w.(type) {
	case *output.Emitter:
		return &EmitNDJSON{emitter: v}
	}
	return &EmitNDJSON{}
}

func (e *EmitNDJSON) E() *output.Emitter { return e.emitter }
