package main

import (
	"cmp"
	"container/heap"
	"context"
	"fmt"
	"log"
	"math/bits"
	"math/rand/v2"
	"slices"
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

type scoreData struct {
	score uint16
	used  [][]int // dice indices that could be used for the score.
}

func getScoreData(r2 rollV2, c category) scoreData {
	r := r2.dice()
	used := make([][]int, 1, 2)
	used[0] = make([]int, 0, 5)
	switch c {
	case CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES:
		var score uint16
		catScoreVal := uint16(1 + (c - CAT_ONES))
		for d := 0; d < 5; d++ {
			if dieCat(r[d]) == c {
				score += catScoreVal
				used[0] = append(used[0], d)
			}
		}
		if score == 0 {
			return scoreData{}
		}
		return scoreData{
			score: score,
			used:  used,
		}
	case CAT_THREE_OF_A_KIND, CAT_FOUR_OF_A_KIND:
		var freqs [7]byte
		var maxFreq byte
		var maxFreqDieIdx int
		var sum uint16
		for d := 0; d < 5; d++ {
			freqs[r[d]]++
			if freqs[r[d]] > maxFreq {
				maxFreq = freqs[r[d]]
				maxFreqDieIdx = d
			}
			maxFreq = max(maxFreq, freqs[r[d]])
			sum += uint16(r[d])
		}
		req := byte(3 + (c - CAT_THREE_OF_A_KIND))
		if maxFreq >= req {
			for d := range 5 {
				if r[d] == r[maxFreqDieIdx] {
					used[0] = append(used[0], d)
				}
			}
			return scoreData{
				score: sum,
				used:  used,
			}
		}
		return scoreData{}
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
			return scoreData{
				score: 25,
				used:  [][]int{{0, 1, 2, 3, 4}},
			}
		}
		return scoreData{}
	case CAT_SMALL_STRAIGHT, CAT_LARGE_STRAIGHT:
		type indexedDie struct {
			val die
			idx int
		}

		rSorted := make([]indexedDie, len(r))
		for i, v := range r {
			rSorted[i] = indexedDie{v, i}
		}

		slices.SortFunc(rSorted, func(i, j indexedDie) int {
			return cmp.Compare(i.val, j.val)
		})

		valToIdx := make(map[die][]int)
		for _, rs := range rSorted {
			valToIdx[rs.val] = append(valToIdx[rs.val], rs.idx)
		}

		getConseq := func(idx int, cnt int) ([][]int, bool) {
			last := rSorted[idx].val

			conseq := 1
			seqs := [][]int{{rSorted[idx].idx}}
			for conseq < cnt {
				indices, ok := valToIdx[last+1]
				if !ok {
					break
				}
				switch len(indices) {
				case 2:
					if len(seqs) == 0 {
						for _, idx := range indices {
							seqs = append(seqs, []int{idx})
						}
					} else {
						seqs = append(seqs, []int{})
						seqs[1] = append(seqs[1], seqs[0]...)
						for i, idx := range indices {
							seqs[i] = append(seqs[i], idx)
						}
					}
				case 1:
					if len(seqs) == 0 {
						seqs = append(seqs, []int{indices[0]})
					} else {

						for i, s := range seqs {
							seqs[i] = append(s, indices[0])
						}
					}
				}

				conseq++
				last++
			}

			if conseq >= cnt {
				return seqs, true
			} else {
				return nil, false
			}
		}

		switch c {
		case CAT_SMALL_STRAIGHT:
			used1, ok1 := getConseq(0, 4)
			used2, ok2 := getConseq(1, 4)
			if !ok1 && !ok2 {
				break
			}
			switch {
			case ok1 && ok2:
				used = [][]int{used1[0], used2[0]}
			case ok1:
				used = used1
			case ok2:
				used = used2
			}
			return scoreData{
				score: 30,
				used:  used,
			}
			return scoreData{
				score: 30,
				used:  used,
			}
		case CAT_LARGE_STRAIGHT:
			if used, ok := getConseq(0, 5); ok {
				return scoreData{
					score: 40,
					used:  used,
				}
			}
		}
		return scoreData{}
	case CAT_YATZY:
		for d := 1; d < 5; d++ {
			if r[d-1] != r[d] {
				return scoreData{}
			}
		}
		return scoreData{
			score: 50,
			used:  [][]int{{0, 1, 2, 3, 4}},
		}
	case CAT_CHANCE:
		var sum uint16
		for d := 0; d < 5; d++ {
			sum += uint16(r[d])
		}
		return scoreData{
			score: sum,
			used:  [][]int{{0, 1, 2, 3, 4}},
		}
	default:
		panic("unimplemented")
	}
}

