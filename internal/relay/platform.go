package relay

// Platform is the interface every chat platform adapter must implement.
type Platform interface {
	// Run starts the platform's event loop. It blocks until stop is closed.
	Run(stop <-chan struct{})
}
