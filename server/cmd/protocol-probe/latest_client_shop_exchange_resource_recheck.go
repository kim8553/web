package main

import "path/filepath"

// recheckCurrentShopExchangeResources closes the obvious sync.Once cache gap:
// the original authority builder checks resource fingerprints only once, but
// an operator could change an INI on disk before a later 0x4F request. This
// read-only preflight fails closed on any missing/changed input. Because the
// five files are not read as an atomic snapshot, successful checks are NOT a
// sufficient transaction/TOCTOU guarantee and cannot enable item mutations.
func recheckCurrentShopExchangeResources() error {
	checks := []struct {
		path string
		allowed []string
	}{
		{defaultShopINIPath, []string{exactCurrentShopINISHA256}},
		{defaultExchangeItemINIPath, []string{exactCurrentExchangeItemSHA256}},
		{defaultConditionINIPath, []string{exactCurrentConditionINISHA256}},
		{defaultConditionFormulaINIPath, []string{exactCurrentFormulaSHA256A, exactCurrentFormulaSHA256B}},
		{filepath.Join(defaultModernShareRoot, "skill", "skill_maxlevel.ini"), []string{exactCurrentSkillMaxLevelSHA256}},
	}
	for _, check := range checks {
		if err := requireFileSHA256(check.path, check.allowed...); err != nil {
			return err
		}
	}
	return nil
}
