package extractor

import (
	"encoding/json"
	"fmt"
	"github.com/fe-dox/go-pbn"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Extractor struct {
	UserAgent string
	client    http.Client
}

func NewExtractor(userAgent string, timeout time.Duration) *Extractor {
	return &Extractor{
		UserAgent: userAgent,
		client:    http.Client{Timeout: timeout},
	}
}

type RawTournamentSettings struct {
	BoardsNumbers []int  `json:"BoardsNumbers"`
	FullName      string `json:"FullName"`
}

type TournamentSettings struct {
	StartBoardNumber int
	EndBoardNumber   int
	EventName        string
}

func (e *Extractor) ExtractSettingsFromUrl(baseUrl string) (TournamentSettings, error) {
	settingsUrl, err := url.JoinPath(baseUrl, "settings.json")
	if err != nil {
		return TournamentSettings{}, err
	}

	request, err := http.NewRequest("GET", settingsUrl, nil)
	if err != nil {
		return TournamentSettings{}, err
	}

	request.Header.Add("User-Agent", e.UserAgent)
	response, err := e.client.Do(request)
	if err != nil {
		return TournamentSettings{}, err
	}
	if response.StatusCode != http.StatusOK {
		return TournamentSettings{}, ErrUnexpectedStatusCode
	}

	defer response.Body.Close()

	var data RawTournamentSettings
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return TournamentSettings{}, err
	}

	ts := TournamentSettings{
		StartBoardNumber: data.BoardsNumbers[0],
		EndBoardNumber:   data.BoardsNumbers[len(data.BoardsNumbers)-1],
		EventName:        data.FullName,
	}
	return ts, nil
}

func (e *Extractor) ExtractFromUrl(url string, start int, end int) ([]pbn.Board, map[int]error) {
	boards := make([]pbn.Board, 0, end-start+1)
	var errors map[int]error
	for i := start; i <= end; i++ {
		tmpBoards, err := e.ExtractOneFromUrl(url, i)
		if err != nil {
			errors[i] = err
			continue
		}
		boards = append(boards, tmpBoards...)
	}
	return boards, errors
}

type RawBoardData struct {
	Dealer        int    `json:"Dealer"`
	HandE         Hand   `json:"HandE"`
	HandN         Hand   `json:"HandN"`
	HandS         Hand   `json:"HandS"`
	HandW         Hand   `json:"HandW"`
	MiniMax       string `json:"MiniMax"`
	TricksFromE   Tricks `json:"TricksFromE"`
	TricksFromN   Tricks `json:"TricksFromN"`
	TricksFromS   Tricks `json:"TricksFromS"`
	TricksFromW   Tricks `json:"TricksFromW"`
	Vulnerability int    `json:"Vulnerability"`
	Declarer      int    `json:"_declarer"`
}

type Hand struct {
	Clubs    string `json:"Clubs"`
	Diamonds string `json:"Diamonds"`
	Hearts   string `json:"Hearts"`
	Spades   string `json:"Spades"`
}

type Tricks struct {
	Clubs    int `json:"Clubs"`
	Diamonds int `json:"Diamonds"`
	Hearts   int `json:"Hearts"`
	Nt       int `json:"Nt"`
	Spades   int `json:"Spades"`
}

type RawProtocol struct {
	ScoringGroups []struct {
		Distribution struct {
			Number         int          `json:"Number"`
			NumberAsPlayed int          `json:"_numberAsPlayed"`
			BoardData      RawBoardData `json:"_handRecord"`
		} `json:"Distribution"`
	} `json:"ScoringGroups"`
}

