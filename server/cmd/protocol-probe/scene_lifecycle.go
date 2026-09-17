package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

type sceneMessageConnection interface{ WriteFrame([]byte) error }
type sceneLifecycle struct {
	remote              string
	conn                sceneMessageConnection
	mu                  sync.Mutex
	armed               bool
	combatActive        bool
	scene               *worldcore.Scene
	viewport            *worldcore.Viewport
	entities            map[worldcore.EntityID]sceneEntity
	combatActors        map[worldcore.EntityID]*entity.Actor
	combatStates        map[worldcore.EntityID]npcCombatState
	closed              bool
	epoch               uint64
	timers              []*time.Timer
	npcFuncs            *npcfunc.Registry
	patrolStarted       bool
	pathDir             string
	patrolRoutes        []patrolRoute
	mapPathDir          string
	mapPathRoutes       map[string][]mapPathPoint
	selectedObject      uint32
	selectedOwner       uint32
	lastMenuObject      uint32
	lastMenuOwner       uint32
	lastMenuAt          time.Time
	lastMenuMovie       bool
	activeShop          currentShopExchangeSession
	portalCooldown      time.Time
	transportHandler    func(transPathRec) error
	transportMenuActive bool
	transportMenu       []transPathRec
	registry            *sceneNPCRegistry
	quests              *questRuntime
	taskNPCCount        uint32
	player              *playerActor
	sceneConfig         string
	sceneResource       string
}
type sceneEntity struct {
	id             uint32
	description    string
	payload        func() []byte
	ownerID        uint32
	transform      worldcore.Transform
	properties     []clientdata.IndexedProperty
	patrolSign     float32
	configID       string
	scriptClass    string
	interaction    npcInteraction
	funcs          []npcfunc.Binding
	services       []npcService
	portal         *scenePortalRoute
	fixedRole      bool
	npcType        int32
	desc           string
	pathID         string
	patrolAssigned bool
}
type npcInteraction uint8

const (
	npcInteractionSelectOnly npcInteraction = iota
	npcInteractionTalk
	npcInteractionCombat
)

func (n npcInteraction) String() string {
	switch n {
	case npcInteractionTalk:
		return "talk"
	case npcInteractionCombat:
		return "combat"
	default:
		return "select-only"
	}
}

type npcService struct {
	mark   uint16
	label  string
	value  string
	source string
}

const (
	markShop      uint16 = 0x1001
	markDepot     uint16 = 0x1002
	markTransport uint16 = 0x1003
	markJob       uint16 = 0x1004
	markGem       uint16 = 0x1005
	markGuild     uint16 = 0x1006
	markHome      uint16 = 0x1007
	markTask      uint16 = 0x1008
)

type sceneLocationReplay struct {
	id      uint32
	payload []byte
}

var actor2LocationReplayDelays = []time.Duration{250 * time.Millisecond, 750 * time.Millisecond, 1500 * time.Millisecond, 3 * time.Second, 5 * time.Second}

