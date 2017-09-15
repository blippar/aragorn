package reporter

// Reporter is used to report failures.
type Reporter interface {
	// Report is called each time an expectation fails.
	Report(err error)
	// Reportf is a helper function that report an error with the same syntax as fmt.Errorf.
	Reportf(format string, args ...interface{})
}
