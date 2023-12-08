package data

type ResultsCache interface {
	Get(key string) Result
	Set(key string, value Result)
}
