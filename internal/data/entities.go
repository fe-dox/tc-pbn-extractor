package data

import (
	"crypto/md5"
	"fmt"
)

const GENERATOR = "pbnextractor.fedox.pl"

type Options struct {
	BaseUrl                string
	EventName              string
	BoardsRange            string
	SplitOnDiscontinuation bool
	ForceRefresh           bool
	FillMissing            bool
}

func (o *Options) Hash() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s %s %v %s %v", o.BaseUrl, o.EventName, o.SplitOnDiscontinuation, o.BoardsRange, o.FillMissing))))
}

type Result struct {
	Success   bool
	BoardSets []string
	Errors    []error
	EventName string
}

func NewResult() *Result {
	return &Result{
		BoardSets: make([]string, 0),
		Errors:    make([]error, 0),
		Success:   false,
		EventName: "",
	}
}

func (r *Result) WithError(err error) *Result {
	r.Errors = []error{err}
	return r
}

func (r *Result) AddBoardSet(boardSet string) {
	r.BoardSets = append(r.BoardSets, boardSet)
}

func (r *Result) AddError(err error) {
	r.Errors = append(r.Errors, err)
}
