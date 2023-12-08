package app

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/fe-dox/go-pbn"
	"github.com/fe-dox/tc-pbn-extractor/internal/data"
	"github.com/fe-dox/tc-pbn-extractor/internal/extractor"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ExtractionService struct {
	ex *extractor.Extractor
	pc data.ResultsCache
}

var (
	ErrJobIsStillBeingProcessed = errors.New("job is still being processed")
	ErrJobAlreadyProcessing     = errors.New("job is already being processed")
	ErrJobNotFound              = errors.New("job not found")
)

func (es ExtractionService) QueueJob(options data.Options) (string, error) {
	jobHash := options.Hash()
	status, err := es.pc.GetStatus(jobHash)
	if err != nil {
		return "", err
	}
	if status == data.JobProcessing {
		return "", ErrJobAlreadyProcessing
	}
	if status == data.JobDone && !options.ForceRefresh {
		return jobHash, nil
	}
	err = es.pc.SetStatusProcessing(jobHash)
	if err != nil {
		return "", err
	}
	go func() {
		result := es.Extract(options)
		err := es.pc.SaveResult(jobHash, *result)
		if err != nil {
			log.Printf("Job %s db save failed: %v", jobHash, err)
		}
	}()
	return jobHash, nil
}

func (es ExtractionService) GetJob(jobHash string) (data.Result, error) {
	status, result, err := es.pc.Get(jobHash)
	if err != nil {
		return data.Result{}, err
	}
	if status == data.JobNotFound {
		return data.Result{}, ErrJobNotFound
	}
	if status == data.JobProcessing {
		return data.Result{}, ErrJobIsStillBeingProcessed
	}
	return result, nil
}

func (es ExtractionService) Extract(options data.Options) *data.Result {
	if _, err := url.Parse(options.BaseUrl); err != nil {
		return data.NewResult().WithError(ErrInvalidBaseUrl)
	}
	settings, err := es.ex.ExtractSettingsFromUrl(options.BaseUrl)
	if err != nil {
		return data.NewResult().WithError(err)
	}

	if options.EventName == "" {
		options.EventName = settings.EventName
	}

	boardRanges, err := getBoardsToExtract(options.BoardsRange, settings.StartBoardNumber, settings.EndBoardNumber)
	if err != nil {
		return data.NewResult().WithError(err)
	}

	type extractionResult struct {
		Board []pbn.Board
		Err   error
	}
	ch := make(chan extractionResult, 1)

	go func() {
		for _, boardRange := range boardRanges {
			for i := boardRange[0]; i <= boardRange[1]; i++ {
				board, err := es.ex.ExtractOneFromUrl(options.BaseUrl, i)
				if err != nil {
					board = []pbn.Board{{Number: i}}
				}
				ch <- extractionResult{
					Err:   err,
					Board: board,
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		close(ch)
	}()

	result := data.NewResult()
	result.EventName = options.EventName

	var prevBoardNumber int

	var b = bytes.NewBufferString("")

	for boardResults := range ch {
		if boardResults.Err != nil {
			result.AddError(fmt.Errorf("Failed to extract Board %d: %v\n", boardResults.Board[0].Number, boardResults.Err))
			if !options.FillMissing {
				continue
			}
		}
		for _, board := range boardResults.Board {
			if options.SplitOnDiscontinuation {
				if prevBoardNumber > board.Number {
					result.AddBoardSet(b.String())
					b = bytes.NewBufferString("")
				}
				prevBoardNumber = board.Number
			}
			board.EventName = options.EventName
			board.Generator = data.GENERATOR
			err = board.Serialize(b, true)
			if err != nil {
				result.AddError(fmt.Errorf("Failed to serialize Board %d (number as played): %v\n", board.Number, err))
				continue
			}
		}
	}
	result.Success = true
	result.AddBoardSet(b.String())
	return result
}

var (
	ErrInvalidBaseUrl                       = errors.New("invalid base URL")
	ErrInvalidBoardsRange                   = errors.New("invalid boards range")
	ErrSelectedBoardsDoNotExistInTournament = errors.New("selected boards do not exist in tournament")
	ErrBoardsNotInOrder                     = errors.New("selected boards are not in order")
)

func getBoardsToExtract(str string, start int, end int) ([][2]int, error) {
	if str == "" {
		return [][2]int{{start, end}}, nil
	}
	boards := make([][2]int, 0, 1)
	ranges := strings.Split(str, ",")
	var currentHighest int
	if len(ranges) == 0 {
		return nil, ErrInvalidBoardsRange
	}
	for _, r := range ranges {
		if strings.Contains(r, "-") {
			parts := strings.Split(r, "-")
			if len(parts) != 2 {
				return nil, ErrInvalidBoardsRange
			}
			from, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, ErrInvalidBoardsRange
			}
			if from <= currentHighest {
				return nil, ErrBoardsNotInOrder
			}
			currentHighest = from
			if from < start {
				return nil, ErrSelectedBoardsDoNotExistInTournament
			}
			to, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, ErrInvalidBoardsRange
			}
			if to <= currentHighest {
				return nil, ErrBoardsNotInOrder
			}
			currentHighest = to
			if to > end {
				return nil, ErrSelectedBoardsDoNotExistInTournament
			}
			if from > to {
				return nil, ErrBoardsNotInOrder
			}
			boards = append(boards, [2]int{from, to})
		} else {
			board, err := strconv.Atoi(r)
			if err != nil {
				return nil, ErrInvalidBoardsRange
			}
			if board <= currentHighest {
				return nil, ErrBoardsNotInOrder
			}
			if board < start || board > end {
				return nil, ErrSelectedBoardsDoNotExistInTournament
			}
			currentHighest = board
			boards = append(boards, [2]int{board, board})
		}
	}
	return boards, nil
}
