package main

import (
	"encoding/binary"
	"fmt"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/world"
	worldcore "github.com/local/9yin-go-server/internal/world"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	playerObjectID                uint32 = 0x11000001
	playerOwnerID                 uint32 = 0x007A5C0B
	propLastObject                uint16 = 100
	propNeigongPKStatus           uint16 = 208
	propDead                      uint16 = 209
	propCantUseSkill              uint16 = 210
	propCurSkillID                uint16 = 211
	propPauseTime                 uint16 = 212
	propHPHeartSpeed              uint16 = 213
	propHPHeartSpeedAdd           uint16 = 214
	propMPHeartSpeed              uint16 = 215
	propMPHeartSpeedAdd           uint16 = 216
	propHPUpSpeed                 uint16 = 217
	propHPUpSpeedAdd              uint16 = 218
	propMPUpSpeed                 uint16 = 219
	propMPUpSpeedAdd              uint16 = 220
	propCurSkillEffectID          uint16 = 221
	propCurSkillLevel             uint16 = 222
	propCurSkillTarget            uint16 = 223
	propModifySkillLockTime       uint16 = 224
	propSkillCanUse               uint16 = 225
	propForce                     uint16 = 226
	propNewSchool                 uint16 = 227
	bufferListPropertyIndex       uint16 = 143
	bufferInfoFirstPropertyIndex  uint16 = 629
	bufferInfoSlotCount           uint16 = 24
	redSuperArmorBuffSlot         uint16 = 1
	waterDriftBuffSlot            uint16 = 2
	waterJumpBuffSlot             uint16 = 3
	innerPowerBuffSlot            uint16 = 4
	yiHuaSkillBuffSlot            uint16 = 5
	yiHuaSkill05BuffSlot          uint16 = 6
	taiJiStanceBuffSlot           uint16 = 7
	taiJiSkill05BuffSlot          uint16 = 8
	taiJiSkill07BuffSlot          uint16 = 9
	yihuaPassiveBuffSlot          uint16 = 10
	redSuperArmorStaticData       uint32 = 8682
	waterDriftStaticData          uint32 = 216
	waterJumpStaticData           uint32 = 5786
	innerPowerBuffLease                  = 24 * time.Hour
	maxPlayerSP                   int32  = 100
	sitcrossFullRecoverySeconds   int32  = 10
	sitcrossHeartMillis           int32  = 1000
	combatBruiseRecoveryPerSecond int32  = 10
)

var nativeBuffSlots = func() []uint16 {
	slots := make([]uint16, bufferInfoSlotCount)
	for index := range slots {
		slots[index] = uint16(index + 1)
	}
	return slots
}()

func bufferInfoPropertyIndex(slot uint16) uint16 {
	if slot < 1 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return bufferInfoFirstPropertyIndex + slot - 1
}
func bufferInfoPropertyName(slot uint16) string {
	if slot < 1 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return fmt.Sprintf("BufferInfo%d", slot)
}

type playerActor struct {
	actor                 *entity.Actor
	name                  string
	silver                int32
	gold                  int32
	silverCard            int32
	silverTicket          int32
	shopWallet            *currencySnapshot // last persisted NPC shop wallet or conflict reconciliation
	faction               string
	mu                    sync.Mutex
	motion                playerMotion
	buffs                 map[uint16]activePlayerBuff
	activeQingGong        map[string]struct{}
	activeQingGongOrder   []string
	activeJingMai         map[string]struct{}
	activeJingMaiOrder    []string
	curJingMai            string
	lastJingMai           string
	zhenQiDayValue        int32
	zqActValue            int32
	zqUnUsedValue         int32
	jingMaiProgress       map[string]jingMaiBookProgress
	shortcuts             []playerShortcut
	lastShortcutRows      []playerShortcut
	keybind               string
	learnedSkills         map[string]int32
	skillCooldowns        map[string]time.Time
	skillTeamCooldowns    map[int32]time.Time
	fwzCollected          map[string]struct{}
	currentSkillID        string
	currentSkillEffectID  string
	currentSkillLevel     int32
	currentSkillTarget    uint64
	parrying              bool
	lastBlockToggle       time.Time
	blockFlipsInWindow    int
	blockLocked           bool
	releaseCount          int
	lastReleaseTime       time.Time
	groundYSet            bool
	groundY               float32
	lastY                 float32
	positionX             float32
	positionY             float32
	positionZ             float32
	leapOrigin            world.Transform
	leapActive            bool
	combatPresence        bool
	skillLockUntil        time.Time
	reviveAt              time.Time
	innerPowerProcReadyAt time.Time
	innerPowerHealReadyAt time.Time
	healOT                healOverTime
	yufengMode            int
	yufengStageAt         time.Time
	yufengLastMp          time.Time
	yufengRunAccum        time.Duration
	yufengLastRunAt       time.Time
	yufengLastX           float32
	yufengLastZ           float32
	progress              playerProgress
	bagItems              []bagItem
	equipItems            []wornEquipItem
	weapon                string
	weaponItemType        uint8
	weaponMode            string
	weaponHeldMode        string
	sex                   uint8
	appearance            roleVisual
	defaultAppearance     roleVisual
}
type playerMotion struct {
	moveSpeed float32
	runSpeed  float32
	gravity   float32
}
type activePlayerBuff struct {
	staticData uint32
	expiresUTC time.Time
	configID   string
	level      int32
}
type playerBuffRules struct {
	CantHitEffect     bool
	CantBeSkillLocked bool
}

