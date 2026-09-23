package main

import (
	"fmt"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
	"log"
	"math"
	"sort"
	"time"
)

const (
	clientCustomUseSkill             int32  = 211
	serverNativeSkillActionMessage   int32  = 20
	yiHuaSkill04BuffStaticData       uint32 = 15101
	yiHuaSkill04BuffLease                   = 35 * time.Second
	yiHuaSkill05BuffStaticData       uint32 = 15105
	yiHuaSkill05BuffLease                   = 5 * time.Second
	taiJiDefenceStanceBuffStaticData uint32 = 6013
	taiJiAttackStanceBuffStaticData  uint32 = 6055
	taiJiSkill05BuffStaticData       uint32 = 6043
	taiJiSkill07BuffStaticData       uint32 = 6050
	taiJiStanceBuffLease                    = 24 * time.Hour
	taiJiSkill05BuffLease                   = 15 * time.Second
	taiJiSkill07BuffLease                   = 8 * time.Second
)

type skillActionSegment struct {
	action   string
	at       time.Duration
	duration time.Duration
}
type skillTargetMode uint8

const (
	skillTargetSelf skillTargetMode = iota
	skillTargetSelected
	skillTargetCasterCircle
	skillTargetCasterSector
	skillTargetCasterRectangle
	skillTargetSelectedArea
)

type skillEffectKind uint8

const (
	skillEffectDamage skillEffectKind = iota
	skillEffectPlayerBuff
	skillEffectTargetControl
	skillEffectRedArmor
	skillEffectYellowArmor
	skillEffectStanceCycle
)

type skillBuffDefinition struct {
	configID      string
	staticData    uint32
	level         int32
	lifetime      time.Duration
	preferredSlot uint16
}
type combatSkillEffect struct {
	kind         skillEffectKind
	amount       int32
	buff         skillBuffDefinition
	buffVariants []skillBuffDefinition
}
type combatSkillDefinition struct {
	id               string
	staticData       int32
	script           string
	taoLu            string
	attribute        string
	level            int32
	cooldownCategory int32
	cooldownTeam     int32
	personalCD       time.Duration
	publicCD         time.Duration
	mpCost           int32
	spCost           int32
	qgCost           int32
	baseDamage       int32
	requiresTarget   bool
	targetMode       skillTargetMode
	range_           float32
	areaRadius       float32
	areaHeight       float32
	sectorAngle      float32
	areaWidth        float32
	actionName       string
	followupActions  []skillActionSegment
	hitFrames        []time.Duration
	actionDuration   time.Duration
	effects          []combatSkillEffect
}

func (definition combatSkillDefinition) totalActionDuration() time.Duration {
	duration := definition.actionDuration
	for _, segment := range definition.followupActions {
		if segment.at+segment.duration > duration {
			duration = segment.at + segment.duration
		}
	}
	return duration
}
func skillCooldownFrame(definition combatSkillDefinition, now time.Time) ([]byte, error) {
	return skillCooldownFrameWithDuration(definition, now, definition.personalCD)
}
func nativeSkillActionFrame(skillID string, targetID, targetOwnerID uint32) ([]byte, error) {
	return nativeSkillActionFrameAt(skillID, targetID, targetOwnerID, false, 0, 0, 0)
}
func skillActionPreparationFrame() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 18, Name: "ActionSet", Value: clientdata.StringValue("0h")}, {Index: 19, Name: "State", Value: clientdata.StringValue("stand")}})
}
func playerActionStateFrame(action string) ([]byte, error) {
	if action == "" {
		return nil, fmt.Errorf("player action state is empty")
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 19, Name: "State", Value: clientdata.StringValue(action)}})
}
func nativeGenericActionFrame(action string) ([]byte, error) {
	if action == "" {
		return nil, fmt.Errorf("native generic action is empty")
	}
	return serverModernCustomStringMessage("action", customObject(playerObjectID, playerOwnerID), customString(action), customString("1"))
}
func combatSkillRequestDefinition(id string, catalog *combatSkillCatalog, installed map[string]combatSkillDefinition) (combatSkillDefinition, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return combatSkillDefinition{}, false
	}
	if definition, exists := installed[id]; exists {
		return definition, true
	}
	if catalog != nil {
		if _, replacement := catalog.noConditionReplacementSource(id); replacement {
			// The replacement may intentionally lack its own player-action section.
			// Return only the requested id here; the learned level and executable
			// definition are resolved after the player authority check below.
			return combatSkillDefinition{id: id}, true
		}
	}
	return combatSkillDefinition{}, false
}

func requestedSkillAuthority(player *playerActor, id string, catalog *combatSkillCatalog) (int32, string, bool) {
	if player == nil {
		return 0, "", false
	}
	if level, learned := player.learnedSkillLevel(id); learned {
		return level, "", true
	}
	if catalog == nil {
		return 0, "", false
	}
	baseID, replacement := catalog.noConditionReplacementSource(id)
	if !replacement {
		return 0, "", false
	}
	level, learned := player.learnedSkillLevel(baseID)
	if !learned {
		return 0, baseID, false
	}
	return level, baseID, true
}

