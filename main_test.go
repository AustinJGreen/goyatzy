package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetRollScoreForCategory(t *testing.T) {
	for _, tt := range []struct {
		r roll
		c category
		w uint16
	}{
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_ONES,
			w: 0,
		},
		{
			r: [5]die{DIE_ONE, DIE_ONE, DIE_TWO, DIE_ONE, DIE_ONE},
			c: CAT_ONES,
			w: 4,
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_TWOS,
			w: 6,
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_THREES,
			w: 3,
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_FOURS,
			w: 4,
		},
		{
			r: [5]die{DIE_TWO, DIE_SIX, DIE_FIVE, DIE_FIVE, DIE_FOUR},
			c: CAT_FIVES,
			w: 10,
		},
		{
			r: [5]die{DIE_TWO, DIE_SIX, DIE_FIVE, DIE_FIVE, DIE_FOUR},
			c: CAT_SIXES,
			w: 6,
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_THREE_OF_A_KIND,
			w: 13,
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO, DIE_FOUR},
			c: CAT_THREE_OF_A_KIND,
			w: 12,
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_THREE, DIE_TWO, DIE_FOUR},
			c: CAT_THREE_OF_A_KIND,
			w: 0,
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_SIX, DIE_ONE, DIE_ONE},
			c: CAT_FOUR_OF_A_KIND,
			w: 0,
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_ONE, DIE_ONE, DIE_ONE},
			c: CAT_FOUR_OF_A_KIND,
			w: 6,
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_ONE, DIE_ONE, DIE_TWO},
			c: CAT_FULL_HOUSE,
			w: 25,
		},
		{
			r: [5]die{DIE_FIVE, DIE_TWO, DIE_ONE, DIE_ONE, DIE_TWO},
			c: CAT_FULL_HOUSE,
			w: 0,
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_SMALL_STRAIGHT,
			w: 30,
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_THREE},
			c: CAT_SMALL_STRAIGHT,
			w: 30,
		},
		{
			r: [5]die{DIE_THREE, DIE_THREE, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_SMALL_STRAIGHT,
			w: 0,
		},
		{
			r: [5]die{DIE_SIX, DIE_TWO, DIE_THREE, DIE_FOUR, DIE_FIVE},
			c: CAT_LARGE_STRAIGHT,
			w: 40,
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_LARGE_STRAIGHT,
			w: 40,
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_FOUR, DIE_FOUR, DIE_FIVE},
			c: CAT_LARGE_STRAIGHT,
			w: 0,
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_CHANCE,
			w: 15,
		},
		{
			r: [5]die{DIE_THREE, DIE_THREE, DIE_THREE, DIE_THREE, DIE_THREE},
			c: CAT_YATZY,
			w: 50,
		},
		{
			r: [5]die{DIE_FOUR, DIE_THREE, DIE_THREE, DIE_THREE, DIE_FOUR},
			c: CAT_YATZY,
			w: 0,
		},
		{
			r: [5]die{DIE_THREE, DIE_THREE, DIE_THREE, DIE_THREE, DIE_FOUR},
			c: CAT_YATZY,
			w: 0,
		},
	} {
		got := getRollScoreForCategory(tt.r, tt.c)
		if got != tt.w {
			t.Errorf("roll score does not match: got %d; want %d", got, tt.w)
		}
	}
}

