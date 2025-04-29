package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func DieCatTest(t *testing.T) {
	for _, tt := range []struct {
		i die
		w category
	}{
		{DIE_ONE, CAT_ONES},
		{DIE_TWO, CAT_TWOS},
		{DIE_SIX, CAT_SIXES},
	} {
		got := dieCat(tt.i)
		if got != tt.w {
			t.Errorf("for dieCat(%s) got %s; want %s", tt.i, got, tt.w)
		}
	}
}

func TestGetRollScoreForCategory(t *testing.T) {
	opts := []cmp.Option{
		cmp.AllowUnexported(scoreData{}),
	}
	for i, tt := range []struct {
		r [5]die
		c category
		w scoreData
	}{
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_THREE},
			c: CAT_SMALL_STRAIGHT,
			w: scoreData{30, [][]int{{2, 1, 0, 3}, {2, 1, 4, 3}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_ONES,
		},
		{
			r: [5]die{DIE_ONE, DIE_ONE, DIE_TWO, DIE_ONE, DIE_ONE},
			c: CAT_ONES,
			w: scoreData{4, [][]int{{0, 1, 3, 4}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_TWOS,
			w: scoreData{6, [][]int{{0, 1, 2}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_THREES,
			w: scoreData{3, [][]int{{3}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_FOURS,
			w: scoreData{4, [][]int{{4}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_SIX, DIE_FIVE, DIE_FIVE, DIE_FOUR},
			c: CAT_FIVES,
			w: scoreData{10, [][]int{{2, 3}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_SIX, DIE_FIVE, DIE_FIVE, DIE_FOUR},
			c: CAT_SIXES,
			w: scoreData{6, [][]int{{1}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_THREE, DIE_FOUR},
			c: CAT_THREE_OF_A_KIND,
			w: scoreData{13, [][]int{{0, 1, 2}}},
		},
		{
			r: [5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO, DIE_FOUR},
			c: CAT_THREE_OF_A_KIND,
			w: scoreData{12, [][]int{{0, 1, 2, 3}}},
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_THREE, DIE_TWO, DIE_FOUR},
			c: CAT_THREE_OF_A_KIND,
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_SIX, DIE_ONE, DIE_ONE},
			c: CAT_FOUR_OF_A_KIND,
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_ONE, DIE_ONE, DIE_ONE},
			c: CAT_FOUR_OF_A_KIND,
			w: scoreData{6, [][]int{{0, 2, 3, 4}}},
		},
		{
			r: [5]die{DIE_ONE, DIE_TWO, DIE_ONE, DIE_ONE, DIE_TWO},
			c: CAT_FULL_HOUSE,
			w: scoreData{25, [][]int{{0, 1, 2, 3, 4}}},
		},
		{
			r: [5]die{DIE_FIVE, DIE_TWO, DIE_ONE, DIE_ONE, DIE_TWO},
			c: CAT_FULL_HOUSE,
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_SMALL_STRAIGHT,
			w: scoreData{30, [][]int{{2, 1, 0, 3}, {1, 0, 3, 4}}},
		},
		{
			r: [5]die{DIE_THREE, DIE_THREE, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_SMALL_STRAIGHT,
		},
		{
			r: [5]die{DIE_SIX, DIE_TWO, DIE_THREE, DIE_FOUR, DIE_FIVE},
			c: CAT_LARGE_STRAIGHT,
			w: scoreData{40, [][]int{{1, 2, 3, 4, 0}}},
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_LARGE_STRAIGHT,
			w: scoreData{40, [][]int{{2, 1, 0, 3, 4}}},
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_FOUR, DIE_FOUR, DIE_FIVE},
			c: CAT_LARGE_STRAIGHT,
		},
		{
			r: [5]die{DIE_THREE, DIE_TWO, DIE_ONE, DIE_FOUR, DIE_FIVE},
			c: CAT_CHANCE,
			w: scoreData{15, [][]int{{0, 1, 2, 3, 4}}},
		},
		{
			r: [5]die{DIE_THREE, DIE_THREE, DIE_THREE, DIE_THREE, DIE_THREE},
			c: CAT_YATZY,
			w: scoreData{50, [][]int{{0, 1, 2, 3, 4}}},
		},
		{
			r: [5]die{DIE_FOUR, DIE_THREE, DIE_THREE, DIE_THREE, DIE_FOUR},
			c: CAT_YATZY,
		},
		{
			r: [5]die{DIE_THREE, DIE_THREE, DIE_THREE, DIE_THREE, DIE_FOUR},
			c: CAT_YATZY,
		},
	} {
		r2 := newRollV2_2(tt.r)
		got := scoresByRoll[r2][tt.c]
		if diff := cmp.Diff(got, tt.w, opts...); diff != "" {
			t.Errorf("roll score does not match (-got, +want): input[%d] %+v, %d;\n%s", i, r2.dice(), r2, diff)
		}
	}
}

func TestPlayerScorecardUpdate(t *testing.T) {
	opts := []cmp.Option{
		cmp.AllowUnexported(playerScorecard{}),
	}
	var ps playerScorecard

	t.Run("seriesA", func(t *testing.T) {
		// 1. 1, 2, 3, 4, 5 - large straight
		got := ps.update(newRollV2_2([5]die{DIE_ONE, DIE_TWO, DIE_THREE, DIE_FOUR, DIE_FIVE}), CAT_LARGE_STRAIGHT)
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
		got = got.update(newRollV2_2([5]die{DIE_ONE, DIE_ONE, DIE_ONE, DIE_ONE, DIE_ONE}), CAT_ONES)
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
		got = got.update(newRollV2_2([5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO}), CAT_YATZY)
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

		// 4. All six - cannot use joker on full house (must take six) + see 100 bonus
		got = got.update(newRollV2_2([5]die{DIE_SIX, DIE_SIX, DIE_SIX, DIE_SIX, DIE_SIX}), CAT_FULL_HOUSE)
		want = playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_ONES:           5,
				CAT_FULL_HOUSE:     0,
				CAT_LARGE_STRAIGHT: 40,
				CAT_YATZY:          150,
			},
			catMask: 5377,
		}
		if diff := cmp.Diff(got, want, opts...); diff != "" {
			t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
		}
	})

	t.Run("seriesB", func(t *testing.T) {
		// 1. 1, 1, 1, 1, 1
		got := ps.update(newRollV2_2([5]die{DIE_ONE, DIE_ONE, DIE_ONE, DIE_ONE, DIE_TWO}), CAT_ONES)
		if diff := cmp.Diff(ps, playerScorecard{}, opts...); diff != "" {
			t.Errorf("original scorecard was modified (-got, +want):\n%s", diff)
		}
		want := playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_ONES: 4,
			},
			catMask: 1,
		}
		if diff := cmp.Diff(got, want, opts...); diff != "" {
			t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
		}

		// 2. 1, 3, 4, 5, 6 - take yatzy
		got = got.update(newRollV2_2([5]die{DIE_ONE, DIE_TWO, DIE_THREE, DIE_FIVE, DIE_SIX}), CAT_YATZY)
		want = playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_ONES:  4,
				CAT_YATZY: 0,
			},
			catMask: 4097,
		}
		if diff := cmp.Diff(got, want, opts...); diff != "" {
			t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
		}

		// 3. 2, 2, 2, 2, 2 - take large straight
		got = got.update(newRollV2_2([5]die{DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO, DIE_TWO}), CAT_LARGE_STRAIGHT)
		want = playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_ONES:           4,
				CAT_LARGE_STRAIGHT: 0,
				CAT_YATZY:          0,
			},
			catMask: 5121,
		}
		if diff := cmp.Diff(got, want, opts...); diff != "" {
			t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
		}

		// 4. All ones - use joker on full house
		got = got.update(newRollV2_2([5]die{DIE_ONE, DIE_ONE, DIE_ONE, DIE_ONE, DIE_ONE}), CAT_FULL_HOUSE)
		want = playerScorecard{
			scoresByCategory: [13]uint16{
				CAT_ONES:           4,
				CAT_FULL_HOUSE:     25,
				CAT_LARGE_STRAIGHT: 0,
				CAT_YATZY:          0,
			},
			catMask: 5377,
		}
		if diff := cmp.Diff(got, want, opts...); diff != "" {
			t.Errorf("scorecards do not match (-got, +want):\n%s", diff)
		}
	})
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
	got := ps.getNext(newRollV2_2([5]die{DIE_ONE, DIE_TWO, DIE_THREE, DIE_FOUR, DIE_FIVE}))
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

