package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/fe-dox/go-pbn"
	"github.com/fe-dox/tc-pbn-extractor/internal/extractor"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	var writeToStdOut bool
	flag.BoolVar(&writeToStdOut, "stdout", false, "Write PBN to stdout instead of file")
	var output string
	flag.StringVar(&output, "out", "", "File to write PBN to, if empty will write to <event-name>.pbn")
	var eventName string
	flag.StringVar(&eventName, "event", "", "Event name to use in PBN, if empty will be extracted from tournament settings")
	var generatorName string
	flag.StringVar(&generatorName, "generator", "tc-pbn-extractor", "Generator name to use in PBN")
	var boardsToExtract string
	flag.StringVar(&boardsToExtract, "boards", "", "Boards to extract, if empty will extract all boards. Valid notation <from>-<to>,<single>,<from>-<to>")
	var splitOnDiscontinuation bool
	flag.BoolVar(&splitOnDiscontinuation, "split", false, "Split boards to different files on numeration discontinuation (untested)")
	var userAgent string
	flag.StringVar(&userAgent, "agent", "tc-pbn-extractor", "User-Agent header to use for requests")
	var timeout time.Duration
	flag.DurationVar(&timeout, "timeout", 1*time.Second, "Timeout for HTTP requests")
	var baseUrl string
	flag.StringVar(&baseUrl, "url", "", "URL to extract PBN from")
	var fillMissing bool
	flag.BoolVar(&fillMissing, "fill-missing", false, "Fill missing boards with empty boards")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of tc-pbn-extractor:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\ttcpbn.exe <url>\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\ttcpbn.exe [options]\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if baseUrl == "" {
		baseUrl = flag.Arg(0)
		if baseUrl == "" {
			flag.Usage()
			return
		}
	}
	_, err := url.ParseRequestURI(baseUrl)
	if err != nil {
		log.Fatal("URL is invalid")
		return
	}

	ext := extractor.NewExtractor(userAgent, timeout)
	settings, err := ext.ExtractSettingsFromUrl(baseUrl)
	if err != nil {
		log.Fatalf("Failed to extract settings: %v\n", err)
		return
	}

	if output == "" {
		output = fmt.Sprintf("%s.pbn", settings.EventName)
	}

	if eventName == "" {
		eventName = settings.EventName
	}

	boardRanges, err := getBoardsToExtract(boardsToExtract, settings.StartBoardNumber, settings.EndBoardNumber)
	if err != nil {
		log.Fatal(err)
	}

	type extractionResult struct {
		Board []pbn.Board
		Err   error
	}
	ch := make(chan extractionResult, 1)

	var successes int
	var failures int
	go func() {
		for _, boardRange := range boardRanges {
			for i := boardRange[0]; i <= boardRange[1]; i++ {
				board, err := ext.ExtractOneFromUrl(baseUrl, i)
				if err != nil {
					board = []pbn.Board{{Number: i}}
				}
				ch <- extractionResult{
					Err:   err,
					Board: board,
				}
			}
		}
		close(ch)
	}()
	var prevBoardNumber int
	var currentSplit int
	var w *os.File
	if writeToStdOut {
		w = os.Stdout
	} else {
		w, err = os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err != nil {
		log.Fatalf("Failed to open file: %v\n", err)
		return
	}
	if splitOnDiscontinuation && !strings.Contains(output, "%d") {
		output = strings.TrimSuffix(output, ".pbn")
		output = output + "-%d.pbn"
	}
	for boardResults := range ch {
		if boardResults.Err != nil {
			log.Printf("Failed to extract Board %d: %v\n", boardResults.Board[0].Number, boardResults.Err)
			failures++
			if !fillMissing {
				continue
			}
		}
		for _, board := range boardResults.Board {
			if splitOnDiscontinuation && !writeToStdOut {
				if prevBoardNumber >= board.Number {
					err := w.Close()
					if err != nil {
						log.Fatalf("Failed to close file: %v\n", err)
						return
					}
					w, err = os.OpenFile(fmt.Sprintf(output, currentSplit), os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Fatalf("Failed to open file: %v\n", err)
						return
					}
					currentSplit++
				}
				prevBoardNumber = board.Number
			}
			board.EventName = eventName
			board.Generator = generatorName
			err = board.Serialize(w, true)
			if err != nil {
				log.Printf("Failed to serialize Board %d (number as played): %v\n", board.Number, err)
				failures += 1
				continue
			}
			successes += 1
		}
	}
	w.Close()
	log.Printf("Extracted %d boards succesfully. Failed %d times. ¯\\_(ツ)_/¯\n", successes, failures)

}

var ErrInvalidBoardsRange = errors.New("invalid boards range")
var ErrSelectedBoardsDoNotExistInTournament = errors.New("selected boards do not exist in tournament")
var ErrBoardsNotInOrder = errors.New("selected boards are not in order")

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
