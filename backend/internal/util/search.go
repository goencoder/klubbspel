package util

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SearchKeys contains precomputed search keys for fuzzy matching
type SearchKeys struct {
	Normalized string   `bson:"normalized" json:"normalized"`
	Prefixes   []string `bson:"prefixes" json:"prefixes"`
	Trigrams   []string `bson:"trigrams" json:"trigrams"`
	Consonants string   `bson:"consonants" json:"consonants"`
	Phonetics  []string `bson:"phonetics" json:"phonetics"`
}

// NormalizeText converts text to lowercase and removes diacritics
func NormalizeText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)
	
	// Remove diacritics using unicode normalization
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, text)
	
	// Clean up extra spaces and special characters
	re := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	result = re.ReplaceAllString(result, " ")
	
	// Normalize spaces
	re = regexp.MustCompile(`\s+`)
	result = re.ReplaceAllString(result, " ")
	
	return strings.TrimSpace(result)
}

// isMn reports whether the rune is a nonspacing mark.
func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

// GeneratePrefixes creates prefixes up to maxLength characters for autocomplete
func GeneratePrefixes(text string, maxLength int) []string {
	normalized := NormalizeText(text)
	words := strings.Fields(normalized)
	
	var prefixes []string
	prefixSet := make(map[string]bool)
	
	for _, word := range words {
		// Generate prefixes for each word
		for i := 2; i <= len(word) && i <= maxLength; i++ {
			prefix := word[:i]
			if !prefixSet[prefix] {
				prefixes = append(prefixes, prefix)
				prefixSet[prefix] = true
			}
		}
		
		// Add full word if longer than maxLength
		if len(word) > maxLength && !prefixSet[word] {
			prefixes = append(prefixes, word)
			prefixSet[word] = true
		}
	}
	
	// Also generate combined prefixes for multi-word names
	if len(words) > 1 {
		combined := strings.Join(words, "")
		for i := 2; i <= len(combined) && i <= maxLength; i++ {
			prefix := combined[:i]
			if !prefixSet[prefix] {
				prefixes = append(prefixes, prefix)
				prefixSet[prefix] = true
			}
		}
	}
	
	return prefixes
}

// GenerateTrigrams creates character trigrams for typo tolerance
func GenerateTrigrams(text string) []string {
	normalized := NormalizeText(text)
	words := strings.Fields(normalized)
	
	var trigrams []string
	trigramSet := make(map[string]bool)
	
	for _, word := range words {
		// Pad word for boundary trigrams
		padded := "  " + word + "  "
		
		// Extract trigrams
		for i := 0; i <= len(padded)-3; i++ {
			trigram := padded[i : i+3]
			if !trigramSet[trigram] {
				trigrams = append(trigrams, trigram)
				trigramSet[trigram] = true
			}
		}
	}
	
	return trigrams
}

// GenerateConsonants creates consonant skeleton by removing vowels
func GenerateConsonants(text string) string {
	normalized := NormalizeText(text)
	
	// Define vowels (including Swedish vowels)
	vowels := "aeiouåäöyAEIOUÅÄÖY"
	
	var consonants strings.Builder
	for _, r := range normalized {
		if !strings.ContainsRune(vowels, r) && unicode.IsLetter(r) {
			consonants.WriteRune(r)
		} else if unicode.IsSpace(r) {
			consonants.WriteRune(' ')
		}
	}
	
	// Clean up extra spaces
	result := regexp.MustCompile(`\s+`).ReplaceAllString(consonants.String(), " ")
	return strings.TrimSpace(result)
}

// DoubleMetaphone implements simplified Double Metaphone algorithm
// This is a basic implementation - in production, consider using a full library
func DoubleMetaphone(text string) []string {
	normalized := NormalizeText(text)
	words := strings.Fields(normalized)
	
	var phonetics []string
	phoneticSet := make(map[string]bool)
	
	for _, word := range words {
		primary, secondary := doubleMetaphoneWord(word)
		
		if primary != "" && !phoneticSet[primary] {
			phonetics = append(phonetics, primary)
			phoneticSet[primary] = true
		}
		
		if secondary != "" && secondary != primary && !phoneticSet[secondary] {
			phonetics = append(phonetics, secondary)
			phoneticSet[secondary] = true
		}
	}
	
	return phonetics
}

