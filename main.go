package main

// yatzy (cardgames.io version not wikipedia yatzy, but somewhat confusingly the Yahtzee wikipedia game??)
// components:
// - rng
// - dice
// - probabilities
// - terminology
//   - upper section
//     - ones, twos, threes, ...
//     - 63 bonus - 35
//   - lower section
//     - three of a kind
//     - four of a kind
//     - full house
//     - small straight
//     - large straight
//     - chance
//     - yatzy
// - simulation and CLI
//
// goal is to be able to simulate and measure perfromance
// by average score and input rolls to play against bill.
//
// decisions to make in this game:
// - what category to choose.
// - whether you want to roll again.
//
// potential optimizations:
// - faster rand (or different ones)
// - pools for object allocation
//


type die byte

const (
	DIE_UNSET die = iota
	DIE_ONE
	DIE_TWO
	DIE_THREE
	DIE_FOUR
	DIE_FIVE
	DIE_SIX
)

const dieNames = [...]string{
	"unset",
	"one",
	"two",
	"three",
	"four",
	"five",
	"six",
}

func (d die) String() string {
	return dieNames[d]
}

func randDie() die {
	return die(1+rand.Intn(6))
}


type roll [5]die

func randRoll() roll {
	return [5]die{randDie(), randDie(), randDie(), randDie(), randDie()}
}

type rollMask uint16

func (r roll) getMask() rollMask {
	var m rollMask
	for d := range 5 {
		m |= ((r[d] & 0b111) << (d * 3))
	}
	return m
}

type category uint16

const (
	CAT_ONES = iota
	CAT_TWOS
	CAT_THREES
	CAT_FOURS
	CAT_FIVES
	CAT_SIXES
	CAT_THREE_OF_KIND
	CAT_FOUR_OF_KIND
	CAT_FULL_HOUSE
	CAT_SMALL_STRAIGHT
	CAT_LARGE_STRAIGHT
	CAT_CHANCE
	CAT_YATZY
)

const catNames = [13]string{
	"ones",
	"twos",
	"threes",
	"fours",
	"fives",
	"sixes",
	"three of a kind",
	"four of a kind",
	"full house",
	"small straight",
	"large straight",
	"chance",
	"yatzy",
}

func (c category) String() string {
	return catNames[c]
}

type playerScorecard struct {
	scoresByCategory [13]uint8
	// catMask is the mask representing which categories
	// have been filled/used (so can no longer be set).
	catMask uint8
}

// getRollScoreForCategory returns the score
// that a roll would earn for a category.
func getRollScoreForCategory(r roll, c category) uint8 {
	var score uint8
	switch c {
	case CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES:
		val := c+1 // hack: based on cat_xxx index.
		for d := range 5 {
			if r[d] == val {
				score += val
			}
		}
	}
}


// getNext checks all available scorecards that would be
// available for a given roll. The number of scorecards
// returned is N+1 where N is the number of turns left.
func (ps playerScorecard) getNext(r roll) []playerScorecard {

}

type turn struct {
	currentRoll roll
	// the number of rolls used in a turn (up to 3).
	// a turn that has just started will have a rollCnt
	// of zero.
	rollCnt int
}


type game struct {
	scorecards []playerScorecard
	curTurn *turn
	curScorecardIdx int
}

func newGame(numPlayers int) *game {
	return &game{
		scorecards: make([]playerScorecard, numPlayers),
		curTurn: new(turn),
	}
}


func main() {
	log.SetFlags(0)

	// just simulation for now
	pvpGame := newGame(2)

	// cardgames.io has human start first.
}