var diceCombinations [][]int

// might want to consider also iterating on this and just doing rand on some entire slice.
// var rollToHash map[roll]uint16
// var rollHashScores map[uint16][13]uint16

type rollV2 uint16

func newRollV2(a, b, c, d, e die) rollV2 {
	return rollV2(uint16(a&7) | uint16(b&7)<<3 | uint16(c&7)<<6 | uint16(d&7)<<9 | uint16(e&7)<<12)
}

func newRollV2_2(d [5]die) rollV2 {
	return newRollV2(d[0], d[1], d[2], d[3], d[4])
}

func (r2 rollV2) die(idx int) die {
	return die((r2 >> (idx * 3)) & 7)
}

func (r2 rollV2) dice() [5]die {
	a := die(r2 & 7)
	b := die((r2 >> 3) & 7)
	c := die((r2 >> 6) & 7)
	d := die((r2 >> 9) & 7)
	e := die((r2 >> 12) & 7)
	return [5]die{a, b, c, d, e}
}

func (r2 rollV2) String() string {
	a := die(r2 & 7)
	b := die((r2 >> 3) & 7)
	c := die((r2 >> 6) & 7)
	d := die((r2 >> 9) & 7)
	e := die((r2 >> 12) & 7)
	return fmt.Sprintf("%s,%s,%s,%s,%s", a, b, c, d, e)
}

var rolls []rollV2
var scoresByRoll map[rollV2][13]scoreData

// subset die -> EV by category
// ex: [one one one one] -> yatzy would be 1/6 so EV would be (50 * 1/6)
// ex: [one two three four] -> large straight would be (1/6 * 40)
// ex: [one two three four] -> yatzy would be 0
var possibleSubDieByScore [13]map[int][]die

// var expectedValueBySubset map[int][13]float64

func getDiceCombos(n int) [][]die {
	if n == 0 {
		return nil
	} else if n == 1 {
		return [][]die{{1}, {2}, {3}, {4}, {5}, {6}}
	}

	var next [][]die
	left := getDiceCombos(n - 1)
	for i := range 6 {
		d := die(i + 1)
		for _, l := range left {
			next = append(next, append([]die{d}, l...))
		}
	}
	return next
}

