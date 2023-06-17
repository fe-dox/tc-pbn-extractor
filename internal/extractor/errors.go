package extractor

import "errors"

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	ErrNoDistributionData   = errors.New("no distribution data")
)
