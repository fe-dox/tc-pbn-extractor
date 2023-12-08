package data

type ExtractionService interface {
	Extract(url string, options Options) (Result, error)
}
