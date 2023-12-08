package extractor

import "errors"

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	ErrSettingsFileNotFound = errors.New("settings.json does not exist")
	ErrNoDistributionData   = errors.New("no distribution data")
)
