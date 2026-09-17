package main

// currentShopExchangeConditionObservation describes the existing current-client
// 508 root evaluations at one instant. It is diagnostic only: the purchase
// semantics of Condition versus Condition2, a coherent player-state snapshot,
// inventory cost, binding, and durable commit remain unproven. In particular,
// neither AllSupported nor zero UnsatisfiedOrUnresolved proves eligibility.
type currentShopExchangeConditionObservation struct {
	RootCount                 int
	AllSupported              bool
	UnsatisfiedOrUnresolved  int
}

// summarizeCurrentShopExchangeConditionResults counts the already-evaluated
// root booleans without guessing their native conjunction/alternative rules.
// An unsupported root is already false in evaluateExchangeConditionDetails.
// Never use this summary to authorize inventory mutation or grant.
func summarizeCurrentShopExchangeConditionResults(satisfied []bool, allSupported bool) currentShopExchangeConditionObservation {
	observation := currentShopExchangeConditionObservation{
		RootCount: len(satisfied), AllSupported: allSupported,
	}
	for _, value := range satisfied {
		if !value {
			observation.UnsatisfiedOrUnresolved++
		}
	}
	return observation
}
