package main

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"math/bits"
	"math/rand/v2"
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

func (g *game) randDie() die {
	return die(1 + g.rng.IntN(6))
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

func hash(r []die) int {
	counts := [6]int{}
	for _, val := range r {
		counts[val-1]++
	}
	base := 6
	hash := 0
	for i := 0; i < 6; i++ {
		hash = hash*base + counts[i]
	}
	return hash
}

func (g *game) randRoll() roll {
	return [5]die{g.randDie(), g.randDie(), g.randDie(), g.randDie(), g.randDie()}
}

func (g *game) randRollWithKept(hold []die) roll {
	var r [5]die
	var i int
	for ; i < len(hold); i++ {
		r[i] = hold[i]
	}
	for ; i < 5; i++ {
		r[i] = g.randDie()
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
		catUsed := ps.catMask&(1<<cat) != 0
		var catUsedStr string
		if catUsed {
			catUsedStr = "*"
		}
		bldr.WriteString(fmt.Sprintf("┃ %-17s ┃ %-10d ┃\n", fmt.Sprintf("%s%s", category(cat), catUsedStr), score))
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

func (ps playerScorecard) maxTheoreticalScore() uint16 {
	var filledTotal uint16
	for _, score := range ps.scoresByCategory {
		filledTotal += score
	}

	// For every empty category, assume we score the best possible score.
	var theoreticalMaxLeft uint16
	unusedMask := ps.catMask
	for unusedMask > 0 {
		cat := category(bits.TrailingZeros16(unusedMask))
		switch cat {
		case CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES:
		}
		unusedMask |= (1 << cat)
	}

	return filledTotal + theoreticalMaxLeft
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
	hold      []die // dice to keep
	from      *playerScorecard
	selection *playerScorecard // cat selection (if any)
	reroll    bool             // whether to re-roll
}

func (m move) cat() category {
	return category(bits.TrailingZeros16(m.selection.catMask ^ m.from.catMask)) // get what flipped
}

func (m move) String() string {
	if m.reroll {
		var hold []string
		for _, d := range m.hold {
			hold = append(hold, d.String())
		}
		return fmt.Sprintf("reroll holding %s", strings.Join(hold, ","))
	}
	c := m.cat()
	return fmt.Sprintf("select %s for %d", c, m.selection.scoresByCategory[c])
}

type player interface {
	pickMove(ctx context.Context, g *game, moves []*move) int
}

type game struct {
	scorecards   []playerScorecard
	curTurn      *turn
	players      []player
	curPlayerIdx int
	rng          *rand.Rand
}

func newGame(rng *rand.Rand, players []player) *game {
	return &game{
		scorecards: make([]playerScorecard, len(players)),
		curTurn:    new(turn),
		players:    players,
		rng:        rng,
	}
}

func (g *game) clone() *game {
	scorecards := make([]playerScorecard, len(g.scorecards))
	copy(scorecards, g.scorecards)
	turn := &turn{
		currentRoll: g.curTurn.currentRoll,
		rollCnt:     g.curTurn.rollCnt,
	}
	return &game{
		scorecards:   scorecards,
		curTurn:      turn,
		players:      g.players, // not cloned (they don't have state)
		curPlayerIdx: g.curPlayerIdx,
		rng:          g.rng,
	}
}

func (g *game) getMovesForCurrentPlayer(r roll) []*move {
	pIdx := g.curPlayerIdx
	ps := g.scorecards[pIdx]
	catMoves := ps.getNext(r)

	var moves []*move
	for _, catPs := range catMoves {
		moves = append(moves, &move{selection: &catPs, from: &ps})
	}

	canRollAgain := g.curTurn.rollCnt < 3
	if canRollAgain {
		mHashes := make(map[int]struct{})
		for _, c := range diceCombinations {
			if len(c) == 5 {
				continue // can't keep all and re-roll.
			}
			var hold []die
			for _, idx := range c {
				hold = append(hold, r[idx])
			}
			hHash := hash(hold)
			if _, ok := mHashes[hHash]; !ok {
				mHashes[hHash] = struct{}{}
				moves = append(moves, &move{
					hold:   hold,
					reroll: true,
				})
			}
		}
	}
	return moves
}

func (g *game) runSimulation(ctx context.Context) {
	var gameOver bool
	// Start round.
	for !gameOver {
		select {
		case <-ctx.Done():
			return
		default:
		}
		r := g.randRoll()

		g.curTurn.rollCnt = 1
		pIdx := g.curPlayerIdx
		ps := g.scorecards[pIdx]

		// Start player turn.
		for {
			moves := g.getMovesForCurrentPlayer(r)
			moveIdx := g.players[g.curPlayerIdx].pickMove(ctx, g, moves)
			move := moves[moveIdx]
			if move.reroll {
				r = g.randRollWithKept(move.hold)
				g.curTurn.rollCnt += 1
				continue
			} else {
				next := move.selection
				turnsLeft := ps.getTurnsLeft()
				gameOver = pIdx == len(g.players)-1 && turnsLeft <= 1
				g.scorecards[pIdx] = *next
				g.curPlayerIdx = (g.curPlayerIdx + 1) % len(g.players)
				g.curTurn.reset()
				break
			}
		}
	}
}

// doPly runs a single ply for the current player. Returns whether
// the game is over.
func (g *game) doPly() bool {
	r := g.randRoll()
	g.curTurn.rollCnt = 1
	pIdx := g.curPlayerIdx
	ps := g.scorecards[pIdx]
	for {
		log.Printf("player [%d]: rolled %s", pIdx, r)
		moves := g.getMovesForCurrentPlayer(r)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		moveIdx := g.players[g.curPlayerIdx].pickMove(ctx, g, moves)
		cancel()
		move := moves[moveIdx]
		log.Printf("player [%d]: %s", pIdx, move)
		if move.reroll {
			r = g.randRollWithKept(move.hold)
			g.curTurn.rollCnt += 1
			continue
		} else {
			next := move.selection
			log.Print(next.pretty())
			log.Printf("player [%d]: has %d points", pIdx, next.score())
			turnsLeft := ps.getTurnsLeft()
			gameOver := pIdx == len(g.players)-1 && turnsLeft <= 1
			g.scorecards[pIdx] = *next
			g.curPlayerIdx = (g.curPlayerIdx + 1) % len(g.players)
			g.curTurn.reset()
			return gameOver
		}
	}
}

type randomPlayer struct {
	rng *rand.Rand
}

func (rp *randomPlayer) pickMove(_ context.Context, _ *game, moves []*move) int {
	return rp.rng.IntN(len(moves))
}

type monteCarloPlayer struct {
	rng *rand.Rand
}

type result struct {
	moveIdx int
	score   uint16
	won     bool
}

// A MinHeap implements heap.Interface and holds Items.
type minHeap []result

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].score < h[j].score } // Min-heap
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(result)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// topN stores the top N largest values seen.
type topN struct {
	heap  minHeap
	limit int
}

func newTopN(n int) *topN {
	h := make(minHeap, 0, n)
	heap.Init(&h)
	return &topN{
		heap:  h,
		limit: n,
	}
}

func (t *topN) insert(r result) {
	if t.heap.Len() < t.limit {
		heap.Push(&t.heap, r)
	} else if r.score > t.heap[0].score {
		t.heap[0] = r
		heap.Fix(&t.heap, 0)
	}
}

func (tn topN) avg() float64 {
	var total uint64
	for _, r := range tn.heap {
		total += uint64(r.score)
	}
	return float64(total) / float64(tn.heap.Len())
}

// montecarlo runs a monte-carlo simulation, returning which move to pick or whether
// to re-roll given a list of moves and a context.
func (mcp *monteCarloPlayer) pickMove(ctx context.Context, g *game, moves []*move) int {
	start := time.Now()
	if len(g.players) != 2 {
		panic("unsupported")
	}
	// Run N workers.
	const workers = 64

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
			for moveIdx := range moveCh {
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
					&randomPlayer{mcp.rng},
					&randomPlayer{mcp.rng},
				}

				sg.runSimulation(ctx)
				selfScore := sg.scorecards[playerIdx].score()
				opponentScore := sg.scorecards[playerIdx^1].score()
				select {
				case <-ctx.Done():
					return
				case results <- result{
					moveIdx: moveIdx,
					score:   selfScore,
					won:     selfScore >= opponentScore, // anything that isn't a loss is a win?
				}:
				}
			}
		}(ctx)
	}

	type stats struct {
		totalScore uint64
		totalGames uint64
		totalWon   uint64
		maxScore   uint16
		topScores  *topN
	}

	statsByMove := make(map[int]*stats)
	for i := range moves {
		statsByMove[i] = &stats{
			topScores: newTopN(500),
		}
	}

