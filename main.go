package main

import (
	"context"
	"fmt"
	"log"
	"math/bits"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

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
// - precomputed scores per unique roll (could hash roll)

// []{}
var diceCombinations [][]int

func init() {
	// We want
	// 0b11111
	for i := 0; i <= 0b11111; i++ {
		var indices []int
		for j := 0; j < 5; j++ {
			if i&(1<<j) != 0 {
				indices = append(indices, j)
			}
		}
		if len(indices) == 0 {
			continue
		}
		diceCombinations = append(diceCombinations, indices)
	}
}

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

func (d die) String() string {
	return [...]string{
		"unset",
		"one",
		"two",
		"three",
		"four",
		"five",
		"six",
	}[d]
}

func randDie() die {
	return die(1 + rand.Intn(6))
}

type roll [5]die

func (r roll) String() string {
	var bldr strings.Builder
	for i, d := range r {
		if i != 0 {
			bldr.WriteRune(',')
		}
		bldr.WriteString(d.String())
	}
	return bldr.String()
}

func randRoll() roll {
	return [5]die{randDie(), randDie(), randDie(), randDie(), randDie()}
}

func randRollWithKept(hold []die) roll {
	var r [5]die
	var i int
	for ; i < len(hold); i++ {
		r[i] = hold[i]
	}
	for ; i < 5; i++ {
		r[i] = randDie()
	}
	return r
}

type category uint16

const (
	CAT_ONES = iota
	CAT_TWOS
	CAT_THREES
	CAT_FOURS
	CAT_FIVES
	CAT_SIXES
	CAT_THREE_OF_A_KIND
	CAT_FOUR_OF_A_KIND
	CAT_FULL_HOUSE
	CAT_SMALL_STRAIGHT
	CAT_LARGE_STRAIGHT
	CAT_CHANCE
	CAT_YATZY
)

func (c category) String() string {
	return [13]string{
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
	}[c]
}

// getRollScoreForCategory returns the score
// that a roll would earn for a category.
func getRollScoreForCategory(r roll, c category) uint16 {
	switch c {
	case CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES:
		var score uint16
		val := die(c + 1) // hack: based on cat_xxx index.
		for d := 0; d < 5; d++ {
			if r[d] == val {
				score += uint16(val)
			}
		}
		return score
	case CAT_THREE_OF_A_KIND, CAT_FOUR_OF_A_KIND:
		var freqs [7]byte
		var maxFreq byte
		var sum uint16
		for d := 0; d < 5; d++ {
			freqs[r[d]]++
			maxFreq = max(maxFreq, freqs[r[d]])
			sum += uint16(r[d])
		}
		req := byte(3 + (c - CAT_THREE_OF_A_KIND))
		if maxFreq >= req {
			return sum
		}
		return 0
	case CAT_FULL_HOUSE:
		var freqs [7]byte
		for d := 0; d < 5; d++ {
			freqs[r[d]]++
		}
		var hasTwo, hasThree bool
		for _, f := range freqs {
			if f == 2 {
				hasTwo = true
			} else if f == 3 {
				hasThree = true
			}
		}
		if hasTwo && hasThree {
			return 25
		}
		return 0
	case CAT_SMALL_STRAIGHT, CAT_LARGE_STRAIGHT:
		var minV, maxV int
		var freqs [7]byte
		for d := 0; d < 5; d++ {
			c := int(r[d])
			if d == 0 {
				minV = c
				maxV = c
			} else {
				minV = min(minV, c)
				maxV = max(maxV, c)
			}
			freqs[r[d]]++
		}

		onMinRun, onMaxRun := true, true
		req := int(4 + (c - CAT_SMALL_STRAIGHT))
		for i := 0; i < req; i++ {
			if minV+i >= 7 || freqs[minV+i] == 0 {
				onMinRun = false
			}
			if maxV-i <= 0 || freqs[maxV-i] == 0 {
				onMaxRun = false
			}
		}
		if onMinRun || onMaxRun {
			switch c {
			case CAT_SMALL_STRAIGHT:
				return 30
			default:
				return 40
			}
		}
		return 0
	case CAT_YATZY:
		for d := 1; d < 5; d++ {
			if r[d-1] != r[d] {
				return 0
			}
		}
		return 50
	case CAT_CHANCE:
		var sum uint16
		for d := 0; d < 5; d++ {
			sum += uint16(r[d])
		}
		return sum
	default:
		panic("unimplemented")
	}
}

const AllFilled = 0x1FFF // 13 categories

type playerScorecard struct {
	scoresByCategory [13]uint16
	// catMask is the mask representing which categories
	// have been filled/used (so can no longer be set).
	catMask uint16
}

func (ps playerScorecard) pretty() string {
	var bldr strings.Builder
	bldr.WriteString("┏━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━┓\n")
	bldr.WriteString(fmt.Sprintf("┃ %-17s ┃ %-10s ┃\n", "Category", "Score"))
	bldr.WriteString("┣━━━━━━━━━━━━━━━━━━━╋━━━━━━━━━━━━┫\n")
	for cat, score := range ps.scoresByCategory {
		bldr.WriteString(fmt.Sprintf("┃ %-17s ┃ %-10d ┃\n", category(cat), score))
	}

	bldr.WriteString("┗━━━━━━━━━━━━━━━━━━━┻━━━━━━━━━━━━┛")
	return bldr.String()
}

func (ps playerScorecard) score() uint16 {
	var total uint16
	for _, score := range ps.scoresByCategory {
		total += score
	}
	var upperScoreTotal uint16
	for _, cat := range []category{
		CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES,
	} {
		upperScoreTotal += ps.scoresByCategory[cat]
	}
	if upperScoreTotal >= 63 {
		total += 35
	}
	return total
}

func (ps playerScorecard) getTurnsLeft() int {
	return 13 - bits.OnesCount16(ps.catMask)
}

const (
	categories              = 13
	upperSectionMinBonusSum = 63
	upperSectionBonus       = 35
	maxReRolls              = 3
	yatzyBonus              = 100
)

// update gets the next scorecard calculated after a roll and category are chosen.
// This function does not check that the category has not been used.
func (ps playerScorecard) update(r roll, c category) playerScorecard {
	// debug: check catMask does not have c set.
	var next playerScorecard
	next.catMask = ps.catMask
	next.scoresByCategory = ps.scoresByCategory
	next.scoresByCategory[c] = getRollScoreForCategory(r, c)
	next.catMask = ps.catMask | uint16(1<<c)

	rs := getRollScoreForCategory(r, CAT_YATZY)
	if rs == 0 {
		return next
	}
	hasScoredYatzy := ps.scoresByCategory[CAT_YATZY] > 0
	if !hasScoredYatzy {
		return next // no joker if you haven't scored already
	}

	next.scoresByCategory[CAT_YATZY] += yatzyBonus
	switch c {
	case CAT_FULL_HOUSE:
		next.scoresByCategory[c] = 25
	case CAT_SMALL_STRAIGHT:
		next.scoresByCategory[c] = 30
	case CAT_LARGE_STRAIGHT:
		next.scoresByCategory[c] = 40
	}
	return next
}

// getNext checks all available scorecards that would be
// available for a given roll. The number of scorecards
// returned is N+1 where N is the number of turns left.
func (ps playerScorecard) getNext(r roll) []playerScorecard {
	turnsLeft := ps.getTurnsLeft()
	scorecards := make([]playerScorecard, turnsLeft)
	cur := ^ps.catMask
	for i := 0; i < turnsLeft; i++ {
		idx := bits.TrailingZeros16(cur)
		cur ^= (1 << idx)
		scorecards[i] = ps.update(r, category(idx))
	}
	return scorecards
}

type turn struct {
	currentRoll roll
	// the number of rolls used in a turn (up to 3).
	// a turn that has just started will have a rollCnt
	// of zero.
	rollCnt int
}

func (t *turn) reset() {
	t.rollCnt = 0
}

type move struct {
	hold      []die            // dice to keep
	selection *playerScorecard // cat selection (if any)
	reroll    bool             // whether to re-roll
}

type player interface {
	pickMove(ctx context.Context, g *game, moves []*move) int
}

type game struct {
	scorecards   []playerScorecard
	curTurn      *turn
	players      []player
	curPlayerIdx int
}

func newGame(players []player) *game {
	return &game{
		scorecards: make([]playerScorecard, len(players)),
		curTurn:    new(turn),
		players:    players,
	}
}

// doPly runs a single ply for the current player. Returns whether
// the game is over.
func (g *game) doPly() bool {
	r := randRoll()
	g.curTurn.rollCnt = 1
	pIdx := g.curPlayerIdx
	ps := g.scorecards[pIdx]
	for {
		log.Printf("player [%d]: rolled %s", pIdx, r)

		turnsLeft := ps.getTurnsLeft()
		catMoves := ps.getNext(r)

		var moves []*move
		for _, catPs := range catMoves {
			moves = append(moves, &move{selection: &catPs})
		}

		canRollAgain := g.curTurn.rollCnt < 3
		if canRollAgain {
			for _, c := range diceCombinations {
				if len(c) == 5 {
					continue // can't keep all and re-roll.
				}
				var hold []die
				for _, idx := range c {
					hold = append(hold, r[idx])
				}
				moves = append(moves, &move{
					hold:   hold,
					reroll: true,
				})
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		moveIdx := g.players[g.curPlayerIdx].pickMove(ctx, g, moves)
		cancel()
		move := moves[moveIdx]
		if move.reroll {
			var hold []string
			for _, d := range move.hold {
				hold = append(hold, d.String())
			}
			log.Printf("player [%d]: selects to roll again, holding %s.", pIdx, strings.Join(hold, ","))
			r = randRollWithKept(move.hold)
			g.curTurn.rollCnt += 1
			continue
		} else {
			next := move.selection
			cat := category(bits.TrailingZeros16(next.catMask ^ ps.catMask)) // get what flipped
			log.Print(next.pretty())
			log.Printf("player [%d]: selects to choose category %s (+%d)", pIdx, cat, next.score()-ps.score())
			log.Printf("player [%d]: has %d points", pIdx, next.score())
			gameOver := pIdx == len(g.players)-1 && turnsLeft <= 1
			g.scorecards[pIdx] = *next
			g.curPlayerIdx = (g.curPlayerIdx + 1) % len(g.players)
			g.curTurn.reset()
			return gameOver
		}
	}
}

func (g *game) clone() *game {
	scorecards := make([]playerScorecard, len(g.scorecards))
	for i, s := range g.scorecards {
		scorecards[i] = s
	}
	turn := &turn{
		currentRoll: g.curTurn.currentRoll,
		rollCnt:     g.curTurn.rollCnt,
	}
	return &game{
		scorecards:   scorecards,
		curTurn:      turn,
		players:      g.players, // not cloned (they don't have state)
		curPlayerIdx: g.curPlayerIdx,
	}
}

func (g *game) runSimulation(ctx context.Context) {
	var gameOver bool
	for !gameOver {
		select {
		case <-ctx.Done():
			return
		default:
		}
		r := randRoll()

		g.curTurn.rollCnt = 1
		pIdx := g.curPlayerIdx
		ps := g.scorecards[pIdx]
		for !gameOver {
			turnsLeft := ps.getTurnsLeft()
			catMoves := ps.getNext(r)

			var moves []*move
			for _, catPs := range catMoves {
				moves = append(moves, &move{selection: &catPs})
			}

			canRollAgain := g.curTurn.rollCnt < 3
			if canRollAgain {
				for _, c := range diceCombinations {
					if len(c) == 5 {
						continue // can't keep all and re-roll.
					}
					var hold []die
					for _, idx := range c {
						hold = append(hold, r[idx])
					}
					moves = append(moves, &move{
						hold:   hold,
						reroll: true,
					})
				}
			}

			moveIdx := g.players[g.curPlayerIdx].pickMove(ctx, g, moves)
			move := moves[moveIdx]
			if move.reroll {
				r = randRollWithKept(move.hold)
				g.curTurn.rollCnt += 1
				continue
			} else {
				next := move.selection
				gameOver = pIdx == len(g.players)-1 && turnsLeft <= 1
				g.scorecards[pIdx] = *next
				g.curPlayerIdx = (g.curPlayerIdx + 1) % len(g.players)
				g.curTurn.reset()
			}
		}
	}
}

type randomPlayer struct{}

func (*randomPlayer) pickMove(_ context.Context, _ *game, moves []*move) int {
	return rand.Intn(len(moves))
}

type monteCarloPlayer struct {
}

// montecarlo runs a monte-carlo simulation, returning which move to pick or whether
// to re-roll given a list of moves and a context.
func (*monteCarloPlayer) pickMove(ctx context.Context, g *game, moves []*move) int {
	if len(g.players) != 2 {
		panic("unsupported")
	}
	// Run N workers.
	const workers = 32

	type result struct {
		moveIdx int
		score   uint16
		won     bool
	}

	var wg sync.WaitGroup
	results := make(chan result)
	moveCh := make(chan int)
	defer close(results)

	playerIdx := g.curPlayerIdx
	log.Printf("Thinking for player %d with %d workers.", playerIdx, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			fmt.Println(1)
			for moveIdx := range moveCh {
				fmt.Println(2)
				// run out game by randomly making moves...
				// for example, at the start of the game,
				// say we roll a large straight.
				// We will explore every move, including taking it.
				// If we don't though, do we finish with a higher average score?
				// The early stages will probably not have a high average, but later
				// they might become more accurate.
				// TODO: Create HeuristicPlayer who chooses things based
				// on probability.
				sg := g.clone()
				sg.players = []player{
					new(randomPlayer),
					new(randomPlayer),
				}

				sg.runSimulation(ctx)
				selfScore := sg.scorecards[playerIdx].score()
				opponentScore := sg.scorecards[^playerIdx].score()
				fmt.Println(3)
				select {
				case <-ctx.Done():
					fmt.Printf("exit\n")
					return
				case results <- result{
					moveIdx: moveIdx,
					score:   sg.scorecards[playerIdx].score(),
					won:     selfScore >= opponentScore, // anything that isn't a loss is a win?
				}:
				}
				fmt.Println(4)
			}
		}(ctx)
	}

	type stats struct {
		totalScore uint64
		totalGames uint64
		totalWon   uint64
	}

	statsByMove := make(map[int]stats)
think:
	for {
		select {
		case <-ctx.Done():
			log.Printf("HERE?")
			break think
		case r := <-results:
			var wonInc uint64
			if r.won {
				wonInc = 1
			}
			cur := statsByMove[r.moveIdx]
			statsByMove[r.moveIdx] = stats{
				totalScore: cur.totalScore + uint64(r.score),
				totalGames: cur.totalGames + 1,
				totalWon:   cur.totalWon + wonInc,
			}
			next := statsByMove[r.moveIdx]
			fmt.Printf("%d, %d, %d\n", next.totalScore, next.totalGames, next.totalWon)
		case moveCh <- rand.Intn(len(moves)):
		}
	}
	close(moveCh)

	type moveWithStats struct {
		moveIdx int
		stats   stats
	}

	var sMoves []moveWithStats

	// Evaluate options.
	for moveIdx, stats := range statsByMove {
		sMoves = append(sMoves, moveWithStats{
			moveIdx: moveIdx,
			stats:   stats,
		})
	}

	sort.Slice(sMoves, func(i, j int) bool {
		return sMoves[i].stats.totalScore > sMoves[j].stats.totalScore
	})

	fmt.Print("waiting")
	wg.Wait() // Wait for threads.
	fmt.Printf("???")
	return sMoves[0].moveIdx
}

func main() {
	log.SetFlags(0)

	// just simulation for now
	g := newGame([]player{new(randomPlayer), new(monteCarloPlayer)})

	// cardgames.io has human start first.
	for !g.doPly() {
		log.Println("Player finished turn.")
	}
}