func (e *Extractor) ExtractOneFromUrl(baseUrl string, boardNumber int) ([]pbn.Board, error) {
	settingsUrl, err := url.JoinPath(baseUrl, fmt.Sprintf("p%d.json", boardNumber))
	var data RawProtocol
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", settingsUrl, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("User-Agent", e.UserAgent)
	response, err := e.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, ErrUnexpectedStatusCode
	}
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	boards := make([]pbn.Board, 0, len(data.ScoringGroups))
	if len(data.ScoringGroups) < 1 {
		return nil, ErrNoDistributionData
	}
	for _, group := range data.ScoringGroups {
		if group.Distribution.BoardData.HandN.Spades == "" &&
			group.Distribution.BoardData.HandN.Hearts == "" &&
			group.Distribution.BoardData.HandN.Diamonds == "" &&
			group.Distribution.BoardData.HandN.Clubs == "" {
			continue
		}
		tmpBoard := pbn.Board{
			Number:     group.Distribution.NumberAsPlayed,
			Dealer:     pbn.DealerFromBoardNumber(group.Distribution.NumberAsPlayed),
			Vulnerable: pbn.Vulnerability(group.Distribution.BoardData.Vulnerability),
			EventName:  "",
			Generator:  "",
			Hands: map[pbn.Direction]pbn.Hand{
				pbn.North: {
					pbn.Clubs:    parseCardsString(strings.Replace(group.Distribution.BoardData.HandN.Clubs, "10", "T", 1)),
					pbn.Diamonds: parseCardsString(strings.Replace(group.Distribution.BoardData.HandN.Diamonds, "10", "T", 1)),
					pbn.Hearts:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandN.Hearts, "10", "T", 1)),
					pbn.Spades:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandN.Spades, "10", "T", 1)),
				},
				pbn.East: {
					pbn.Clubs:    parseCardsString(strings.Replace(group.Distribution.BoardData.HandE.Clubs, "10", "T", 1)),
					pbn.Diamonds: parseCardsString(strings.Replace(group.Distribution.BoardData.HandE.Diamonds, "10", "T", 1)),
					pbn.Hearts:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandE.Hearts, "10", "T", 1)),
					pbn.Spades:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandE.Spades, "10", "T", 1)),
				},
				pbn.South: {
					pbn.Clubs:    parseCardsString(strings.Replace(group.Distribution.BoardData.HandS.Clubs, "10", "T", 1)),
					pbn.Diamonds: parseCardsString(strings.Replace(group.Distribution.BoardData.HandS.Diamonds, "10", "T", 1)),
					pbn.Hearts:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandS.Hearts, "10", "T", 1)),
					pbn.Spades:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandS.Spades, "10", "T", 1)),
				},
				pbn.West: {
					pbn.Clubs:    parseCardsString(strings.Replace(group.Distribution.BoardData.HandW.Clubs, "10", "T", 1)),
					pbn.Diamonds: parseCardsString(strings.Replace(group.Distribution.BoardData.HandW.Diamonds, "10", "T", 1)),
					pbn.Hearts:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandW.Hearts, "10", "T", 1)),
					pbn.Spades:   parseCardsString(strings.Replace(group.Distribution.BoardData.HandW.Spades, "10", "T", 1)),
				},
			},
			Ability: pbn.Ability{
				pbn.North: map[pbn.Suit]int{
					pbn.Clubs:    group.Distribution.BoardData.TricksFromN.Clubs,
					pbn.Diamonds: group.Distribution.BoardData.TricksFromN.Diamonds,
					pbn.Hearts:   group.Distribution.BoardData.TricksFromN.Hearts,
					pbn.Spades:   group.Distribution.BoardData.TricksFromN.Spades,
					pbn.NoTrump:  group.Distribution.BoardData.TricksFromN.Nt,
				},
				pbn.East: map[pbn.Suit]int{
					pbn.Clubs:    group.Distribution.BoardData.TricksFromE.Clubs,
					pbn.Diamonds: group.Distribution.BoardData.TricksFromE.Diamonds,
					pbn.Hearts:   group.Distribution.BoardData.TricksFromE.Hearts,
					pbn.Spades:   group.Distribution.BoardData.TricksFromE.Spades,
					pbn.NoTrump:  group.Distribution.BoardData.TricksFromE.Nt,
				},
				pbn.South: map[pbn.Suit]int{
					pbn.Clubs:    group.Distribution.BoardData.TricksFromS.Clubs,
					pbn.Diamonds: group.Distribution.BoardData.TricksFromS.Diamonds,
					pbn.Hearts:   group.Distribution.BoardData.TricksFromS.Hearts,
					pbn.Spades:   group.Distribution.BoardData.TricksFromS.Spades,
					pbn.NoTrump:  group.Distribution.BoardData.TricksFromS.Nt,
				},
				pbn.West: map[pbn.Suit]int{
					pbn.Clubs:    group.Distribution.BoardData.TricksFromW.Clubs,
					pbn.Diamonds: group.Distribution.BoardData.TricksFromW.Diamonds,
					pbn.Hearts:   group.Distribution.BoardData.TricksFromW.Hearts,
					pbn.Spades:   group.Distribution.BoardData.TricksFromW.Spades,
					pbn.NoTrump:  group.Distribution.BoardData.TricksFromW.Nt,
				},
			},
			OptimumScore: struct {
				Direction pbn.Direction
				Score     int
			}{},
			MinimaxScore: pbn.Contract{},
		}
		if group.Distribution.BoardData.MiniMax != "" {
			rawMinimaxData := group.Distribution.BoardData.MiniMax
			rawMinimaxData = strings.Replace(rawMinimaxData, "nt", "n", 1)
			rawMinimaxData = strings.Replace(rawMinimaxData, "NT", "n", 1)
			rawMinimaxData = strings.Replace(rawMinimaxData, " ", "", -1)
			rawMinimax := strings.Split(rawMinimaxData, "")
			tmpBoard.MinimaxScore.Level, err = strconv.Atoi(rawMinimax[0])
			tmpBoard.MinimaxScore.Suit = pbn.SuitFromSting(rawMinimax[1])
			directionIndex := 2
			if rawMinimax[2] == "X" || rawMinimax[2] == "x" {
				tmpBoard.MinimaxScore.Doubled = true
				directionIndex = 3
			}
			tmpBoard.MinimaxScore.Direction = pbn.DirectionFromString(rawMinimax[directionIndex])
			tmpBoard.MinimaxScore.Score, _ = strconv.Atoi(strings.Join(rawMinimax[directionIndex+1:], ""))
			tmpBoard.OptimumScore.Score = tmpBoard.MinimaxScore.Score
			if tmpBoard.MinimaxScore.Direction != pbn.North && tmpBoard.MinimaxScore.Direction != pbn.South {
				tmpBoard.OptimumScore.Score = -tmpBoard.MinimaxScore.Score
			}
			tmpBoard.OptimumScore.Direction = pbn.North

		}
		boards = append(boards, tmpBoard)
	}
	if len(boards) == 0 {
		return nil, ErrNoDistributionData
	}
	return boards, nil
}

func parseCardsString(str string) []pbn.CardValue {
	cards := make([]pbn.CardValue, 0, len(str))
	for _, c := range str {
		cards = append(cards, pbn.CardValueFromRune(c))
	}
	return cards
}