func init() {
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

	for c := CAT_ONES; c <= CAT_YATZY; c++ {
		possibleSubDieByScore[c] = make(map[int][]die)
	}

	scoresByRoll = make(map[rollV2][13]scoreData)
	for _, combos := range getDiceCombos(5) {
		r2 := newRollV2(combos[0], combos[1], combos[2], combos[3], combos[4])
		rolls = append(rolls, r2)

		var scores [13]scoreData
		for c := CAT_ONES; c <= CAT_YATZY; c++ {
			scoreData := getScoreData(r2, category(c))
			scores[c] = scoreData

			if scoreData.score > 0 {
				for _, used := range scoreData.used {
					var usedToScore []die
					for _, idx := range used {
						usedToScore = append(usedToScore, r2.die(idx))
					}
					possibleSubDieByScore[c][hash(usedToScore)] = usedToScore
				}
			}
		}
		scoresByRoll[r2] = scores
	}

	/*
		for c, m := range possibleSubDieByScore {
			for _, d := range m {
				fmt.Printf("can score %s with %+v\n", category(c), d)
			}
		}
	*/
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

func (g *game) randRollV2() rollV2 {
	return rolls[g.rng.IntN(len(rolls))]
}

func (g *game) randRollV2WithKept(hold []die) rollV2 {
	var r [5]die
	var i int
	for ; i < len(hold); i++ {
		r[i] = hold[i]
	}
	for ; i < 5; i++ {
		r[i] = g.randDie()
	}
	return newRollV2_2(r)
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

func dieCat(d die) category {
	return [...]category{CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES}[d-DIE_ONE]
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

	var upperScoreTotal uint16
	var maxUpperScoreLeft uint16
	for _, cat := range []category{
		CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES,
	} {
		if ps.catMask&(1<<cat) != 0 {
			upperScoreTotal += ps.scoresByCategory[cat]
		} else {
			maxUpperScoreLeft += uint16(die(cat+1) * 5)
		}
	}

	if upperScoreTotal+maxUpperScoreLeft >= 63 {
		filledTotal += 35
	}

	// For every empty category, assume we score the best possible score.
	var theoreticalMaxLeft uint16
	unusedMask := (^ps.catMask & 0x1FFF)
	for unusedMask > 0 {
		cat := category(bits.TrailingZeros16(unusedMask))
		switch cat {
		case CAT_ONES, CAT_TWOS, CAT_THREES, CAT_FOURS, CAT_FIVES, CAT_SIXES:
			val := 1 + (cat - CAT_ONES)
			theoreticalMaxLeft += uint16(val * 5)
		case CAT_THREE_OF_A_KIND, CAT_FOUR_OF_A_KIND, CAT_SMALL_STRAIGHT, CAT_CHANCE:
			theoreticalMaxLeft += 30
		case CAT_LARGE_STRAIGHT:
			theoreticalMaxLeft += 40
		case CAT_FULL_HOUSE:
			theoreticalMaxLeft += 25
		case CAT_YATZY:
			movesLeft := ps.getTurnsLeft()
			theoreticalMaxLeft += uint16(50 + ((movesLeft - 1) * 100))
		}
		unusedMask ^= (1 << cat)
	}
	if ps.scoresByCategory[CAT_YATZY] > 0 {
		movesLeft := ps.getTurnsLeft()
		theoreticalMaxLeft += uint16((movesLeft - 1) * 100)
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
func (ps playerScorecard) update(r rollV2, c category) playerScorecard {
	var next playerScorecard
	rs := scoresByRoll[r][c].score
	next.catMask = ps.catMask
	next.scoresByCategory = ps.scoresByCategory
	next.scoresByCategory[c] = rs
	next.catMask = ps.catMask | uint16(1<<c)

	if scoresByRoll[r][CAT_YATZY].score == 0 {
		return next
	}

	hasScoredYatzy := ps.scoresByCategory[CAT_YATZY] > 0
	if hasScoredYatzy {
		next.scoresByCategory[CAT_YATZY] += yatzyBonus
	}

	// Only allow a joker in the lower section if the respective upper section
	// category is filled.
	die := r.die(0)
	dieCat := dieCat(die)
	if ps.catMask&(1<<dieCat) == 0 {
		return next // must take points in upper section
	}

	switch c {
	// other c's already covered -- add ones that joker helps.
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
func (ps playerScorecard) getNext(r rollV2) []playerScorecard {
	turnsLeft := ps.getTurnsLeft()
	scorecards := make([]playerScorecard, turnsLeft)
	cur := (^ps.catMask & 0x1FFF)
	for i := 0; i < turnsLeft; i++ {
		idx := bits.TrailingZeros16(cur)
		cur ^= (1 << idx)
		scorecards[i] = ps.update(r, category(idx))
	}
	return scorecards
}

type turn struct {
	currentRoll rollV2
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

func (g *game) getMovesForCurrentPlayer(r rollV2) []*move {
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
				hold = append(hold, r.die(idx))
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

func (g *game) doMove(m *move) bool {
	if m.reroll {
		g.curTurn.currentRoll = g.randRollV2WithKept(m.hold)
		g.curTurn.rollCnt += 1
		return false
	}
	pIdx := g.curPlayerIdx
	next := m.selection
	turnsLeft := g.scorecards[pIdx].getTurnsLeft()
	gameOver := pIdx == len(g.players)-1 && turnsLeft <= 1
	g.scorecards[pIdx] = *next
	g.curPlayerIdx = (pIdx + 1) % len(g.players)
	g.curTurn.reset()
	return gameOver
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

		// Start player turn.
		g.curTurn.currentRoll = g.randRollV2()
		g.curTurn.rollCnt = 1
		curPlayer := g.curPlayerIdx
		for g.curPlayerIdx == curPlayer && !gameOver {
			moves := g.getMovesForCurrentPlayer(g.curTurn.currentRoll)
			moveIdx := g.players[curPlayer].pickMove(ctx, g, moves)
			move := moves[moveIdx]
			gameOver = g.doMove(move)
		}
	}
}

// doPly runs a single ply for the current player. Returns whether
// the game is over.
func (g *game) doPly() bool {
	// Start player turn.
	g.curTurn.currentRoll = g.randRollV2()
	g.curTurn.rollCnt = 1
	curPlayer := g.curPlayerIdx
	for g.curPlayerIdx == curPlayer {
		log.Printf("player [%s]: rolled %s", g.players[curPlayer], g.curTurn.currentRoll)
		moves := g.getMovesForCurrentPlayer(g.curTurn.currentRoll)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		moveIdx := g.players[curPlayer].pickMove(ctx, g, moves)
		cancel()
		move := moves[moveIdx]
		log.Printf("player [%s]: %s", g.players[curPlayer], move)
		if !move.reroll {
			log.Print(move.selection.pretty())
			log.Printf("player [%s]: has %d points", g.players[curPlayer], move.selection.score())
		}
		if g.doMove(move) {
			return true
		}
	}
	return false
}

type randomPlayer struct {
	rng *rand.Rand
}

func (rp *randomPlayer) String() string { return "random player" }

func (rp *randomPlayer) pickMove(_ context.Context, _ *game, moves []*move) int {
	return rp.rng.IntN(len(moves))
}

type monteCarloPlayer struct {
	rng *rand.Rand
}

func (mcp *monteCarloPlayer) String() string { return "MC" }

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
	const workers = 100

	var wg sync.WaitGroup
	results := make(chan result)
	done := make(chan struct{})
	defer close(done)

	// organize selections by used dice.
	// i.e. we know that using a single 3 from a first dice roll will have a worst-opportunity cost
	// than re-rolling with any 3 as we could always fallback and select our original choice.

	playerIdx := g.curPlayerIdx
	log.Printf("Thinking with %d workers.", workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			for {
				moveIdx := g.rng.IntN(len(moves))

				sg := g.clone()
				sg.players = []player{
					&randomPlayer{mcp.rng},
					&randomPlayer{mcp.rng},
				}
				if !sg.doMove(moves[moveIdx]) {
					sg.runSimulation(ctx)
				}

				selfScore := sg.scorecards[playerIdx].score()
				opponentScore := sg.scorecards[playerIdx^1].score()
				select {
				case <-ctx.Done():
					return
				case results <- result{
					moveIdx: moveIdx,
					score:   selfScore,
					won:     selfScore >= opponentScore, // anything that isn't a loss is a win :)
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
			topScores: newTopN(50),
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
		}
	}

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

	// Try ordering rerolls by ones that allow for any available category left.
	/*
		ps := g.scorecards[playerIdx]
		rerollPotentials := make(map[int]int) // moveIdx -> EV
		for _, m := range moves {
			if m.reroll {
				for cat, score := range ps.scoresByCategory {
					catUsed := ps.catMask&(1<<cat) != 0
					if !catUsed {
						// expectedValueBySubset[moveHas(m)] = EV
					}
				}
			}
		}*/

	sort.Slice(sMoves, func(i, j int) bool {
		is, js := sMoves[i].stats, sMoves[j].stats
		return is.topScores.avg() > js.topScores.avg()
		// return is.maxScore > js.maxScore
	})

	wg.Wait() // Wait for threads.
	took := time.Since(start)
	fmt.Printf("Stopped. Explored %d games (%.2f g/s)\n", totalGamesExplored, float64(totalGamesExplored)/took.Seconds())
	for i, sm := range sMoves {
		stats := sm.stats
		avgScore := float64(stats.totalScore) / float64(stats.totalGames)
		wonPct := float64(stats.totalWon) / float64(stats.totalGames)
		move := moves[sm.moveIdx]
		fmt.Printf("[%d]: %s (%d games) (%.4f avg) (%d max) (%.4f top n avg) (%.2f won pct)\n", i, move, stats.totalGames, avgScore, stats.maxScore, stats.topScores.avg(), wonPct)
	}
	return sMoves[0].moveIdx
}

func main() {
	log.SetFlags(0)
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))

	// just simulation for now
	g := newGame(r, []player{&randomPlayer{r}, &monteCarloPlayer{r}})

	// cardgames.io has human start first.
	for !g.doPly() {
		log.Println("Player finished turn.")
	}
}
