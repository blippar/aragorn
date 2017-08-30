package reporter

// Reporter is used to report failures.
type Reporter interface {
	// Report is called each time an expectation failed.
	Report(err error)
}
