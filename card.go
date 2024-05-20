package main

import (
	"fmt"
	"math/rand"
	"slices"
)

const (
	Diamonds     Suite = 1 << iota
	Clubs        Suite = 1 << iota
	Hearts       Suite = 1 << iota
	Spades       Suite = 1 << iota
	InvalidSuite Suite = 0
)

const (
	InvalidPokerHand PokerHand = iota
	HighCard         PokerHand = iota
	OnePair          PokerHand = iota
	ThreeOfAKind     PokerHand = iota
	Straight         PokerHand = iota
	Flush            PokerHand = iota
	FullHouse        PokerHand = iota
	FourOfAKind      PokerHand = iota
	StraightFlush    PokerHand = iota
	FiveOfAKind      PokerHand = iota
)

const (
	Three       Rank = 1 << iota
	Four        Rank = 1 << iota
	Five        Rank = 1 << iota
	Six         Rank = 1 << iota
	Seven       Rank = 1 << iota
	Eight       Rank = 1 << iota
	Nine        Rank = 1 << iota
	Ten         Rank = 1 << iota
	Jack        Rank = 1 << iota
	Queen       Rank = 1 << iota
	King        Rank = 1 << iota
	Ace         Rank = 1 << iota
	Two         Rank = 1 << iota
	InvalidRank Rank = 0
)

const (
	ThreeStraight uint = 31
	FourStraight  uint = 62
	FiveStraight  uint = 124
	SixStraight   uint = 248
	SevenStraight uint = 496
	EightStraight uint = 992
	NineStraight  uint = 1984
	TenStraight   uint = 3968
	TwoStraight   uint = 4111
	AceStraight   uint = 6151
)

type Card = uint
type Rank = uint
type Suite = uint
type PokerHand = uint
type PokerHandRank = uint

func NewCard(r Rank, s Suite) Card {
	return r<<8 | s
}

// Byte shift 13 to the left to get suite
func GetSuite(c Card) Suite {
	// bitmask first 4 bits of uint to get LSBs (suite)
	suite := c & 15
	fmt.Println(c)
	fmt.Printf("suite %d\n", suite)
	if suite == Diamonds || suite == Hearts || suite == Spades || suite == Clubs {
		return suite
	}
	return 0
}

// Byte shift 8 to the right to get rank
// https://stackoverflow.com/questions/13420241/check-if-only-one-single-bit-is-set-within-an-integer-whatever-its-position
func GetRank(c Card) uint {
	rank := c >> 8
	// Check if power of 2
	if rank != 0 && ((rank & (rank - 1)) != 0) {
		return 0
	}
	return rank
}

func getPokerHand(c []Card) (PokerHand, PokerHandRank) {
	switch len(c) {
	case 1:
		return HighCard, c[0]
	case 2, 3, 4:
		return getNOfAKind(c)
	case 5:
		// flush, royal flush, straight, double pair, fullhouse, five of a kind
		return get5CardPokerHand(c)

	}
	return InvalidPokerHand, 0
}

func get5CardPokerHand(c []Card) (PokerHand, PokerHandRank) {
	ph, phr := getNOfAKind(c)
	if ph == FiveOfAKind {
		return ph, phr
	}
	ph, phr = getStraightFlush(c)
	if ph == StraightFlush {
		return ph, phr
	}
	ph, phr = getFullHouse(c)
	if ph == FullHouse {
		return ph, phr
	}
	ph, phr = getStraight(c)
	if ph == Straight {
		return ph, phr
	}
	return InvalidPokerHand, InvalidRank
}

func getFullHouse(c []Card) (PokerHand, PokerHandRank) {
	m := make(map[uint]uint)
	for _, card := range c {
		val, ok := m[GetRank(card)]
		if ok {
			m[GetRank(card)] = val + 1
		} else {
			m[GetRank(card)] = 1
		}
	}
	triple := false
	double := false
	var tripleRank uint
	for k, v := range m {
		if v == 3 {
			triple = true
			tripleRank = k
		}
		if v == 2 {
			double = true
		}
	}

	if triple && double {
		return FullHouse, tripleRank
	}
	return InvalidPokerHand, InvalidRank
}

