package entity

import (
	"math"
	"sync"
	"time"
)

const (
	LogicStateNormal   uint8 = 0
	LogicStateFighting uint8 = 1
	LogicStateSitcross uint8 = 102
	LogicStateDied     uint8 = 120
)

type Actor struct {
	mu                 sync.RWMutex
	objectID           uint32
	ownerID            uint32
	name               string
	hp                 int32
	baseMaxHP          int32
	maxHP              int32
	hitHP              int32
	mp                 int32
	baseMaxMP          int32
	maxMP              int32
	qingGong           int32
	maxQingGong        int32
	maxQingGongAdd     int32
	qingGongConsumeMul float32
	qingGongUpdatedAt  time.Time
	speedRatio         int32
	logic              uint8
}
type ActorState struct {
	ObjectID, OwnerID    uint32
	Name                 string
	HP, BaseMaxHP, MaxHP int32
	HitHP                int32
	MP, BaseMaxMP, MaxMP int32
	QingGongPoint        int32
	MaxQingGongPoint     int32
	MaxQingGongPointAdd  int32
	QingGongConsumeMul   float32
	SpeedRatio           int32
	LogicState           uint8
}

func NewActor(objectID, ownerID uint32, name string, maxHP, maxMP int32) *Actor {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	const defaultMaxQingGong = 150
	return &Actor{objectID: objectID, ownerID: ownerID, name: name, hp: maxHP, baseMaxHP: maxHP, maxHP: maxHP, hitHP: maxHP, mp: maxMP, baseMaxMP: maxMP, maxMP: maxMP, qingGong: defaultMaxQingGong, maxQingGong: defaultMaxQingGong, qingGongUpdatedAt: time.Now(), speedRatio: 100}
}
func NewPlayer(objectID, ownerID uint32, name string) *Actor {
	return NewActor(objectID, ownerID, name, 1000, 500)
}
func (a *Actor) Snapshot() ActorState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return ActorState{ObjectID: a.objectID, OwnerID: a.ownerID, Name: a.name, HP: a.hp, BaseMaxHP: a.baseMaxHP, MaxHP: a.maxHP, HitHP: a.hitHP, MP: a.mp, BaseMaxMP: a.baseMaxMP, MaxMP: a.maxMP, QingGongPoint: a.qingGong, MaxQingGongPoint: a.maxQingGong, MaxQingGongPointAdd: a.maxQingGongAdd, QingGongConsumeMul: a.qingGongConsumeMul, SpeedRatio: a.speedRatio, LogicState: a.logic}
}
func (a *Actor) ApplyDamage(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.hp == 0 {
		return false
	}
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	a.hp -= amount
	if a.hp <= 0 {
		a.hp = 0
		a.hitHP = 0
		a.logic = LogicStateDied
	} else {
		a.logic = LogicStateFighting
	}
	return true
}
func (a *Actor) ApplyHeal(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic == LogicStateDied || a.hp >= a.maxHP {
		return false
	}
	before := a.hp
	a.hp += amount
	if a.hp > a.maxHP {
		a.hp = a.maxHP
	}
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	return a.hp != before
}
func (a *Actor) RecoverBruisedHPWhileFighting(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic != LogicStateFighting || a.hp <= 0 || a.hitHP <= a.hp {
		return false
	}
	a.hitHP -= amount
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	return true
}
func (a *Actor) RestoreWhileSit(hpAmount, mpAmount int32) bool {
	if hpAmount < 0 || mpAmount < 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic != LogicStateSitcross || a.hp <= 0 {
		return false
	}
	changed := false
	if hpAmount > 0 && a.hp < a.maxHP {
		a.hp += hpAmount
		if a.hp > a.maxHP {
			a.hp = a.maxHP
		}
		if a.hitHP < a.hp {
			a.hitHP = a.hp
		}
		changed = true
	}
	if mpAmount > 0 && a.mp < a.maxMP {
		a.mp += mpAmount
		if a.mp > a.maxMP {
			a.mp = a.maxMP
		}
		changed = true
	}
	return changed
}
func (a *Actor) SetDerivedMaxima(maxHP, maxMP int32) bool {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.maxHP == maxHP && a.maxMP == maxMP {
		return false
	}
	hpWasFull := a.hp >= a.maxHP
	mpWasFull := a.mp >= a.maxMP
	a.maxHP = maxHP
	a.maxMP = maxMP
	if hpWasFull {
		a.hp = maxHP
	} else if a.hp > maxHP {
		a.hp = maxHP
	}
	if a.hitHP > maxHP {
		a.hitHP = maxHP
	}
	if mpWasFull {
		a.mp = maxMP
	} else if a.mp > maxMP {
		a.mp = maxMP
	}
	return true
}
func (a *Actor) SpendMP(cost int32) bool {
	if cost < 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mp < cost {
		return false
	}
	a.mp -= cost
	return true
}
func (a *Actor) UseQingGong(baseCost int32, now time.Time) (cost int32, ok bool) {
	if baseCost < 0 {
		return 0, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.restoreQingGongLocked(now)
	multiplier := float64(1 + a.qingGongConsumeMul)
	if multiplier < 0 {
		multiplier = 0
	}
	cost = int32(math.Ceil(float64(baseCost)*multiplier - 1e-6))
	if a.qingGong < cost {
		return cost, false
	}
	a.qingGong -= cost
	if a.qingGong < 50 {
		a.qingGong = a.maxQingGong + a.maxQingGongAdd
		a.qingGongUpdatedAt = now
	}
	return cost, true
}
func (a *Actor) RestoreQingGong(now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.restoreQingGongLocked(now)
}
func (a *Actor) restoreQingGongLocked(now time.Time) bool {
	if now.Before(a.qingGongUpdatedAt) {
		return false
	}
	points := int32(now.Sub(a.qingGongUpdatedAt) / (100 * time.Millisecond))
	if points <= 0 {
		return false
	}
	a.qingGongUpdatedAt = a.qingGongUpdatedAt.Add(time.Duration(points) * 100 * time.Millisecond)
	before := a.qingGong
	a.qingGong += points
	max := a.maxQingGong + a.maxQingGongAdd
	if a.qingGong > max {
		a.qingGong = max
	}
	return a.qingGong != before
}
func (a *Actor) SetQingGongConsumeMultiplier(multiplier float32) {
	a.mu.Lock()
	a.qingGongConsumeMul = multiplier
	a.mu.Unlock()
}
func (a *Actor) SetSpeedRatio(ratio int32) {
	if ratio < 0 {
		ratio = 0
	}
	a.mu.Lock()
	a.speedRatio = ratio
	a.mu.Unlock()
}
func (a *Actor) SetLogicState(state uint8) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logic = state
}
func (a *Actor) Revive() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hp = a.maxHP
	a.hitHP = a.maxHP
	a.mp = a.maxMP
	a.qingGong = a.maxQingGong + a.maxQingGongAdd
	a.qingGongUpdatedAt = time.Now()
	a.speedRatio = 100
	a.logic = LogicStateNormal
}