func newSceneLifecycle(remote string, conn sceneMessageConnection, registries ...*npcfunc.Registry) *sceneLifecycle {
	var funcs *npcfunc.Registry
	if len(registries) > 0 {
		funcs = registries[0]
	}
	return &sceneLifecycle{remote: remote, conn: conn, scene: worldcore.NewScene(), viewport: worldcore.NewViewport(), entities: make(map[worldcore.EntityID]sceneEntity), combatActors: make(map[worldcore.EntityID]*entity.Actor), combatStates: make(map[worldcore.EntityID]npcCombatState), npcFuncs: funcs, quests: newQuestRuntime()}
}
func (s *sceneLifecycle) begin(config, resource string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.sceneConfig = config
	s.sceneResource = resource
	if _, err := s.scene.Begin(config, resource); err != nil {
		return err
	}
	clear(s.entities)
	clear(s.combatActors)
	clear(s.combatStates)
	s.armed = false
	s.combatActive = false
	s.patrolStarted = false
	s.selectedObject = 0
	s.selectedOwner = 0
	s.lastMenuObject = 0
	s.lastMenuOwner = 0
	s.lastMenuAt = time.Time{}
	s.lastMenuMovie = false
	s.activeShop = currentShopExchangeSession{}
	s.epoch++
	s.stopReplayTimersLocked()
	if s.pathDir != "" && resource != "" {
		routes, routeErr := loadScenePatrolRoutes(s.pathDir, resource)
		if routeErr != nil {
			log.Printf("%s: patrol routes for scene %q unavailable: %v", s.remote, resource, routeErr)
			s.patrolRoutes = nil
		} else {
			s.patrolRoutes = routes
			total := 0
			for _, route := range routes {
				total += len(route.Points)
			}
			if len(routes) > 0 {
				log.Printf("%s: loaded %d patrol route(s) for scene %q (%d waypoints)", s.remote, len(routes), resource, total)
			}
		}
	} else {
		s.patrolRoutes = nil
	}
	if s.mapPathDir != "" && resource != "" {
		mapRoutes, mapErr := loadSceneMapPathRoutes(s.mapPathDir, resource)
		if mapErr != nil {
			log.Printf("%s: mappath routes for scene %q unavailable: %v", s.remote, resource, mapErr)
			s.mapPathRoutes = nil
		} else {
			s.mapPathRoutes = mapRoutes
		}
	} else {
		s.mapPathRoutes = nil
	}
	return nil
}
func (s *sceneLifecycle) registerNPC(id uint32, npc npcSpawn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == 0 || id == 1 {
		return fmt.Errorf("NPC scene identity %d collides with reserved player identity", id)
	}
	npc.synchronizeTransform()
	orderedProperties, err := npc.resolved.OrderedProperties(clientdata.VisibleNPCModernV1())
	if err != nil {
		return fmt.Errorf("resolve NPC %s properties: %w", npc.resolved.ConfigID, err)
	}
	payload, err := npcAddObject(id, npc, 0)
	if err != nil {
		return fmt.Errorf("encode NPC %s: %w", npc.resolved.ConfigID, err)
	}
	properties := make(worldcore.PropertyArchive)
	for _, property := range orderedProperties {
		value, err := worldValue(property.Value)
		if err != nil {
			return fmt.Errorf("convert NPC %s property %s: %w", npc.resolved.ConfigID, property.Name, err)
		}
		if err := properties.Set(property.Name, value); err != nil {
			return err
		}
	}
	entityID := worldcore.EntityID(id)
	if err := s.scene.Add(worldcore.Entity{ID: entityID, OwnerID: 0, Kind: worldcore.EntityNPC, Transform: worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient}, Properties: properties}); err != nil {
		return err
	}
	services := modernNPCServices(npc)
	portal, hasPortal, portalErr := doorPortalRoute(npc)
	if portalErr != nil {
		return fmt.Errorf("resolve portal %s: %w", npc.resolved.ConfigID, portalErr)
	}
	funcs := s.npcFuncs.FuncsForNPC(npc.resolved.ConfigID)
	interaction := npcInteractionFor(npc, services)
	sceneMeta := sceneEntity{id: id, description: "npc:" + npc.resolved.ConfigID, payload: func() []byte {
		return append([]byte(nil), payload...)
	}, ownerID: 1, transform: worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient}, properties: cloneIndexedProperties(orderedProperties), configID: npc.resolved.ConfigID, scriptClass: npc.resolved.ScriptClass, interaction: interaction, funcs: funcs, services: services, fixedRole: isFixedRoleNPC(npc), npcType: npc.resolved.Properties["NpcType"].Value.I32, desc: npc.resolved.Extensions["creator.desc"], pathID: npcPatrolPathID(npc)}
	if hasPortal {
		sceneMeta.portal = &portal
	}
	s.entities[entityID] = sceneMeta
	name := npc.resolved.ConfigID
	maxHP := int32(1000)
	maxMP := int32(100)
	for _, property := range orderedProperties {
		switch property.Name {
		case "Name":
			if property.Value.Text != "" {
				name = property.Value.Text
			}
		case "MaxHP":
			if property.Value.I32 > 0 {
				maxHP = property.Value.I32
			}
		case "MaxMP":
			if property.Value.I32 > 0 {
				maxMP = property.Value.I32
			}
		}
	}
	actor := entity.NewActor(id, 1, name, maxHP, maxMP)
	s.combatActors[entityID] = actor
	if interaction == npcInteractionCombat {
		s.combatStates[entityID] = newNPCCombatState(npc, sceneMeta.transform)
	}
	return nil
}
func (s *sceneLifecycle) portalRoute(request clientObjectRequest) (scenePortalRoute, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return scenePortalRoute{}, false, nil
	}
	entity, exists := s.entities[worldcore.EntityID(request.ObjectID)]
	if !exists {
		return scenePortalRoute{}, false, fmt.Errorf("unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if request.OwnerID != 0 && request.OwnerID != entity.ownerID {
		return scenePortalRoute{}, false, fmt.Errorf("unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if entity.portal != nil {
		return *entity.portal, true, nil
	}
	return scenePortalRoute{}, false, nil
}
func (s *sceneLifecycle) portalAt(position role.Position, now time.Time) (scenePortalRoute, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || now.Before(s.portalCooldown) {
		return scenePortalRoute{}, false
	}
	for _, entity := range s.entities {
		if entity.portal == nil {
			continue
		}
		route := *entity.portal
		dx := position.X - route.trigger.X
		dz := position.Z - route.trigger.Z
		if dx*dx+dz*dz > route.triggerRadius*route.triggerRadius {
			continue
		}
		s.portalCooldown = now.Add(2 * time.Second)
		return route, true
	}
	return scenePortalRoute{}, false
}
func npcInteractionFor(npc npcSpawn, services []npcService) npcInteraction {
	class := strings.ToLower(strings.TrimSpace(npc.resolved.ScriptClass))
	switch class {
	case "bossnpc", "attacknpc":
		return npcInteractionCombat
	case "door", "beimg", "boxnpc", "cannpc", "gather", "weapon", "doornpc", "minenpc", "blocknpc", "eventnpc", "tablenpc", "farmmanage", "machinenpc", "triggernpc", "adventuredoor", "teamleadernpc", "escortobserver":
		return npcInteractionSelectOnly
	}
	if npcHasCombatSkill(npc) {
		return npcInteractionCombat
	}
	return npcInteractionTalk
}
func (s *sceneLifecycle) unregister(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entityID := worldcore.EntityID(id)
	if err := s.scene.Remove(entityID); err != nil {
		return err
	}
	if s.activeShop.NPCObjectID == id {
		s.activeShop = currentShopExchangeSession{}
	}
	delete(s.entities, entityID)
	delete(s.combatActors, entityID)
	delete(s.combatStates, entityID)
	return nil
}
func (s *sceneLifecycle) clientReady() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.armed = true
	log.Printf("%s: ClientReady; armed scene objects pending post-ready client activity", s.remote)
	return nil
}
func (s *sceneLifecycle) clientActivity() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.armed {
		return nil
	}
	log.Printf("%s: post-ready client activity; materializing scene objects", s.remote)
	if _, err := s.reconcileViewportLocked(); err != nil {
		return err
	}
	s.armed = false
	s.combatActive = true
	return nil
}
func (s *sceneLifecycle) reconcileViewport() error {
	s.mu.Lock()
	newPatrol, err := s.reconcileViewportLocked()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	for _, id := range newPatrol {
		s.assignPatrolForEntity(id)
	}
	return nil
}
func (s *sceneLifecycle) reconcileViewportLocked() ([]worldcore.EntityID, error) {
	delta := s.viewport.Plan(s.scene.Snapshot())
	for _, update := range delta.Updates {
		entity, exists := s.entities[update.Entity.ID]
		if !exists {
			return nil, fmt.Errorf("missing property encoder for scene object %d", update.Entity.ID)
		}
		properties := sceneObjectTransformProperties(entity.id, entity.ownerID, entity.transform)
		if err := s.conn.WriteFrame(properties); err != nil {
			return nil, fmt.Errorf("update scene object %d: %w", update.Entity.ID, err)
		}
		log.Printf("%s: updated scene object transform id=%d kind=%s", s.remote, entity.id, entity.description)
	}
	if delta.PreviousGeneration == delta.Generation {
		for _, removal := range delta.Removes {
			payload := sceneRemoveObject(uint32(removal.Entity.ID), uint32(removal.Entity.OwnerID))
			if err := s.conn.WriteFrame(payload); err != nil {
				return nil, fmt.Errorf("remove scene object %d: %w", removal.Entity.ID, err)
			}
		}
	}
	sort.Slice(delta.Adds, func(i, j int) bool {
		return delta.Adds[i].Entity.ID < delta.Adds[j].Entity.ID
	})
	if len(delta.Adds) != 0 || len(delta.Removes) != 0 {
		log.Printf("%s: scene viewport delta adds=%d removes=%d", s.remote, len(delta.Adds), len(delta.Removes))
	}
	// LATEST-CLIENT COMPATIBILITY A/B ONLY. This is deliberately not part of
	// the exact-current-EXE authority reconstruction. Historical successful
	// LIVE captures show that the client receives an immediate ServerLocation
	// after each newly materialized Actor2 and five delayed location replays
	// before it emits the final stage_main ClientReady. The September live
	// client reaches the same pre-ready traffic but stalls before that boundary.
	locationReplays := make([]sceneLocationReplay, 0, len(delta.Adds))
	newPatrol := make([]worldcore.EntityID, 0, len(delta.Adds))
	for _, addition := range delta.Adds {
		id := addition.Entity.ID
		entity, exists := s.entities[id]
		if !exists {
			return nil, fmt.Errorf("missing payload encoder for scene object %d", id)
		}
		payload := entity.payload()
		digest := sha256.Sum256(payload)
		log.Printf("%s: materialize scene object id=%d kind=%s opcode=0x%02X payload_len=%d payload_sha256=%x", s.remote, id, entity.description, payload[0], len(payload), digest)
		if err := s.conn.WriteFrame(payload); err != nil {
			return nil, fmt.Errorf("add scene object %d: %w", id, err)
		}
		if entity.ownerID != 0 {
			location := serverLocation(entity.id, entity.ownerID, entity.transform)
			if err := s.conn.WriteFrame(location); err != nil {
				return nil, fmt.Errorf("latest-client compat locate scene object %d: %w", id, err)
			}
			locationReplays = append(locationReplays, sceneLocationReplay{
				id: uint32(id), payload: append([]byte(nil), location...),
			})
		}
		log.Printf("%s: sent modern scene object id=%d via ServerAddObject + latest-client location compatibility", s.remote, id)
		if patrolCandidateMeta(entity) && !entity.patrolAssigned {
			newPatrol = append(newPatrol, id)
		}
	}
	if err := s.viewport.Commit(delta); err != nil {
		return nil, err
	}
	if len(locationReplays) != 0 {
		s.scheduleLocationReplaysLocked(locationReplays)
		log.Printf("%s: latest-client scene compatibility armed location replays objects=%d delays=%v", s.remote, len(locationReplays), actor2LocationReplayDelays)
	}
	return newPatrol, nil
}
func (s *sceneLifecycle) objectRequest(request clientObjectRequest) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", nil
	}
	// A new NPC interaction invalidates the previous shop, even when the
	// client keeps an old shop window open. No speculative timeout is used.
	s.activeShop = currentShopExchangeSession{}
	entity, exists := s.entities[worldcore.EntityID(request.ObjectID)]
	if !exists {
		return "", fmt.Errorf("unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if request.OwnerID != 0 && request.OwnerID != entity.ownerID {
		return "", fmt.Errorf("unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if s.lastMenuObject == entity.id && s.lastMenuOwner == entity.ownerID && time.Since(s.lastMenuAt) < time.Second {
		log.Printf("%s: coalesced duplicate object request sequence=%d id=%d owner=%d", s.remote, request.Sequence, entity.id, entity.ownerID)
		return entity.description, nil
	}
	if err := s.conn.WriteFrame(sceneObjectObjectProperty(playerObjectID, playerOwnerID, uint32(propLastObject), entity.id, entity.ownerID)); err != nil {
		return "", fmt.Errorf("set player LastObject=%d-%d: %w", entity.id, entity.ownerID, err)
	}
	s.selectedObject = entity.id
	s.selectedOwner = entity.ownerID
	if entity.interaction != npcInteractionTalk {
		log.Printf("%s: object request sequence=%d id=%d config=%s script_class=%s interaction=%s; selected without dialogue", s.remote, request.Sequence, entity.id, entity.configID, entity.scriptClass, entity.interaction)
		return entity.description, nil
	}
	items := talkMenuItems(playableNPCServices(entity.services), nil)
	accept, submit := questsForNPC(entity.configID)
	if len(accept) != 0 || len(submit) != 0 {
		items = appendQuestMenuItems(items, accept, submit)
	}
	frames, err := encodeCinematicTalkMenu(entity.id, entity.ownerID, "", items)
	if err != nil {
		return "", fmt.Errorf("encode cinematic menu for %s: %w", sceneIdent(entity.id, entity.ownerID), err)
	}
	for _, frame := range frames {
		if err := s.conn.WriteFrame(frame); err != nil {
			return "", fmt.Errorf("send cinematic menu for %s: %w", sceneIdent(entity.id, entity.ownerID), err)
		}
	}
	s.lastMenuObject = entity.id
	s.lastMenuOwner = entity.ownerID
	s.lastMenuAt = time.Now()
	s.lastMenuMovie = true
	log.Printf("%s: object request sequence=%d id=%d config=%s script_class=%s interaction=%s services=%v funcs=%v; sent numeric cinematic talk frames=%d ident=%s", s.remote, request.Sequence, entity.id, entity.configID, entity.scriptClass, entity.interaction, entity.services, npcFuncIDs(entity.funcs), len(frames), sceneIdent(entity.id, entity.ownerID))
	return entity.description, nil
}
func (s *sceneLifecycle) openDepotLocked() error {
	frame := serverCreateView(serverViewSpec{ID: 4, Capacity: 18})
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("create depot view: %w", err)
	}
	return nil
}
func (s *sceneLifecycle) openShopLocked(shopID string) error {
	items, shopType, pageCount, err := shopCatalogItems(defaultShopINIPath, shopID)
	if err != nil {
		return err
	}
	frame, err := serverCreateViewWithProperties(serverViewSpec{ID: 61, Capacity: 100}, latestClientShopViewProperties(shopID, shopType, pageCount))
	if err != nil {
		return fmt.Errorf("encode shop view %q: %w", shopID, err)
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("create shop view %q: %w", shopID, err)
	}
	for _, item := range items {
		if !currentShopItemVisibleInView(item) {
			continue
		}
		objectIndex, ok := currentShopViewObjectIndex(item)
		if !ok {
			return fmt.Errorf("shop %q item %q has invalid current-client page=%d position=%d", shopID, item.configID, item.page, item.position)
		}
		properties := latestClientShopItemProperties(item)
		itemFrame, encodeErr := serverViewAdd(61, objectIndex, properties)
		if encodeErr != nil {
			return fmt.Errorf("encode shop %q item %q: %w", shopID, item.configID, encodeErr)
		}
		if writeErr := s.conn.WriteFrame(itemFrame); writeErr != nil {
			return fmt.Errorf("add shop %q item %q: %w", shopID, item.configID, writeErr)
		}
	}
	return nil
}