func getStraightFlush(c []Card) (PokerHand, PokerHandRank) {
	// if is straight and is flush return StraightFlush + rank
	// rank is determined by suite and top card
	// eg: top card = Queen suite = spades
	// rank = top card shift 8, xor suite
	if isFlush(c) && isStraight(c) {
		slices.Sort(c)
		slices.Reverse(c)
		rank := GetRank(c[0])
		return StraightFlush, rank<<8 | GetSuite(c[0])
	}
	return InvalidPokerHand, InvalidRank
}

func getFlush(c []Card) (PokerHand, PokerHandRank) {
	if isFlush(c) {
		slices.Sort(c)
		slices.Reverse(c)
		rank := GetRank(c[0])
		suite := GetSuite(c[0])
		return Flush, NewCard(rank, suite)
	}
	return InvalidPokerHand, InvalidRank
}

func getStraight(c []Card) (PokerHand, PokerHandRank) {
	if isStraight(c) {
		maxCard := slices.Max(c)
		return Straight, NewCard(GetRank(maxCard), GetSuite(maxCard))
	}
	return InvalidPokerHand, InvalidRank
}

func isFlush(c []Card) bool {
	target := GetSuite(c[0])
	for _, card := range c {
		if GetSuite(card) != target {
			return false
		}
	}
	return true
}

func isStraight(c []Card) bool {
	var straight uint
	for i, card := range c {
		fmt.Println(card)
		if i == 0 {
			straight = card
		} else {
			straight = straight | card
		}
	}

	fmt.Println(straight)
	straight = straight >> 8
	fmt.Println(straight)
	switch straight {
	case ThreeStraight, FourStraight, FiveStraight, SixStraight, SevenStraight, EightStraight, NineStraight, TenStraight, TwoStraight, AceStraight:
		return true
	}
	return false
}

// flush -> check if all suites are same as suite of 1st card
// royal flush -> flush + all cards are royals

func getNOfAKind(cards []Card) (PokerHand, PokerHandRank) {
	rank := cards[0]
	if len(cards) == 1 {
		return HighCard, rank
	}
	ph := true
	for _, c := range cards {
		ph = ph && GetRank(c) == rank
	}
	if ph {
		switch len(cards) {
		case 2:
			return OnePair, rank
		case 3:
			return ThreeOfAKind, rank
		case 4:
			return FourOfAKind, rank
		case 5:
			return FiveOfAKind, rank
		}
	}
	return InvalidPokerHand, 0
}

func shuffleDeck() []Card {
	var deck []Card
	for j := 0; j < 13; j++ {
		for k := 0; k < 4; k++ {
			deck = append(deck, NewCard(1<<j, 1<<k))
		}
	}
	// Fisher Yates shuffle
	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func dealCards(deck []Card) []Card {
	var x []Card
	x, deck = deck[0:12], deck[1:]
	return x
}

func cardsToString(cards []Card) []string {
	var x []string
	for _, j := range cards {
		suite := GetSuite(j)
		var s string
		if suite == Diamonds {
			s = "♦"
		} else if suite == Hearts {
			s = "♥"
		} else if suite == Clubs {
			s = "♣"
		} else if suite == Spades {
			s = "♠"
		}
		rank := GetRank(j)
		var r string
		if rank == Three {
			r = "3"
		} else if rank == Four {
			r = "4"
		} else if rank == Five {
			r = "5"
		} else if rank == Six {
			r = "6"
		} else if rank == Seven {
			r = "7"
		} else if rank == Eight {
			r = "8"
		} else if rank == Nine {
			r = "9"
		} else if rank == Ten {
			r = "10"
		} else if rank == Jack {
			r = "J"
		} else if rank == Queen {
			r = "Q"
		} else if rank == King {
			r = "K"
		} else if rank == Ace {
			r = "A"
		}
		x = append(x, s+r)
	}
	return x
}
