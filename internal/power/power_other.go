//go:build !darwin

package power

// Other platforms have no wake notification here, so the periodic check stays
// the only trigger.
func watch(func()) (stop func()) {
	return func() {}
}