func parseUseSkillCustom(custom clientCustomMessage) (combatSkillDefinition, bool) {
	if len(custom.Values) < 2 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomUseSkill || custom.Values[1].Type != 6 || custom.Values[1].Text == "" {
		return combatSkillDefinition{}, false
	}
	return combatSkillRequestDefinition(custom.Values[1].Text, installedCombatSkillCatalog, combatSkills)
}
func useSkillCastTransform(custom clientCustomMessage) (worldcore.Transform, bool) {
	if len(custom.Values) < 5 || custom.Values[2].Type != 4 || custom.Values[3].Type != 4 || custom.Values[4].Type != 4 {
		return worldcore.Transform{}, false
	}
	orient := float32(0)
	if len(custom.Values) > 5 && custom.Values[5].Type == 4 {
		orient = custom.Values[5].Float32
	}
	return worldcore.Transform{X: custom.Values[2].Float32, Y: custom.Values[3].Float32, Z: custom.Values[4].Float32, Orient: orient}, true
}
func (p *playerActor) beginSkillUse(definition combatSkillDefinition, target uint64, now time.Time) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.skillLockUntil.After(now) {
		return false, fmt.Sprintf("skill locked remaining=%s", p.skillLockUntil.Sub(now).Round(time.Millisecond))
	}
	if ready := p.skillCooldowns[definition.id]; ready.After(now) {
		return false, fmt.Sprintf("personal cooldown remaining=%s", ready.Sub(now).Round(time.Millisecond))
	}
	teamKey := definition.teamKey()
	if ready := p.skillTeamCooldowns[teamKey]; ready.After(now) {
		return false, fmt.Sprintf("taolu switch lock team=%d remaining=%s", teamKey, ready.Sub(now).Round(time.Millisecond))
	}
	state := p.actor.Snapshot()
	if state.LogicState == 0x78 || state.HP <= 0 {
		return false, "player is dead"
	}
	if state.MP < definition.mpCost {
		return false, fmt.Sprintf("MP %d < %d", state.MP, definition.mpCost)
	}
	if p.progress.attributes.sp < definition.spCost {
		return false, fmt.Sprintf("SP %d < %d", p.progress.attributes.sp, definition.spCost)
	}
	if definition.qgCost > 0 {
		if cost, ok := p.actor.UseQingGong(definition.qgCost, now); !ok {
			return false, fmt.Sprintf("QingGongPoint cannot pay %d (resolved cost %d)", definition.qgCost, cost)
		}
	}
	if !p.actor.SpendMP(definition.mpCost) {
		return false, "MP spend rejected"
	}
	p.progress.attributes.sp -= definition.spCost
	p.skillCooldowns[definition.id] = now.Add(definition.personalCD)
	p.lockOtherSkillTeams(teamKey, now)
	p.currentSkillID = definition.id
	p.currentSkillEffectID = definition.id
	p.currentSkillLevel = definition.level
	p.currentSkillTarget = target
	return true, ""
}
func (p *playerActor) skillSP() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.attributes.sp
}
func (s *sceneLifecycle) selectedCombatTarget() (uint32, uint32, bool) {
	if s == nil {
		return 0, 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	actor := s.combatActors[id]
	if actor == nil || actor.Snapshot().HP <= 0 {
		return 0, 0, false
	}
	meta, exists := s.entities[id]
	if !exists || meta.ownerID != s.selectedOwner || meta.interaction != npcInteractionCombat {
		return 0, 0, false
	}
	return uint32(id), meta.ownerID, true
}
func (s *sceneLifecycle) applySelectedSkillDamage(amount int32, now time.Time) ([]byte, uint32, int32, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	actor := s.combatActors[id]
	meta, exists := s.entities[id]
	if actor == nil || !exists || meta.ownerID != s.selectedOwner || meta.interaction != npcInteractionCombat || actor.Snapshot().HP <= 0 {
		return nil, 0, 0, false, nil
	}
	if !actor.ApplyDamage(amount) {
		return nil, uint32(id), 0, false, nil
	}
	state := actor.Snapshot()
	if combat, isCombat := s.combatStates[id]; isCombat {
		combat.threat += amount
		if state.HP <= 0 {
			combat.threat = 0
			combat.respawnAt = now.Add(combat.respawnAfter)
			combat.leaveCombatAt = now.Add(5 * time.Second)
		}
		s.combatStates[id] = combat
	}
	frame, err := sceneObjectProperties(uint32(id), meta.ownerID, 0, npcCombatProperties(state))
	loot := int32(0)
	if state.HP <= 0 {
		loot = state.MaxHP / 100
		if loot <= 0 {
			loot = 1
		}
		if loot > 99 {
			loot = 99
		}
	}
	return frame, uint32(id), loot, state.HP <= 0, err
}

type areaSkillDamageResult struct {
	frame   []byte
	id      uint32
	ownerID uint32
	loot    int32
	killed  bool
}

func (s *sceneLifecycle) applyAreaSkillDamage(center worldcore.Transform, radius, height float32, amount int32, now time.Time) ([]areaSkillDamageResult, error) {
	return s.applyShapedSkillDamage(center, radius, height, 0, 0, amount, now)
}
func (s *sceneLifecycle) applyShapedSkillDamage(center worldcore.Transform, radius, height, sectorAngle, rectangleWidth float32, amount int32, now time.Time) ([]areaSkillDamageResult, error) {
	if s == nil || radius <= 0 || amount <= 0 {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]int, 0, len(s.combatActors))
	radiusSquared := radius * radius
	for id, actor := range s.combatActors {
		meta, exists := s.entities[id]
		if actor == nil || !exists || meta.interaction != npcInteractionCombat || actor.Snapshot().HP <= 0 {
			continue
		}
		dx := meta.transform.X - center.X
		dz := meta.transform.Z - center.Z
		dy := meta.transform.Y - center.Y
		if dy < 0 {
			dy = -dy
		}
		skipReason := ""
		if height > 0 && dy > height {
			skipReason = fmt.Sprintf("height dy=%.1f>%.1f", dy, height)
		} else if rectangleWidth > 0 {
			orientation := float64(center.Orient)
			forward := float64(dx)*math.Sin(orientation) + float64(dz)*math.Cos(orientation)
			side := -float64(dx)*math.Cos(orientation) + float64(dz)*math.Sin(orientation)
			if forward < 0 || forward > float64(radius) || math.Abs(side) > float64(rectangleWidth)*0.5 {
				skipReason = fmt.Sprintf("rect forward=%.1f side=%.1f r=%.1f w=%.1f", forward, side, radius, rectangleWidth)
			}
		} else {
			distanceSquared := dx*dx + dz*dz
			if distanceSquared > radiusSquared {
				skipReason = fmt.Sprintf("dist %.1f>%.1f", float32(math.Sqrt(float64(distanceSquared))), radius)
			} else if sectorAngle > 0 {
				delta := math.Atan2(float64(dx), float64(dz)) - float64(center.Orient)
				for delta > math.Pi {
					delta -= 2 * math.Pi
				}
				for delta < -math.Pi {
					delta += 2 * math.Pi
				}
				halfAngle := float64(sectorAngle) * math.Pi / 360
				if math.Abs(delta) > halfAngle {
					skipReason = fmt.Sprintf("angle deg=%.1f>%.1f", 180*math.Abs(delta)/math.Pi, float64(sectorAngle)*0.5)
				}
			}
		}
		if skipReason != "" {
			log.Printf("area hit MISS id=%d cfg=%s center=(%.1f,%.1f,%.1f)o=%.2f npc=(%.1f,%.1f,%.1f) %s", uint32(id), meta.configID, center.X, center.Y, center.Z, center.Orient, meta.transform.X, meta.transform.Y, meta.transform.Z, skipReason)
			continue
		}
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	results := make([]areaSkillDamageResult, 0, len(ids))
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		actor := s.combatActors[id]
		meta := s.entities[id]
		if !actor.ApplyDamage(amount) {
			continue
		}
		state := actor.Snapshot()
		if combat, isCombat := s.combatStates[id]; isCombat {
			combat.threat += amount
			if state.HP <= 0 {
				combat.threat = 0
				combat.respawnAt = now.Add(combat.respawnAfter)
				combat.leaveCombatAt = now.Add(5 * time.Second)
			}
			s.combatStates[id] = combat
		}
		frame, err := sceneObjectProperties(uint32(id), meta.ownerID, 0, npcCombatProperties(state))
		if err != nil {
			return nil, err
		}
		loot := int32(0)
		if state.HP <= 0 {
			loot = state.MaxHP / 100
			s.recordQuestKill(meta.configID)
			if loot <= 0 {
				loot = 1
			}
			if loot > 99 {
				loot = 99
			}
		}
		results = append(results, areaSkillDamageResult{frame: frame, id: uint32(id), ownerID: meta.ownerID, loot: loot, killed: state.HP <= 0})
	}
	return results, nil
}
func (s *sceneLifecycle) selectedCombatTargetTransform() (worldcore.Transform, bool) {
	if s == nil {
		return worldcore.Transform{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	meta, exists := s.entities[id]
	actor := s.combatActors[id]
	return meta.transform, exists && actor != nil && actor.Snapshot().HP > 0
}
func (s *sceneLifecycle) applyNPCControl(ids []uint32, lifetime time.Duration, now time.Time) {
	if s == nil || lifetime <= 0 || len(ids) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	until := now.Add(lifetime)
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		state, exists := s.combatStates[id]
		if !exists {
			continue
		}
		if until.After(state.controlledUntil) {
			state.controlledUntil = until
			s.combatStates[id] = state
		}
	}
}

type appliedSkillBuff struct {
	slot       uint16
	staticData uint32
	expires    time.Time
	uniqueID   uint64
}

func stagePlayerSkillEffects(player *playerActor, definition combatSkillDefinition, now time.Time, remote string) []appliedSkillBuff {
	applied := make([]appliedSkillBuff, 0, 3)
	for _, effect := range definition.effects {
		var buff skillBuffDefinition
		switch effect.kind {
		case skillEffectPlayerBuff:
			buff = effect.buff
		case skillEffectStanceCycle:
			var ok bool
			buff, ok = player.nextSkillBuffVariant(effect.buffVariants, now)
			if !ok {
				continue
			}
		case skillEffectRedArmor:
			buff = skillBuffDefinition{configID: "buf_fixed_red", staticData: 8682, lifetime: definition.totalActionDuration(), preferredSlot: 1}
		case skillEffectYellowArmor:
			buff = skillBuffDefinition{configID: "buf_fixed_yel_ng", staticData: 11017, lifetime: definition.totalActionDuration(), preferredSlot: 15}
		default:
			continue
		}
		slot, state, expires, ok := player.activateSkillBuff(buff, now)
		if !ok {
			log.Printf("%s: no native BuffInfo slot available skill=%s buff=%s", remote, definition.id, buff.configID)
			continue
		}
		uniqueID := uint64(now.UnixMilli())
		applied = append(applied, appliedSkillBuff{slot: slot, staticData: buff.staticData, expires: expires, uniqueID: uniqueID})
		log.Printf("%s: staged compiled skill Buff skill=%s buff=%s property=%d value=%q expiry=%d uid=%d", remote, definition.id, buff.configID, bufferInfoPropertyIndex(slot), state, expires.UnixMilli(), uniqueID)
	}
	return applied
}
func handleSkillCustom(link sceneMessageConnection, player *playerActor, world *sceneLifecycle, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomUseSkill {
		return false, nil
	}
	definition, ok := parseUseSkillCustom(custom)
	if !ok {
		log.Printf("%s: reject malformed or unlearned USE_SKILL values=%v", remote, custom.Values)
		return true, nil
	}
	if player == nil {
		log.Printf("%s: reject skill before player spawn id=%s", remote, definition.id)
		return true, nil
	}
	level, replacementBaseID, learned := requestedSkillAuthority(player, definition.id, installedCombatSkillCatalog)
	if learned && replacementBaseID != "" {
		log.Printf("%s: authorize condition-zero replacement id=%s from learned base=%s level=%d", remote, definition.id, replacementBaseID, level)
	}
	if !learned {
		log.Printf("%s: reject unlearned installed skill id=%s", remote, definition.id)
		return true, nil
	}
	executable, ok := installedCombatSkillCatalog.definition(definition.id, level)
	if !ok {
		replacementDefinition, baseID, replacementOK := installedCombatSkillCatalog.noConditionReplacementDefinition(definition.id, level)
		if replacementOK {
			executable = replacementDefinition
			ok = true
			if replacementBaseID == "" {
				replacementBaseID = baseID
			}
			log.Printf("%s: compiled condition-zero replacement id=%s base_action=%s level=%d", remote, definition.id, baseID, level)
		}
	}
	if !ok {
		log.Printf("%s: reject learned skill without compiled level response id=%s level=%d", remote, definition.id, level)
		return true, nil
	}
	definition = executable
	if mult := player.skillElementMult(definition); mult != 1 {
		effects := append([]combatSkillEffect(nil), definition.effects...)
		for i := range effects {
			if effects[i].kind == skillEffectDamage && effects[i].amount > 0 {
				effects[i].amount = int32(math.Round(float64(effects[i].amount) * mult))
			}
		}
		definition.effects = effects
		log.Printf("%s: element affinity skill=%s attr=%s neigong=%s mult=%.2f", remote, definition.id, definition.attribute, player.currentNeiGongAttribute(), mult)
	}
	attackBonus := playerSkillAttackBonus(player)
	targetID, targetOwnerID := uint32(playerObjectID), uint32(playerOwnerID)
	if definition.requiresTarget {
		found := false
		if objectID, ownerID, hasObject := useSkillTargetObject(custom); hasObject {
			targetID, targetOwnerID, found = world.targetIfCombat(objectID, ownerID)
		}
		if !found {
			targetID, targetOwnerID, found = world.selectedCombatTarget()
		}
		if !found && definition.targetMode != skillTargetSelectedArea {
			log.Printf("%s: reject skill id=%s without living selected target", remote, definition.id)
			return true, nil
		}
		if !found {
			targetID, targetOwnerID = uint32(playerObjectID), uint32(playerOwnerID)
		}
	}
	if definition.range_ > 0 && skillTargetOutOfRange(player.authoritativePosition(), world, definition.range_) {
		castTransform, hasCastTransform := useSkillCastTransform(custom)
		if !hasCastTransform || skillTargetOutOfRange(castTransform, world, definition.range_) {
			log.Printf("%s: reject skill id=%s target out of range (max %.1fm)", remote, definition.id, definition.range_)
			return true, nil
		}
	}
	now := time.Now()
	target := uint64(targetID) | uint64(targetOwnerID)<<32
	started, reason := player.beginSkillUse(definition, target, now)
	if !started {
		log.Printf("%s: reject skill id=%s: %s", remote, definition.id, reason)
		return true, nil
	}
	appliedBuffs := stagePlayerSkillEffects(player, definition, now, remote)
	uniqueID := uint64(now.UnixMilli())
	if currentSlot := currentParryStanceSlot(definition, appliedBuffs); currentSlot != 0 {
		for _, slot := range parryStanceBuffSlots {
			if slot == currentSlot {
				continue
			}
			removed, removedOK := player.takeBuffSlot(slot, now)
			if !removedOK {
				continue
			}
			clearFrame, err := skillCurEffectClearFrame()
			if err != nil {
				return true, fmt.Errorf("encode parry stance clear: %w", err)
			}
			if err := link.WriteFrame(clearFrame); err != nil {
				return true, err
			}
			name := player.actor.Snapshot().Name
			removeFrame, err := skillBufferFrame(1, removed, name, uniqueID, 0)
			if err != nil {
				return true, fmt.Errorf("encode parry stance remove: %w", err)
			}
			if err := link.WriteFrame(removeFrame); err != nil {
				return true, err
			}
			log.Printf("%s: 架招互斥 overwrote previous taolu parry stance skill=%s slot=%d buff=%q", remote, definition.id, slot, removed)
		}
	}
	cooldownFrame, err := skillCooldownFrame(definition, now)
	if err != nil {
		return true, fmt.Errorf("encode skill cooldown %s: %w", definition.id, err)
	}
	if err := link.WriteFrame(cooldownFrame); err != nil {
		return true, err
	}
	switchFrames, err := player.taoLuSwitchCooldownFrames(definition.teamKey(), now)
	if err != nil {
		return true, fmt.Errorf("encode taolu switch cooldowns %s: %w", definition.id, err)
	}
	for _, frame := range switchFrames {
		if err := link.WriteFrame(frame); err != nil {
			return true, err
		}
	}
	vitalFrame, err := player.vitalUpdate()
	if err != nil {
		return true, err
	}
	if err := link.WriteFrame(vitalFrame); err != nil {
		return true, err
	}
	hasRedArmor := false
	for _, buff := range appliedBuffs {
		if buff.staticData != 8682 {
			continue
		}
		hasRedArmor = true
		effectFrame, err := skillCurEffectFrame(buff.staticData, definition.level, uniqueID)
		if err != nil {
			return true, fmt.Errorf("encode skill CurSkillEffect: %w", err)
		}
		if err := link.WriteFrame(effectFrame); err != nil {
			return true, err
		}
		name := player.actor.Snapshot().Name
		buffFrame, err := skillBufferFrame(0, "buf_fixed_red", name, uniqueID, 0)
		if err != nil {
			return true, fmt.Errorf("encode skill red super armor add: %w", err)
		}
		if err := link.WriteFrame(buffFrame); err != nil {
			return true, err
		}
		log.Printf("%s: sent CurSkillEffect + event-169 buf_fixed_red add skill=%s static=%X", remote, definition.id, buff.staticData)
		break
	}
	for _, buff := range appliedBuffs {
		if buff.staticData != 11017 {
			continue
		}
		effectFrame, err := skillCurEffectFrame(buff.staticData, definition.level, uniqueID)
		if err != nil {
			return true, fmt.Errorf("encode skill CurSkillEffect: %w", err)
		}
		if err := link.WriteFrame(effectFrame); err != nil {
			return true, err
		}
		name := player.actor.Snapshot().Name
		buffFrame, err := skillBufferFrame(0, "buf_fixed_yel_ng", name, uniqueID, 0)
		if err != nil {
			return true, fmt.Errorf("encode skill yellow super armor add: %w", err)
		}
		if err := link.WriteFrame(buffFrame); err != nil {
			return true, err
		}
		log.Printf("%s: sent CurSkillEffect + event-169 buf_fixed_yel_ng add skill=%s static=%X", remote, definition.id, buff.staticData)
		break
	}
	if definition.actionName != "" {
		switch definition.targetMode {
		case skillTargetSelected:
			transform, _ := useSkillCastTransform(custom)
			prepareFrame, err := skillCastPreparationFrame(transform)
			if err != nil {
				return true, fmt.Errorf("encode skill cast preparation: %w", err)
			}
			if err := link.WriteFrame(prepareFrame); err != nil {
				return true, err
			}
			if isJumpLeapSkill(definition.id) {
				if targetTransform, targetOK := world.selectedCombatTargetTransform(); targetOK {
					origin := player.authoritativePosition()
					player.setLeapOrigin(origin)
					if err := sendPlayerLocationAndVitals(link, player, role.Position{X: targetTransform.X, Y: targetTransform.Y, Z: targetTransform.Z}); err != nil {
						return true, err
					}
					log.Printf("%s: leap skill=%s moved to target (%.1f,%.1f,%.1f)", remote, definition.id, targetTransform.X, targetTransform.Y, targetTransform.Z)
				}
			}
			actionFrame, err := nativeSkillActionFrameAt(definition.id, targetID, targetOwnerID, false, 0, 0, 0)
			if err != nil {
				return true, fmt.Errorf("encode skill native action %s (%s): %w", definition.id, definition.actionName, err)
			}
			if err := link.WriteFrame(actionFrame); err != nil {
				return true, err
			}
		case skillTargetSelf:
			actionFrame, err := nativeSkillActionFrameAt(definition.id, uint32(playerObjectID), uint32(playerOwnerID), false, 0, 0, 0)
			if err != nil {
				return true, fmt.Errorf("encode skill native action %s (%s): %w", definition.id, definition.actionName, err)
			}
			if err := link.WriteFrame(actionFrame); err != nil {
				return true, err
			}
			for _, buff := range appliedBuffs {
				if buff.staticData == 8682 {
					continue
				}
				effectFrame, err := skillCurEffectFrame(buff.staticData, definition.level, uniqueID)
				if err != nil {
					return true, fmt.Errorf("encode skill CurSkillEffect: %w", err)
				}
				if err := link.WriteFrame(effectFrame); err != nil {
					return true, err
				}
				buffID := player.buffConfigID(buff.slot)
				if buffID == "" {
					continue
				}
				name := player.actor.Snapshot().Name
				buffFrame, err := skillBufferFrame(0, buffID, name, uniqueID, 0)
				if err != nil {
					return true, fmt.Errorf("encode skill buff add: %w", err)
				}
				if err := link.WriteFrame(buffFrame); err != nil {
					return true, err
				}
				skillFrame, err := skillEffectFrame(definition.id, uint32(playerObjectID), uint32(playerOwnerID))
				if err != nil {
					return true, fmt.Errorf("encode skill effect %s: %w", definition.id, err)
				}
				if err := link.WriteFrame(skillFrame); err != nil {
					return true, err
				}
				log.Printf("%s: sent CurSkillEffect + event-169 + event-205 for self skill=%s buff=%s static=%X", remote, definition.id, buffID, buff.staticData)
				break
			}
		case skillTargetSelectedArea:
			if hasRedArmor {
				actionFrame, err := nativeSkillActionFrameAt(definition.id, 0, 0, false, 0, 0, 0)
				if err != nil {
					return true, fmt.Errorf("encode skill native action %s (%s): %w", definition.id, definition.actionName, err)
				}
				if err := link.WriteFrame(actionFrame); err != nil {
					return true, err
				}
				break
			}
			transform, _ := useSkillCastTransform(custom)
			prepareFrame, err := skillCastPreparationFrame(transform)
			if err != nil {
				return true, fmt.Errorf("encode skill cast preparation: %w", err)
			}
			if err := link.WriteFrame(prepareFrame); err != nil {
				return true, err
			}
			tx, ty, tz, hasPos := useSkillDragTarget(custom)
			actionFrame, err := nativeSkillActionFrameAt(definition.id, 0, 0, hasPos, tx, ty, tz)
			if err != nil {
				return true, fmt.Errorf("encode skill native action %s (%s): %w", definition.id, definition.actionName, err)
			}
			if err := link.WriteFrame(actionFrame); err != nil {
				return true, err
			}
			log.Printf("%s: sent drag-area skill action skill=%s pos=(%.2f,%.2f,%.2f) has_pos=%t", remote, definition.id, tx, ty, tz, hasPos)
		case skillTargetCasterCircle, skillTargetCasterSector, skillTargetCasterRectangle:
			if !hasRedArmor {
				transform, _ := useSkillCastTransform(custom)
				prepareFrame, err := skillCastPreparationFrame(transform)
				if err != nil {
					return true, fmt.Errorf("encode skill cast preparation: %w", err)
				}
				if err := link.WriteFrame(prepareFrame); err != nil {
					return true, err
				}
			}
			actionFrame, err := nativeSkillActionFrameAt(definition.id, 0, 0, false, 0, 0, 0)
			if err != nil {
				return true, fmt.Errorf("encode skill native action %s (%s): %w", definition.id, definition.actionName, err)
			}
			if err := link.WriteFrame(actionFrame); err != nil {
				return true, err
			}
		}
		log.Printf("%s: started verified State skill action skill=%s action=%s followups=%d total=%s caster=%d-%d target=%d-%d", remote, definition.id, definition.actionName, len(definition.followupActions), definition.totalActionDuration(), playerObjectID, playerOwnerID, targetID, targetOwnerID)
	}
	followupDamage := int32(0)
	for _, effect := range definition.effects {
		if effect.kind == skillEffectDamage && effect.amount > 0 {
			followupDamage = effect.amount + attackBonus
			break
		}
	}
	if len(definition.followupActions) > 0 && definition.targetMode == skillTargetSelected {
		scheduleSkillFollowupActions(link, player, world, definition, targetID, targetOwnerID, followupDamage)
	}
	if len(definition.hitFrames) > 0 {
		center, hasCenter := skillAreaCastCenter(custom, definition, world)
		scheduleSkillHitFrames(link, player, world, definition, targetID, targetOwnerID, followupDamage, center, hasCenter, appliedBuffs)
	}
	if definition.targetMode == skillTargetSelected && targetID != 0 && targetID != uint32(playerObjectID) {
		for _, effect := range definition.effects {
			if effect.kind == skillEffectDamage && effect.amount > 0 {
				player.enterCombatPresence()
				break
			}
		}
	}
	hitIDs := make([]uint32, 0, 8)
	var damageTargetID uint32
	damageTargetCount := 0
	var totalLoot int32
	if len(definition.hitFrames) > 0 {
		for _, effect := range definition.effects {
			if effect.kind == skillEffectTargetControl {
				log.Printf("%s: compiled target control skill=%s skipped (multi-hit segment-driven) buff=%s", remote, definition.id, effect.buff.configID)
			}
		}
	} else {
		for _, effect := range definition.effects {
			if effect.kind != skillEffectDamage || effect.amount <= 0 || world == nil {
				continue
			}
			damage := effect.amount + attackBonus
			switch definition.targetMode {
			case skillTargetSelected:
				targetFrame, id, loot, killed, err := world.applySelectedSkillDamage(damage, now)
				if err != nil {
					return true, err
				}
				if targetFrame == nil {
					continue
				}
				player.enterCombatPresence()
				hitIDs = append(hitIDs, id)
				if err := link.WriteFrame(targetFrame); err != nil {
					return true, err
				}
				if err := writeSkillHitEvents(link, player, world, definition, id, targetOwnerID, damage, false); err != nil {
					return true, fmt.Errorf("write skill hit events: %w", err)
				}
				for _, buff := range appliedBuffs {
					if buff.staticData == 8682 {
						continue
					}
					effectFrame, err := skillCurEffectFrame(buff.staticData, definition.level, uniqueID)
					if err != nil {
						return true, fmt.Errorf("encode skill CurSkillEffect: %w", err)
					}
					if err := link.WriteFrame(effectFrame); err != nil {
						return true, err
					}
					buffID := player.buffConfigID(buff.slot)
					if buffID == "" {
						break
					}
					name := player.actor.Snapshot().Name
					buffFrame, err := skillBufferFrame(0, buffID, name, uniqueID, 0)
					if err != nil {
						return true, fmt.Errorf("encode skill buff add: %w", err)
					}
					if err := link.WriteFrame(buffFrame); err != nil {
						return true, err
					}
				}
				skillFrame, err := skillEffectFrame(definition.id, id, targetOwnerID)
				if err != nil {
					return true, fmt.Errorf("encode skill effect %s: %w", definition.id, err)
				}
				if err := link.WriteFrame(skillFrame); err != nil {
					return true, err
				}
				damageTargetCount++
				damageTargetID = id
				if killed {
					totalLoot += loot
				}
			case skillTargetCasterCircle, skillTargetCasterSector, skillTargetCasterRectangle, skillTargetSelectedArea:
				center, hasCenter := useSkillCastTransform(custom)
				if definition.targetMode == skillTargetSelectedArea {
					castCenterOK := hasCenter
					tx, ty, tz, hasDragTarget := useSkillDragTarget(custom)
					if hasDragTarget {
						center = worldcore.Transform{X: tx, Y: ty, Z: tz}
						hasCenter = castCenterOK
					} else {
						center, hasCenter = world.selectedCombatTargetTransform()
					}
				}
				if !hasCenter {
					log.Printf("%s: compiled area skill id=%s has no center; no targets settled", remote, definition.id)
					continue
				}
				angle := float32(0)
				width := float32(0)
				if definition.targetMode == skillTargetCasterSector {
					angle = definition.sectorAngle
				} else if definition.targetMode == skillTargetCasterRectangle {
					width = definition.areaWidth
				}
				results, err := world.applyShapedSkillDamage(center, definition.areaRadius, definition.areaHeight, angle, width, damage, now)
				if err != nil {
					return true, err
				}
				for _, result := range results {
					player.enterCombatPresence()
					if damageTargetID == 0 {
						damageTargetID = result.id
					}
					damageTargetCount++
					hitIDs = append(hitIDs, result.id)
					if err := link.WriteFrame(result.frame); err != nil {
						return true, err
					}
					if err := writeSkillHitEvents(link, player, world, definition, result.id, result.ownerID, damage, false); err != nil {
						return true, fmt.Errorf("write area skill hit events: %w", err)
					}
					skillFrame, err := skillEffectFrame(definition.id, result.id, result.ownerID)
					if err != nil {
						return true, fmt.Errorf("encode area skill effect %s: %w", definition.id, err)
					}
					if err := link.WriteFrame(skillFrame); err != nil {
						return true, err
					}
					if result.killed {
						totalLoot += result.loot
					}
				}
				log.Printf("%s: settled compiled area skill id=%s mode=%d center=(%.2f,%.2f,%.2f) radius=%.1f angle=%.1f width=%.1f hits=%d damage_each=%d", remote, definition.id, definition.targetMode, center.X, center.Y, center.Z, definition.areaRadius, angle, width, len(results), damage)
			}
		}
	}
	for _, effect := range definition.effects {
		if effect.kind != skillEffectTargetControl {
			continue
		}
		world.applyNPCControl(hitIDs, effect.buff.lifetime, now)
		casterName := player.actor.Snapshot().Name
		for _, hitID := range hitIDs {
			ownerID, ok := world.npcOwnerID(hitID)
			if !ok {
				continue
			}
			buffFrame, err := skillTargetBufferFrame(0, effect.buff.configID, hitID, ownerID, casterName, uniqueID)
			if err != nil {
				return true, fmt.Errorf("encode target DBUFF %s: %w", effect.buff.configID, err)
			}
			if err := link.WriteFrame(buffFrame); err != nil {
				return true, err
			}
		}
		log.Printf("%s: settled compiled target control skill=%s buff=%s targets=%d lifetime=%s (+target event-169)", remote, definition.id, effect.buff.configID, len(hitIDs), effect.buff.lifetime)
	}
	if totalLoot > 0 {
		player.addSilver(totalLoot)
		silverFrame, err := player.silverUpdate()
		if err != nil {
			return true, err
		}
		if err := link.WriteFrame(silverFrame); err != nil {
			return true, err
		}
		log.Printf("%s: skill defeated NPCs targets=%d silver_drop=%d", remote, damageTargetCount, totalLoot)
	}
	for _, buff := range appliedBuffs {
		schedulePlayerBuffExpiry(link, player, buff.slot, buff.staticData, buff.expires)
	}
	scheduleCurrentSkillClear(link, player, definition, appliedBuffs)
	state := player.actor.Snapshot()
	sp := player.skillSP()
	log.Printf("%s: accepted skill id=%s mp=%d sp=%d qg=%d damage=%d target=%d targets=%d personal_cd=%s public_cd=%s", remote, definition.id, state.MP, sp, state.QingGongPoint, definition.baseDamage, damageTargetID, damageTargetCount, definition.personalCD, definition.publicCD)
	return true, nil
}
func scheduleSkillFollowupActions(conn sceneMessageConnection, player *playerActor, world *sceneLifecycle, definition combatSkillDefinition, targetID, targetOwnerID uint32, damage int32) {
	for _, segment := range definition.followupActions {
		delay := segment.at
		if delay < 0 {
			delay = 0
		}
		time.AfterFunc(delay, func() {
			frame, err := playerActionStateFrame(segment.action)
			if err != nil {
				log.Printf("skill follow-up State encode id=%s action=%s: %v", definition.id, segment.action, err)
				return
			}
			if err := conn.WriteFrame(frame); err != nil {
				log.Printf("skill follow-up State write id=%s action=%s: %v", definition.id, segment.action, err)
				return
			}
			log.Printf("skill follow-up State sent id=%s action=%s at=%s", definition.id, segment.action, delay)
			applyOffensiveSegmentDamage(conn, player, world, definition, targetID, targetOwnerID, damage, delay, worldcore.Transform{}, false)
		})
	}
}
func scheduleCurrentSkillClear(conn sceneMessageConnection, player *playerActor, definition combatSkillDefinition, appliedBuffs []appliedSkillBuff) {
	delay := definition.totalActionDuration()
	if delay <= 0 {
		delay = time.Millisecond
	}
	time.AfterFunc(delay, func() {
		if !player.clearCurrentSkill(definition.id) {
			return
		}
		if _, ok := jumpLeapSkillIDs[definition.id]; ok {
			if origin, ok := player.takeLeapOrigin(); ok {
				err := sendPlayerLocationAndVitals(conn, player, role.Position{X: origin.X, Y: origin.Y, Z: origin.Z})
				if err != nil {
					log.Printf("skill leap return write id=%s: %v", definition.id, err)
				} else {
					log.Printf("sent leap return id=%s origin=(%.1f,%.1f,%.1f)", definition.id, origin.X, origin.Y, origin.Z)
				}
			}
		}
		for _, buff := range appliedBuffs {
			if buff.staticData == 8682 {
				effectFrame, effectErr := skillCurEffectClearFrame()
				if effectErr != nil {
					log.Printf("skill CurSkillEffect clear encode id=%s: %v", definition.id, effectErr)
				} else if err := conn.WriteFrame(effectFrame); err != nil {
					log.Printf("skill CurSkillEffect clear write id=%s: %v", definition.id, err)
				}
				name := player.actor.Snapshot().Name
				buffFrame, buffErr := skillBufferFrame(1, "buf_fixed_red", name, buff.uniqueID, 0)
				if buffErr != nil {
					log.Printf("skill red super armor remove encode id=%s: %v", definition.id, buffErr)
				} else if err := conn.WriteFrame(buffFrame); err != nil {
					log.Printf("skill red super armor remove write id=%s: %v", definition.id, err)
				} else {
					log.Printf("sent CurSkillEffect clear + event-169 buf_fixed_red remove id=%s", definition.id)
				}
			}
			if buff.staticData == 11017 {
				effectFrame, effectErr := skillCurEffectClearFrame()
				if effectErr != nil {
					log.Printf("skill CurSkillEffect clear encode id=%s: %v", definition.id, effectErr)
				} else if err := conn.WriteFrame(effectFrame); err != nil {
					log.Printf("skill CurSkillEffect clear write id=%s: %v", definition.id, err)
				}
				name := player.actor.Snapshot().Name
				buffFrame, buffErr := skillBufferFrame(1, "buf_fixed_yel_ng", name, buff.uniqueID, 0)
				if buffErr != nil {
					log.Printf("skill yellow super armor remove encode id=%s: %v", definition.id, buffErr)
				} else if err := conn.WriteFrame(buffFrame); err != nil {
					log.Printf("skill yellow super armor remove write id=%s: %v", definition.id, err)
				} else {
					log.Printf("sent CurSkillEffect clear + event-169 buf_fixed_yel_ng remove id=%s", definition.id)
				}
			}
		}
		update, err := player.currentSkillClearUpdate()
		if err != nil {
			log.Printf("skill state clear encode id=%s: %v", definition.id, err)
			return
		}
		if err := conn.WriteFrame(update); err != nil {
			log.Printf("skill state clear write id=%s: %v", definition.id, err)
			return
		}
		stand, err := playerActionStateFrame("stand")
		if err != nil {
			log.Printf("skill stand restore encode id=%s: %v", definition.id, err)
			return
		}
		if err := conn.WriteFrame(stand); err != nil {
			log.Printf("skill stand restore write id=%s: %v", definition.id, err)
			return
		}
		log.Printf("skill state cleared and stand restored id=%s total_action=%s", definition.id, delay)
	})
}