func newPlayerActor(name string, silver int32) *playerActor {
	learnedSkills := make(map[string]int32, len(starterSkillViews))
	for _, skill := range starterSkillViews {
		learnedSkills[skill.configID] = skill.level
	}
	p := &playerActor{actor: entity.NewActor(playerObjectID, playerOwnerID, name, 1000, 500), name: name, silver: silver, motion: playerMotion{moveSpeed: 3.9, runSpeed: 4.0, gravity: 10.0}, buffs: make(map[uint16]activePlayerBuff), activeQingGong: make(map[string]struct{}), activeJingMai: make(map[string]struct{}), zhenQiDayValue: zhenQiDailyCap, jingMaiProgress: make(map[string]jingMaiBookProgress), learnedSkills: learnedSkills, skillCooldowns: make(map[string]time.Time), skillTeamCooldowns: make(map[int32]time.Time), fwzCollected: make(map[string]struct{}), progress: newPlayerProgress()}
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.syncEquippedNeiGongResourcesLocked()
	return p
}
func (p *playerActor) learnedSkillLevel(configID string) (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	level, ok := p.learnedSkills[configID]
	return level, ok && level > 0
}
func (p *playerActor) learnSkill(configID string, level int32) {
	if configID == "" || level <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.learnedSkills[configID] = level
}
func (p *playerActor) progressProperties() []clientdata.IndexedProperty {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.properties()
}
func (p *playerActor) setFaction(faction string) ([]byte, error) {
	p.mu.Lock()
	p.faction = faction
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, factionProperties(faction))
}
func factionProperties(faction string) []clientdata.IndexedProperty {
	school, force, newSchool := "", "", ""
	if strings.HasPrefix(faction, "force_") {
		force = faction
	} else if strings.HasPrefix(faction, "newschool_") {
		newSchool = faction
	} else {
		school = faction
	}
	return []clientdata.IndexedProperty{{Index: 189, Name: "School", Value: clientdata.StringValue(school)}, {Index: 194, Name: "Force", Value: clientdata.StringValue(force)}, {Index: 190, Name: "NewSchool", Value: clientdata.StringValue(newSchool)}}
}

type npcAttackOutcome struct {
	stateChanged        bool
	parried             bool
	dodged              bool
	dead                bool
	innerPowerTriggered bool
	innerPowerHeal      int32
}