// doubleMetaphoneWord processes a single word - simplified implementation
func doubleMetaphoneWord(word string) (primary, secondary string) {
	if len(word) == 0 {
		return "", ""
	}
	
	word = strings.ToUpper(word)
	
	// Very basic phonetic mapping for common Swedish/English patterns
	var primary strings.Builder
	var secondary strings.Builder
	
	i := 0
	for i < len(word) {
		switch word[i] {
		case 'B':
			primary.WriteByte('P')
			secondary.WriteByte('P')
		case 'C':
			if i+1 < len(word) && (word[i+1] == 'H' || word[i+1] == 'K') {
				primary.WriteByte('K')
				secondary.WriteByte('K')
				if word[i+1] == 'H' {
					i++ // skip H
				}
			} else {
				primary.WriteByte('K')
				secondary.WriteByte('S')
			}
		case 'D':
			primary.WriteByte('T')
			secondary.WriteByte('T')
		case 'F':
			primary.WriteByte('F')
			secondary.WriteByte('F')
		case 'G':
			primary.WriteByte('K')
			secondary.WriteByte('J')
		case 'H':
			if i == 0 || isVowel(word[i-1]) {
				primary.WriteByte('H')
				secondary.WriteByte('H')
			}
		case 'J':
			primary.WriteByte('J')
			secondary.WriteByte('Y')
		case 'K':
			primary.WriteByte('K')
			secondary.WriteByte('K')
		case 'L':
			primary.WriteByte('L')
			secondary.WriteByte('L')
		case 'M':
			primary.WriteByte('M')
			secondary.WriteByte('M')
		case 'N':
			primary.WriteByte('N')
			secondary.WriteByte('N')
		case 'P':
			if i+1 < len(word) && word[i+1] == 'H' {
				primary.WriteByte('F')
				secondary.WriteByte('F')
				i++ // skip H
			} else {
				primary.WriteByte('P')
				secondary.WriteByte('P')
			}
		case 'Q':
			primary.WriteByte('K')
			secondary.WriteByte('K')
		case 'R':
			primary.WriteByte('R')
			secondary.WriteByte('R')
		case 'S':
			if i+1 < len(word) && word[i+1] == 'H' {
				primary.WriteByte('X')
				secondary.WriteByte('X')
				i++ // skip H
			} else {
				primary.WriteByte('S')
				secondary.WriteByte('S')
			}
		case 'T':
			if i+1 < len(word) && word[i+1] == 'H' {
				primary.WriteByte('0') // TH sound
				secondary.WriteByte('T')
				i++ // skip H
			} else {
				primary.WriteByte('T')
				secondary.WriteByte('T')
			}
		case 'V':
			primary.WriteByte('F')
			secondary.WriteByte('V')
		case 'W':
			primary.WriteByte('V')
			secondary.WriteByte('W')
		case 'X':
			primary.WriteByte('K')
			secondary.WriteByte('K')
		case 'Y':
			primary.WriteByte('Y')
			secondary.WriteByte('Y')
		case 'Z':
			primary.WriteByte('S')
			secondary.WriteByte('T')
		case 'Å', 'Ä':
			primary.WriteByte('A')
			secondary.WriteByte('E')
		case 'Ö':
			primary.WriteByte('O')
			secondary.WriteByte('U')
		default:
			if isVowel(word[i]) {
				if i == 0 {
					primary.WriteByte('A')
					secondary.WriteByte('A')
				}
			}
		}
		i++
	}
	
	return primary.String(), secondary.String()
}

// isVowel checks if a character is a vowel (including Swedish vowels)
func isVowel(c byte) bool {
	vowels := "AEIOUÅÄÖY"
	return strings.ContainsRune(vowels, rune(c))
}

// GenerateSearchKeys creates all search keys for a given text
func GenerateSearchKeys(text string) SearchKeys {
	return SearchKeys{
		Normalized: NormalizeText(text),
		Prefixes:   GeneratePrefixes(text, 6), // Up to 6 characters for prefixes
		Trigrams:   GenerateTrigrams(text),
		Consonants: GenerateConsonants(text),
		Phonetics:  DoubleMetaphone(text),
	}
}

// CalculateMatchScore calculates a similarity score between 0.0 and 1.0
func CalculateMatchScore(query string, target SearchKeys) float64 {
	queryKeys := GenerateSearchKeys(query)
	
	var score float64
	var weights float64
	
	// Exact normalized match (highest weight)
	if queryKeys.Normalized == target.Normalized {
		score += 1.0 * 0.4
		weights += 0.4
	}
	
	// Prefix matching
	prefixScore := calculatePrefixScore(queryKeys.Prefixes, target.Prefixes)
	score += prefixScore * 0.25
	weights += 0.25
	
	// Trigram overlap
	trigramScore := calculateOverlapScore(queryKeys.Trigrams, target.Trigrams)
	score += trigramScore * 0.2
	weights += 0.2
	
	// Consonant skeleton matching
	if queryKeys.Consonants == target.Consonants && queryKeys.Consonants != "" {
		score += 1.0 * 0.1
		weights += 0.1
	}
	
	// Phonetic matching
	phoneticScore := calculateOverlapScore(queryKeys.Phonetics, target.Phonetics)
	score += phoneticScore * 0.05
	weights += 0.05
	
	if weights == 0 {
		return 0.0
	}
	
	return score / weights
}

// calculatePrefixScore calculates score based on prefix matching
func calculatePrefixScore(queryPrefixes, targetPrefixes []string) float64 {
	if len(queryPrefixes) == 0 || len(targetPrefixes) == 0 {
		return 0.0
	}
	
	var matches int
	for _, qp := range queryPrefixes {
		for _, tp := range targetPrefixes {
			if strings.HasPrefix(tp, qp) || strings.HasPrefix(qp, tp) {
				matches++
				break
			}
		}
	}
	
	return float64(matches) / float64(len(queryPrefixes))
}

// calculateOverlapScore calculates Jaccard similarity (intersection/union)
func calculateOverlapScore(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0.0
	}
	
	setA := make(map[string]bool)
	for _, item := range a {
		setA[item] = true
	}
	
	setB := make(map[string]bool)
	for _, item := range b {
		setB[item] = true
	}
	
	intersection := 0
	for item := range setA {
		if setB[item] {
			intersection++
		}
	}
	
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}
	
	return float64(intersection) / float64(union)
}