func TestPlayerScorecardUpdate(t *testing.T) {
	opts := []cmp.Option{
		cmp.AllowUnexported(playerScorecard{}),
	}
	var ps playerScorecard

	// 1. 1, 2, 3, 4, 5 - large straight
	got := ps.update([5]die{DIE_ONE, DIE_TWO, DIE_THREE, DIE_FOUR, DIE_FIVE}, CAT_LARGE_STRAIGHT)
	if diff := cmp.Diff(ps, playerScorecard{}, opts...); diff != "" {
		t.Errorf("original scorecard was modified (-got, +want):\n%s", diff)
	}
	want := playerScorecard{
		scoresByCategory: [13]uint16{
			CAT_LARGE_STRAIGHT: 40,
		},
		catMask: 1024,
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
	}

	// 2. 1, 1, 1, 1, 1 - take ones
	got = got.update([5]die{DIE_ONE, DIE_ONE, DIE_ONE, DIE_ONE, DIE_ONE}, CAT_ONES)
	want = playerScorecard{
		scoresByCategory: [13]uint16{
			CAT_ONES:           5,
			CAT_LARGE_STRAIGHT: 40,
		},
		catMask: 1025,
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
	}

	// 3. 2, 2, 2, 2, 2 - take yatzy
	got = got.update([5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO}, CAT_YATZY)
	want = playerScorecard{
		scoresByCategory: [13]uint16{
			CAT_ONES:           5,
			CAT_LARGE_STRAIGHT: 40,
			CAT_YATZY:          50,
		},
		catMask: 5121,
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
	}

	// 4. All six - use joker on full house + see 100 bonus
	got = got.update([5]die{DIE_SIX, DIE_SIX, DIE_SIX, DIE_SIX, DIE_SIX}, CAT_FULL_HOUSE)
	want = playerScorecard{
		scoresByCategory: [13]uint16{
			CAT_ONES:           5,
			CAT_FULL_HOUSE:     25,
			CAT_LARGE_STRAIGHT: 40,
			CAT_YATZY:          150,
		},
		catMask: 5377,
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
	}
}

func TestGetTurnsLeft(t *testing.T) {
	ps := playerScorecard{
		scoresByCategory: [13]uint16{
			CAT_ONES:           5,
			CAT_LARGE_STRAIGHT: 40,
		},
		catMask: 1025,
	}
	if got, want := ps.getTurnsLeft(), 11; got != want {
		t.Errorf("turns left do not match: got %d; want %d", got, want)
	}
}

func TestPlayerScorecardGetNext(t *testing.T) {
	opts := []cmp.Option{
		cmp.AllowUnexported(playerScorecard{}),
	}
	var ps playerScorecard
	got := ps.getNext([5]die{DIE_ONE, DIE_TWO, DIE_THREE, DIE_FOUR, DIE_FIVE})
	want := []playerScorecard{
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_ONES: 1,
			},
			catMask: 1,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_TWOS: 2,
			},
			catMask: 2,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_THREES: 3,
			},
			catMask: 4,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_FOURS: 4,
			},
			catMask: 8,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_FIVES: 5,
			},
			catMask: 16,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_SIXES: 0,
			},
			catMask: 32,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_THREE_OF_A_KIND: 0,
			},
			catMask: 64,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_FOUR_OF_A_KIND: 0,
			},
			catMask: 128,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_FULL_HOUSE: 0,
			},
			catMask: 256,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_SMALL_STRAIGHT: 30,
			},
			catMask: 512,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_LARGE_STRAIGHT: 40,
			},
			catMask: 1024,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_CHANCE: 15,
			},
			catMask: 2048,
		},
		playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_YATZY: 0,
			},
			catMask: 4096,
		},
	}
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Errorf("next scorecards do not match (-got, +want):\n%s", diff)
	}
}

func TestPlayerScorecardScore(t *testing.T) {
	ps := playerScorecard{
		scoresByCategory: [13]uint16{
			CAT_ONES:            4,
			CAT_TWOS:            6,
			CAT_THREES:          12,
			CAT_FOURS:           16,
			CAT_FIVES:           20,
			CAT_SIXES:           18,
			CAT_THREE_OF_A_KIND: 28,
			CAT_FOUR_OF_A_KIND:  29,
			CAT_FULL_HOUSE:      25,
			CAT_YATZY:           50,
		},
		catMask: 0x1FFF,
	}

	if got, want := ps.score(), uint16(243); got != want {
		t.Errorf("player scorecard scores do not match: got %d; want %d", got, want)
	}
}

/*
func TestGameGetMovesForCurrentPlayer(t *testing.T) {
	g := &game{
		curTurn: new(turn),
		scorecards: []playerScorecard{
			playerScorecard{},
		},
	}

	moves := g.getMovesForCurrentPlayer([5]die{DIE_SIX, DIE_FIVE, DIE_FOUR, DIE_THREE, DIE_ONE})
	want := []*move{}
	if diff := cmp.Diff(moves, want); diff != "" {
		t.Errorf("moves do not match (-got, +want):\n%s", diff)
	}
}
*/