func TestGameGetMovesForCurrentPlayer(t *testing.T) {
	g := &game{
		curTurn: new(turn),
		scorecards: []playerScorecard{
			playerScorecard{},
		},
	}

	moves := g.getMovesForCurrentPlayer(newRollV2_2([5]die{DIE_SIX, DIE_FIVE, DIE_FOUR, DIE_THREE, DIE_ONE}))
	var movesStr []string
	for _, m := range moves {
		movesStr = append(movesStr, m.String())
	}

	want := []string{
		"select ones for 1",
		"select twos for 0",
		"select threes for 3",
		"select fours for 4",
		"select fives for 5",
		"select sixes for 6",
		"select three of a kind for 0",
		"select four of a kind for 0",
		"select full house for 0",
		"select small straight for 30",
		"select large straight for 0",
		"select chance for 19",
		"select yatzy for 0",
		"reroll holding six",
		"reroll holding five",
		"reroll holding six,five",
		"reroll holding four",
		"reroll holding six,four",
		"reroll holding five,four",
		"reroll holding six,five,four",
		"reroll holding three",
		"reroll holding six,three",
		"reroll holding five,three",
		"reroll holding six,five,three",
		"reroll holding four,three",
		"reroll holding six,four,three",
		"reroll holding five,four,three",
		"reroll holding six,five,four,three",
		"reroll holding one",
		"reroll holding six,one",
		"reroll holding five,one",
		"reroll holding six,five,one",
		"reroll holding four,one",
		"reroll holding six,four,one",
		"reroll holding five,four,one",
		"reroll holding six,five,four,one",
		"reroll holding three,one",
		"reroll holding six,three,one",
		"reroll holding five,three,one",
		"reroll holding six,five,three,one",
		"reroll holding four,three,one",
		"reroll holding six,four,three,one",
		"reroll holding five,four,three,one",
	}
	if diff := cmp.Diff(movesStr, want); diff != "" {
		t.Errorf("moves do not match (-got, +want):\n%s", diff)
	}
}

