package util

import (
	"strings"
	"unicode"
)

// StringSimilarity calculates a similarity score between two strings (0.0 to 1.0)
// Uses a combination of different algorithms for best name matching
func StringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Normalize strings for comparison
	norm1 := normalizeForComparison(s1)
	norm2 := normalizeForComparison(s2)

	if norm1 == norm2 {
		return 1.0
	}

	if len(norm1) == 0 || len(norm2) == 0 {
		return 0.0
	}

	// Calculate multiple similarity metrics and combine them
	levenshtein := levenshteinSimilarity(norm1, norm2)
	jaro := jaroSimilarity(norm1, norm2)

	// Weight the different algorithms
	// Jaro is better for names, Levenshtein catches typos
	combined := 0.7*jaro + 0.3*levenshtein

	return combined
}

// normalizeForComparison prepares strings for comparison
func normalizeForComparison(s string) string {
	// Convert to lowercase and remove extra whitespace
	s = strings.ToLower(strings.TrimSpace(s))

	// Remove non-letter characters except spaces
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	// Normalize multiple spaces to single spaces
	normalized := strings.Join(strings.Fields(result.String()), " ")

	return normalized
}

// levenshteinSimilarity calculates similarity based on Levenshtein distance
func levenshteinSimilarity(s1, s2 string) float64 {
	distance := levenshteinDistance(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(distance)/maxLen
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}

	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// jaroSimilarity calculates the Jaro similarity between two strings
func jaroSimilarity(s1, s2 string) float64 {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	if len1 == 0 && len2 == 0 {
		return 1.0
	}
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	// Maximum allowed distance for matches
	matchWindow := max(len1, len2)/2 - 1
	if matchWindow < 0 {
		matchWindow = 0
	}

	s1Matches := make([]bool, len1)
	s2Matches := make([]bool, len2)

	matches := 0

	// Find matches
	for i := 0; i < len1; i++ {
		start := max(0, i-matchWindow)
		end := min(i+matchWindow+1, len2)

		for j := start; j < end; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	// Count transpositions
	transpositions := 0
	k := 0
	for i := 0; i < len1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	jaro := (float64(matches)/float64(len1) +
		float64(matches)/float64(len2) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0

	return jaro
}

// Helper functions for Go < 1.21 compatibility
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min3(a, b, c int) int {
	return min(min(a, b), c)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
