package main

import (
	"github.com/local/9yin-go-server/internal/exchangebinding"
	"github.com/local/9yin-go-server/internal/exchangegate"
	"github.com/local/9yin-go-server/internal/role"
)

// currentShopExchangeMutationGate converts the already-computed read-only
// preflight into one fail-closed transaction decision. It deliberately accepts
// final binding and persistence readiness as separate explicit inputs because
// neither may be guessed from unrelated preflight fields.
func currentShopExchangeMutationGate(preflight shopExchangeReadOnlyPreflight, binding exchangebinding.Decision, persistenceReady bool) exchangegate.Decision {
	return exchangegate.Evaluate(exchangegate.Evidence{
		ConditionSupported: preflight.ConditionSupported,
		ConditionSatisfied: preflight.ConditionSatisfied,
		PropertySupported:  preflight.PropertySupported,
		PropertySatisfied:  preflight.PropertySatisfied,
		MaterialsSatisfied: preflight.MaterialPlan.Satisfied,
		CapacitySupported:  preflight.CapacitySupported,
		CapacitySatisfied:  preflight.CapacitySatisfied,
		BindingKnown:       binding.Known(),
		PersistenceReady:   persistenceReady,
	})
}

// currentShopExchangePersistenceReady is stricter than bagStore != nil: it also
// rejects a typed-nil persistence implementation hidden inside the interface and
// refuses zero role IDs.
func currentShopExchangePersistenceReady(roleID role.RoleID, bagStore bagStoreIface) bool {
	return roleID != 0 && exchangegate.PersistenceTargetReady(bagStore)
}