func TestMaxTheoreticalScore(t *testing.T) {
	for _, tt := range []struct {
		ps   playerScorecard
		want uint16
	}{
		{
			ps: playerScorecard{
				scoresByCategory: [13]uint16{
					CAT_TWOS:   4,
					CAT_THREES: 6,
					CAT_FOURS:  12,
					CAT_FIVES:  15,
					CAT_SIXES:  18,
				},
				catMask: 0xFFE,
			},
			want: 210,
		},
		{
			ps: playerScorecard{
				scoresByCategory: [13]uint16{
					CAT_THREES: 6,
					CAT_FOURS:  12,
					CAT_FIVES:  15,
					CAT_SIXES:  18,
				},
				catMask: 0xFFC,
			},
			want: 351,
		},
		{
			ps: playerScorecard{
				scoresByCategory: [13]uint16{
					CAT_THREES: 6,
					CAT_FOURS:  12,
					CAT_FIVES:  15,
					CAT_SIXES:  18,
					CAT_YATZY:  50,
				},
				catMask: 0x1FFC,
			},
			want: 251,
		},
		{
			ps: playerScorecard{
				catMask: 0x1FFF,
			},
			want: 0,
		},
		{
			ps: playerScorecard{
				catMask: 0x1FFE,
			},
			want: 5,
		},
		{
			ps: playerScorecard{
				catMask: 0x1FFC,
			},
			want: 15,
		},
		{
			ps: playerScorecard{
				catMask: 0x17FF,
			},
			want: 30,
		},
		{
			ps: playerScorecard{
				catMask: 0xFFF,
			},
			want: 50,
		},
	} {
		got := tt.ps.maxTheoreticalScore()
		if got != tt.want {
			t.Errorf("ps:\n%s\ngot %d; want %d", tt.ps.pretty(), got, tt.want)
		}
	}
}
