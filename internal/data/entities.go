package data

type Options struct {
	SplitOnDiscontinuation bool
	EventName              string
	ForceRefresh           bool
}

type Result struct {
	BoardSets []string
	Errors    []error
	EventName string
}