type talkMenuItem struct {
	funcID int32
	textID string
}

const transportMenuFuncBase int32 = 860001000

func transportMenuFuncID(index int) int32 {
	return transportMenuFuncBase + int32(index)
}

func sceneIdent(objectID uint32, ownerID uint32) string {
	return fmt.Sprintf("%d-%d", objectID, ownerID)
}
func (s *sceneLifecycle) sendTalkMenuLocked(objectID, ownerID uint32, items []serverMenuItem) error {
	frame, err := serverMenu(objectID, ownerID, items)
	if err != nil {
		return fmt.Errorf("encode 0x1C SERVER_MENU for %s: %w", sceneIdent(objectID, ownerID), err)
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send 0x1C SERVER_MENU for %s: %w", sceneIdent(objectID, ownerID), err)
	}
	return nil
}
func encodeCinematicTalkMenu(objectID, ownerID uint32, titleID string, items []talkMenuItem) ([][]byte, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("talk menu has no items")
	}
	out := make([][]byte, 0, 3+len(items))
	begin, err := serverCustomIntMessage(4)
	if err != nil {
		return nil, err
	}
	out = append(out, begin)
	if titleID != "" {
		title, err := serverCustomIntMessage(5, customString(titleID))
		if err != nil {
			return nil, err
		}
		out = append(out, title)
	}
	for _, item := range items {
		pkt, err := serverCustomIntMessage(6, customInt(item.funcID), customString(item.textID))
		if err != nil {
			return nil, err
		}
		out = append(out, pkt)
	}
	end, err := serverCustomIntMessage(7, customObject(objectID, ownerID))
	if err != nil {
		return nil, err
	}
	out = append(out, end)
	return out, nil
}
func (s *sceneLifecycle) selectNPCMenu(request clientObjectRequest) error {
	s.mu.Lock()
	entity, exists := s.entities[worldcore.EntityID(request.ObjectID)]
	if !exists || (request.OwnerID != 0 && request.OwnerID != entity.ownerID) {
		s.mu.Unlock()
		return fmt.Errorf("select unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if s.lastMenuObject != entity.id || s.lastMenuOwner != entity.ownerID {
		s.mu.Unlock()
		return fmt.Errorf("select menu mark=%d without active menu for id=%d-%d", request.FuncID, entity.id, entity.ownerID)
	}
	if s.lastMenuMovie {
		action, err := s.selectCinematicTalkLocked(entity, request)
		s.mu.Unlock()
		if err != nil {
			return err
		}
		if action != nil {
			return s.performQuestMenuAction(*action)
		}
		return nil
	}
	if uint32(request.FuncID) > 0xffff {
		s.mu.Unlock()
		return fmt.Errorf("invalid menu mark=%d for id=%d-%d", request.FuncID, entity.id, entity.ownerID)
	}
	mark := uint16(request.FuncID)
	if mark == 0 {
		if err := s.clearMenuLocked(); err != nil {
			s.mu.Unlock()
			return err
		}
		s.lastMenuObject = 0
		s.lastMenuOwner = 0
		s.lastMenuAt = time.Time{}
		s.lastMenuMovie = false
		log.Printf("%s: NPC menu dismissed id=%d config=%s", s.remote, entity.id, entity.configID)
		s.mu.Unlock()
		return nil
	}
	var service npcService
	found := false
	for _, candidate := range entity.services {
		if candidate.mark == mark {
			service = candidate
			found = true
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return fmt.Errorf("NPC %s has no implemented modern service for mark=%d", entity.configID, mark)
	}
	if err := s.clearMenuLocked(); err != nil {
		s.mu.Unlock()
		return err
	}
	s.lastMenuObject = 0
	s.lastMenuOwner = 0
	s.lastMenuAt = time.Time{}
	s.lastMenuMovie = false
	if err := s.openSelectedServiceLocked(entity, service); err != nil {
		s.mu.Unlock()
		return err
	}
	log.Printf("%s: NPC menu selection id=%d config=%s mark=%d source=%s completed", s.remote, entity.id, entity.configID, mark, service.source)
	s.mu.Unlock()
	return nil
}
func (s *sceneLifecycle) selectCinematicTalkLocked(entity sceneEntity, request clientObjectRequest) (*questMenuAction, error) {
	if request.FuncID == 600000000 {
		if err := s.closeCinematicTalkLocked(); err != nil {
			return nil, err
		}
		s.clearActiveMenuLocked()
		s.transportMenuActive = false
		s.transportMenu = nil
		return nil, nil
	}
	if s.transportMenuActive {
		index, ok := transportMenuIndex(request.FuncID)
		if !ok || index >= len(s.transportMenu) {
			return nil, fmt.Errorf("NPC %s invalid transport destination func_id=%d", entity.configID, request.FuncID)
		}
		dest := s.transportMenu[index]
		if s.transportHandler == nil {
			return nil, fmt.Errorf("NPC %s transport service is not wired", entity.configID)
		}
		if err := s.closeCinematicTalkLocked(); err != nil {
			return nil, err
		}
		s.clearActiveMenuLocked()
		s.transportMenuActive = false
		s.transportMenu = nil
		if err := s.transportHandler(dest); err != nil {
			return nil, fmt.Errorf("NPC %s transport to %s: %w", entity.configID, dest.ID, err)
		}
		log.Printf("%s: NPC %s transport destination %s (%s) completed", s.remote, entity.configID, dest.ID, dest.TextID)
		return nil, nil
	}
	if isTalkAction(request.FuncID) {
		if entity.desc != "" {
			frame, err := encodeNPCBubble(entity.id, entity.ownerID, entity.desc)
			if err != nil {
				return nil, err
			}
			if err := s.conn.WriteFrame(frame); err != nil {
				return nil, err
			}
		}
		if err := s.closeCinematicTalkLocked(); err != nil {
			return nil, err
		}
		s.clearActiveMenuLocked()
		log.Printf("%s: NPC %s 交谈 desc=%q", s.remote, entity.configID, entity.desc)
		return nil, nil
	}
	if action, ok := questMenuActionFromFuncID(request.FuncID); ok {
		_, exists := questCatalog[action.taskID]
		if !exists {
			log.Printf("%s: NPC %s func_id=%d targets unknown quest %d; closing menu", s.remote, entity.configID, request.FuncID, action.taskID)
			if err := s.closeCinematicTalkLocked(); err != nil {
				return nil, err
			}
			s.clearActiveMenuLocked()
			return nil, nil
		}
		if err := s.clearMenuLocked(); err != nil {
			return nil, err
		}
		s.clearActiveMenuLocked()
		return &action, nil
	}
	mark, ok := cinematicTalkMark(request.FuncID)
	if !ok {
		log.Printf("%s: NPC %s func_id=%d has no implemented cinematic service; closing menu", s.remote, entity.configID, request.FuncID)
		if err := s.closeCinematicTalkLocked(); err != nil {
			return nil, err
		}
		s.clearActiveMenuLocked()
		return nil, nil
	}
	service, ok := findNPCService(entity.services, mark)
	if !ok {
		return nil, fmt.Errorf("NPC %s does not advertise cinematic service mark=%d", entity.configID, mark)
	}
	if err := s.closeCinematicTalkLocked(); err != nil {
		return nil, err
	}
	s.clearActiveMenuLocked()
	if err := s.openSelectedServiceLocked(entity, service); err != nil {
		return nil, err
	}
	log.Printf("%s: cinematic NPC menu selection id=%d config=%s func_id=%d source=%s completed", s.remote, entity.id, entity.configID, request.FuncID, service.source)
	return nil, nil
}
func cinematicTalkMark(funcID int32) (uint16, bool) {
	switch funcID {
	case 805000000:
		return 0x1001, true
	case 807000000:
		return 0x1002, true
	case 860000022:
		return 0x1003, true
	default:
		return 0, false
	}
}
func (s *sceneLifecycle) openSelectedServiceLocked(entity sceneEntity, service npcService) error {
	// Called under s.mu; retain authorization only for a successfully
	// opened shop service, never for a menu advertisement alone.
	s.activeShop = currentShopExchangeSession{}
	switch service.mark {
	case 0x1001:
		if service.value == "" {
			return fmt.Errorf("NPC %s shop service has no ShopID", entity.configID)
		}
		if err := s.openShopLocked(service.value); err != nil {
			return err
		}
		s.activeShop = currentShopExchangeSession{ShopID: service.value, NPCObjectID: entity.id, NPCOwnerID: entity.ownerID, NPCConfigID: entity.configID, SceneEpoch: s.epoch}
	case 0x1002:
		if err := s.openDepotLocked(); err != nil {
			return err
		}
	case 0x1003:
		catalog, catErr := loadNPCCatalogTransportData()
		if catErr != nil {
			return fmt.Errorf("load transport catalog: %w", catErr)
		}
		dests := catalog.destinations(service.value)
		if len(dests) == 0 {
			return fmt.Errorf("NPC %s TransFuncID %s has no destinations", entity.configID, service.value)
		}
		items := make([]talkMenuItem, 0, len(dests)+1)
		for index, dest := range dests {
			label := dest.TextID
			if label == "" {
				label = "目的地 " + dest.ID
			}
			items = append(items, talkMenuItem{funcID: transportMenuFuncID(index), textID: label})
		}
		items = append(items, talkMenuItem{funcID: 600000000, textID: "ui_form_talk_leave"})
		frames, menuErr := encodeCinematicTalkMenu(entity.id, entity.ownerID, "", items)
		if menuErr != nil {
			return menuErr
		}
		for _, frame := range frames {
			if err := s.conn.WriteFrame(frame); err != nil {
				return fmt.Errorf("send transport menu: %w", err)
			}
		}
		s.lastMenuObject = entity.id
		s.lastMenuOwner = entity.ownerID
		s.lastMenuAt = time.Now()
		s.lastMenuMovie = true
		s.transportMenuActive = true
		s.transportMenu = dests
		log.Printf("%s: NPC %s transport menu opened TransFuncID=%s destinations=%d", s.remote, entity.configID, service.value, len(dests))
		return nil
	default:
		return fmt.Errorf("NPC %s service %q (mark=%d) is discovered but not yet implemented", entity.configID, service.source, service.mark)
	}
	return nil
}
func (s *sceneLifecycle) clearMenuLocked() error {
	if err := s.conn.WriteFrame([]byte{0x1D}); err != nil {
		return fmt.Errorf("send SERVER_CLEAR_MENU: %w", err)
	}
	return nil
}
func (s *sceneLifecycle) clearActiveMenuLocked() {
	s.lastMenuObject = 0
	s.lastMenuOwner = 0
	s.lastMenuAt = time.Time{}
	s.lastMenuMovie = false
}
func (s *sceneLifecycle) closeCinematicTalkLocked() error {
	frame, err := serverCustomIntMessageWithOpcode(0x1E, 0x08)
	if err != nil {
		return fmt.Errorf("encode cinematic closemenu: %w", err)
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send cinematic closemenu: %w", err)
	}
	return nil
}
func findNPCService(services []npcService, mark uint16) (npcService, bool) {
	for _, service := range services {
		if service.mark == mark {
			return service, true
		}
	}
	return npcService{}, false
}
func npcServiceMenu(services []npcService, bindings []npcfunc.Binding) []serverMenuItem {
	if len(services) != 0 {
		items := make([]serverMenuItem, 0, len(services)+1)
		for _, service := range services {
			items = append(items, serverMenuItem{Type: 0, Mark: service.mark, Content: service.label})
		}
		return append(items, serverMenuItem{Type: 0, Mark: 0, Content: "离开"})
	}
	seen := make(map[int]struct{}, len(bindings))
	items := make([]serverMenuItem, 0, len(bindings)+1)
	for _, binding := range bindings {
		if _, duplicate := seen[binding.FuncID]; duplicate {
			continue
		}
		seen[binding.FuncID] = struct{}{}
		items = append(items, serverMenuItem{Type: 0, Mark: uint16(binding.FuncID), Content: npcFuncMenuText(binding.FuncID)})
	}
	if len(items) == 0 {
		items = append(items, serverMenuItem{Type: 0, Mark: 1, Content: "此 NPC 暂无 Npc_Func 配置"})
	}
	items = append(items, serverMenuItem{Type: 0, Mark: 0, Content: "离开"})
	return items
}
func talkMenuItems(services []npcService, bindings []npcfunc.Binding) []talkMenuItem {
	items := make([]talkMenuItem, 0, len(services)+len(bindings)+1)
	for _, service := range services {
		var funcID int32
		switch service.mark {
		case 0x1001:
			funcID = 805000000
		case 0x1002:
			funcID = 807000000
		case 0x1003:
			funcID = 860000022
		case 0x1004:
			funcID = 806000036
		case 0x1005:
			funcID = 806000013
		case 0x1006:
			funcID = 809000000
		case 0x1007:
			funcID = 881000001
		case 0x1008:
			funcID = 100000001
		default:
			funcID = 860000000 + int32(service.mark)
		}
		var textID string
		switch service.mark {
		case 0x1001:
			textID = "ui_shop"
		case 0x1002:
			textID = "ui_storewithdraw"
		case 0x1003:
			textID = "ui_chuansongmen"
		case 0x1004:
			textID = "jianghu_job"
		case 0x1005:
			textID = "ui_xiangqian"
		case 0x1006:
			textID = "ui_gonghui"
		case 0x1007:
			textID = "ui_hp004"
		case 0x1008:
			textID = "ui_form_talk_receive_task"
		default:
			textID = "ui_talk"
		}
		items = append(items, talkMenuItem{funcID: funcID, textID: textID})
	}
	if len(items) == 0 {
		seen := make(map[int]struct{}, len(bindings))
		for _, binding := range bindings {
			if _, duplicate := seen[binding.FuncID]; duplicate {
				continue
			}
			seen[binding.FuncID] = struct{}{}
			var funcID int32
			switch binding.FuncID {
			case 9, 17, 18:
				funcID = 805000000 + int32(binding.FuncID)
			case 20:
				funcID = 807000000
			case 24:
				funcID = 809000000
			case 36:
				funcID = 806000036
			case 48:
				funcID = 881000001
			default:
				funcID = 860000000 + int32(binding.FuncID)
			}
			var textID string
			switch binding.FuncID {
			case 9:
				textID = "ui_shop"
			case 20:
				textID = "ui_storewithdraw"
			case 22:
				textID = "ui_chuansongmen"
			case 24:
				textID = "ui_gonghui"
			case 36:
				textID = "jianghu_job"
			case 47:
				textID = "ui_talk"
			case 48:
				textID = "ui_hp004"
			default:
				textID = "ui_talk"
			}
			items = append(items, talkMenuItem{funcID: funcID, textID: textID})
		}
	}
	if len(items) == 0 {
		items = append(items, talkMenuItem{funcID: 860000001, textID: "ui_talk"})
	}
	items = append(items, talkMenuItem{funcID: 600000000, textID: "ui_form_talk_leave"})
	return items
}
func modernTalkFuncID(mark uint16) int32 {
	switch mark {
	case 0x1001:
		return 805000000
	case 0x1002:
		return 807000000
	case 0x1003:
		return 860000022
	case 0x1004:
		return 806000036
	case 0x1005:
		return 806000013
	case 0x1006:
		return 809000000
	case 0x1007:
		return 881000001
	case 0x1008:
		return 100000001
	default:
		return 860000000 + int32(mark)
	}
}
func modernTalkTextID(mark uint16) string {
	switch mark {
	case 0x1001:
		return "ui_shop"
	case 0x1002:
		return "ui_depot"
	case 0x1003:
		return "ui_transport"
	case 0x1004:
		return "ui_life_job"
	case 0x1005:
		return "ui_equip_gem"
	case 0x1006:
		return "ui_guild"
	case 0x1007:
		return "ui_home_point"
	case 0x1008:
		return "ui_form_talk_receive_task"
	default:
		return "ui_talk"
	}
}
func legacyTalkFuncID(funcID int) int32 {
	switch funcID {
	case 9, 17, 18:
		return 805000000 + int32(funcID)
	case 20:
		return 807000000
	case 24:
		return 809000000
	case 36:
		return 806000036
	case 48:
		return 881000001
	default:
		return 860000000 + int32(funcID)
	}
}
func legacyTalkTextID(funcID int) string {
	switch funcID {
	case 9:
		return "ui_shop"
	case 20:
		return "ui_depot"
	case 22:
		return "ui_transport"
	case 24:
		return "ui_guild"
	case 36:
		return "ui_life_job"
	case 47:
		return "ui_skill_trainer"
	case 48:
		return "ui_home_point"
	default:
		return "ui_talk"
	}
}
func modernNPCServices(npc npcSpawn) []npcService {
	businessValue := func(key string) (string, string) {
		return effectiveNPCBusinessValue(npc, key)
	}
	appendBusiness := func(items []npcService, mark uint16, label string, key string) []npcService {
		value, source := businessValue(key)
		if value == "" || value == "0" {
			return items
		}
		return append(items, npcService{mark: mark, label: label, value: value, source: source})
	}
	items := make([]npcService, 0, 6)
	items = appendBusiness(items, 0x1001, "商店", "ShopID")
	items = appendBusiness(items, 0x1002, "仓库", "DepotID")
	items = appendBusiness(items, 0x1003, "传送", "TransFuncID")
	items = appendBusiness(items, 0x1005, "宝石服务", "GemGameNpcID")
	items = appendBusiness(items, 0x1006, "帮会", "GuildFunc")
	items = appendBusiness(items, 0x1007, "回城点", "HomePoint")
	job := npc.int32Property("Job")
	if job != 0 {
		items = append(items, npcService{mark: 0x1004, label: "生活职业", value: fmt.Sprintf("%d", job), source: "Job"})
	}
	_, acceptSource := businessValue("table@TaskCanAccept")
	_, submitSource := businessValue("table@TaskCanSubmit")
	if acceptSource != "" || submitSource != "" {
		source := acceptSource
		if source == "" {
			source = submitSource
		}
		items = append(items, npcService{mark: 0x1008, label: "任务", source: source})
	}
	return items
}
func playableNPCServices(services []npcService) []npcService {
	playable := make([]npcService, 0, len(services))
	for _, service := range services {
		switch service.mark {
		case 0x1002:
			playable = append(playable, service)
		case 0x1001:
			if _, _, _, err := shopCatalogItems(defaultShopINIPath, service.value); err == nil {
				playable = append(playable, service)
			}
		case 0x1003:
			playable = append(playable, service)
		}
	}
	return playable
}
func effectiveNPCBusinessValue(npc npcSpawn, key string) (value string, source string) {
	for _, prefix := range []string{"creator.", "template."} {
		configured := strings.TrimSpace(npc.resolved.Extensions[prefix+key])
		if configured != "" {
			return configured, prefix + key
		}
	}
	return "", ""
}
func npcFuncMenuText(funcID int) string {
	labels := map[int]string{6: "江湖/PVP", 7: "交谈", 9: "商店", 10: "强化", 12: "合成", 13: "宝石合成", 14: "开孔", 17: "拍卖行", 18: "黑市", 19: "转职", 20: "仓库", 22: "传送", 23: "召唤载具", 24: "帮会", 32: "制造", 35: "物品绑定", 36: "生活职业", 40: "PVP 任务", 41: "PVP 任务", 43: "神秘人", 44: "向导", 45: "典狱官", 46: "监察员", 47: "技能训练", 48: "回城点", 49: "PVP 任务", 50: "载具管理", 51: "赠礼", 111: "个人机械管理"}
	if label, ok := labels[funcID]; ok {
		return label
	}
	return fmt.Sprintf("配置功能 #%d", funcID)
}
func npcFuncIDs(bindings []npcfunc.Binding) []int {
	ids := make([]int, 0, len(bindings))
	for _, binding := range bindings {
		ids = append(ids, binding.FuncID)
	}
	return ids
}
func (s *sceneLifecycle) moveNPC(id uint32, destination worldcore.Transform, arrival time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	entityID := worldcore.EntityID(id)
	meta, exists := s.entities[entityID]
	if !exists {
		return fmt.Errorf("move unknown scene object %d", id)
	}
	current, err := s.scene.Get(entityID)
	if err != nil {
		return err
	}
	current.Entity.Transform = destination
	for _, property := range []struct {
		name  string
		value float32
	}{{"PosiX", destination.X}, {"PosiY", destination.Y}, {"PosiZ", destination.Z}, {"Orient", destination.Orient}} {
		if err := current.Entity.Properties.Set(property.name, worldcore.Float32(property.value)); err != nil {
			return err
		}
	}
	if err := s.scene.Replace(current.Entity); err != nil {
		return err
	}
	meta.transform = destination
	s.entities[entityID] = meta
	if err := s.conn.WriteFrame(serverMoving(meta.id, meta.ownerID, motionFromDest(destination))); err != nil {
		return fmt.Errorf("send NPC moving %d: %w", id, err)
	}
	if _, err := s.reconcileViewportLocked(); err != nil {
		return err
	}
	if arrival <= 0 {
		arrival = 1500 * time.Millisecond
	}
	epoch := s.epoch
	timer := time.AfterFunc(arrival, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.epoch != epoch {
			return
		}
		current, exists := s.entities[entityID]
		if !exists {
			return
		}
		if err := s.conn.WriteFrame(serverLocation(current.id, current.ownerID, current.transform)); err != nil {
			log.Printf("%s: confirm NPC move id=%d: %v", s.remote, current.id, err)
			return
		}
		log.Printf("%s: NPC id=%d arrived at (%.2f, %.2f, %.2f)", s.remote, current.id, current.transform.X, current.transform.Y, current.transform.Z)
	})
	s.timers = append(s.timers, timer)
	log.Printf("%s: NPC id=%d moving to (%.2f, %.2f, %.2f)", s.remote, id, destination.X, destination.Y, destination.Z)
	return nil
}
func (s *sceneLifecycle) startPatrol(count int) {
	if count <= 0 {
		return
	}
	s.mu.Lock()
	if s.closed || s.patrolStarted {
		s.mu.Unlock()
		return
	}
	s.patrolStarted = true
	ids := make([]int, 0, len(s.entities))
	for id, meta := range s.entities {
		if meta.interaction != npcInteractionCombat {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	if count > len(ids) {
		count = len(ids)
	}
	selected := make([]uint32, count)
	for index := range selected {
		selected[index] = uint32(ids[index])
	}
	s.mu.Unlock()
	for index, id := range selected {
		s.schedulePatrolStep(id, 3*time.Second+time.Duration(index)*700*time.Millisecond)
	}
	log.Printf("%s: started low-rate patrol for %d nearby NPCs", s.remote, len(selected))
}
func (s *sceneLifecycle) schedulePatrolStep(id uint32, delay time.Duration) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	epoch := s.epoch
	timer := time.AfterFunc(delay, func() {
		s.mu.Lock()
		if s.closed || s.epoch != epoch {
			s.mu.Unlock()
			return
		}
		entityID := worldcore.EntityID(id)
		meta, exists := s.entities[entityID]
		if !exists {
			s.mu.Unlock()
			return
		}
		if meta.patrolSign == 0 {
			meta.patrolSign = 1
		}
		destination := meta.transform
		destination.X += meta.patrolSign * 1.5
		meta.patrolSign = -meta.patrolSign
		s.entities[entityID] = meta
		s.mu.Unlock()
		if err := s.moveNPC(id, destination, 1500*time.Millisecond); err != nil {
			log.Printf("%s: patrol move id=%d: %v", s.remote, id, err)
			return
		}
		s.schedulePatrolStep(id, 5*time.Second)
	})
	s.timers = append(s.timers, timer)
	s.mu.Unlock()
}
func (entity sceneEntity) propertiesForTransform() []clientdata.IndexedProperty {
	properties := cloneIndexedProperties(entity.properties)
	for index := range properties {
		switch properties[index].Name {
		case "PosiX":
			properties[index].Value = clientdata.Float32Value(entity.transform.X)
		case "PosiY":
			properties[index].Value = clientdata.Float32Value(entity.transform.Y)
		case "PosiZ":
			properties[index].Value = clientdata.Float32Value(entity.transform.Z)
		case "Orient":
			properties[index].Value = clientdata.Float32Value(entity.transform.Orient)
		}
	}
	return properties
}
func cloneIndexedProperties(source []clientdata.IndexedProperty) []clientdata.IndexedProperty {
	return append([]clientdata.IndexedProperty(nil), source...)
}
func worldValue(value clientdata.Value) (worldcore.Value, error) {
	switch value.Type {
	case clientdata.WireByte:
		return worldcore.Byte(value.U8), nil
	case clientdata.WireWord:
		return worldcore.Word(value.U16), nil
	case clientdata.WireInt32:
		return worldcore.Int32(value.I32), nil
	case clientdata.WireFloat32:
		return worldcore.Float32(value.F32), nil
	case clientdata.WireString:
		return worldcore.String(value.Text), nil
	case clientdata.WireWideString:
		return worldcore.WideString(value.Text), nil
	default:
		return worldcore.Value{}, fmt.Errorf("unsupported wire type %d", value.Type)
	}
}
func (s *sceneLifecycle) scheduleLocationReplaysLocked(replays []sceneLocationReplay) {
	epoch := s.epoch
	for _, delay := range actor2LocationReplayDelays {
		delay := delay
		batch := make([]sceneLocationReplay, len(replays))
		for i, replay := range replays {
			batch[i] = sceneLocationReplay{id: replay.id, payload: append([]byte(nil), replay.payload...)}
		}
		timer := time.AfterFunc(delay, func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.closed || s.epoch != epoch {
				return
			}
			sent := 0
			for _, replay := range batch {
				if _, exists := s.entities[worldcore.EntityID(replay.id)]; !exists {
					continue
				}
				if err := s.conn.WriteFrame(replay.payload); err != nil {
					log.Printf("%s: delayed ServerLocation id=%d after %s: %v", s.remote, replay.id, delay, err)
					return
				}
				sent++
			}
			if sent != 0 {
				log.Printf("%s: replayed ServerLocation for %d scene objects after %s", s.remote, sent, delay)
			}
		})
		s.timers = append(s.timers, timer)
	}
}
func (s *sceneLifecycle) stopReplayTimersLocked() {
	for _, timer := range s.timers {
		timer.Stop()
	}
	s.timers = nil
}
func sceneRemoveObject(objectID uint32, ownerID uint32) []byte {
	msg := make([]byte, 9)
	msg[0] = 0x0E
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], ownerID)
	return msg
}
func (s *sceneLifecycle) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.epoch++
	s.stopReplayTimersLocked()
}
