package main

import (
	"github.com/local/9yin-go-server/internal/exchangebinding"
	"github.com/local/9yin-go-server/internal/exchangegate"
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
