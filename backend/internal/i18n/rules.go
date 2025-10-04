package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed rules_en.json
var rulesEN []byte

//go:embed rules_sv.json
var rulesSV []byte

type RuleExample struct {
	Scenario string `json:"scenario"`
	Outcome  string `json:"outcome"`
}

type RulesContent struct {
	Title    string        `json:"title"`
	Summary  string        `json:"summary"`
	Rules    []string      `json:"rules"`
	Examples []RuleExample `json:"examples"`
}

type RulesData struct {
	FreePlay         RulesContent `json:"free_play"`
	LadderClassic    RulesContent `json:"ladder_classic"`
	LadderAggressive RulesContent `json:"ladder_aggressive"`
}

var rulesCache = make(map[string]*RulesData)

// LoadRules loads rules for the specified locale
func LoadRules(locale string) (*RulesData, error) {
	// Check cache first
	if cached, ok := rulesCache[locale]; ok {
		return cached, nil
	}

	var data []byte
	switch locale {
	case "sv":
		data = rulesSV
	case "en":
		data = rulesEN
	default:
		// Default to Swedish
		data = rulesSV
	}

	var rules RulesData
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules for locale %s: %w", locale, err)
	}

	// Cache the loaded rules
	rulesCache[locale] = &rules

	return &rules, nil
}

// GetFreePlayRules returns the rules for free play format
func GetFreePlayRules(locale string) (*RulesContent, error) {
	rules, err := LoadRules(locale)
	if err != nil {
		return nil, err
	}
	return &rules.FreePlay, nil
}

// GetLadderRules returns the rules for ladder format
func GetLadderRules(locale string, aggressive bool) (*RulesContent, error) {
	rules, err := LoadRules(locale)
	if err != nil {
		return nil, err
	}

	if aggressive {
		return &rules.LadderAggressive, nil
	}
	return &rules.LadderClassic, nil
}