func (p *playerActor) resolveNPCAttack(attackerID uint32, damage int32, now time.Time) npcAttackOutcome {
	if damage <= 0 || p.actor.Snapshot().HP <= 0 {
		return npcAttackOutcome{}
	}
	if p.parryingSnapshot() {
		return npcAttackOutcome{parried: true}
	}
	profile, hasProfile := p.equippedInnerPowerPassive()
	activePassive, triggered := false, false
	if hasProfile && profile.mode == innerPowerModeIncomingDodgeHeal {
		activePassive, triggered = p.applyIncomingDodgePassive(profile, attackerID, now)
	}
	dodgeChance := int32(10)
	if activePassive {
		dodgeChance += profile.dodgeAdd
	}
	if combatPercentRoll(attackerID, now, 18) < dodgeChance {
		healed := int32(0)
		if hasProfile && profile.mode == innerPowerModeIncomingDodgeHeal && !p.innerPowerHealCoolingDown(now) {
			state := p.actor.Snapshot()
			amount := (state.MaxHP*profile.healPercent + 99) / 100
			if p.actor.ApplyHeal(amount) {
				healed = amount
			}
			p.setInnerPowerHealCooldown(now.Add(profile.healCooldown))
		}
		return npcAttackOutcome{stateChanged: triggered || healed > 0, dodged: true, innerPowerTriggered: triggered, innerPowerHeal: healed}
	}
	changed := p.actor.ApplyDamage(damage)
	if !changed {
		return npcAttackOutcome{}
	}
	rules := p.buffRules(now)
	if !rules.CantBeSkillLocked {
		p.mu.Lock()
		p.skillLockUntil = now.Add(500 * time.Millisecond)
		p.mu.Unlock()
	}
	dead := p.actor.Snapshot().HP <= 0
	if dead {
		p.mu.Lock()
		p.reviveAt = now.Add(playerAutoReviveDelay)
		p.mu.Unlock()
	}
	return npcAttackOutcome{stateChanged: true, dead: dead}
}
func combatPercentRoll(attackerID uint32, now time.Time, salt uint64) int32 {
	value := uint64(attackerID)*1103515245 + uint64(now.Unix())*2654435761 + salt*97
	return int32(value % 100)
}
func (p *playerActor) applyIncomingDodgePassive(profile innerPowerPassiveProfile, attackerID uint32, now time.Time) (active, triggered bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	current := p.buffs[yihuaPassiveBuffSlot]
	active = current.staticData == profile.procStatic && current.expiresUTC.After(now.UTC())
	if active || p.innerPowerProcReadyAt.After(now) || combatPercentRoll(attackerID, now, 17) >= profile.procChance {
		return active, false
	}
	expires := now.UTC().Add(profile.procLifetime)
	p.buffs[yihuaPassiveBuffSlot] = activePlayerBuff{staticData: profile.procStatic, expiresUTC: expires, configID: profile.procBuffID, level: profile.buffLevel}
	p.innerPowerProcReadyAt = now.Add(profile.procCooldown)
	return true, true
}
func (p *playerActor) innerPowerHealCoolingDown(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.innerPowerHealReadyAt.After(now)
}
func (p *playerActor) setInnerPowerHealCooldown(next time.Time) {
	p.mu.Lock()
	p.innerPowerHealReadyAt = next
	p.mu.Unlock()
}
func (p *playerActor) expireInnerPowerPassive(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[10]
	if !ok || buff.expiresUTC.After(now.UTC()) {
		return false
	}
	delete(p.buffs, 10)
	return true
}
func (p *playerActor) skillLocked(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.skillLockUntil.After(now)
}
func (p *playerActor) reviveWhenDue(now time.Time) bool {
	p.mu.Lock()
	if p.reviveAt.IsZero() || now.Before(p.reviveAt) {
		p.mu.Unlock()
		return false
	}
	p.reviveAt = time.Time{}
	p.skillLockUntil = time.Time{}
	p.mu.Unlock()
	p.actor.Revive()
	return true
}
func (p *playerActor) progressViewFrames() ([][]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.viewFrames()
}
func (p *playerActor) equipNeiGong(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.equip(id); err != nil {
		return err
	}
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.syncEquippedNeiGongResourcesLocked()
	return nil
}
func (p *playerActor) syncEquippedNeiGongResourcesLocked() {
	hpAdd, mpAddRaw := p.progress.totalResourceAdds()
	base := p.actor.Snapshot()
	p.actor.SetDerivedMaxima(base.BaseMaxHP+hpAdd, base.BaseMaxMP+mpAddRaw)
}
func (p *playerActor) syncEquippedNeiGongBuffLocked(now time.Time) {
	book := p.progress.book(p.progress.curNeiGong)
	if book == nil || book.buffID == "" || book.buffStaticData == 0 || book.buffLevel <= 0 {
		delete(p.buffs, innerPowerBuffSlot)
		return
	}
	displayID := neiGongDisplayBuffID(book.buffID)
	staticData := buffStaticDataForID(displayID)
	if staticData == 0 {
		staticData = book.buffStaticData
	}
	p.buffs[innerPowerBuffSlot] = activePlayerBuff{staticData: staticData, expiresUTC: now.UTC().Add(24 * time.Hour), configID: displayID, level: book.buffLevel}
}
func (p *playerActor) facultyReady(id string) (uint16, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.facultyReady(id)
}
func (p *playerActor) facultyBegin(style int32) (uint16, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.facultyBegin(style, time.Now()); err != nil {
		return 0, false, err
	}
	if style == 2 {
		p.actor.SetLogicState(108)
	} else {
		p.actor.SetLogicState(0)
	}
	return 0, false, nil
}
func (p *playerActor) facultyExit() {
	p.mu.Lock()
	p.progress.facultyExit()
	p.mu.Unlock()
	p.actor.SetLogicState(0)
}
func (p *playerActor) progressBookFrame(slot uint16) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.bookFrame(slot)
}
func (p *playerActor) facultyTick(now time.Time) (slot uint16, changed bool, isQingGong bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot, changed = p.progress.advanceFaculty(now)
	if changed {
		_, isQingGong = qgStaticData[p.progress.facultyName]
		if !isQingGong {
			p.syncEquippedNeiGongBuffLocked(now)
			p.syncEquippedNeiGongResourcesLocked()
		}
	}
	return slot, changed, isQingGong
}
func (p *playerActor) facultySnapshot() facultyProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.snapshot()
}
func (p *playerActor) restoreFaculty(value facultyProgressSnapshot, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.restore(value, now)
	if p.progress.facultyState == 2 && p.progress.facultyStyle == 2 {
		p.actor.SetLogicState(108)
	} else {
		p.actor.SetLogicState(0)
	}
	p.syncEquippedNeiGongBuffLocked(now)
	p.syncEquippedNeiGongResourcesLocked()
}
func (p *playerActor) activateQingGong(id string) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.activeQingGong[id]; exists {
		return 0, false
	}
	p.activeQingGong[id] = struct{}{}
	p.activeQingGongOrder = append(p.activeQingGongOrder, id)
	return len(p.activeQingGongOrder) - 1, true
}
func (p *playerActor) stateProperties() []clientdata.IndexedProperty {
	state := p.actor.Snapshot()
	hpRecovery := sitcrossRecoveryPerSecond(state.MaxHP)
	mpRecovery := sitcrossRecoveryPerSecond(state.MaxMP)
	currentSkillID, _, _, currentSkillTarget := p.currentSkillStateSnapshot()
	hpRatio := int32(int64(state.HP) * 100 / int64(state.MaxHP))
	hitHPRatio := int32(int64(state.HitHP) * 100 / int64(state.MaxHP))
	mpRatio := int32(int64(state.MP) * 100 / int64(state.MaxMP))
	dead := uint8(0)
	if state.HP <= 0 {
		dead = 1
	}
	inParry := int32(0)
	if p.parryingSnapshot() {
		inParry = 1
	}
	properties := []clientdata.IndexedProperty{{Index: 463, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)}, {Index: 469, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)}, {Index: 470, Name: "MPRatio", Value: clientdata.Int32Value(mpRatio)}, {Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)}, {Index: 579, Name: "MaxMP", Value: clientdata.Int32Value(state.BaseMaxMP)}, {Index: 465, Name: "HP", Value: clientdata.Int32Value(state.HP)}, {Index: 466, Name: "MP", Value: clientdata.Int32Value(state.MP)}, {Index: 390, Name: "SpeedRatio", Value: clientdata.Int32Value(state.SpeedRatio)}, {Index: 481, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)}, {Index: 482, Name: "HitHPRatio", Value: clientdata.Int32Value(hitHPRatio)}, {Index: 692, Name: "QingGongPoint", Value: clientdata.Int32Value(state.QingGongPoint)}, {Index: 693, Name: "MaxQingGongPoint", Value: clientdata.Int32Value(state.MaxQingGongPoint)}, {Index: 694, Name: "MaxQingGongPointAdd", Value: clientdata.Int32Value(state.MaxQingGongPointAdd)}, {Index: 85, Name: "NeigongPKStatus", Value: clientdata.WordValue(0)}, {Index: 464, Name: "Dead", Value: clientdata.ByteValue(dead)}, {Index: 471, Name: "InParry", Value: clientdata.Int32Value(inParry)}, {Index: 460, Name: "CurSkillID", Value: clientdata.StringValue(currentSkillID)}, {Index: 627, Name: "CurSkillTarget", Value: clientdata.ObjectValue(uint32(currentSkillTarget), uint32(currentSkillTarget>>32))}, {Index: 221, Name: "ModifySkillLockTime", Value: clientdata.Int32Value(5000)}, {Index: 617, Name: "HPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)}, {Index: 621, Name: "HPHeartSpeedAdd", Value: clientdata.Int32Value(0)}, {Index: 618, Name: "MPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)}, {Index: 622, Name: "MPHeartSpeedAdd", Value: clientdata.Int32Value(0)}, {Index: 615, Name: "HPUpSpeed", Value: clientdata.Int32Value(hpRecovery)}, {Index: 619, Name: "HPUpSpeedAdd", Value: clientdata.Int32Value(0)}, {Index: 616, Name: "MPUpSpeed", Value: clientdata.Int32Value(mpRecovery)}, {Index: 620, Name: "MPUpSpeedAdd", Value: clientdata.Int32Value(0)}, {Index: 178, Name: "Name", Value: clientdata.WideStringValue(state.Name)}, {Index: 417, Name: "CapitalType1", Value: clientdata.Int64Value(int64(p.silver))}, {Index: 416, Name: "CapitalType0", Value: clientdata.Int64Value(int64(p.gold))}, {Index: 418, Name: "CapitalType2", Value: clientdata.Int64Value(int64(p.silverCard))}, {Index: 420, Name: "CapitalType4", Value: clientdata.Int64Value(int64(p.silverTicket))}}
	properties = append(properties, factionProperties(p.faction)...)
	for _, slot := range nativeBuffSlots {
		properties = append(properties, clientdata.IndexedProperty{Index: bufferInfoPropertyIndex(slot), Name: bufferInfoPropertyName(slot), Value: clientdata.StringValue(p.bufferInfo(slot, time.Now()))})
	}
	properties = append(properties, baseMotionProperties(p.motionSnapshot())...)
	properties = append(properties, qinggongMotionProperties()...)
	p.mu.Lock()
	curJingMai := p.curJingMai
	lastJingMai := p.lastJingMai
	jmActCount := len(p.activeJingMaiOrder)
	if p.zhenQiDayValue <= 0 {
		p.zhenQiDayValue = zhenQiDailyCap
	}
	zhenQiDay := p.zhenQiDayValue
	zqAct := p.zqActValue
	zqUnused := p.zqUnUsedValue
	p.mu.Unlock()
	properties = append(properties, clientdata.IndexedProperty{Index: 849, Name: "CurJingMai", Value: clientdata.StringValue(curJingMai)}, clientdata.IndexedProperty{Index: 850, Name: "LastJingMai", Value: clientdata.StringValue(lastJingMai)}, clientdata.IndexedProperty{Index: 851, Name: "jmActCount", Value: clientdata.Int32Value(int32(jmActCount))}, clientdata.IndexedProperty{Index: 852, Name: "ZhenQiDayValue", Value: clientdata.Int32Value(zhenQiDay)}, clientdata.IndexedProperty{Index: 855, Name: "ZQPLValue", Value: clientdata.Int32Value(0)}, clientdata.IndexedProperty{Index: 856, Name: "ZQActValue", Value: clientdata.Int32Value(zqAct)}, clientdata.IndexedProperty{Index: 857, Name: "ZQUnUsedValue", Value: clientdata.Int32Value(zqUnused)})
	properties = append(properties, p.progressProperties()...)
	return properties
}
func boolInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
func baseMotionProperties(motion playerMotion) []clientdata.IndexedProperty {
	return []clientdata.IndexedProperty{{Index: 677, Name: "Gravity", Value: clientdata.Float32Value(motion.gravity)}, {Index: 678, Name: "GravityAdd", Value: clientdata.Float32Value(0)}, {Index: 389, Name: "DropHeightPub", Value: clientdata.Float32Value(100000)}, {Index: 681, Name: "JumpSpeed", Value: clientdata.Float32Value(8)}}
}
func (p *playerActor) motionSnapshot() playerMotion {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.motion
}
func (p *playerActor) setMoveSpeed(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.moveSpeed = value
	p.motion.runSpeed = value
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 384, Name: "MoveSpeed", Value: clientdata.Float32Value(value)}, {Index: 386, Name: "RunSpeed", Value: clientdata.Float32Value(value)}})
}
func (p *playerActor) setGravity(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.gravity = value
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 677, Name: "Gravity", Value: clientdata.Float32Value(value)}})
}
func (p *playerActor) activateRedSuperArmor(now time.Time) (string, time.Time) {
	return p.activateBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, 20*time.Second, now)
}
func (p *playerActor) activateBuff(slot uint16, staticData uint32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	expires := now.Add(lifetime)
	p.buffs[slot] = activePlayerBuff{staticData: staticData, expiresUTC: expires, level: 1}
	return p.bufferInfoLocked(slot, now), expires
}
func (p *playerActor) activateNamedBuff(slot uint16, staticData uint32, configID string, lifetime time.Duration, now time.Time) (string, time.Time) {
	return p.activateNamedBuffAtLevel(slot, staticData, configID, 1, lifetime, now)
}
func (p *playerActor) activateNamedBuffAtLevel(slot uint16, staticData uint32, configID string, level int32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if level < 1 {
		level = 1
	}
	expires := now.UTC().Add(lifetime)
	p.buffs[slot] = activePlayerBuff{staticData: staticData, expiresUTC: expires, configID: configID, level: level}
	return p.bufferInfoLocked(slot, now), expires
}
func (p *playerActor) activateSkillBuff(definition skillBuffDefinition, now time.Time) (uint16, string, time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := definition.preferredSlot
	if slot == 0 {
		for candidate, active := range p.buffs {
			if active.configID == definition.configID {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 {
		for candidate := uint16(11); candidate <= 24; candidate++ {
			active, occupied := p.buffs[candidate]
			if !occupied || !active.expiresUTC.After(now.UTC()) {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 || slot > 24 {
		return 0, "", time.Time{}, false
	}
	lifetime := definition.lifetime
	if lifetime <= 0 {
		lifetime = time.Millisecond
	}
	expires := now.UTC().Add(lifetime)
	level := definition.level
	if level < 1 {
		level = 1
	}
	p.buffs[slot] = activePlayerBuff{staticData: definition.staticData, expiresUTC: expires, configID: definition.configID, level: level}
	return slot, p.bufferInfoLocked(slot, now), expires, true
}
func (p *playerActor) nextSkillBuffVariant(variants []skillBuffDefinition, now time.Time) (skillBuffDefinition, bool) {
	if len(variants) == 0 {
		return skillBuffDefinition{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := variants[0].preferredSlot
	active, exists := p.buffs[slot]
	if !exists || !active.expiresUTC.After(now.UTC()) {
		return variants[0], true
	}
	for index, variant := range variants {
		if active.staticData == variant.staticData {
			return variants[(index+1)%len(variants)], true
		}
	}
	return variants[0], true
}
func (p *playerActor) bufferInfo(slot uint16, now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferInfoLocked(slot, now)
}
func (p *playerActor) bufferInfoLocked(slot uint16, now time.Time) string {
	buff, ok := p.buffs[slot]
	if !ok || !buff.expiresUTC.After(now.UTC()) {
		return ""
	}
	level := buff.level
	if level < 1 {
		level = 1
	}
	return fmt.Sprintf("%X,%X,%X,%X,%X,%X", buff.staticData, playerObjectID, playerOwnerID, level, buff.expiresUTC.UnixMilli(), 1)
}
func (p *playerActor) bufferListString(now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferListStringLocked(now)
}
func (p *playerActor) bufferListStringLocked(now time.Time) string {
	var list strings.Builder
	for _, slot := range nativeBuffSlots {
		buff, ok := p.buffs[slot]
		if !ok || !buff.expiresUTC.After(now.UTC()) {
			continue
		}
		configID := buff.configID
		if configID == "" {
			configID, ok = activeBuffConfigID(buff.staticData)
		}
		if !ok {
			continue
		}
		level := buff.level
		if level < 1 {
			level = 1
		}
		fmt.Fprintf(&list, "%s,local-%d,%d,%d-%d,%d,1,0|", configID, slot, level, playerObjectID, playerOwnerID, buff.expiresUTC.UnixMilli())
	}
	return list.String()
}
func activeBuffConfigID(staticData uint32) (string, bool) {
	switch staticData {
	case 216:
		return "buf_qg_drift", true
	case 4558:
		return "BuffInParry", true
	case 5786:
		return "buf_qg_dst", true
	case 8682:
		return "buf_fixed_red", true
	case 11017:
		return "buf_fixed_yel_ng", true
	default:
		return "", false
	}
}
func (p *playerActor) hasActiveBuff(slot uint16, staticData uint32, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	return ok && buff.staticData == staticData && buff.expiresUTC.After(now.UTC())
}
func (p *playerActor) clearBuff(slot uint16, expectedStaticData uint32) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}
func (p *playerActor) expireBuff(slot uint16, expectedStaticData uint32, expected time.Time, now time.Time) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData || !buff.expiresUTC.Equal(expected) || buff.expiresUTC.After(now.UTC()) {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}
func (p *playerActor) buffRules(now time.Time) playerBuffRules {
	red := p.hasActiveBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, now)
	yellow := p.hasActiveBuff(15, 11017, now)
	armored := red || yellow
	return playerBuffRules{CantHitEffect: armored, CantBeSkillLocked: armored}
}
func qinggongMotionProperties() []clientdata.IndexedProperty {
	const (
		waterSpeed         float32 = 7.5
		waterJumpSpeed     float32 = 6.0
		secondJumpSpeed    float32 = 6.0
		thirdJumpSpeed     float32 = 6.0
		airRushSpeed       float32 = 15
		airRushDist        float32 = 12
		landRushSpeed      float32 = 25
		landRushDist       float32 = 6.5
		climbSpeed         float32 = 7.5
		groundRushRange    float32 = 5.0
		airRushRange       float32 = 8.0
		waterRushRange     float32 = 8.0
		groundRushRangeAdd float32 = 1.5
		airRushRangeAdd    float32 = 6.5
		waterRushRangeAdd  float32 = 6.5
	)
	f := func(index uint16, name string, value float32) clientdata.IndexedProperty {
		return clientdata.IndexedProperty{Index: index, Name: name, Value: clientdata.Float32Value(value)}
	}
	return []clientdata.IndexedProperty{f(0x0184, "RunSpeedAdd", 0), f(0x02AA, "JumpSpeedAdd", 0), f(0x028F, "DriftSpeed", waterSpeed), f(0x0290, "DriftSpeedAdd", 0), f(0x0293, "DriftJumpSpeed", waterJumpSpeed), f(0x0294, "DriftJumpSpeedAdd", 0), f(0x0295, "SndJumpSpeed", secondJumpSpeed), f(0x0296, "SndJumpSpeedAdd", 0), f(0x0297, "ThdJumpSpeed", thirdJumpSpeed), f(0x0298, "ThdJumpSpeedAdd", 0), f(0x0299, "AirRushSpeed", airRushSpeed), f(0x029A, "AirRushSpeedAdd", 0), f(0x029B, "AirRushDist", airRushDist), f(0x029C, "AirRushDistAdd", 0), f(0x029D, "LandRushSpeed", landRushSpeed), f(0x029E, "LandRushSpeedAdd", 0), f(0x029F, "LandRushDist", landRushDist), f(0x02A0, "LandRushDistAdd", 0), f(0x02A7, "ClimbSpeed", climbSpeed), f(0x02A8, "ClimbSpeedAdd", 0), f(0x0369, "GRushRange", groundRushRange), f(0x036C, "GRushRangeAdd", groundRushRangeAdd), f(0x036A, "ARushRange", airRushRange), f(0x036D, "ARushRangeAdd", airRushRangeAdd), f(0x036B, "WRushRange", waterRushRange), f(0x036E, "WRushRangeAdd", waterRushRangeAdd)}
}
func (p *playerActor) vitalUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, p.stateProperties())
}
func (p *playerActor) sitcrossTick() bool {
	state := p.actor.Snapshot()
	return p.actor.RestoreWhileSit(sitcrossRecoveryPerSecond(state.MaxHP), sitcrossRecoveryPerSecond(state.MaxMP))
}
func sitcrossRecoveryPerSecond(maximum int32) int32 {
	if maximum <= 0 {
		return 0
	}
	return (maximum + sitcrossFullRecoverySeconds - 1) / sitcrossFullRecoverySeconds
}
func (p *playerActor) combatBruiseTick() bool {
	return p.actor.RecoverBruisedHPWhileFighting(combatBruiseRecoveryPerSecond)
}
func (p *playerActor) setCombatPresence(active bool) bool {
	state := p.actor.Snapshot()
	if state.HP <= 0 {
		return false
	}
	p.mu.Lock()
	p.combatPresence = active
	p.mu.Unlock()
	if active {
		if state.LogicState == 1 {
			return false
		}
		p.actor.SetLogicState(1)
		return true
	}
	if state.LogicState != 1 {
		return false
	}
	p.actor.SetLogicState(0)
	return true
}
func (p *playerActor) currentSkillSnapshot() string {
	id, _, _, _ := p.currentSkillStateSnapshot()
	return id
}
func (p *playerActor) currentSkillStateSnapshot() (string, string, int32, uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentSkillID, p.currentSkillEffectID, p.currentSkillLevel, p.currentSkillTarget
}
func (p *playerActor) clearCurrentSkill(expected string) bool {
	p.mu.Lock()
	if p.currentSkillID != expected {
		p.mu.Unlock()
		return false
	}
	p.currentSkillID = ""
	p.currentSkillEffectID = ""
	p.currentSkillLevel = 0
	p.currentSkillTarget = 0
	inCombat := p.combatPresence
	p.mu.Unlock()
	if inCombat {
		p.actor.SetLogicState(1)
	} else {
		p.actor.SetLogicState(0)
	}
	return true
}
func (p *playerActor) currentSkillClearUpdate() ([]byte, error) {
	p.mu.Lock()
	inCombat := p.combatPresence
	p.mu.Unlock()
	logicState := uint8(0)
	if inCombat {
		logicState = 1
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 463, Name: "LogicState", Value: clientdata.ByteValue(logicState)}, {Index: 460, Name: "CurSkillID", Value: clientdata.StringValue("")}, {Index: 627, Name: "CurSkillTarget", Value: clientdata.ObjectValue(0, 0)}})
}
func (p *playerActor) facultyNameResetFrame() ([]byte, bool, error) {
	p.mu.Lock()
	hasFacultyName := p.progress.facultyName != ""
	p.mu.Unlock()
	if !hasFacultyName {
		return nil, false, nil
	}
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 760, Name: "FacultyName", Value: clientdata.StringValue("")}})
	return frame, true, err
}
func (p *playerActor) setSilver(value int32) {
	p.mu.Lock()
	p.silver = clampCapital(p.silver, value-p.silver, 8000000)
	p.mu.Unlock()
}
func clampPlayerSP(value int32) int32 {
	if value < 0 {
		return 0
	}
	if value > maxPlayerSP {
		return maxPlayerSP
	}
	return value
}
func (p *playerActor) setSP(value int32) (int32, []byte, error) {
	p.mu.Lock()
	if value < 0 {
		value = 0
	} else if value > 100 {
		value = 100
	}
	p.progress.attributes.sp = value
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 467, Name: "SP", Value: clientdata.Int32Value(sp)}})
	return sp, frame, err
}
func (p *playerActor) addSP(delta int32) (int32, []byte, error) {
	p.mu.Lock()
	next := int64(p.progress.attributes.sp) + int64(delta)
	if next < 0 {
		next = 0
	}
	if next > 100 {
		next = 100
	}
	p.progress.attributes.sp = int32(next)
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 467, Name: "SP", Value: clientdata.Int32Value(sp)}})
	return sp, frame, err
}
func (p *playerActor) addSilver(amount int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.silver = clampCapital(p.silver, amount, 8000000)
	return p.silver
}
func (p *playerActor) silverUpdate() ([]byte, error) {
	p.mu.Lock()
	silver := p.silver
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 417, Name: "CapitalType1", Value: clientdata.Int64Value(int64(silver))}})
}
func (p *playerActor) addObject(location role.Position, visual roleVisual) []byte {
	state := p.actor.Snapshot()
	motion := p.motionSnapshot()
	msg := make([]byte, 0x3f, 0x120)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], state.ObjectID)
	binary.LittleEndian.PutUint32(msg[5:], state.OwnerID)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(location.X))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(location.Y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(location.Z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(location.Orient))
	count := uint16(0)
	addByte := func(index uint16, value byte) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = append(msg, value)
		count++
	}
	addInt := func(index uint16, value int32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(value))
		count++
	}
	addFloat := func(index uint16, value float32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(value))
		count++
	}
	addString := func(index uint16, value string) {
		msg = appendStringProperty(msg, index, value)
		count++
	}
	addObject := func(index uint16, value uint64) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint64(msg, value)
		count++
	}
	addByte(0, 2)
	addByte(1, visual.sex)
	addString(2, visual.photo)
	addInt(3, 0)
	addInt(4, 0)
	addInt(5, 0)
	addInt(6, 1)
	addFloat(9, location.X)
	addFloat(10, location.Y)
	addFloat(11, location.Z)
	addFloat(12, location.Orient)
	addString(13, visual.hair)
	addString(14, visual.face)
	addString(15, visual.cloth)
	addString(16, visual.pants)
	addString(17, visual.shoes)
	addString(18, visual.actionSet)
	addString(19, "stand")
	addInt(20, 0)
	addByte(21, state.LogicState)
	addFloat(22, 8.0)
	addFloat(23, motion.moveSpeed)
	addFloat(24, 7.5)
	addFloat(25, motion.runSpeed)
	addInt(26, state.SpeedRatio)
	addByte(27, 0)
	addInt(28, int32(int64(state.HP)*100/int64(state.MaxHP)))
	addInt(29, int32(int64(state.MP)*100/int64(state.MaxMP)))
	addInt(30, state.BaseMaxHP)
	addInt(31, state.BaseMaxMP)
	addInt(32, state.HP)
	addInt(33, state.MP)
	addInt(112, state.HitHP)
	addInt(113, int32(int64(state.HitHP)*100/int64(state.MaxHP)))
	addInt(114, state.QingGongPoint)
	addInt(115, state.MaxQingGongPoint)
	addInt(116, state.MaxQingGongPointAdd)
	addInt(propNeigongPKStatus, 0)
	addInt(propDead, boolInt32(state.HP <= 0))
	addInt(propCantUseSkill, boolInt32(p.skillLocked(time.Now())))
	currentSkillID, currentSkillEffectID, currentSkillLevel, currentSkillTarget := p.currentSkillStateSnapshot()
	addString(propCurSkillID, currentSkillID)
	addString(propCurSkillEffectID, currentSkillEffectID)
	addInt(propCurSkillLevel, currentSkillLevel)
	addObject(propCurSkillTarget, currentSkillTarget)
	addInt(propModifySkillLockTime, 5000)
	addString(bufferListPropertyIndex, p.bufferListString(time.Now()))
	for _, property := range baseMotionProperties(motion) {
		addFloat(property.Index, property.Value.F32)
	}
	for _, property := range qinggongMotionProperties() {
		addFloat(property.Index, property.Value.F32)
	}
	addInt(111, p.silver)
	for _, slot := range nativeBuffSlots {
		addString(bufferInfoPropertyIndex(slot), p.bufferInfo(slot, time.Now()))
	}
	for _, property := range p.progressProperties() {
		msg = appendNPCProperty(msg, property)
		count++
	}
	addString(35, visual.hair)
	msg = appendWideStringProperty(msg, 36, state.Name)
	count++
	msg = appendObjectProperty(msg, 100, 0, 0)
	count++
	binary.LittleEndian.PutUint16(msg[0x3d:], count)
	return msg
}
func sendPlayerSpawn(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	player.updatePosition(location.X, location.Y, location.Z)
	log.Printf("native Buff spawn player=%d-%d red=%q drift=%q jump=%q neigong=%q", playerObjectID, playerOwnerID, player.bufferInfo(redSuperArmorBuffSlot, time.Now()), player.bufferInfo(waterDriftBuffSlot, time.Now()), player.bufferInfo(waterJumpBuffSlot, time.Now()), player.bufferInfo(innerPowerBuffSlot, time.Now()))
	if err := sendPlayerAddObject(conn, player, location, visual); err != nil {
		return err
	}
	snapshot, err := player.playerSnapshot608(location, visual)
	if err != nil {
		return err
	}
	if err := conn.WriteFrame(snapshot); err != nil {
		return err
	}
	if appearance, err := player.playerAppearanceFrame(visual); err == nil {
		if err := conn.WriteFrame(appearance); err != nil {
			return err
		}
	}
	return sendPlayerLocationAndVitals(conn, player, location)
}
func sendPlayerAddObject(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	rawProperties := player.playerBirthProperties(location, visual)
	wireProperties := latestClientPlayerWireProperties(rawProperties)
	log.Printf("latest-client PLAYER-PROPERTY-ORDINAL birth raw=%d wire=%d dropped=%d table=%d", len(rawProperties), len(wireProperties), len(rawProperties)-len(wireProperties), latestClientPlayerWirePropertyTableCount)
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, wireProperties)
	if err != nil {
		return err
	}
	return conn.WriteFrame(frame)
}
func sendPlayerLocationAndVitals(conn sceneMessageConnection, player *playerActor, location role.Position) error {
	if err := conn.WriteFrame(serverLocation(playerObjectID, playerOwnerID, world.Transform{X: location.X, Y: location.Y, Z: location.Z, Orient: location.Orient})); err != nil {
		return err
	}
	vitals, err := player.vitalUpdate()
	if err != nil {
		return err
	}
	return conn.WriteFrame(vitals)
}
func worldTransform(position role.Position) worldcore.Transform {
	return worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
}
