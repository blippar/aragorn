package reporter

// Reporter is used to report failures.
type Reporter interface {
	// Errorf reports an error with the same syntax as fmt.Errorf.
	Errorf(format string, args ...interface{})
}
