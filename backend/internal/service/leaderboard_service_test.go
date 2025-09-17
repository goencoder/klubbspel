package service

import (
	"math"
	"testing"
)

const epsilon = 1e-6

func TestCalculateELOExpectedProbability(t *testing.T) {
	ratingA := 1200.0
	ratingB := 1000.0

	newRatingA, newRatingB := calculateELO(ratingA, ratingB, 11, 5)

	expectedA := 1 / (1 + math.Pow(10, (ratingB-ratingA)/400))
	expectedB := 1 / (1 + math.Pow(10, (ratingA-ratingB)/400))

	derivedExpectedA := 1 - (newRatingA-ratingA)/32
	derivedExpectedB := -(newRatingB - ratingB) / 32

	if math.Abs(derivedExpectedA-expectedA) > epsilon {
		t.Fatalf("expected player A win probability %.6f, got %.6f", expectedA, derivedExpectedA)
	}
	if math.Abs(derivedExpectedB-expectedB) > epsilon {
		t.Fatalf("expected player B win probability %.6f, got %.6f", expectedB, derivedExpectedB)
	}

	if math.Abs(expectedA-0.7597469) > 1e-3 {
		t.Fatalf("expected probability for player A to be close to 0.76, got %.6f", expectedA)
	}
	if math.Abs(expectedB-0.2402531) > 1e-3 {
		t.Fatalf("expected probability for player B to be close to 0.24, got %.6f", expectedB)
	}
}

func TestCalculateELORatingUpdate(t *testing.T) {
	ratingA := 1000.0
	ratingB := 1200.0

	newRatingA, newRatingB := calculateELO(ratingA, ratingB, 7, 11)

	expectedA := 1 / (1 + math.Pow(10, (ratingB-ratingA)/400))
	expectedB := 1 / (1 + math.Pow(10, (ratingA-ratingB)/400))

	const K = 32.0
	actualA := 0.0
	actualB := 1.0

	wantRatingA := ratingA + K*(actualA-expectedA)
	wantRatingB := ratingB + K*(actualB-expectedB)

	if math.Abs(newRatingA-wantRatingA) > epsilon {
		t.Fatalf("expected new rating for player A %.6f, got %.6f", wantRatingA, newRatingA)
	}
	if math.Abs(newRatingB-wantRatingB) > epsilon {
		t.Fatalf("expected new rating for player B %.6f, got %.6f", wantRatingB, newRatingB)
	}
}
