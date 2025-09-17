package service

import (
	"math"
	"testing"
)

const (
	epsilon = 1e-6
	// expectedProbabilityDiff200Favorite represents the favourite's win probability when leading by 200 Elo points.
	// Derived from the standard Elo expected score formula: 1 / (1 + 10^(-200/400)) ≈ 0.759746926647958.
	expectedProbabilityDiff200Favorite = 0.759746926647958
	// expectedProbabilityDiff200Underdog complements the favourite's probability (≈ 0.240253073352042).
	expectedProbabilityDiff200Underdog = 1 - expectedProbabilityDiff200Favorite
	eloKFactor                         = 32.0
)

func TestCalculateELOExpectedProbability(t *testing.T) {
	ratingFavorite := 1200.0
	ratingUnderdog := 1000.0

	probabilityFavorite := 1 / (1 + math.Pow(10, (ratingUnderdog-ratingFavorite)/400))
	probabilityUnderdog := 1 / (1 + math.Pow(10, (ratingFavorite-ratingUnderdog)/400))

	if math.Abs(probabilityFavorite-expectedProbabilityDiff200Favorite) > epsilon {
		t.Fatalf("expected favourite win probability %.6f, got %.6f", expectedProbabilityDiff200Favorite, probabilityFavorite)
	}
	if math.Abs(probabilityUnderdog-expectedProbabilityDiff200Underdog) > epsilon {
		t.Fatalf("expected underdog win probability %.6f, got %.6f", expectedProbabilityDiff200Underdog, probabilityUnderdog)
	}
}

func TestCalculateELORatingUpdate(t *testing.T) {
	ratingFavorite := 1200.0
	ratingUnderdog := 1000.0

	t.Run("favourite wins", func(t *testing.T) {
		newRatingFavorite, newRatingUnderdog := calculateELO(ratingFavorite, ratingUnderdog, 11, 5)

		wantRatingFavorite := ratingFavorite + eloKFactor*(1-expectedProbabilityDiff200Favorite)
		wantRatingUnderdog := ratingUnderdog - eloKFactor*expectedProbabilityDiff200Underdog

		if math.Abs(newRatingFavorite-wantRatingFavorite) > epsilon {
			t.Fatalf("expected favourite rating %.6f, got %.6f", wantRatingFavorite, newRatingFavorite)
		}
		if math.Abs(newRatingUnderdog-wantRatingUnderdog) > epsilon {
			t.Fatalf("expected underdog rating %.6f, got %.6f", wantRatingUnderdog, newRatingUnderdog)
		}
	})

	t.Run("underdog wins", func(t *testing.T) {
		newRatingFavorite, newRatingUnderdog := calculateELO(ratingFavorite, ratingUnderdog, 5, 11)

		wantRatingFavorite := ratingFavorite - eloKFactor*expectedProbabilityDiff200Favorite
		wantRatingUnderdog := ratingUnderdog + eloKFactor*(1-expectedProbabilityDiff200Underdog)

		if math.Abs(newRatingFavorite-wantRatingFavorite) > epsilon {
			t.Fatalf("expected favourite rating %.6f, got %.6f", wantRatingFavorite, newRatingFavorite)
		}
		if math.Abs(newRatingUnderdog-wantRatingUnderdog) > epsilon {
			t.Fatalf("expected underdog rating %.6f, got %.6f", wantRatingUnderdog, newRatingUnderdog)
		}
	})
}