think:
	for {
		select {
		case <-ctx.Done():
			break think
		case r := <-results:
			var wonInc uint64
			if r.won {
				wonInc = 1
			}
			s := statsByMove[r.moveIdx]
			s.totalScore += uint64(r.score)
			s.totalGames += 1
			s.totalWon += wonInc
			s.maxScore = max(s.maxScore, r.score)
			s.topScores.insert(r)
		case moveCh <- mcp.rng.IntN(len(moves)):
		}
	}
	close(moveCh)

	type moveWithStats struct {
		moveIdx int
		stats   *stats
	}

	var totalGamesExplored uint64
	var sMoves []moveWithStats

	// Evaluate options.
	for moveIdx, stats := range statsByMove {
		sMoves = append(sMoves, moveWithStats{
			moveIdx: moveIdx,
			stats:   stats,
		})
		totalGamesExplored += stats.totalGames
	}

	sort.Slice(sMoves, func(i, j int) bool {
		is, js := sMoves[i].stats, sMoves[j].stats
		return is.topScores.avg() > js.topScores.avg()
	})

	wg.Wait() // Wait for threads.
	took := time.Since(start)
	fmt.Printf("Stopped. Explored %d games (%.2f g/s)\n", totalGamesExplored, float64(totalGamesExplored)/took.Seconds())
	for i, sm := range sMoves {
		stats := sm.stats
		avgScore := float64(stats.totalScore) / float64(stats.totalGames)
		wonPct := float64(stats.totalWon) / float64(stats.totalGames)
		move := moves[sm.moveIdx]
		fmt.Printf("[%d]: %s (%.2f avg) (%d max) (%.2f top n avg) (%.2f won pct)\n", i, move, avgScore, stats.maxScore, stats.topScores.avg(), wonPct)
	}
	return sMoves[0].moveIdx
}

func main() {
	log.SetFlags(0)
	r := rand.New(rand.NewPCG(uint64(1234), 0))

	// just simulation for now
	g := newGame(r, []player{&randomPlayer{r}, &monteCarloPlayer{r}})

	// cardgames.io has human start first.
	for !g.doPly() {
		log.Println("Player finished turn.")
	}
}
