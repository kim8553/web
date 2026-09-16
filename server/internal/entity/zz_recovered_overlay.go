package entity

func (a *Actor) ApplyMPRestore(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic == LogicStateDied || a.mp >= a.maxMP {
		return false
	}
	before := a.mp
	a.mp += amount
	if a.mp > a.maxMP {
		a.mp = a.maxMP
	}
	return a.mp != before
}
