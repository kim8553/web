package skillreplace

import (
	"strconv"
	"strings"
)

type Rule struct {
	BaseID        string
	ConditionID   int32
	ReplacementID string
	Flag          int32
}

func Parse(baseID, key, value string) (Rule, bool) {
	condition, err := strconv.ParseInt(strings.TrimSpace(key), 10, 32)
	if err != nil {
		return Rule{}, false
	}
	parts := strings.Split(value, ",")
	replacementID := strings.TrimSpace(parts[0])
	if replacementID == "" {
		return Rule{}, false
	}
	var flag int32
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		if parseErr != nil {
			return Rule{}, false
		}
		flag = int32(parsed)
	}
	return Rule{
		BaseID:        strings.TrimSpace(baseID),
		ConditionID:   int32(condition),
		ReplacementID: replacementID,
		Flag:          flag,
	}, true
}

// IndexBases maps an authored replacement target back to its source skill.
// It deliberately does not evaluate ConditionID or Flag and therefore never
// decides when a replacement should occur. Ambiguous targets fail closed.
func IndexBases(rules []Rule) map[string]string {
	result := make(map[string]string)
	conflicts := make(map[string]struct{})
	for _, rule := range rules {
		key := strings.ToLower(strings.TrimSpace(rule.ReplacementID))
		if key == "" || strings.TrimSpace(rule.BaseID) == "" {
			continue
		}
		if existing, ok := result[key]; ok && !strings.EqualFold(existing, rule.BaseID) {
			delete(result, key)
			conflicts[key] = struct{}{}
			continue
		}
		if _, conflict := conflicts[key]; conflict {
			continue
		}
		result[key] = strings.TrimSpace(rule.BaseID)
	}
	return result
}
