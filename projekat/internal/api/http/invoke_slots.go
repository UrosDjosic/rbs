package httpapi

// NewInvokeSlots creates a buffered channel limiting concurrent in-flight invokes.
// capacity <= 0 returns nil (unlimited).
func NewInvokeSlots(capacity int) chan struct{} {
	if capacity <= 0 {
		return nil
	}
	return make(chan struct{}, capacity)
}

func (s *Server) acquireInvokeSlot() (release func(), ok bool) {
	if s.InvokeSlots == nil {
		return func() {}, true
	}
	select {
	case s.InvokeSlots <- struct{}{}:
		return func() { <-s.InvokeSlots }, true
	default:
		return nil, false
	}
}
