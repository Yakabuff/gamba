package main

import (
	"fmt"
	"testing"
)

func TestNewCard(t *testing.T) {
	c := NewCard(Three, Spades)
	suite := GetSuite(c)
	rank := GetRank(c)
	if suite != Spades || rank != Three {
		t.Errorf("Invalid Card")
	}
}

func TestStraight(t *testing.T) {
	c := NewCard(Three, Spades)
	c1 := NewCard(Four, Spades)
	c2 := NewCard(Five, Spades)
	c3 := NewCard(Six, Spades)
	c4 := NewCard(Seven, Spades)
	s := []Card{c, c1, c2, c3, c4}
	res := isStraight(s)
	if res == false {
		t.Errorf("Not straight")
	}
	x, y := getStraight(s)
	if x != Straight {
		t.Errorf("Expected straight")
	}
	if y != NewCard(Seven, Spades) {
		t.Errorf("Expected 7 spades as max")
	}
}

func TestFlush(t *testing.T) {
	c := NewCard(Three, Spades)
	c1 := NewCard(Four, Spades)
	c2 := NewCard(Five, Spades)
	c3 := NewCard(Six, Spades)
	c4 := NewCard(Seven, Spades)
	s := []Card{c, c1, c2, c3, c4}
	res := isFlush(s)
	if res == false {
		t.Errorf("Expected flush")
	}
	x, y := getFlush(s)
	if x != Flush {
		t.Errorf("expected flush")
	}
	if y != NewCard(Seven, Spades) {
		t.Errorf("expected top card as seven of spades")
	}
}

func TestFullHouse(t *testing.T) {

	c := NewCard(Three, Spades)
	c1 := NewCard(Three, Clubs)
	c2 := NewCard(Three, Diamonds)
	c3 := NewCard(Six, Spades)
	c4 := NewCard(Six, Clubs)
	s := []Card{c, c1, c2, c3, c4}
	x, y := getFullHouse(s)

	if x != FullHouse {
		t.Errorf("Expected full house. Got %d", x)
	}
	if y != Three {
		t.Errorf("Expected rank of 3 (1). Got %d", y)
	}
}

func TestShuffleDeck(t *testing.T) {
	res := shuffleDeck()
	fmt.Println(res)
	if len(res) != 52 {
		t.Errorf("invalid shuffle")
	}
	s := GetSuite(res[0])
	r := GetRank(res[0])

	if s != Diamonds && s != Clubs && s != Hearts && s != Spades {
		t.Errorf("invalid suite")
	}
	if r != Three && r != Four && r != Five && r != Six && r != Seven && r != Eight && r != Nine && r != Ten && r != Jack && r != Queen && r != King && r != Ace && r != Two {
		t.Errorf("invalid rank")
	}
}
