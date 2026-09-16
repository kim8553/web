package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Hiroko103/go-quicklz"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/transport"
	"github.com/local/9yin-go-server/internal/world"
	worldcore "github.com/local/9yin-go-server/internal/world"
	"golang.org/x/text/encoding/simplifiedchinese"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func handleBlockStateCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0x36 {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("block state before scene player exists")
	}
	if len(custom.Values) < 2 {
		return true, fmt.Errorf("block state requires a value")
	}
	value := custom.Values[1]
	var holding bool
	switch value.Type {
	case 2:
		holding = value.Int32 == 0
	case 3:
		holding = value.Int64 == 0
	default:
		return true, fmt.Errorf("block state value must be integral, got %s", value)
	}
	if !player.setPlayerBlock(holding, time.Now()) {
		return true, nil
	}
	mode := int32(1)
	if player.parryingSnapshot() {
		mode = 0
	}
	state := player.actor.Snapshot()
	buffEvent, buffErr := skillBufferFrame(mode, "BuffInParry", state.Name, uint64(time.Now().UnixMilli()), 0)
	if buffErr != nil {
		return true, fmt.Errorf("encode BuffInParry event: %w", buffErr)
	}
	if err := link.WriteFrame(buffEvent); err != nil {
		return true, fmt.Errorf("write BuffInParry event: %w", err)
	}
	update, err := player.blockStateFrame(player.parryingSnapshot(), time.Now())
	if err != nil {
		return true, fmt.Errorf("encode block state: %w", err)
	}
	if err := link.WriteFrame(update); err != nil {
		return true, fmt.Errorf("write block state: %w", err)
	}
	log.Printf("%s: player block %s (heartbeat=%d)", remote, map[bool]string{true: "engaged", false: "released"}[holding], value.Int64)
	return true, nil
}
func enrichBagItem(item bagItem, itemCatalog *itemCatalog, equipCatalog *equipCatalog) bagItem {
	if itemCatalog != nil {
		if ti, ok := itemCatalog.byID[item.ConfigID]; ok {
			if item.ItemType == 0 {
				item.ItemType = ti.ItemType
			}
			if item.ViewID == 0 {
				item.ViewID = ti.ViewID
			}
			if item.Amount == 0 {
				item.Amount = ti.Amount
			}
			if item.MaxAmount == 0 {
				item.MaxAmount = ti.MaxAmount
			}
			if item.FuncPack == 0 {
				item.FuncPack = ti.FuncPack
			}
			if item.LogicPack == 0 {
				item.LogicPack = ti.LogicPack
			}
			if item.PropModifyPack == 0 {
				item.PropModifyPack = ti.PropModifyPack
			}
			if item.TextureType == 0 {
				item.TextureType = ti.TextureType
			}
			if item.ToolUseEffect == "" {
				item.ToolUseEffect = ti.ToolUseEffect
			}
			if item.FuncBuffer == "" {
				item.FuncBuffer = ti.FuncBuffer
			}
			if item.CardID == 0 {
				item.CardID = ti.CardID
			}
			if item.Name == "" {
				item.Name = ti.Name
			}
		}
	}
	if equipCatalog != nil {
		if eq, ok := equipCatalog.byID[item.ConfigID]; ok {
			if item.ItemType == 0 {
				item.ItemType = eq.ItemType
			}
			if item.ViewID == 0 {
				item.ViewID = eq.ViewID
			}
			if item.EquipType == "" {
				item.EquipType = eq.EquipType
			}
			if item.ArtPack == 0 {
				item.ArtPack = eq.ArtPack
			}
			if item.ColorLevel == 0 {
				item.ColorLevel = eq.ColorLevel
			}
			if item.Hardiness == 0 {
				item.Hardiness = eq.Hardiness
			}
			if item.MaxHardiness == 0 {
				item.MaxHardiness = eq.MaxHardiness
			}
			if item.MaxMeleeDamage == 0 {
				item.MaxMeleeDamage = eq.MaxMeleeDamage
			}
			if item.MinMeleeDamage == 0 {
				item.MinMeleeDamage = eq.MinMeleeDamage
			}
			if item.Name == "" {
				item.Name = eq.Name
			}
		}
	}
	return item
}
func grantBagItems(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, equipCatalog *equipCatalog, remote string) error {
	items := player.bagSnapshot()
	for _, spec := range starterBagViews() {
		var frame []byte
		var err error
		if spec.BaseCap > 0 {
			baseCap1 := int32(spec.BaseCap)
			baseCap2 := int32(spec.BaseCap)
			frame, err = serverCreateViewWithProperties(spec, []serverViewProperty{viewInt(0x00E8, baseCap1), viewInt(0x08A7, baseCap2)})
		} else {
			frame, err = serverCreateViewWithProperties(spec, nil)
		}
		if err != nil {
			return err
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	assigned := make(map[uint16]map[int32]struct{})
	assignSlot := func(item bagItem) (uint16, int32) {
		view := bagViewForViewID(item.ViewID)
		slot := item.Slot
		if slot <= 0 || slot > 65535 {
			slot = nextBagSlotFrom(assigned[view])
		}
		if assigned[view] == nil {
			assigned[view] = make(map[int32]struct{})
		}
		assigned[view][slot] = struct{}{}
		return view, slot
	}
	views := make(map[uint16]struct{})
	for _, item := range items {
		item = enrichBagItem(item, itemCatalog, equipCatalog)
		view, slot := assignSlot(item)
		if view == 0 {
			return fmt.Errorf("latest-client bag item %s has unsupported ViewID category %d", item.ConfigID, item.ViewID)
		}
		views[view] = struct{}{}
		frames, err := latestClientCurrentBagFrames(view, uint16(slot), item.ViewID, bagItemProps(view, item))
		if err != nil {
			return fmt.Errorf("latest-client bag item %s view=%d slot=%d: %w", item.ConfigID, view, slot, err)
		}
		log.Printf("%s: latest-client CURRENT-BAG-FULL item=%s category=%d container=%d slot=%d table=%d frame=%x", remote, item.ConfigID, item.ViewID, view, uint16(slot), latestClientPlayerWirePropertyTableCount, frames[0])
		for _, frame := range frames {
			if err := link.WriteFrame(frame); err != nil {
				return err
			}
		}
	}
	if len(items) != 0 {
		log.Printf("%s: granted %d bag items across %d views", remote, len(items), len(views))
	}
	return nil
}
func nextBagSlotFrom(assigned map[int32]struct{}) int32 {
	for slot := int32(1); ; slot++ {
		if _, taken := assigned[slot]; !taken {
			return slot
		}
	}
}

const bagStoreVersion = 1

type bagStore struct {
	mu    sync.Mutex
	path  string
	roles map[string][]bagItem
}
type bagStoreFile struct {
	Version int                  `json:"version"`
	Roles   map[string][]bagItem `json:"roles"`
}

func bagViewForViewID(viewID int32) uint16 {
	view, ok := latestClientCurrentBagContainer(viewID)
	if !ok {
		return 0
	}
	return view
}
func openBagStore() (*bagStore, error) {
	return openBagStoreAt(filepath.Join(runtimeProjectRoot, "data", "bag_items.json"))
}
func openBagStoreAt(path string) (*bagStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("bag store: empty path")
	}
	path = filepath.Clean(path)
	store := &bagStore{path: path, roles: make(map[string][]bagItem)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read bag store: %w", err)
	}
	var document bagStoreFile
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode bag store: %w", err)
	}
	if document.Version != bagStoreVersion {
		return nil, fmt.Errorf("decode bag store: unsupported version %d", document.Version)
	}
	if document.Roles != nil {
		store.roles = document.Roles
	}
	return store, nil
}
func (store *bagStore) Load(roleID role.RoleID) ([]bagItem, bool) {
	if store == nil || roleID == 0 {
		return nil, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	items, ok := store.roles[strconv.FormatUint(uint64(roleID), 10)]
	if !ok {
		return nil, false
	}
	return append([]bagItem(nil), items...), true
}
func (store *bagStore) Save(roleID role.RoleID, items []bagItem) error {
	if store == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("bag store: zero role id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := strconv.FormatUint(uint64(roleID), 10)
	previous, existed := store.roles[key]
	store.roles[key] = items
	if err := store.persistLocked(); err != nil {
		if existed {
			store.roles[key] = previous
		} else {
			delete(store.roles, key)
		}
		return err
	}
	return nil
}
func (store *bagStore) persistLocked() error {
	document := bagStoreFile{Version: bagStoreVersion, Roles: store.roles}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode bag store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return fmt.Errorf("mkdir bag store: %w", err)
	}
	tmp := store.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write bag store: %w", err)
	}
	return os.Rename(tmp, store.path)
}
func npcAttackSkill(npc npcSpawn) string {
	raw := strings.TrimSpace(npc.resolved.Properties["template.NormalSkill"].Value.Text)
	raw = strings.Trim(raw, `"`)
	if raw == "" || raw == "0" {
		return "jn_mon_hand_006"
	}
	return raw
}
func npcDamageSkill(npc npcSpawn) string {
	raw := strings.TrimSpace(npc.resolved.Properties["template.table@SkillRec"].Value.Text)
	raw = strings.Trim(raw, `"`)
	if raw == "" || raw == "0" {
		return "default_normal_skill"
	}
	parts := strings.SplitN(raw, ";", 2)
	first := strings.TrimSpace(parts[0])
	skill := strings.TrimSpace(strings.SplitN(first, ",", 2)[0])
	if skill == "" {
		return "default_normal_skill"
	}
	return skill
}
func npcCombatName(npc npcSpawn) string {
	if name := npc.resolved.Properties["Name"].Value.Text; name != "" {
		return name
	}
	return npc.resolved.ConfigID
}
func (s *sceneLifecycle) writeNPCAttackFrames(meta sceneEntity, state npcCombatState, player *playerActor, landed bool) error {
	if player == nil {
		return nil
	}
	playerName := player.actor.Snapshot().Name
	prep, err := skillNPCCastPreparationFrame(meta.id, meta.ownerID, meta.transform)
	if err != nil {
		return err
	}
	if err := s.conn.WriteFrame(prep); err != nil {
		return err
	}
	animation, err := skillAnimationFrame(meta.id, meta.ownerID, state.attackSkill, playerObjectID, playerOwnerID)
	if err != nil {
		return err
	}
	if err := s.conn.WriteFrame(animation); err != nil {
		return err
	}
	if !landed {
		return nil
	}
	hit, err := skillNPCAttackHitFrame(playerObjectID, playerOwnerID, meta.id, meta.ownerID, state.damageSkill, playerName, state.name, state.damage)
	if err != nil {
		return err
	}
	if err := s.conn.WriteFrame(hit); err != nil {
		return err
	}
	playerState := player.actor.Snapshot()
	healthPercent := int32(100)
	if playerState.MaxHP > 0 {
		healthPercent = int32(int64(playerState.HP) * 100 / int64(playerState.MaxHP))
	}
	ratio := entityPropertyBatch(uint64(playerObjectID)|uint64(playerOwnerID)<<32, []entityRatioProperty{{id: 28, value: healthPercent}, {id: 113, value: healthPercent}})
	err = s.conn.WriteFrame(ratio)
	return err
}
func (s *sceneLifecycle) combatMoveTick(player *playerActor, position role.Position, now time.Time) error {
	if s == nil || player == nil {
		return nil
	}
	playerState := player.actor.Snapshot()
	if playerState.HP <= 0 {
		return nil
	}
	playerTransform := worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
	s.mu.Lock()
	if s.closed || !s.combatActive || s.combatStates == nil {
		s.mu.Unlock()
		return nil
	}
	moves := make([]npcMoveUpdate, 0, 4)
	stops := make([]npcMoveUpdate, 0, 2)
	for rawID := range s.combatStates {
		id := rawID
		state, exists := s.combatStates[id]
		if !exists {
			continue
		}
		if state.threat == 0 || now.Before(state.controlledUntil) {
			continue
		}
		actor := s.combatActors[id]
		meta, exists := s.entities[id]
		if actor == nil || !exists || actor.Snapshot().HP <= 0 {
			continue
		}
		distance := horizontalDistance(meta.transform, playerTransform)
		if distance > state.chaseRange {
			continue
		}
		if distance <= state.attackRange {
			if state.moving {
				stops = append(stops, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, from: meta.transform, destination: meta.transform})
				state.moving = false
			}
		} else {
			step := state.moveSpeed * 0.15
			if remain := distance - state.attackRange; remain < step {
				step = remain
			}
			destination := stepToward(meta.transform, playerTransform, step)
			if err := s.setNPCTransformLocked(id, destination); err != nil {
				s.mu.Unlock()
				return err
			}
			moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, from: meta.transform, destination: destination, speed: state.moveSpeed})
			state.moving = true
		}
		s.combatStates[id] = state
	}
	s.mu.Unlock()
	for _, move := range moves {
		speed := move.speed
		if speed <= 0 {
			speed = 1.77
		}
		heading := headingTowards(move.from, move.destination)
		if err := s.conn.WriteFrame(serverEntityMove([]entityMove{{id: move.id, position: move.from, movement: makeMovementData(speed, heading)}})); err != nil {
			return fmt.Errorf("combat chase NPC %d: %w", move.id, err)
		}
	}
	for _, stop := range stops {
		if err := s.conn.WriteFrame(serverEntityMove([]entityMove{{id: stop.id, position: stop.destination}})); err != nil {
			return fmt.Errorf("combat stop NPC %d: %w", stop.id, err)
		}
	}
	return nil
}

const currencyStoreVersion = 1

type currencySnapshot struct {
	Silver       int32 `json:"silver"`
	Gold         int32 `json:"gold"`
	SilverCard   int32 `json:"silver_card"`
	SilverTicket int32 `json:"silver_ticket"`
}
type currencyStore struct {
	mu    sync.Mutex
	path  string
	roles map[string]currencySnapshot
}
type currencyStoreFile struct {
	Version int                         `json:"version"`
	Roles   map[string]currencySnapshot `json:"roles"`
}

func (c *currencySnapshot) fromActor(player *playerActor) {
	c.Silver, c.Gold, c.SilverCard, c.SilverTicket = player.currencySnapshot()
}
func (c *currencySnapshot) toActor(player *playerActor) {
	if player == nil {
		return
	}
	player.setSilver(c.Silver)
	player.setGold(c.Gold)
	player.setSilverCard(c.SilverCard)
	player.setSilverTicket(c.SilverTicket)
}
func newMySQLCurrencyStore(db *sql.DB) *mysqlCurrencyStore {
	return &mysqlCurrencyStore{db: db}
}
func (store *mysqlCurrencyStore) Load(roleID role.RoleID) (currencySnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return currencySnapshot{}, false
	}
	var encoded []byte
	err := store.db.QueryRowContext(context.Background(), "SELECT snapshot FROM role_currency WHERE role_id = ?", roleID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return currencySnapshot{}, false
	}
	if err != nil {
		return currencySnapshot{}, false
	}
	var value currencySnapshot
	if err := json.Unmarshal(encoded, &value); err != nil {
		return currencySnapshot{}, false
	}
	return value, true
}
func (store *mysqlCurrencyStore) Save(roleID role.RoleID, value currencySnapshot) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("currency store: zero role id")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_currency WHERE role_id = ?", roleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO role_currency(role_id, snapshot) VALUES (?, ?)", roleID, encoded); err != nil {
		return err
	}
	return tx.Commit()
}
func openCurrencyStore() (*currencyStore, error) {
	return openCurrencyStoreAt(filepath.Join(runtimeProjectRoot, "data", "currency.json"))
}
func openCurrencyStoreAt(path string) (*currencyStore, error) {
	path = filepath.Clean(path)
	store := &currencyStore{path: path, roles: make(map[string]currencySnapshot)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read currency store: %w", err)
	}
	var document currencyStoreFile
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode currency store: %w", err)
	}
	if document.Roles != nil {
		store.roles = document.Roles
	}
	return store, nil
}
func (store *currencyStore) Load(roleID role.RoleID) (currencySnapshot, bool) {
	if store == nil || roleID == 0 {
		return currencySnapshot{}, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	value, ok := store.roles[fmt.Sprintf("%d", roleID)]
	return value, ok
}
func (store *currencyStore) Save(roleID role.RoleID, value currencySnapshot) error {
	if store == nil || roleID == 0 {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.roles[fmt.Sprintf("%d", roleID)] = value
	return store.persistLocked()
}
func (store *currencyStore) persistLocked() error {
	document := currencyStoreFile{Version: currencyStoreVersion, Roles: store.roles}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(store.path, data, 0o644)
}

var defaultDropTablePath = filepath.Join(defaultModernShareRoot, "item", "drop_table.json")

type dropEntry struct {
	ConfigID string `json:"config_id"`
	Amount   int32  `json:"amount"`
	Weight   int32  `json:"weight,omitempty"`
}
type dropTableSection struct {
	Mode    string      `json:"mode,omitempty"`
	Rolls   int32       `json:"rolls,omitempty"`
	Entries []dropEntry `json:"entries"`
}
type dropTable struct{ byDropID map[string]*dropTableSection }

func loadDropTable(path string) (*dropTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t := &dropTable{byDropID: make(map[string]*dropTableSection)}
	if err := json.Unmarshal(data, &t.byDropID); err != nil {
		return nil, fmt.Errorf("decode drop table: %w", err)
	}
	for dropID, section := range t.byDropID {
		if section == nil {
			return nil, fmt.Errorf("drop table %s: empty section", dropID)
		}
		switch strings.ToLower(section.Mode) {
		case "", "random":
			section.Mode = "random"
			if section.Rolls <= 0 {
				section.Rolls = 1
			}
		case "fixed":
			section.Rolls = 0
		default:
			return nil, fmt.Errorf("drop table %s: unknown mode %q", dropID, section.Mode)
		}
		if len(section.Entries) == 0 {
			return nil, fmt.Errorf("drop table %s: no entries", dropID)
		}
		for i := range section.Entries {
			entry := section.Entries[i]
			if strings.TrimSpace(entry.ConfigID) == "" {
				return nil, fmt.Errorf("drop table %s: entry %d empty config_id", dropID, i)
			}
			if entry.Amount <= 0 {
				return nil, fmt.Errorf("drop table %s: entry %s amount=%d", dropID, entry.ConfigID, entry.Amount)
			}
		}
	}
	return t, nil
}
func (t *dropTable) Has(dropID string) bool {
	if t == nil {
		return false
	}
	_, ok := t.byDropID[dropID]
	return ok
}
func entryWeight(e dropEntry) int64 {
	if e.Weight <= 0 {
		return 1
	}
	return int64(e.Weight)
}
func (t *dropTable) roll(dropID string, rng *rand.Rand) ([]bagItem, bool) {
	if t == nil {
		return nil, false
	}
	section, ok := t.byDropID[dropID]
	if !ok || section == nil {
		return nil, false
	}
	out := make([]bagItem, 0, len(section.Entries))
	if section.Mode == "fixed" {
		for _, entry := range section.Entries {
			out = append(out, bagItem{ConfigID: entry.ConfigID, Amount: entry.Amount})
		}
		return out, true
	}
	total := int64(0)
	for _, entry := range section.Entries {
		total += entryWeight(entry)
	}
	if total <= 0 {
		return nil, false
	}
	for i := int32(0); i < section.Rolls; i++ {
		pick := rng.Int63n(total)
		acc := int64(0)
		for _, entry := range section.Entries {
			acc += entryWeight(entry)
			if pick < acc {
				out = append(out, bagItem{ConfigID: entry.ConfigID, Amount: entry.Amount})
				break
			}
		}
	}
	return out, true
}

var defaultPlayerWeaponDir = filepath.Join(defaultModernShareRoot, "ini", "effect", "playerweapon")
var defaultEquipmentINI = filepath.Join(defaultModernShareRoot, "item", "equipment.ini")
var defaultItemArtStaticINI = filepath.Join(defaultModernShareRoot, "item", "itemartstatic.ini")

type equipItem struct {
	ConfigID       string
	Name           string
	ItemType       int32
	EquipType      string
	ViewID         int32
	ColorLevel     int32
	ArtPack        int32
	Hardiness      int32
	MaxHardiness   int32
	MinMeleeDamage int32
	MaxMeleeDamage int32
	MaxHPAdd       int32
	MaxMPAdd       int32
	PhyDef         int32
	MaxParry       int32
}
type equipCatalog struct {
	byID          map[string]equipItem
	artModels     map[int32]string
	weaponModels  map[string]string
	weaponHeld    map[string]string
	artActionSets map[int32]string
}

func (e *equipCatalog) weaponModelName(artPack int32) string {
	if e == nil {
		return ""
	}
	return e.artModels[artPack]
}
func (e *equipCatalog) actionSetForArtPack(artPack int32) string {
	if e == nil {
		return ""
	}
	mode, ok := e.artActionSets[artPack]
	if !ok {
		return ""
	}
	return mode
}
func loadPlayerWeaponModels(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	models := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "weapon_") || !strings.HasSuffix(entry.Name(), ".ini") {
			continue
		}
		file, openErr := os.Open(filepath.Join(dir, entry.Name()))
		if openErr != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		current := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "/") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				current = strings.TrimSpace(line[1 : len(line)-1])
				continue
			}
			if current == "" {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "Model") {
				continue
			}
			model := strings.TrimSpace(value)
			if strings.HasSuffix(model, ".xmod") {
				model = model[:len(model)-len(".xmod")]
			}
			if model == "" {
				continue
			}
			if _, exists := models[current]; !exists {
				models[current] = model
			}
		}
		scanErr := scanner.Err()
		_ = file.Close()
		if scanErr != nil {
			return nil, scanErr
		}
	}
	return models, nil
}
func loadWeaponHeldModes(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	held := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "weapon_") || !strings.HasSuffix(entry.Name(), ".ini") {
			continue
		}
		mode := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "weapon_"), ".ini")
		if mode == "" {
			continue
		}
		file, openErr := os.Open(filepath.Join(dir, entry.Name()))
		if openErr != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		current := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "/") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				current = strings.TrimSpace(line[1 : len(line)-1])
				continue
			}
			if current == "" {
				continue
			}
			key, _, ok := strings.Cut(line, "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "Model") {
				continue
			}
			if _, exists := held[current]; !exists {
				held[current] = mode
			}
		}
		scanErr := scanner.Err()
		_ = file.Close()
		if scanErr != nil {
			return nil, scanErr
		}
	}
	return held, nil
}
func loadItemArtStatic(path string) (map[int32]string, map[int32]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	decoder := simplifiedchinese.GBK.NewDecoder()
	models := make(map[int32]string)
	actionSets := make(map[int32]string)
	current := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if current == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		artPack64, parseErr := strconv.ParseInt(strings.TrimSpace(current), 10, 32)
		if parseErr != nil {
			continue
		}
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		artPack := int32(artPack64)
		if strings.EqualFold(k, "MaleModel") {
			decoded, decodeErr := decoder.String(v)
			if decodeErr == nil {
				models[artPack] = decoded
			} else {
				models[artPack] = v
			}
		} else if strings.EqualFold(k, "ActionSet") {
			actionSets[artPack] = v
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return models, actionSets, nil
}
func loadEquipCatalog(path string) (*equipCatalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := simplifiedchinese.GBK.NewDecoder()
	catalog := &equipCatalog{byID: make(map[string]equipItem), artModels: make(map[int32]string)}
	current := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			if current == "" {
				continue
			}
			catalog.byID[current] = equipItem{ConfigID: current, ItemType: 100, ViewID: 2}
			continue
		}
		if current == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		entry := catalog.byID[current]
		switch strings.TrimSpace(key) {
		case "ItemType":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.ItemType = int32(n)
			}
		case "EquipType":
			entry.EquipType = strings.TrimSpace(value)
		case "ColorLevel":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.ColorLevel = int32(n)
			}
		case "ViewID":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.ViewID = int32(n)
			}
		case "ArtPack":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.ArtPack = int32(n)
			}
		case "Hardiness":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.Hardiness = int32(n)
			}
		case "MaxHardiness":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.MaxHardiness = int32(n)
			}
		case "MinMeleeDamage":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.MinMeleeDamage = int32(n)
			}
		case "MaxMeleeDamage":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.MaxMeleeDamage = int32(n)
			}
		case "MaxHP":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.MaxHPAdd = int32(n)
			}
		case "MaxMP":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.MaxMPAdd = int32(n)
			}
		case "PhyDef":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.PhyDef = int32(n)
			}
		case "MaxParry":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.MaxParry = int32(n)
			}
		case "Name":
			if decoded, err := decoder.String(strings.TrimSpace(value)); err == nil {
				entry.Name = decoded
			}
		}
		catalog.byID[current] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	models, actionSets, err := loadItemArtStatic(defaultItemArtStaticINI)
	if err != nil {
		return nil, fmt.Errorf("load itemartstatic: %w", err)
	}
	catalog.artModels = models
	catalog.artActionSets = actionSets
	weaponModels, weaponErr := loadPlayerWeaponModels(defaultPlayerWeaponDir)
	if weaponErr != nil {
		return nil, fmt.Errorf("load playerweapon models: %w", weaponErr)
	}
	catalog.weaponModels = weaponModels
	heldModes, heldErr := loadWeaponHeldModes(defaultPlayerWeaponDir)
	if heldErr != nil {
		return nil, fmt.Errorf("load playerweapon held modes: %w", heldErr)
	}
	catalog.weaponHeld = heldModes
	return catalog, nil
}
func (e *equipCatalog) Lookup(configID string) (equipItem, bool) {
	if e == nil {
		return equipItem{}, false
	}
	item, ok := e.byID[configID]
	return item, ok
}
func (e *equipCatalog) Count() int {
	if e == nil || e.byID == nil {
		return 0
	}
	return len(e.byID)
}
func grantEquipView(link sceneMessageConnection, player *playerActor, equipCatalog *equipCatalog, remote string) error {
	if err := link.WriteFrame(serverCreateView(serverViewSpec{ID: 1, Capacity: 40})); err != nil {
		return err
	}
	for _, item := range player.equipSnapshot() {
		slot := item.Slot
		if slot <= 0 {
			continue
		}
		if correct := equipBodySlot(item.EquipType); correct > 0 {
			slot = correct
		}
		bag := bagItem{ConfigID: item.ConfigID, ItemType: item.ItemType, Hardiness: item.Hardiness, MaxHardiness: item.MaxHardiness, EquipType: item.EquipType, ArtPack: item.ArtPack}
		if equipCatalog != nil {
			if cat, ok := equipCatalog.Lookup(item.ConfigID); ok {
				if bag.ColorLevel == 0 {
					bag.ColorLevel = cat.ColorLevel
				}
				if bag.ItemType == 0 {
					bag.ItemType = cat.ItemType
				}
				if bag.MaxMeleeDamage == 0 {
					bag.MaxMeleeDamage = cat.MaxMeleeDamage
				}
				if bag.MinMeleeDamage == 0 {
					bag.MinMeleeDamage = cat.MinMeleeDamage
				}
			}
		}
		bodyProps := equipItemProps(bag)
		if isWeaponEquip(item.EquipType) {
			bodyProps = weaponRowProps(bag)
		}
		frame, err := serverViewAdd(1, uint16(slot), bodyProps)
		if err != nil {
			return err
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
		if index, name := appearanceIndexFor(item.EquipType); index != 0 {
			if model := equipmentModels.resolve(item.ConfigID, player.sex); model != "" {
				if appFrame, appErr := player.setAppearanceModel(index, name, model); appErr == nil {
					if err := link.WriteFrame(appFrame); err != nil {
						return err
					}
				}
			}
		}
	}
	log.Printf("%s: created VIEWPORT_EQUIP=1 view with %d worn items", remote, len(player.equipSnapshot()))
	return nil
}

const equipStoreVersion = 1

type equipStore struct {
	mu    sync.Mutex
	path  string
	roles map[string][]wornEquipItem
}
type equipStoreFile struct {
	Version int                        `json:"version"`
	Roles   map[string][]wornEquipItem `json:"roles"`
}

func openEquipStore() (*equipStore, error) {
	return openEquipStoreAt(filepath.Join(runtimeProjectRoot, "data", "equip_items.json"))
}
func openEquipStoreAt(path string) (*equipStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("equip store: empty path")
	}
	path = filepath.Clean(path)
	store := &equipStore{path: path, roles: make(map[string][]wornEquipItem)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	var document equipStoreFile
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if document.Version != equipStoreVersion {
		return nil, errors.New("equip store: unsupported version")
	}
	if document.Roles != nil {
		store.roles = document.Roles
	}
	return store, nil
}
func (store *equipStore) Load(roleID role.RoleID) ([]wornEquipItem, bool) {
	if store == nil {
		return nil, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	items, ok := store.roles[strconv.FormatUint(uint64(roleID), 10)]
	return append([]wornEquipItem(nil), items...), ok
}
func (store *equipStore) Save(roleID role.RoleID, items []wornEquipItem) error {
	if store == nil {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := strconv.FormatUint(uint64(roleID), 10)
	copied := append([]wornEquipItem(nil), items...)
	store.roles[key] = copied
	document := equipStoreFile{Version: equipStoreVersion, Roles: store.roles}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(store.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data = append(data, '\n')
	return os.WriteFile(store.path, data, 0o644)
}
func equipBodySlot(equipType string) int32 {
	switch equipType {
	case "Hat", "InnerHat", "FashionHat":
		return 1
	case "Mask":
		return 2
	case "Cloth", "Mantle", "InnerCloth", "InnerMantle", "FashionCoat", "NewFashionCoat":
		return 3
	case "Pants", "InnerPants":
		return 4
	case "Wrist":
		return 6
	case "Leg":
		return 7
	case "Shoes", "InnerShoes", "FashionShoes":
		return 8
	case "Earring":
		return 11
	case "Necklace", "InnerNecklace":
		return 13
	case "Fingerring", "InnerFingerring":
		return 14
	case "Weapon", "InnerWeapon":
		return 22
	case "ShotWeapon":
		return 23
	case "Treasure", "NewTreasure", "FacultyBook", "FacultyPaint", "TaoShaNeiGongBook", "TaoShaZhaoShiBook":
		return 24
	default:
		return 0
	}
}
func isWeaponEquip(equipType string) bool {
	switch equipType {
	case "Weapon", "ShotWeapon", "InnerWeapon", "FacultyWeapon":
		return true
	default:
		return false
	}
}
func bagUniqueID(item bagItem) string {
	hash := uint32(2166136261)
	for i := 0; i < len(item.ConfigID); i++ {
		hash ^= uint32(item.ConfigID[i])
		hash *= 16777619
	}
	return fmt.Sprintf("7100568-%03d-%010d-%04d", (hash>>20)%1000, uint64(hash), hash%10000)
}
func bagItemProps(view uint16, item bagItem) []serverViewProperty {
	common := []serverViewProperty{viewString(0x0007, item.ConfigID), viewInt(0x0761, item.ItemType)}
	switch view {
	case 121:
		if isWeaponEquip(item.EquipType) {
			return weaponRowProps(item)
		}
		return append(common, viewInt(0x0763, item.ViewID), viewInt(0x0764, item.ColorLevel), viewString(0x0765, bagUniqueID(item)), viewInt(0x0766, item.Amount), viewInt(0x0767, item.MaxAmount), viewInt(0x076A, 0), viewInt(0x076D, 0), viewInt(0x0779, item.LogicPack), viewInt(0x077A, item.ArtPack), viewInt(0x0786, 1), viewInt(0x0787, 1), viewInt(0x0788, 1), viewInt(0x078F, item.Hardiness), viewInt(0x0790, item.MaxHardiness), viewByte(0x0791, 2), viewString(0x0792, item.EquipType), viewInt(0x01FD, 0), viewInt(0x01FE, 0), viewByte(0x05B9, 0x29), viewInt(0x079D, 0), viewInt(0x079E, 0), viewInt(0x07A0, 0), viewInt(0x07A1, 0), viewInt(0x07A3, 0), viewInt(0x07B1, 0), viewByte(0x07B2, 0))
	case 123:
		return append(common, viewInt(0x0762, item.TextureType), viewInt(0x0763, item.ViewID), viewInt(0x0764, item.ColorLevel), viewString(0x0765, bagUniqueID(item)), viewInt(0x0766, item.Amount), viewInt(0x0767, item.MaxAmount), viewInt(0x076D, 0), viewInt(0x0779, item.LogicPack), viewInt(0x0787, 1), viewInt(0x07DA, item.FuncPack))
	case 125:
		return append(common, viewInt(0x0763, item.ViewID), viewInt(0x0764, item.ColorLevel), viewString(0x0765, bagUniqueID(item)), viewInt(0x0766, item.Amount), viewInt(0x0767, item.MaxAmount), viewInt(0x0768, 1), viewInt(0x076A, 0), viewInt(0x0779, item.LogicPack), viewInt(0x07DA, item.FuncPack))
	default:
		return append(common, viewInt(0x0762, item.TextureType), viewInt(0x0763, item.ViewID), viewInt(0x0764, item.ColorLevel), viewString(0x0765, bagUniqueID(item)), viewInt(0x0766, item.Amount), viewInt(0x0767, item.MaxAmount), viewInt(0x076A, 0), viewInt(0x076D, 0), viewInt(0x0779, item.LogicPack), viewInt(0x0787, 1), viewInt(0x07DA, item.FuncPack), viewInt(0x07E7, item.PropModifyPack), viewInt(0x07EE, 1))
	}
}
func equipItemProps(item bagItem) []serverViewProperty {
	return []serverViewProperty{viewString(0x0007, item.ConfigID), viewInt(0x0761, item.ItemType), viewInt(0x0763, 2), viewInt(0x0764, item.ColorLevel), viewString(0x0765, bagUniqueID(item)), viewInt(0x0766, 1), viewInt(0x0767, item.MaxAmount), viewInt(0x076A, 0), viewInt(0x076D, 0), viewInt(0x0779, item.LogicPack), viewInt(0x077A, item.ArtPack), viewInt(0x0788, 1), viewInt(0x078F, item.Hardiness), viewInt(0x0790, item.MaxHardiness), viewByte(0x0791, 2), viewString(0x0792, item.EquipType), viewString(0x0793, ""), viewByte(0x05B9, 0x29), viewInt(0x079D, 0), viewInt(0x079E, 0), viewInt(0x07A7, 0), viewInt(0x07B1, 0), viewByte(0x07B2, 0), viewString(0x07C0, "")}
}
func weaponRowProps(item bagItem) []serverViewProperty {
	return []serverViewProperty{viewNest(0x05A0, item.ConfigID), viewInt(0x0761, item.ItemType), viewInt(0x0763, 2), viewInt(0x0764, item.ColorLevel), viewString(0x0765, bagUniqueID(item)), viewInt(0x0766, 1), viewInt(0x0767, item.MaxAmount), viewInt(0x076A, 1), viewInt(0x076D, 0), viewInt(0x0779, item.LogicPack), viewInt(0x077A, item.ArtPack), viewInt(0x0786, 1), viewInt(0x0787, 1), viewInt(0x0788, 1), viewInt(0x078F, item.Hardiness), viewInt(0x0790, item.MaxHardiness), viewByte(0x0791, 2), viewString(0x0792, "Weapon"), viewInt(0x01FD, item.MaxMeleeDamage), viewInt(0x01FE, item.MinMeleeDamage), viewByte(0x05B9, 0x15), viewInt(0x079D, 1), viewInt(0x079E, 2), viewInt(0x07A0, 0), viewInt(0x07A1, 0), viewInt(0x07A3, 1), viewInt(0x07B1, 4000), viewByte(0x07B2, 1)}
}
func (p *playerActor) takeBagItem(view uint16, slot int32) (bagItem, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, item := range p.bagItems {
		if bagViewForViewID(item.ViewID) != view || item.Slot != slot {
			continue
		}
		out := item
		p.bagItems = append(p.bagItems[:i], p.bagItems[i+1:]...)
		return out, true
	}
	return bagItem{}, false
}
func (p *playerActor) peekBagItem(view uint16, slot int32) (bagItem, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, item := range p.bagItems {
		if bagViewForViewID(item.ViewID) == view && item.Slot == slot {
			return item, true
		}
	}
	return bagItem{}, false
}
func (p *playerActor) addEquipItem(item wornEquipItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.equipItems = append(p.equipItems, item)
}
func (p *playerActor) takeEquipItem(slot int32) (wornEquipItem, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, item := range p.equipItems {
		if item.Slot != slot {
			continue
		}
		out := item
		p.equipItems = append(p.equipItems[:i], p.equipItems[i+1:]...)
		return out, true
	}
	return wornEquipItem{}, false
}
func (p *playerActor) hasEquipItem(slot int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, item := range p.equipItems {
		if item.Slot == slot {
			return true
		}
	}
	return false
}
func bagSlotFor(player *playerActor, view uint16) int32 {
	used := make(map[int32]bool)
	for _, item := range player.bagSnapshot() {
		if bagViewForViewID(item.ViewID) == view && item.Slot > 0 {
			used[item.Slot] = true
		}
	}
	for slot := int32(1); ; slot++ {
		if !used[slot] {
			return slot
		}
	}
}
func writeFrames(link sceneMessageConnection, frames ...[]byte) error {
	for i, frame := range frames {
		log.Printf("equip/move s2c frame[%d] opcode=0x%02X len=%d hex=%s", i, frame[0], len(frame), hex.EncodeToString(frame))
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write move frames: %w", err)
		}
	}
	return nil
}
func persistBagEquip(bagStore bagStoreIface, equipStore equipStoreIface, roleID role.RoleID, player *playerActor) {
	if bagStore != nil {
		if err := bagStore.Save(roleID, player.bagSnapshot()); err != nil {
			log.Printf("persist bag after move: %v", err)
		}
	}
	if equipStore != nil {
		if err := equipStore.Save(roleID, player.equipSnapshot()); err != nil {
			log.Printf("persist equip after move: %v", err)
		}
	}
}
func handleMoveItemCustom(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, equipCatalog *equipCatalog, bagStore bagStoreIface, equipStore equipStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 5 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 30 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject MOVEITEM before player spawn", remote)
		return true, nil
	}
	srcView := custom.Values[1].Int32
	srcPos := custom.Values[2].Int32
	dstView := custom.Values[3].Int32
	dstPos := custom.Values[4].Int32
	if srcView <= 0 || srcPos <= 0 || dstView <= 0 || dstPos <= 0 {
		log.Printf("%s: reject malformed MOVEITEM (%d,%d)->(%d,%d)", remote, srcView, srcPos, dstView, dstPos)
		return true, nil
	}
	if isBagView(srcView) && dstView == 1 {
		return applyEquip(link, player, itemCatalog, equipCatalog, bagStore, equipStore, roleID, uint16(srcView), srcPos, dstPos, remote)
	}
	if srcView == 1 && isBagView(dstView) {
		return applyUnequip(link, player, equipCatalog, bagStore, equipStore, roleID, srcPos, uint16(dstView), dstPos, remote)
	}
	if isBagView(srcView) && isBagView(dstView) {
		return applyBagMove(link, player, bagStore, roleID, uint16(srcView), srcPos, uint16(dstView), dstPos, remote)
	}
	log.Printf("%s: ignore unsupported MOVEITEM (%d,%d)->(%d,%d)", remote, srcView, srcPos, dstView, dstPos)
	return true, nil
}
func isBagView(view int32) bool {
	switch view {
	case 2, 121, 123, 125:
		return true
	default:
		return false
	}
}
func serverViewRemove(viewID, objectIndex uint16) []byte {
	msg := make([]byte, 5)
	msg[0] = 0x19
	binary.LittleEndian.PutUint16(msg[1:3], viewID)
	binary.LittleEndian.PutUint16(msg[3:5], objectIndex)
	return msg
}
func handleArrangeItemCustom(link sceneMessageConnection, player *playerActor, bagStore bagStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 4 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 36 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject ARANGEITEM before player spawn", remote)
		return true, nil
	}
	srcView := custom.Values[1].Int32
	if srcView <= 0 {
		beginIndex := custom.Values[2].Int32
		endIndex := custom.Values[3].Int32
		log.Printf("%s: reject malformed ARANGEITEM view=%d begin=%d end=%d", remote, srcView, beginIndex, endIndex)
		return true, nil
	}
	if !isBagView(srcView) {
		log.Printf("%s: ignore ARANGEITEM on non-bag view=%d", remote, srcView)
		return true, nil
	}
	arranged, changed := player.arrangeBagView(uint16(srcView))
	if !changed {
		log.Printf("%s: arrange view=%d no change", remote, srcView)
		return true, nil
	}
	frames := make([][]byte, 0, len(arranged.removes)+len(arranged.adds))
	for _, slot := range arranged.removes {
		frames = append(frames, serverViewRemove(uint16(srcView), uint16(slot)))
	}
	for slot, item := range arranged.adds {
		frame, err := serverViewAdd(uint16(srcView), uint16(slot), bagItemProps(uint16(srcView), item))
		if err != nil {
			return true, err
		}
		frames = append(frames, frame)
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	if bagStore != nil {
		if err := bagStore.Save(roleID, player.bagSnapshot()); err != nil {
			log.Printf("persist bag after arrange: %v", err)
		}
	}
	log.Printf("%s: arranged view=%d removes=%d adds=%d", remote, srcView, len(arranged.removes), len(arranged.adds))
	return true, nil
}
func handleDeleteItemCustom(link sceneMessageConnection, player *playerActor, bagStore bagStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 4 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 34 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject DELETEITEM before player spawn", remote)
		return true, nil
	}
	srcView := custom.Values[1].Int32
	srcPos := custom.Values[2].Int32
	amount := custom.Values[3].Int32
	if srcView <= 0 || srcPos <= 0 {
		log.Printf("%s: reject malformed DELETEITEM view=%d pos=%d amount=%d", remote, srcView, srcPos, amount)
		return true, nil
	}
	if !isBagView(srcView) {
		log.Printf("%s: ignore DELETEITEM on non-bag view=%d pos=%d", remote, srcView, srcPos)
		return true, nil
	}
	if amount <= 0 {
		amount = int32(^uint32(0) >> 1)
	}
	item, remaining, consumed, ok := player.decrementBagItem(uint16(srcView), srcPos, amount)
	if !ok {
		log.Printf("%s: DELETEITEM source empty view=%d pos=%d", remote, srcView, srcPos)
		return true, nil
	}
	var frame []byte
	if consumed {
		frame = serverViewRemove(uint16(srcView), uint16(srcPos))
	} else {
		var err error
		frame, err = serverViewAdd(uint16(srcView), uint16(srcPos), bagItemProps(uint16(srcView), item))
		if err != nil {
			return true, fmt.Errorf("encode partial delete: %w", err)
		}
	}
	if err := link.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("write delete item: %w", err)
	}
	if bagStore != nil {
		if err := bagStore.Save(roleID, player.bagSnapshot()); err != nil {
			log.Printf("persist bag after delete: %v", err)
		}
	}
	if consumed {
		log.Printf("%s: deleted item %s view=%d pos=%d amount=%d (stack cleared)", remote, item.ConfigID, srcView, srcPos, amount)
	} else {
		log.Printf("%s: deleted %d of %s view=%d pos=%d (remaining %d)", remote, amount, item.ConfigID, srcView, srcPos, remaining)
	}
	return true, nil
}
func equipSlotFor(itemType int32) int32 {
	if itemType >= 101 && itemType <= 110 {
		return 22
	}
	return 0
}
func handleUseItemCustom(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, equipCatalog *equipCatalog, dropTable *dropTable, bagStore bagStoreIface, equipStore equipStoreIface, fwzCardStore fwzCardStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 3 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 31 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject USEITEM before player spawn", remote)
		return true, nil
	}
	srcView := custom.Values[1].Int32
	srcPos := custom.Values[2].Int32
	if srcView <= 0 || srcPos <= 0 {
		log.Printf("%s: reject malformed USEITEM view=%d pos=%d", remote, srcView, srcPos)
		return true, nil
	}
	item, ok := player.peekBagItem(uint16(srcView), srcPos)
	if !ok {
		log.Printf("%s: USEITEM source empty view=%d pos=%d", remote, srcView, srcPos)
		return true, nil
	}
	item = enrichBagItem(item, itemCatalog, equipCatalog)
	dstPos := equipSlotFor(item.ItemType)
	if dstPos == 0 {
		dstPos = equipBodySlot(item.EquipType)
	}
	if dstPos > 0 {
		log.Printf("%s: USEITEM equip %s type=%d view=%d pos=%d -> body slot %d", remote, item.ConfigID, item.ItemType, srcView, srcPos, dstPos)
		return applyEquip(link, player, itemCatalog, equipCatalog, bagStore, equipStore, roleID, uint16(srcView), srcPos, dstPos, remote)
	}
	return applyUseConsumable(link, player, itemCatalog, equipCatalog, dropTable, bagStore, fwzCardStore, roleID, uint16(srcView), srcPos, item, remote)
}
func applyUseBuffItem(link sceneMessageConnection, player *playerActor, bagStore bagStoreIface, roleID role.RoleID, srcView uint16, srcPos int32, item bagItem, remote string) (bool, error) {
	updated, remaining, consumed, ok := player.decrementBagItem(srcView, srcPos, 1)
	if !ok {
		log.Printf("%s: USEITEM buff item %s source vanished view=%d pos=%d", remote, item.ConfigID, srcView, srcPos)
		return true, nil
	}
	frames := make([][]byte, 0, 2)
	if consumed {
		frames = append(frames, serverViewRemove(srcView, uint16(srcPos)))
	} else {
		row, err := serverViewAdd(srcView, uint16(srcPos), bagItemProps(srcView, updated))
		if err != nil {
			return true, err
		}
		frames = append(frames, row)
	}
	if item.FuncBuffer == "buff_ride_yufeng" {
		if buffFrame := player.startYufengReady(time.Now()); buffFrame != nil {
			frames = append(frames, buffFrame)
		}
	} else {
		log.Printf("%s: USEITEM buff item %s FuncBuffer=%s not wired yet (view=%d pos=%d)", remote, item.ConfigID, item.FuncBuffer, srcView, srcPos)
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, nil, roleID, player)
	log.Printf("%s: USEITEM buff item %s type=%d view=%d pos=%d remaining=%d", remote, item.ConfigID, item.ItemType, srcView, srcPos, remaining)
	return true, nil
}
func applyUseConsumable(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, equipCatalog *equipCatalog, dropTable *dropTable, bagStore bagStoreIface, fwzCardStore fwzCardStoreIface, roleID role.RoleID, srcView uint16, srcPos int32, item bagItem, remote string) (bool, error) {
	if item.CardID > 0 {
		return applyUseFwzCard(link, player, fwzCardStore, bagStore, roleID, srcView, srcPos, item, remote)
	}
	if strings.HasPrefix(item.ConfigID, "book_") {
		return applyUseSkillBook(link, player, bagStore, roleID, srcView, srcPos, item, remote)
	}
	boxDropID := ""
	if itemCatalog != nil {
		if ti, ok := itemCatalog.Lookup(item.ConfigID); ok && ti.Script == "BoxItem" {
			boxDropID = ti.DropID
		}
	}
	if boxDropID != "" || (dropTable != nil && dropTable.Has(item.ConfigID)) {
		return openGiftBox(link, player, itemCatalog, equipCatalog, dropTable, bagStore, roleID, srcView, srcPos, item, boxDropID, remote)
	}
	isPotion := item.ItemType == 1 || item.ItemType == 2 || item.ItemType == 11 || item.ItemType == 12
	if !isPotion && item.ToolUseEffect == "" && item.FuncBuffer == "" {
		log.Printf("%s: USEITEM item %s type=%d not a supported consumable (view=%d pos=%d)", remote, item.ConfigID, item.ItemType, srcView, srcPos)
		return true, nil
	}
	if item.FuncBuffer != "" {
		return applyUseBuffItem(link, player, bagStore, roleID, srcView, srcPos, item, remote)
	}
	healed, healFrame, err := player.applyConsumableHeal(item.ItemType)
	if err != nil {
		return true, err
	}
	if isPotion && !healed && player.actor.Snapshot().LogicState == 120 {
		log.Printf("%s: USEITEM consumable %s ignored while dead", remote, item.ConfigID)
		return true, nil
	}
	updated, remaining, consumed, ok := player.decrementBagItem(srcView, srcPos, 1)
	if !ok {
		log.Printf("%s: USEITEM consumable %s source vanished view=%d pos=%d", remote, item.ConfigID, srcView, srcPos)
		return true, nil
	}
	frames := make([][]byte, 0, 2)
	if consumed {
		frames = append(frames, serverViewRemove(srcView, uint16(srcPos)))
	} else {
		row, err := serverViewAdd(srcView, uint16(srcPos), bagItemProps(srcView, updated))
		if err != nil {
			return true, err
		}
		frames = append(frames, row)
	}
	if healed && len(healFrame) != 0 {
		frames = append(frames, healFrame)
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, nil, roleID, player)
	log.Printf("%s: USEITEM consumable %s type=%d view=%d pos=%d healed=%t remaining=%d", remote, item.ConfigID, item.ItemType, srcView, srcPos, healed, remaining)
	return true, nil
}
func openGiftBox(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, equipCatalog *equipCatalog, dropTable *dropTable, bagStore bagStoreIface, roleID role.RoleID, srcView uint16, srcPos int32, item bagItem, boxDropID string, remote string) (bool, error) {
	key := boxDropID
	if key == "" {
		key = item.ConfigID
	}
	if dropTable == nil || !dropTable.Has(key) {
		log.Printf("%s: open %s rejected: drop table has no entry for %q", remote, item.ConfigID, key)
		return true, nil
	}
	updated, remaining, consumed, ok := player.decrementBagItem(srcView, srcPos, 1)
	if !ok {
		log.Printf("%s: open %s source vanished view=%d pos=%d", remote, item.ConfigID, srcView, srcPos)
		return true, nil
	}
	frames := make([][]byte, 0, 3)
	if consumed {
		frames = append(frames, serverViewRemove(srcView, uint16(srcPos)))
	} else {
		row, err := serverViewAdd(srcView, uint16(srcPos), bagItemProps(srcView, updated))
		if err != nil {
			return true, err
		}
		frames = append(frames, row)
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rewards, ok := dropTable.roll(key, rng)
	if !ok {
		log.Printf("%s: open %s (%q) rolled nothing", remote, item.ConfigID, key)
		rewards = nil
	}
	for _, reward := range rewards {
		reward = enrichBagItem(reward, itemCatalog, equipCatalog)
		slot := player.addBagItem(reward)
		view := bagViewForViewID(reward.ViewID)
		row, err := serverViewAdd(view, uint16(slot), bagItemProps(view, reward))
		if err != nil {
			return true, err
		}
		frames = append(frames, row)
		log.Printf("%s: open %s reward %s x%d -> bag view=%d slot=%d", remote, item.ConfigID, reward.ConfigID, reward.Amount, view, slot)
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, nil, roleID, player)
	log.Printf("%s: open %s (%q) consumed, remaining=%d rewards=%d", remote, item.ConfigID, key, remaining, len(rewards))
	return true, nil
}
func bagViewToViewID(view uint16) int32 {
	switch view {
	case 121:
		return 2
	case 123:
		return 3
	case 125:
		return 4
	default:
		return 1
	}
}
func applyUnequip(link sceneMessageConnection, player *playerActor, equipCatalog *equipCatalog, bagStore bagStoreIface, equipStore equipStoreIface, roleID role.RoleID, srcPos int32, dstView uint16, dstPos int32, remote string) (bool, error) {
	old, ok := player.takeEquipItem(srcPos)
	if !ok {
		log.Printf("%s: unequip body slot %d empty", remote, srcPos)
		return true, nil
	}
	back := bagItem{ConfigID: old.ConfigID, ItemType: old.ItemType, Amount: 1, ViewID: bagViewToViewID(dstView), EquipType: old.EquipType, ArtPack: old.ArtPack, Hardiness: old.Hardiness, MaxHardiness: old.MaxHardiness}
	slot := dstPos
	if slot <= 0 {
		slot = bagSlotFor(player, dstView)
	}
	back.Slot = slot
	player.addBagItem(back)
	frames := [][]byte{serverViewRemove(1, uint16(srcPos))}
	row, err := serverViewAdd(dstView, uint16(slot), bagItemProps(dstView, back))
	if err != nil {
		return true, err
	}
	frames = append(frames, row)
	if isWeaponEquip(old.EquipType) {
		player.setWeaponItemType(0)
		player.setWeaponMode("")
		player.setWeaponHeldMode("")
		weaponFrames, weaponErr := player.weaponMountFrames("")
		if weaponErr == nil {
			frames = append(frames, weaponFrames...)
		}
	}
	if index, name := appearanceIndexFor(old.EquipType); index != 0 {
		frame, appErr := player.resetAppearanceModel(index, name)
		if appErr == nil && frame != nil {
			frames = append(frames, frame)
		}
	}
	player.applyEquipmentStats(equipCatalog)
	player.syncEquippedResourceCaps()
	statsFrame, statsErr := player.equipmentStatsUpdate()
	if statsErr == nil && statsFrame != nil {
		frames = append(frames, statsFrame)
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, equipStore, roleID, player)
	log.Printf("%s: unequipped %s from body slot %d to view=%d slot=%d", remote, old.ConfigID, srcPos, dstView, slot)
	return true, nil
}
func applyBagMove(link sceneMessageConnection, player *playerActor, bagStore bagStoreIface, roleID role.RoleID, srcView uint16, srcPos int32, dstView uint16, dstPos int32, remote string) (bool, error) {
	item, ok := player.takeBagItem(srcView, srcPos)
	if !ok {
		log.Printf("%s: bag move source empty view=%d pos=%d", remote, srcView, srcPos)
		return true, nil
	}
	srcViewID := bagViewToViewID(srcView)
	dstViewID := bagViewToViewID(dstView)
	var displaced *bagItem
	if occupant, occupied := player.peekBagItem(dstView, dstPos); occupied {
		_ = occupant
		disp, _ := player.takeBagItem(dstView, dstPos)
		if dstView == srcView {
			disp.Slot = srcPos
		} else {
			disp.ViewID = srcViewID
			disp.Slot = 0
		}
		player.addBagItem(disp)
		displaced = &disp
	}
	item.ViewID = dstViewID
	item.Slot = dstPos
	player.addBagItem(item)
	frames := [][]byte{serverViewRemove(srcView, uint16(srcPos))}
	if displaced != nil {
		dispSlot := displaced.Slot
		if dstView != srcView {
			row, err := serverViewAdd(srcView, uint16(dispSlot), bagItemProps(srcView, *displaced))
			if err != nil {
				return true, err
			}
			frames = append(frames, row)
		} else {
			frames = append(frames, serverViewRemove(dstView, uint16(dstPos)))
			row, err := serverViewAdd(srcView, uint16(dispSlot), bagItemProps(srcView, *displaced))
			if err != nil {
				return true, err
			}
			frames = append(frames, row)
		}
	}
	row, err := serverViewAdd(dstView, uint16(dstPos), bagItemProps(dstView, item))
	if err != nil {
		return true, err
	}
	frames = append(frames, row)
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, nil, roleID, player)
	if displaced != nil {
		log.Printf("%s: bag move %s view=%d:%d -> view=%d:%d displaced %s to %d:%d", remote, item.ConfigID, srcView, srcPos, dstView, dstPos, displaced.ConfigID, srcView, displaced.Slot)
	} else {
		log.Printf("%s: bag move %s view=%d:%d -> view=%d:%d", remote, item.ConfigID, srcView, srcPos, dstView, dstPos)
	}
	return true, nil
}
func weaponModeFor(itemType int32) string {
	switch itemType {
	case 101:
		return "blade1_1"
	case 102:
		return "sword1_1"
	case 104:
		return "sblade1_1"
	case 105:
		return "ssword1_1"
	default:
		return ""
	}
}
func applyEquip(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, equipCatalog *equipCatalog, bagStore bagStoreIface, equipStore equipStoreIface, roleID role.RoleID, srcView uint16, srcPos int32, dstPos int32, remote string) (bool, error) {
	item, ok := player.takeBagItem(srcView, srcPos)
	if !ok {
		log.Printf("%s: equip source empty view=%d pos=%d", remote, srcView, srcPos)
		return true, nil
	}
	item = enrichBagItem(item, itemCatalog, equipCatalog)
	fixed := equipSlotFor(item.ItemType)
	if fixed == 0 {
		fixed = equipBodySlot(item.EquipType)
	}
	if fixed <= 0 {
		log.Printf("%s: equip %s type=%d has no body slot, reject", remote, item.ConfigID, item.ItemType)
		return true, nil
	}
	if fixed == 11 || fixed == 14 {
		if player.hasEquipItem(fixed) {
			fixed++
		}
	}
	dstPos = fixed
	returnFrames := make([][]byte, 0)
	old, hasOld := player.takeEquipItem(dstPos)
	if hasOld {
		back := bagItem{ConfigID: old.ConfigID, ItemType: old.ItemType, Amount: 1, ViewID: bagViewToViewID(srcView), EquipType: old.EquipType, ArtPack: old.ArtPack, Hardiness: old.Hardiness, MaxHardiness: old.MaxHardiness}
		if slot := player.addBagItem(back); slot > 0 {
			back.Slot = int32(slot)
			frame, err := serverViewAdd(srcView, uint16(slot), bagItemProps(srcView, back))
			if err != nil {
				return true, err
			}
			returnFrames = append(returnFrames, frame)
		}
	}
	eq := wornEquipItem{ConfigID: item.ConfigID, ItemType: item.ItemType, Hardiness: item.Hardiness, MaxHardiness: item.MaxHardiness, Slot: dstPos}
	mountedModel := ""
	if equipCatalog != nil {
		if cat, ok := equipCatalog.Lookup(item.ConfigID); ok {
			eq.EquipType = cat.EquipType
			eq.ArtPack = cat.ArtPack
			mountedModel = equipCatalog.weaponModelName(cat.ArtPack)
		}
	}
	player.addEquipItem(eq)
	frames := [][]byte{serverViewRemove(srcView, uint16(srcPos))}
	if hasOld {
		frames = append(frames, serverViewRemove(1, uint16(dstPos)))
	}
	bodyProps := equipItemProps(item)
	if isWeaponEquip(eq.EquipType) {
		bodyProps = weaponRowProps(item)
	}
	bodyFrame, err := serverViewAdd(1, uint16(dstPos), bodyProps)
	if err != nil {
		return true, err
	}
	frames = append(frames, bodyFrame)
	frames = append(frames, returnFrames...)
	if isWeaponEquip(eq.EquipType) {
		player.setWeaponItemType(uint8(item.ItemType))
		player.setWeaponMode(weaponModeFor(item.ItemType))
		held := ""
		if equipCatalog != nil {
			held = equipCatalog.actionSetForArtPack(eq.ArtPack)
		}
		player.setWeaponHeldMode(held)
		weaponFrames, weaponErr := player.weaponMountFrames(mountedModel)
		if weaponErr == nil {
			frames = append(frames, weaponFrames...)
		} else {
			log.Printf("%s: build weapon mount frames: %v", remote, weaponErr)
		}
	}
	if index, name := appearanceIndexFor(eq.EquipType); index != 0 {
		appModel := equipmentModels.resolve(item.ConfigID, player.sex)
		if appModel != "" {
			frame, appErr := player.setAppearanceModel(index, name, appModel)
			if appErr == nil && frame != nil {
				frames = append(frames, frame)
			}
		}
	}
	player.applyEquipmentStats(equipCatalog)
	player.syncEquippedResourceCaps()
	statsFrame, statsErr := player.equipmentStatsUpdate()
	if statsErr == nil && statsFrame != nil {
		frames = append(frames, statsFrame)
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, equipStore, roleID, player)
	log.Printf("%s: equipped %s (%s) model=%s from view=%d slot=%d to body slot=%d", remote, item.ConfigID, eq.EquipType, mountedModel, srcView, srcPos, dstPos)
	return true, nil
}

const snapshotPowerLevelProp uint16 = 210
const snapshotLevelTitleProp uint16 = 801
const facultyLevelMaxPower int32 = 240

var facultyLevelOnce sync.Once
var facultyLevelTitles map[int32]string
var facultyLevelLastKey int32

func loadFacultyLevelTitles() {
	facultyLevelOnce.Do(func() {
		facultyLevelTitles = make(map[int32]string)
		path := filepath.Join(defaultModernShareRoot, "faculty", "facultylevel.ini")
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		inConfig := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "\ufeff") {
				line = line[len("\ufeff"):]
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				inConfig = strings.EqualFold(line[1:len(line)-1], "config")
				continue
			}
			if !inConfig || !strings.HasPrefix(line, "r=") {
				continue
			}
			parts := strings.Split(line[2:], ",")
			if len(parts) < 3 {
				continue
			}
			power, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}
			title := strings.TrimSpace(parts[2])
			key := int32(power)
			facultyLevelTitles[key] = title
			if key > facultyLevelLastKey {
				facultyLevelLastKey = key
			}
		}
	})
}
func levelTitleFromPower(power int32) string {
	loadFacultyLevelTitles()
	if title, ok := facultyLevelTitles[power]; ok {
		return title
	}
	if len(facultyLevelTitles) == 0 {
		return "title001"
	}
	if power > facultyLevelLastKey {
		return facultyLevelTitles[facultyLevelLastKey]
	}
	return facultyLevelTitles[0]
}
func powerLevelFromBooks(books []learnedNeiGong) int32 {
	var highest int32
	for i := range books {
		power := books[i].neiGongLevel
		if power <= 0 {
			power = books[i].level
		}
		if power > highest {
			highest = power
		}
	}
	if highest > facultyLevelMaxPower {
		highest = facultyLevelMaxPower
	}
	if highest < 0 {
		highest = 0
	}
	return highest
}
func facultyTipSnapshot(books []learnedNeiGong, target *learnedNeiGong) []clientdata.IndexedProperty {
	power := powerLevelFromBooks(books)
	props := []clientdata.IndexedProperty{{Index: snapshotPowerLevelProp, Name: "PowerLevel", Value: clientdata.Int32Value(power)}, {Index: snapshotLevelTitleProp, Name: "LevelTitle", Value: clientdata.StringValue(levelTitleFromPower(power))}, {Index: 818, Name: "SkillUseValue", Value: clientdata.Int32Value(0)}, {Index: 819, Name: "JingMaiTotalLevel", Value: clientdata.Int32Value(0)}}
	famous := ""
	neiGongUse := int32(0)
	if target != nil {
		famous = fmt.Sprintf("%s,%d", target.configID, target.level)
		neiGongUse = target.power
	}
	props = append(props, clientdata.IndexedProperty{Index: 803, Name: "FamousNeiGongName", Value: clientdata.WideStringValue(famous)}, clientdata.IndexedProperty{Index: 817, Name: "NeigongUseValue", Value: clientdata.Int32Value(neiGongUse)})
	return props
}
func facultyLevelLog(books []learnedNeiGong) string {
	power := powerLevelFromBooks(books)
	return fmt.Sprintf("PowerLevel=%d LevelTitle=%s", power, levelTitleFromPower(power))
}
func handleFreshManChoice(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 2 {
		return false, nil
	}
	choice, ok := custom.Values[1].exactInt32()
	if !ok {
		return false, nil
	}
	label := "新手(走引导)"
	if choice == 5 {
		label = "老手(跳过引导)"
	}
	log.Printf("%s: fresh-man choice=%d (%s)", remote, choice, label)
	return true, nil
}

type fwzCardSnapshot struct {
	Used map[int32]struct{}
	Worn []int32
}
type fwzCardStoreIface interface {
	ClearWorn(role.RoleID, int32) error
	Load(role.RoleID) (fwzCardSnapshot, bool)
	SaveUsed(role.RoleID, int32) error
	SaveWorn(role.RoleID, int32) error
}
type mysqlFwzCardStore struct{ db *sql.DB }

func (store *mysqlFwzCardStore) ready() bool {
	return store != nil && store.db != nil
}
func (store *mysqlFwzCardStore) Load(roleID role.RoleID) (fwzCardSnapshot, bool) {
	if !store.ready() || roleID == 0 {
		return fwzCardSnapshot{}, false
	}
	rows, err := store.db.QueryContext(context.Background(), "SELECT card_id, is_used, is_worn, worn_pos FROM role_fwz WHERE role_id = ? ORDER BY worn_pos", roleID)
	if err != nil {
		return fwzCardSnapshot{}, false
	}
	defer rows.Close()
	snap := fwzCardSnapshot{Used: make(map[int32]struct{})}
	found := false
	for rows.Next() {
		var cardID int32
		var isUsed, isWorn bool
		var wornPos int32
		if err := rows.Scan(&cardID, &isUsed, &isWorn, &wornPos); err != nil {
			continue
		}
		found = true
		if isUsed {
			snap.Used[cardID] = struct{}{}
		}
		if isWorn {
			snap.Worn = append(snap.Worn, cardID)
		}
	}
	return snap, found
}
func (store *mysqlFwzCardStore) SaveUsed(roleID role.RoleID, cardID int32) error {
	if !store.ready() || cardID <= 0 {
		return nil
	}
	_, err := store.db.ExecContext(context.Background(), `INSERT INTO role_fwz(role_id, card_id, is_used) VALUES (?, ?, 1)
ON DUPLICATE KEY UPDATE is_used = 1`, roleID, cardID)
	return err
}
func (store *mysqlFwzCardStore) SaveWorn(roleID role.RoleID, cardID int32) error {
	if !store.ready() || cardID <= 0 {
		return nil
	}
	var pos int
	_ = store.db.QueryRowContext(context.Background(), "SELECT COALESCE(MAX(worn_pos),0)+1 FROM role_fwz WHERE role_id = ? AND is_worn = 1", roleID).Scan(&pos)
	_, err := store.db.ExecContext(context.Background(), `INSERT INTO role_fwz(role_id, card_id, is_used, is_worn, worn_pos) VALUES (?, ?, 1, 1, ?)
ON DUPLICATE KEY UPDATE is_worn = 1, worn_pos = ?`, roleID, cardID, pos, pos)
	return err
}
func (store *mysqlFwzCardStore) ClearWorn(roleID role.RoleID, cardID int32) error {
	if !store.ready() || cardID <= 0 {
		return nil
	}
	_, err := store.db.ExecContext(context.Background(), "UPDATE role_fwz SET is_worn = 0, worn_pos = 0 WHERE role_id = ? AND card_id = ?", roleID, cardID)
	return err
}
func handleFwzCardCustom(link sceneMessageConnection, player *playerActor, store fwzCardStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {
	_ = player
	if len(custom.Values) < 3 {
		log.Printf("%s: reject malformed CARD msg values=%d", remote, len(custom.Values))
		return true, nil
	}
	sub := custom.Values[1].Int32
	cardID := custom.Values[2].Int32
	arguments := make([]string, 0, len(custom.Values)-3)
	for _, value := range custom.Values[3:] {
		arguments = append(arguments, value.String())
	}
	log.Printf("%s: client CARD msg sub=%d card_id=%d raw=[%s]", remote, sub, cardID, strings.Join(arguments, ", "))
	switch sub {
	case 0:
		if cardID <= 0 {
			return true, nil
		}
		if store != nil {
			if err := store.SaveWorn(roleID, cardID); err != nil {
				log.Printf("%s: persist fwz wear card=%d: %v", remote, cardID, err)
			}
		}
		value := uint32(cardID)
		zero := uint32(0)
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, 9, []recordCell{{intValue: &value}, {intValue: &zero}})
		if err != nil {
			return true, err
		}
		if err := link.WriteFrame(frame); err != nil {
			return true, err
		}
		log.Printf("%s: fwz wear card=%d saved", remote, cardID)
		return true, nil
	case 1:
		if cardID <= 0 {
			return true, nil
		}
		if store != nil {
			if err := store.ClearWorn(roleID, cardID); err != nil {
				log.Printf("%s: persist fwz unwear card=%d: %v", remote, cardID, err)
			}
		}
		log.Printf("%s: fwz unwear card=%d saved (UseCardRec rebuild on regrant)", remote, cardID)
		return true, nil
	case 7:
		if cardID <= 0 {
			return true, nil
		}
		if store != nil {
			if err := store.SaveUsed(roleID, cardID); err != nil {
				log.Printf("%s: persist fwz collect card=%d: %v", remote, cardID, err)
			}
		}
		if err := link.WriteFrame(serverRecordAddInt32(playerObjectID, playerOwnerID, 8, uint32(cardID))); err != nil {
			return true, err
		}
		log.Printf("%s: fwz collect card=%d saved", remote, cardID)
		return true, nil
	case 8:
		log.Printf("%s: fwz get-award card=%d not implemented", remote, cardID)
		return true, nil
	default:
		log.Printf("%s: fwz card msg sub=%d unsupported", remote, sub)
		return true, nil
	}
}
func applyUseFwzCard(link sceneMessageConnection, player *playerActor, store fwzCardStoreIface, bagStore bagStoreIface, roleID role.RoleID, srcView uint16, srcPos int32, item bagItem, remote string) (bool, error) {
	cardID := item.CardID
	if cardID <= 0 {
		log.Printf("%s: USEITEM fwz card %s missing CardID (view=%d pos=%d)", remote, item.ConfigID, srcView, srcPos)
		return true, nil
	}
	updated, remaining, consumed, ok := player.decrementBagItem(srcView, srcPos, 1)
	if !ok {
		log.Printf("%s: USEITEM fwz card %s source vanished view=%d pos=%d", remote, item.ConfigID, srcView, srcPos)
		return true, nil
	}
	frames := make([][]byte, 0, 2)
	if consumed {
		frames = append(frames, serverViewRemove(srcView, uint16(srcPos)))
	} else {
		frame, err := serverViewAdd(srcView, uint16(srcPos), bagItemProps(srcView, updated))
		if err != nil {
			return true, err
		}
		frames = append(frames, frame)
	}
	if store != nil {
		if err := store.SaveUsed(roleID, cardID); err != nil {
			log.Printf("%s: persist fwz collect %s card=%d: %v", remote, item.ConfigID, cardID, err)
		}
	}
	if bagStore != nil {
		if err := bagStore.Save(roleID, player.bagSnapshot()); err != nil {
			log.Printf("%s: persist bag after fwz collect %s: %v", remote, item.ConfigID, err)
		}
	}
	frames = append(frames, serverRecordAddInt32(playerObjectID, playerOwnerID, 8, uint32(cardID)))
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	log.Printf("%s: USEITEM fwz card %s card_id=%d collected remaining=%d", remote, item.ConfigID, cardID, remaining)
	return true, nil
}
func grantFwzRecords(conn sceneMessageConnection, store fwzCardStoreIface, roleID role.RoleID) error {
	if store == nil || roleID == 0 {
		return nil
	}
	snap, ok := store.Load(roleID)
	if !ok {
		return nil
	}
	for cardID := range snap.Used {
		if err := conn.WriteFrame(serverRecordAddInt32(playerObjectID, playerOwnerID, 8, uint32(cardID))); err != nil {
			return err
		}
	}
	for _, cardID := range snap.Worn {
		value := uint32(cardID)
		zero := uint32(0)
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, 9, []recordCell{{intValue: &value}, {intValue: &zero}})
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	log.Printf("grant fwz records role=%d collected=%d worn=%d", roleID, len(snap.Used), len(snap.Worn))
	return nil
}

type gmCatalogCategory struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}
type gmCatalogCategories struct {
	Equip []gmCatalogCategory `json:"equip"`
	Tool  []gmCatalogCategory `json:"tool"`
}
type gmCatalogItem struct {
	ConfigID string `json:"config_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

var equipTypeLabels = map[string]string{"Weapon": "武器", "ShotWeapon": "暗器", "InnerWeapon": "内功武器", "Hat": "头盔", "InnerHat": "内功帽", "Cloth": "衣服", "InnerCloth": "内功衣", "Pants": "裤子", "InnerPants": "内功裤", "Shoes": "鞋子", "FashionShoes": "时装鞋", "Wrist": "护腕", "Leg": "腿甲", "Necklace": "项链", "Fingerring": "戒指", "Earring": "耳环", "Mask": "面具", "Suit": "套装", "NewSuit": "新套装", "FashionCoat": "时装衣", "FashionHat": "时装帽", "Treasure": "宝藏", "FacultyPaint": "门派画卷", "FacultyBook": "门派武学书", "TaoShaZhaoShiBook": "绝学招式书", "TaoShaNeiGongBook": "绝学内功书", "NewTreasure": "新宝藏", "NewFashionCoat": "新时装衣", "InnerMantle": "内功披风", "InnerNecklace": "内功项链", "InnerFingerring": "内功戒指", "InnerShoes": "内功鞋"}
var toolScriptLabels = map[string]string{"SkillPage": "技能/武学", "ToolItem": "工具", "BoxItem": "礼盒", "SkillBook": "技能书", "CardItem": "卡片", "WuJiItem": "无极", "LifeExpBook": "生活经验书", "CompensateItem": "补偿物品", "ExchangeItem": "兑换物品", "TitleItem": "称号", "SkillBookTag": "技能书标签", "YouLiItem": "阅历", "SkillScroll": "技能卷轴", "CaiYaoItem": "采药", "HomeZS": "家园", "XueWeiItem": "穴位", "fish": "钓鱼", "LifeTool": "生活工具", "Mount": "坐骑", "Enchase": "镶嵌", "Poison": "毒药", "SetWuXueLevelItem": "武学等级", "Seed": "种子", "Bag": "背包", "XunBaoItem": "寻宝物品", "Wine": "酒", "Qipu": "棋谱", "TableItem": "桌案", "LeiTaiItem": "擂台物品", "TrainPatItem": "训练物品", "SecCardItem": "二级卡片", "MutualActItem": "互动物品", "FacultySkillItem": "门派技能", "DareSchoolTask": "挑战门派任务", "GuildItem": "帮派物品", "CommonIncPropItem": "通用属性物品", "Robe": "外装", "AddPlayerTitle": "称号物品", "SkillItem": "技能物品", "PropPointItem": "属性点物品", "ChoiceBoxItem": "选择礼盒", "HomeXL": "家园·修炼", "HomeBJ": "家园·摆设", "HomeZM": "家园·种养", "Sable": "材料", "Tonic": "药酒", "CapitalItem": "资金物品", "GatherVip": "采集VIP", "HomeBuilding": "家园建筑", "PresentToNpcItem": "赠礼NPC", "BingLu": "兵录", "GeneralTaskItem": "通用任务物品", "CarryMoneyItem": "携带金钱", "FortuneTellingItem": "卜卦物品", "HongChenCollectItem": "红尘收集", "HomeCF": "家园·仓房", "VegetableSeed": "蔬菜种子", "SsfItem": "生死符", "WarItem": "战争物品", "MarryItem": "婚恋物品", "HomeCW": "家园·温床", "ForceSchoolItem": "门派物品", "OutlandValueItem": "外域值", "PointItem": "积分物品", "gmp_con_item": "GM补偿", "FarmTools": "农具", "ShijiaHonourItem": "世家荣誉", "IncValueItem": "成长值物品"}

func gmCatalogCategoriesSnapshot(items *itemCatalog, equips *equipCatalog) gmCatalogCategories {
	equipCounts := make(map[string]int)
	if equips != nil {
		for _, entry := range equips.byID {
			equipCounts[entry.EquipType]++
		}
	}
	toolCounts := make(map[string]int)
	if items != nil {
		for _, entry := range items.byID {
			toolCounts[entry.Script]++
		}
	}
	var out gmCatalogCategories
	for key, count := range equipCounts {
		if key == "" {
			continue
		}
		label := key
		if value, ok := equipTypeLabels[key]; ok {
			label = value
		}
		out.Equip = append(out.Equip, gmCatalogCategory{Key: key, Label: label, Count: count})
	}
	sort.Slice(out.Equip, func(i, j int) bool {
		return out.Equip[i].Count > out.Equip[j].Count
	})
	for key, count := range toolCounts {
		if key == "" {
			continue
		}
		label := key
		if value, ok := toolScriptLabels[key]; ok {
			label = value
		}
		out.Tool = append(out.Tool, gmCatalogCategory{Key: key, Label: label, Count: count})
	}
	sort.Slice(out.Tool, func(i, j int) bool {
		return out.Tool[i].Count > out.Tool[j].Count
	})
	return out
}
func gmDisplayName(name, label, configID string, names map[string]string) string {
	trimmed := strings.TrimSpace(name)
	for _, r := range trimmed {
		if r >= '\u4e00' && r <= '\u9fff' {
			return trimmed
		}
	}
	if localized, ok := names[configID]; ok && strings.TrimSpace(localized) != "" {
		return localized
	}
	return label + " " + configID
}
func gmCatalogSearch(entries []gmCatalogItem, query string, limit int) []gmCatalogItem {
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]gmCatalogItem, 0, 64)
	if limit <= 0 {
		limit = 100
	}
	for _, entry := range entries {
		if len(out) >= limit {
			break
		}
		if query == "" || strings.Contains(strings.ToLower(entry.ConfigID), query) || strings.Contains(strings.ToLower(entry.Name), query) {
			out = append(out, entry)
		}
	}
	return out
}

type skillGrantStore struct{ db *sql.DB }

func (store *skillGrantStore) ready() bool {
	return store != nil && store.db != nil
}

type neiGongBookDef struct {
	configID   string
	staticData int32
	itemType   int32
	school     string
}

var neiGongDefsOnce sync.Once
var neiGongDefs []neiGongBookDef
var neiGongDefsErr error
var qingGongAllOnce sync.Once
var qingGongAllIDs []string
var qingGongAllErr error
var anqiAllOnce sync.Once
var anqiAllIDs []string
var anqiAllErr error
var zhenFaAllOnce sync.Once
var zhenFaAllIDs []string
var zhenFaAllErr error
var jingMaiAllOnce sync.Once
var jingMaiAllIDs []string
var jingMaiAllErr error

func (store *skillGrantStore) GrantAllNewRole(roleID role.RoleID) error {
	if !store.ready() {
		return fmt.Errorf("skill grant store unavailable")
	}
	type item struct {
		stype string
		id    string
	}
	items := make([]item, 0, 6000)
	for _, id := range playerWuxueSkillIDs() {
		items = append(items, item{stype: "wuxue", id: id})
	}
	for _, def := range modernNeiGongBookDefs() {
		items = append(items, item{stype: "neigong", id: def.configID})
	}
	for _, id := range allJiangHuQingGongIDs() {
		items = append(items, item{stype: "qinggong", id: id})
	}
	for _, id := range allAnqiIDs() {
		items = append(items, item{stype: "anqi", id: id})
	}
	for _, id := range allZhenFaIDs() {
		items = append(items, item{stype: "zhenfa", id: id})
	}
	for _, id := range allJingMaiIDs() {
		items = append(items, item{stype: "jingmai", id: id})
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin role_skills grant: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO role_skills(role_id, skill_type, skill_id) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE skill_type = VALUES(skill_type)`)
	if err != nil {
		return fmt.Errorf("prepare role_skills grant: %w", err)
	}
	defer stmt.Close()
	for _, item := range items {
		if _, err := stmt.ExecContext(ctx, roleID, item.stype, item.id); err != nil {
			return fmt.Errorf("grant %s %s: %w", item.stype, item.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit role_skills grant: %w", err)
	}
	return nil
}
func playerWuxueSkillIDs() []string {
	ids := make([]string, 0, len(combatSkills))
	for id := range combatSkills {
		if !strings.HasPrefix(id, "CS_") {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func modernNeiGongBookDefs() []neiGongBookDef {
	neiGongDefsOnce.Do(func() {
		path := filepath.Join(defaultModernShareRoot, "skill", "neigong", "neigong.ini")
		table, err := loadINISections(path)
		if err != nil {
			neiGongDefsErr = err
			return
		}
		ids := make([]string, 0, len(table))
		for id := range table {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			fields := table[id]
			configID := iniValue(fields, "QName")
			if configID == "" {
				configID = id
			}
			neiGongDefs = append(neiGongDefs, neiGongBookDef{configID: configID, staticData: iniInt(fields, "StaticData"), itemType: iniInt(fields, "ItemType"), school: iniValue(fields, "School")})
		}
	})
	if neiGongDefsErr != nil {
		return nil
	}
	return neiGongDefs
}
func allJiangHuQingGongIDs() []string {
	qingGongAllOnce.Do(func() {
		path := filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgskill.ini")
		table, err := loadINISections(path)
		if err != nil {
			qingGongAllErr = err
			return
		}
		for id := range table {
			if !strings.HasPrefix(id, "QG_JH_") {
				continue
			}
			qingGongAllIDs = append(qingGongAllIDs, id)
		}
		sort.Strings(qingGongAllIDs)
	})
	if qingGongAllErr != nil {
		return nil
	}
	return qingGongAllIDs
}
func allAnqiIDs() []string {
	anqiAllOnce.Do(func() {
		path := filepath.Join(defaultModernShareRoot, "skill", "shoufa", "shoufa.ini")
		table, err := loadINISections(path)
		if err != nil {
			anqiAllErr = err
			return
		}
		for id := range table {
			anqiAllIDs = append(anqiAllIDs, id)
		}
		sort.Strings(anqiAllIDs)
	})
	if anqiAllErr != nil {
		return nil
	}
	return anqiAllIDs
}
func allZhenFaIDs() []string {
	zhenFaAllOnce.Do(func() {
		path := filepath.Join(defaultModernShareRoot, "skill", "zhenfa.ini")
		table, err := loadINISections(path)
		if err != nil {
			zhenFaAllErr = err
			return
		}
		for id := range table {
			zhenFaAllIDs = append(zhenFaAllIDs, id)
		}
		sort.Strings(zhenFaAllIDs)
	})
	if zhenFaAllErr != nil {
		return nil
	}
	return zhenFaAllIDs
}
func allJingMaiIDs() []string {
	jingMaiAllOnce.Do(func() {
		path := filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai.ini")
		table, err := loadINISections(path)
		if err != nil {
			jingMaiAllErr = err
			return
		}
		for id, fields := range table {
			if !strings.EqualFold(iniValue(fields, "script"), "JingMai") {
				continue
			}
			jingMaiAllIDs = append(jingMaiAllIDs, id)
		}
		sort.Strings(jingMaiAllIDs)
	})
	if jingMaiAllErr != nil {
		return nil
	}
	return jingMaiAllIDs
}

var defaultToolItemINI = filepath.Join(defaultModernShareRoot, "item", "tool_item.ini")

type depotItem struct {
	ConfigID       string
	ItemType       int32
	Amount         int32
	ViewID         int32
	Name           string
	Script         string
	MaxAmount      int32
	FuncPack       int32
	LogicPack      int32
	PropModifyPack int32
	TextureType    int32
	CardID         int32
	ToolUseEffect  string
	FuncBuffer     string
	WindReadyBuff  string
	WindFixSpeed   int32
	WindMaxSpeed   int32
	HoldPowerTime  int32
	DropID         string
}
type itemCatalog struct{ byID map[string]depotItem }

func loadItemCatalog(path string) (*itemCatalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := simplifiedchinese.GBK.NewDecoder()
	catalog := &itemCatalog{byID: make(map[string]depotItem)}
	current := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			if current == "" {
				continue
			}
			catalog.byID[current] = depotItem{ConfigID: current, Amount: 1, ViewID: 1}
			continue
		}
		if current == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		entry := catalog.byID[current]
		switch strings.TrimSpace(key) {
		case "ItemType":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.ItemType = int32(n)
			}
		case "ViewID":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.ViewID = int32(n)
			}
		case "Name":
			if decoded, err := decoder.String(strings.TrimSpace(value)); err == nil {
				entry.Name = decoded
			}
		case "script":
			entry.Script = strings.TrimSpace(value)
		case "Amount":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.Amount = int32(n)
			}
		case "MaxAmount":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.MaxAmount = int32(n)
			}
		case "FuncPack":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.FuncPack = int32(n)
			}
		case "LogicPack":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.LogicPack = int32(n)
			}
		case "PropModifyPack":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.PropModifyPack = int32(n)
			}
		case "TextureType":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.TextureType = int32(n)
			}
		case "CardID":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.CardID = int32(n)
			}
		case "ToolUseEffect":
			entry.ToolUseEffect = strings.TrimSpace(value)
		case "FuncBuffer":
			entry.FuncBuffer = strings.TrimSpace(value)
		case "WindReadyBuff":
			entry.WindReadyBuff = strings.TrimSpace(value)
		case "WindFixSpeed":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.WindFixSpeed = int32(n)
			}
		case "WindMaxSpeed":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.WindMaxSpeed = int32(n)
			}
		case "HoldPowerTime":
			if n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
				entry.HoldPowerTime = int32(n)
			}
		case "DropID":
			entry.DropID = strings.TrimSpace(value)
		}
		catalog.byID[current] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return catalog, nil
}
func (c *itemCatalog) Lookup(configID string) (depotItem, bool) {
	if c == nil {
		return depotItem{}, false
	}
	entry, ok := c.byID[configID]
	return entry, ok
}
func (c *itemCatalog) Count() int {
	if c == nil {
		return 0
	}
	return len(c.byID)
}
func (p *playerActor) jingMaiProgressFor(id string) jingMaiBookProgress {
	p.mu.Lock()
	defer p.mu.Unlock()
	value, ok := p.jingMaiProgress[id]
	if !ok {
		value = jingMaiBookProgress{level: 1, total: 750}
	}
	if value.total <= 0 {
		value.total = 750
	}
	if value.level <= 0 {
		value.level = 1
	}
	return value
}
func (p *playerActor) setJingMaiProgress(id string, value jingMaiBookProgress) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.jingMaiProgress == nil {
		p.jingMaiProgress = make(map[string]jingMaiBookProgress)
	}
	p.jingMaiProgress[id] = value
}
func (p *playerActor) jingMaiProgressSnapshot() map[string]jingMaiBookProgress {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]jingMaiBookProgress, len(p.jingMaiProgress))
	for id, value := range p.jingMaiProgress {
		out[id] = value
	}
	return out
}
func (p *playerActor) activateJingMai(id string) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.activeJingMai[id]; ok {
		return 0, false
	}
	p.activeJingMai[id] = struct{}{}
	p.activeJingMaiOrder = append(p.activeJingMaiOrder, id)
	return len(p.activeJingMaiOrder) - 1, true
}
func (p *playerActor) deactivateJingMai(id string) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.activeJingMai[id]; !ok {
		return 0, false
	}
	delete(p.activeJingMai, id)
	for row, activeID := range p.activeJingMaiOrder {
		if activeID != id {
			continue
		}
		p.activeJingMaiOrder = append(p.activeJingMaiOrder[:row], p.activeJingMaiOrder[row+1:]...)
		return row, true
	}
	return 0, false
}
func (p *playerActor) setCurJingMai(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.curJingMai != "" && p.curJingMai != id {
		p.lastJingMai = p.curJingMai
	}
	p.curJingMai = id
}
func (p *playerActor) jingMaiSnapshot() (cur string, last string, count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.curJingMai, p.lastJingMai, len(p.activeJingMaiOrder)
}
func (p *playerActor) activatedJingMaiOrder() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.activeJingMaiOrder...)
}
func activateJingMaiRecord(conn sceneMessageConnection, player *playerActor, id string) (bool, error) {
	if player == nil {
		return false, fmt.Errorf("character has not entered a scene")
	}
	if id == "" {
		return false, fmt.Errorf("active jingmai id is empty")
	}
	_, activated := player.activateJingMai(id)
	if !activated {
		return false, nil
	}
	frame := serverRecordAddString(playerObjectID, playerOwnerID, 6, id)
	if err := conn.WriteFrame(frame); err != nil {
		return false, fmt.Errorf("add active_jingmai_rec %s: %w", id, err)
	}
	return true, nil
}
func deactivateJingMaiRecord(conn sceneMessageConnection, player *playerActor, id string) error {
	if player == nil {
		return fmt.Errorf("character has not entered a scene")
	}
	if id == "" {
		return fmt.Errorf("active jingmai id is empty")
	}
	row, deactivated := player.deactivateJingMai(id)
	if !deactivated {
		return nil
	}
	frame := serverRecordDelRow(playerObjectID, playerOwnerID, 6, uint16(row))
	if err := conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("delete active_jingmai_rec %s: %w", id, err)
	}
	return nil
}
func handleJingMaiActiveCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0xa1 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject jingmai active before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 3 || custom.Values[1].Type != 2 || (custom.Values[1].Int32 != 1 && custom.Values[1].Int32 != 2) || custom.Values[2].Type != 6 || custom.Values[2].Text == "" {
		log.Printf("%s: reject malformed ACTIVE_JINGMAI values=%v", remote, custom.Values)
		return true, nil
	}
	action := custom.Values[1].Int32
	id := custom.Values[2].Text
	if action == 1 {
		activated, err := activateJingMaiRecord(link, player, id)
		if err != nil {
			return true, err
		}
		if activated {
			log.Printf("%s: activated jingmai id=%s via active_jingmai_rec", remote, id)
		} else {
			log.Printf("%s: active jingmai already present id=%s", remote, id)
		}
	} else {
		if err := deactivateJingMaiRecord(link, player, id); err != nil {
			return true, err
		}
		log.Printf("%s: deactivated jingmai id=%s via active_jingmai_rec", remote, id)
	}
	update, err := player.vitalUpdate()
	if err != nil {
		return true, fmt.Errorf("encode jingmai count update: %w", err)
	}
	if err := link.WriteFrame(update); err != nil {
		return true, fmt.Errorf("write jingmai count update: %w", err)
	}
	return true, nil
}
func handleJingMaiCultivateCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0x96 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject jingmai cultivate before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) < 2 || custom.Values[1].Type != 2 {
		log.Printf("%s: reject malformed JINGMAI_WUJI values=%v", remote, custom.Values)
		return true, nil
	}
	sub := custom.Values[1].Int32
	if len(custom.Values) == 2 && sub == 2 {
		update, err := player.vitalUpdate()
		if err != nil {
			return true, fmt.Errorf("encode jingmai vitality update: %w", err)
		}
		if err := link.WriteFrame(update); err != nil {
			return true, fmt.Errorf("write jingmai vitality update: %w", err)
		}
		log.Printf("%s: refreshed jingmai vitality request (150/2)", remote)
		return true, nil
	}
	if sub != 1 || len(custom.Values) != 3 || custom.Values[2].Type != 6 {
		log.Printf("%s: unhandled jingmai_wuji sub=%d values=%v", remote, sub, custom.Values)
		return true, nil
	}
	id := custom.Values[2].Text
	player.setCurJingMai(id)
	update, err := player.vitalUpdate()
	if err != nil {
		return true, fmt.Errorf("encode jingmai cultivate update: %w", err)
	}
	if err := link.WriteFrame(update); err != nil {
		return true, fmt.Errorf("write jingmai cultivate update: %w", err)
	}
	if id == "" {
		log.Printf("%s: cleared jingmai cultivation", remote)
	} else {
		log.Printf("%s: set jingmai cultivation id=%s", remote, id)
	}
	return true, nil
}
func grantJingMaiProgress(conn sceneMessageConnection, player *playerActor) error {
	if player == nil {
		return fmt.Errorf("character has not entered a scene")
	}
	for _, id := range player.activatedJingMaiOrder() {
		frame := serverRecordAddString(playerObjectID, playerOwnerID, 6, id)
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("replay active_jingmai_rec %s: %w", id, err)
		}
	}
	return nil
}
func (p *playerActor) zhenQiSnapshot() (day int32, act int32, unused int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.zhenQiDayValue <= 0 {
		p.zhenQiDayValue = zhenQiDailyCap
	}
	return p.zhenQiDayValue, p.zqActValue, p.zqUnUsedValue
}
func (p *playerActor) grantZhenQi(amount int32) bool {
	if amount <= 0 {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.zhenQiDayValue <= 0 {
		p.zhenQiDayValue = zhenQiDailyCap
	}
	available := p.zhenQiDayValue - p.zqActValue - p.zqUnUsedValue
	if available <= 0 {
		return false
	}
	if amount > available {
		amount = available
	}
	p.zqUnUsedValue += amount
	return true
}
func (p *playerActor) consumeZhenQi(amount int32) int32 {
	if amount <= 0 {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.zqUnUsedValue <= 0 {
		return 0
	}
	if amount > p.zqUnUsedValue {
		amount = p.zqUnUsedValue
	}
	p.zqUnUsedValue -= amount
	p.zqActValue += amount
	return amount
}
func jingMaiViewUpdateFrame(id string, progress jingMaiBookProgress) ([]byte, error) {
	slot := uint16(0)
	for index, starterID := range starterJingMaiIDs {
		if starterID == id {
			slot = uint16(index + 1)
			break
		}
	}
	if slot == 0 {
		return nil, fmt.Errorf("meridian %s not in starter view", id)
	}
	levelByte := uint8(progress.level)
	return serverObjectProperty(45, slot, []serverViewProperty{{index: 0x05B9, byte1: &levelByte}, {index: 0x0823, int32: &progress.maxLevel}, {index: 0x02FD, int32: &progress.total}})
}
func (p *playerActor) jingMaiCultivateTick(now time.Time) (string, bool) {
	_ = now
	cur, _, _ := p.jingMaiSnapshot()
	if cur == "" {
		return "", false
	}
	progress := p.jingMaiProgressFor(cur)
	if progress.maxLevel > 0 && progress.maxLevel <= progress.level {
		return "", false
	}
	if p.consumeZhenQi(1) <= 0 {
		return "", false
	}
	progress.fill += 5
	leveled := false
	if progress.total > 0 && progress.fill >= progress.total {
		if progress.maxLevel > 0 && progress.maxLevel <= progress.level {
			progress.fill = progress.total
			progress.level = progress.maxLevel
		} else {
			progress.fill -= progress.total
			progress.level++
			leveled = true
		}
	}
	p.setJingMaiProgress(cur, progress)
	return cur, leveled
}

type jingMaiStatic struct {
	staticData int32
	itemType   int32
	maxLevel   int32
}
type zhenFaViewDef struct {
	id         string
	staticData int32
	itemType   int32
	maxLevel   int32
}
type shouFaViewDef struct {
	id         string
	staticData int32
	itemType   int32
	maxLevel   int32
}

var starterJingMaiIDs = []string{"jm_dantian", "jm_zuyangming", "jm_zushaoyin", "jm_shoutaiyin", "jm_shoutaiyang", "jm_zutaiyin", "jm_zushaoyang", "jm_zujueyin", "jm_shoushaoyang", "jm_shouyangming", "jm_zutaiyang", "jm_shoushaoyin", "jm_yinqiao_wei", "jm_yangqiao_wei"}
var starterZhenFaIDs = []zhenFaViewDef{{id: "zhenfa_jh_01", staticData: 21, itemType: 1010, maxLevel: 1}, {id: "zhenfa_jh_02", staticData: 22, itemType: 1010, maxLevel: 1}, {id: "zhenfa_jh_03", staticData: 23, itemType: 1010, maxLevel: 1}}
var starterShouFaIDs = []shouFaViewDef{{id: "zs_hiddenwp00005", staticData: 1, itemType: 1011, maxLevel: 1}, {id: "zs_hiddenwp00007", staticData: 2, itemType: 1011, maxLevel: 1}, {id: "zs_hiddenwp00009", staticData: 3, itemType: 1011, maxLevel: 1}}
var jingMaiOnce sync.Once
var jingMaiLoadErr error
var jingMaiStaticData = make(map[string]jingMaiStatic)
var zhenFaDefsOnce sync.Once
var zhenFaDefsAll []zhenFaViewDef
var zhenFaDefsErr error
var shouFaDefsOnce sync.Once
var shouFaDefsAll []shouFaViewDef
var shouFaDefsErr error

func loadJingMaiCatalog() error {
	jingMaiOnce.Do(func() {
		table, err := loadINISections(filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai.ini"))
		if err != nil {
			jingMaiLoadErr = err
			return
		}
		staticTable, err := loadINISections(filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai_static.ini"))
		if err != nil {
			jingMaiLoadErr = err
			return
		}
		maxLevelForStatic := func(static int32) int32 {
			if static < 0 {
				return 0
			}
			fields, ok := staticTable[strconv.FormatInt(int64(static), 10)]
			if !ok {
				return 0
			}
			maxVar := iniInt(fields, "MaxVarPropNo")
			minVar := iniInt(fields, "MinVarPropNo")
			if maxVar <= 0 || minVar <= 0 {
				return 0
			}
			return maxVar - minVar + 1
		}
		for id, fields := range table {
			if !strings.EqualFold(iniValue(fields, "script"), "JingMai") {
				continue
			}
			static := iniInt(fields, "StaticData")
			jingMaiStaticData[id] = jingMaiStatic{staticData: static, itemType: iniInt(fields, "ItemType"), maxLevel: maxLevelForStatic(static)}
		}
	})
	return jingMaiLoadErr
}
func grantStarterJingMai(conn sceneMessageConnection, player *playerActor) error {
	if err := loadJingMaiCatalog(); err != nil {
		return fmt.Errorf("load jingmai catalog: %w", err)
	}
	ids := starterJingMaiIDs
	if player != nil && player.progress.xiulian > 0 {
		ids = allJingMaiIDs()
	}
	if err := conn.WriteFrame(serverCreateView(serverViewSpec{ID: 45, Capacity: uint16(len(ids))})); err != nil {
		return fmt.Errorf("create JingMaiContainer: %w", err)
	}
	slot := 0
	for _, id := range ids {
		static, ok := jingMaiStaticData[id]
		if !ok {
			continue
		}
		progress := player.jingMaiProgressFor(id)
		maxLevel := static.maxLevel
		if maxLevel <= 0 {
			maxLevel = 216
		}
		if progress.maxLevel <= 0 {
			progress.maxLevel = maxLevel
			player.setJingMaiProgress(id, progress)
		}
		slot++
		frame, err := serverViewAdd(45, uint16(slot), []serverViewProperty{viewString(0x05A0, id), viewInt(0x0761, static.itemType), viewInt(0x08BD, static.staticData), viewByte(0x05B9, byte(progress.level)), viewInt(0x0823, progress.maxLevel), viewInt(0x02FD, progress.total)})
		if err != nil {
			return fmt.Errorf("encode JingMaiContainer %s: %w", id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add JingMaiContainer %s: %w", id, err)
		}
	}
	return nil
}
func allZhenFaViewDefs() []zhenFaViewDef {
	zhenFaDefsOnce.Do(func() {
		table, err := loadINISections(filepath.Join(defaultModernShareRoot, "skill", "zhenfa.ini"))
		if err != nil {
			zhenFaDefsErr = err
			return
		}
		for _, id := range allZhenFaIDs() {
			fields := table[id]
			if len(fields) == 0 {
				continue
			}
			zhenFaDefsAll = append(zhenFaDefsAll, zhenFaViewDef{id: id, staticData: iniInt(fields, "StaticData"), itemType: iniInt(fields, "ItemType"), maxLevel: 1})
		}
	})
	if zhenFaDefsErr != nil {
		return nil
	}
	return zhenFaDefsAll
}
func grantStarterZhenFa(conn sceneMessageConnection, player *playerActor) error {
	defs := starterZhenFaIDs
	if player != nil && player.progress.xiulian > 0 {
		defs = allZhenFaViewDefs()
	}
	if err := conn.WriteFrame(serverCreateView(serverViewSpec{ID: 47, Capacity: uint16(len(defs))})); err != nil {
		return fmt.Errorf("create ZhenFaContainer: %w", err)
	}
	for slot, def := range defs {
		frame, err := serverViewAdd(47, uint16(slot+1), []serverViewProperty{viewString(0x05A0, def.id), viewInt(0x0761, def.itemType), viewInt(0x08BD, def.staticData), viewByte(0x05B9, 1), viewInt(0x0823, def.maxLevel), viewInt(0x02FD, 750)})
		if err != nil {
			return fmt.Errorf("encode ZhenFaContainer %s: %w", def.id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add ZhenFaContainer %s: %w", def.id, err)
		}
	}
	return nil
}
func allShouFaViewDefs() []shouFaViewDef {
	shouFaDefsOnce.Do(func() {
		table, err := loadINISections(filepath.Join(defaultModernShareRoot, "skill", "shoufa", "shoufa.ini"))
		if err != nil {
			shouFaDefsErr = err
			return
		}
		for _, id := range allAnqiIDs() {
			fields := table[id]
			if len(fields) == 0 {
				continue
			}
			shouFaDefsAll = append(shouFaDefsAll, shouFaViewDef{id: id, staticData: iniInt(fields, "StaticData"), itemType: iniInt(fields, "ItemType"), maxLevel: 1})
		}
	})
	if shouFaDefsErr != nil {
		return nil
	}
	return shouFaDefsAll
}
func grantStarterShouFa(conn sceneMessageConnection, player *playerActor) error {
	defs := starterShouFaIDs
	if player != nil && player.progress.xiulian > 0 {
		defs = allShouFaViewDefs()
	}
	if err := conn.WriteFrame(serverCreateView(serverViewSpec{ID: 48, Capacity: uint16(len(defs))})); err != nil {
		return fmt.Errorf("create ShouFaContainer: %w", err)
	}
	for slot, def := range defs {
		frame, err := serverViewAdd(48, uint16(slot+1), []serverViewProperty{viewString(0x05A0, def.id), viewInt(0x0761, def.itemType), viewInt(0x08BD, def.staticData), viewByte(0x05B9, 1), viewInt(0x0823, def.maxLevel), viewInt(0x02FD, 750)})
		if err != nil {
			return fmt.Errorf("encode ShouFaContainer %s: %w", def.id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add ShouFaContainer %s: %w", def.id, err)
		}
	}
	return nil
}

const jingMaiStoreVersion = 1

type jingMaiProgressStore struct {
	mu    sync.Mutex
	path  string
	roles map[string]jingMaiProgressSnapshot
}
type jingMaiProgressSnapshot struct {
	Active []string                      `json:"active"`
	Cur    string                        `json:"cur"`
	Last   string                        `json:"last"`
	ZhenQi zhenQiSnapshot                `json:"zhenqi"`
	Books  map[string]jingMaiBookPersist `json:"books"`
}
type zhenQiSnapshot struct {
	Day    int32 `json:"day"`
	Act    int32 `json:"act"`
	Unused int32 `json:"unused"`
}
type jingMaiBookPersist struct {
	Level    int32 `json:"level"`
	Fill     int32 `json:"fill"`
	Total    int32 `json:"total"`
	MaxLevel int32 `json:"max_level"`
}
type jingMaiStoreFile struct {
	Version int                                `json:"version"`
	Roles   map[string]jingMaiProgressSnapshot `json:"roles"`
}

func openJingMaiProgressStore() (*jingMaiProgressStore, error) {
	return openJingMaiProgressStoreAt(filepath.Join(runtimeProjectRoot, "data", "jingmai.json"))
}
func openJingMaiProgressStoreAt(path string) (*jingMaiProgressStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("jingmai store: empty path")
	}
	path = filepath.Clean(path)
	store := &jingMaiProgressStore{path: path, roles: make(map[string]jingMaiProgressSnapshot)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read jingmai store: %w", err)
	}
	var document jingMaiStoreFile
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode jingmai store: %w", err)
	}
	if document.Version != jingMaiStoreVersion {
		return nil, fmt.Errorf("decode jingmai store: unsupported version %d", document.Version)
	}
	if document.Roles != nil {
		store.roles = document.Roles
	}
	return store, nil
}
func (store *jingMaiProgressStore) Load(roleID role.RoleID) (jingMaiProgressSnapshot, bool) {
	if store == nil || roleID == 0 {
		return jingMaiProgressSnapshot{}, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	value, exists := store.roles[strconv.FormatUint(uint64(roleID), 10)]
	return cloneJingMaiProgress(value), exists
}
func (store *jingMaiProgressStore) Save(roleID role.RoleID, value jingMaiProgressSnapshot) error {
	if store == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("jingmai store: zero role id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := strconv.FormatUint(uint64(roleID), 10)
	previous, existed := store.roles[key]
	store.roles[key] = cloneJingMaiProgress(value)
	if err := store.persistLocked(); err != nil {
		if existed {
			store.roles[key] = previous
		} else {
			delete(store.roles, key)
		}
		return err
	}
	return nil
}
func (store *jingMaiProgressStore) persistLocked() error {
	document := jingMaiStoreFile{Version: jingMaiStoreVersion, Roles: store.roles}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode jingmai store: %w", err)
	}
	data = append(data, '\n')
	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create jingmai store directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".jingmai-*.tmp")
	if err != nil {
		return fmt.Errorf("create jingmai store temporary: %w", err)
	}
	temporaryPath := temporary.Name()
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("write jingmai store temporary: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("sync jingmai store temporary: %w", err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("close jingmai store temporary: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("replace jingmai store: %w", err)
	}
	return nil
}
func cloneJingMaiProgress(value jingMaiProgressSnapshot) jingMaiProgressSnapshot {
	cloned := value
	cloned.Active = append([]string(nil), value.Active...)
	cloned.Books = make(map[string]jingMaiBookPersist, len(value.Books))
	for id, book := range value.Books {
		cloned.Books[id] = book
	}
	return cloned
}
func (p *playerActor) jingMaiProgressSnapshotFull() jingMaiProgressSnapshot {
	day, act, unused := p.zhenQiSnapshot()
	books := make(map[string]jingMaiBookPersist, len(p.jingMaiProgressSnapshot()))
	for id, progress := range p.jingMaiProgressSnapshot() {
		books[id] = jingMaiBookPersist{Level: progress.level, Fill: progress.fill, Total: progress.total, MaxLevel: progress.maxLevel}
	}
	return jingMaiProgressSnapshot{Active: p.activatedJingMaiOrder(), Cur: func() string {
		cur, _, _ := p.jingMaiSnapshot()
		return cur
	}(), Last: func() string {
		_, last, _ := p.jingMaiSnapshot()
		return last
	}(), ZhenQi: zhenQiSnapshot{Day: day, Act: act, Unused: unused}, Books: books}
}
func (p *playerActor) restoreJingMaiProgressFull(value jingMaiProgressSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeJingMai = make(map[string]struct{}, len(value.Active))
	p.activeJingMaiOrder = append([]string(nil), value.Active...)
	for _, id := range value.Active {
		p.activeJingMai[id] = struct{}{}
	}
	p.curJingMai = value.Cur
	p.lastJingMai = value.Last
	if value.ZhenQi.Day > 0 {
		p.zhenQiDayValue = value.ZhenQi.Day
	}
	p.zqActValue = value.ZhenQi.Act
	p.zqUnUsedValue = value.ZhenQi.Unused
	p.jingMaiProgress = make(map[string]jingMaiBookProgress, len(value.Books))
	for id, book := range value.Books {
		p.jingMaiProgress[id] = jingMaiBookProgress{level: book.Level, fill: book.Fill, total: book.Total, maxLevel: book.MaxLevel}
	}
}
func currentCasterPosition(runtime *sceneRuntime) world.Transform {
	if runtime == nil {
		return world.Transform{}
	}
	return world.Transform{X: runtime.activeRole.Location.Position.X, Y: runtime.activeRole.Location.Position.Y, Z: runtime.activeRole.Location.Position.Z}
}
func playerIdentityFields() []clientdata.FieldSpec {
	return []clientdata.FieldSpec{{Index: 0x00AE, Name: "Camp", Type: clientdata.WireByte}, {Index: 0x00B2, Name: "Name", Type: clientdata.WireWideString}, {Index: 0x00B5, Name: "Type", Type: clientdata.WireByte}, {Index: 0x00B6, Name: "Sex", Type: clientdata.WireByte}, {Index: 0x00B7, Name: "State", Type: clientdata.WireString}, {Index: 0x00B8, Name: "Photo", Type: clientdata.WireString}, {Index: 0x00B9, Name: "Face", Type: clientdata.WireString}, {Index: 0x00BC, Name: "Hair", Type: clientdata.WireString}, {Index: 0x00CB, Name: "CantMove", Type: clientdata.WireByte}, {Index: 0x00CC, Name: "CantAttack", Type: clientdata.WireByte}, {Index: 0x0180, Name: "MoveSpeed", Type: clientdata.WireFloat32}, {Index: 0x0181, Name: "WalkSpeed", Type: clientdata.WireFloat32}, {Index: 0x0182, Name: "RunSpeed", Type: clientdata.WireFloat32}, {Index: 0x0187, Name: "Hat", Type: clientdata.WireString}, {Index: 0x0189, Name: "Cloth", Type: clientdata.WireString}, {Index: 0x018A, Name: "Pants", Type: clientdata.WireString}, {Index: 0x018B, Name: "Shoes", Type: clientdata.WireString}, {Index: 0x0196, Name: "ActionSet", Type: clientdata.WireString}, {Index: 0x01CF, Name: "LogicState", Type: clientdata.WireByte}, {Index: 0x01D1, Name: "HP", Type: clientdata.WireInt32}, {Index: 0x01D2, Name: "MP", Type: clientdata.WireInt32}, {Index: 0x01D5, Name: "HPRatio", Type: clientdata.WireInt32}, {Index: 0x01D6, Name: "MPRatio", Type: clientdata.WireInt32}, {Index: 0x0242, Name: "MaxHP", Type: clientdata.WireInt32}, {Index: 0x0243, Name: "MaxMP", Type: clientdata.WireInt32}, {Index: 0x05B9, Name: "Level", Type: clientdata.WireByte}, {Index: 0x0617, Name: "NpcType", Type: clientdata.WireInt32}, {Index: 0x0634, Name: "Job", Type: clientdata.WireString}}
}
func sendCustomizingRestore(link *transport.Connection, store shortcutStoreIface, roleID role.RoleID, remote string) {
	if link == nil || store == nil || roleID == 0 {
		return
	}
	saved, ok := store.LoadCustomizing(roleID)
	if !ok || saved == "" {
		return
	}
	frame, err := serverCustomIntMessageWithOpcode(0x1E, 705, customString(saved))
	if err != nil {
		return
	}
	if err := link.WriteFrame(frame); err != nil {
		log.Printf("%s: write customizing restore: %v", remote, err)
		return
	}
	log.Printf("%s: restored persisted customizing role=%d bytes=%d", remote, roleID, len(saved))
}
func selectedRoleID(selected *role.RoleSnapshot) role.RoleID {
	if selected == nil {
		return 0
	}
	return selected.ID
}

type mapPathPoint struct {
	X           float32
	Y           float32
	Z           float32
	stopMillis  int64
	standAction string
}

func loadSceneMapPathRoutes(dir, resource string) (map[string][]mapPathPoint, error) {
	dir = strings.TrimSpace(dir)
	resource = strings.TrimSpace(resource)
	if dir == "" || resource == "" {
		return nil, nil
	}
	path := filepath.Join(dir, resource+"_npc.ini")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	routes := make(map[string][]mapPathPoint)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	var current string
	var currentPoints []mapPathPoint
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if current != "" && len(currentPoints) >= 2 {
				routes[current] = currentPoints
			}
			current = strings.TrimSpace(line[1 : len(line)-1])
			currentPoints = nil
			continue
		}
		if !strings.HasPrefix(line, "r=") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 7 {
			continue
		}
		x, xerr := strconv.ParseFloat(strings.TrimSpace(fields[1]), 32)
		y, yerr := strconv.ParseFloat(strings.TrimSpace(fields[2]), 32)
		z, zerr := strconv.ParseFloat(strings.TrimSpace(fields[3]), 32)
		if xerr != nil || yerr != nil || zerr != nil {
			continue
		}
		var stop int64
		stopRaw := strings.TrimSpace(fields[5])
		if stopRaw != "" && stopRaw != "-1" {
			if v, err := strconv.ParseInt(stopRaw, 10, 64); err == nil && v > 0 {
				stop = v
			}
		}
		stand := strings.TrimSpace(fields[6])
		if stand == "" || stand == "nil" {
			stand = "stand"
		}
		currentPoints = append(currentPoints, mapPathPoint{X: float32(x), Y: float32(y), Z: float32(z), stopMillis: stop, standAction: stand})
	}
	if current != "" && len(currentPoints) >= 2 {
		routes[current] = currentPoints
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	log.Printf("mappath: loaded %d official patrol routes from %s", len(routes), path)
	return routes, nil
}
func sceneNameID(resource string) string {
	switch strings.ToLower(resource) {
	case "born02":
		return "erengu_scene"
	case "born03":
		return "yanyuzhuang_scene"
	case "born04":
		return "qiandengzhen_scene"
	case "city05":
		return "chengdu_scene"
	case "school09":
		return "yihuagong_scene"
	default:
		return ""
	}
}
func appendViewProperty(msg *[]byte, property serverViewProperty) error {
	if property.nest != nil {
		*msg = binary.LittleEndian.AppendUint16(*msg, 0x00B5)
		*msg = append(*msg, 8)
		*msg = binary.LittleEndian.AppendUint16(*msg, property.nest.subIndex)
		*msg = binary.LittleEndian.AppendUint32(*msg, uint32(len(*property.nest.text)+1))
		*msg = append(*msg, (*property.nest.text)...)
		*msg = append(*msg, 0)
		return nil
	}
	if property.byte1 != nil {
		*msg = binary.LittleEndian.AppendUint16(*msg, property.index)
		*msg = append(*msg, *property.byte1)
		return nil
	}
	if property.text != nil {
		*msg = binary.LittleEndian.AppendUint16(*msg, property.index)
		*msg = binary.LittleEndian.AppendUint32(*msg, uint32(len(*property.text)+1))
		*msg = append(*msg, (*property.text)...)
		*msg = append(*msg, 0)
		return nil
	}
	if property.int32 != nil {
		*msg = binary.LittleEndian.AppendUint16(*msg, property.index)
		*msg = binary.LittleEndian.AppendUint32(*msg, uint32(*property.int32))
		return nil
	}
	if property.real32 != nil {
		*msg = binary.LittleEndian.AppendUint16(*msg, property.index)
		*msg = binary.LittleEndian.AppendUint32(*msg, math.Float32bits(*property.real32))
		return nil
	}
	return fmt.Errorf("property %d has no value", property.index)
}
func serverObjectProperty(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {
	if latestClientStarterBagView(viewID) {
		properties = latestClientBagWireProperties(properties)
	} else if latestClientNonBagViewOrdinalFamily(viewID) {
		var err error
		properties, err = latestClientNonBagViewProperties(viewID, properties)
		if err != nil {
			return nil, err
		}
	}
	if len(properties) > math.MaxUint16 {
		return nil, fmt.Errorf("view %d object %d property count %d exceeds u16", viewID, objectIndex, len(properties))
	}
	msg := make([]byte, 12)
	msg[0] = 0x10
	msg[1] = 0x01
	binary.LittleEndian.PutUint32(msg[2:6], uint32(viewID))
	binary.LittleEndian.PutUint32(msg[6:10], uint32(objectIndex))
	count := 0
	for _, property := range properties {
		count++
		if property.nest != nil {
			count++
		}
	}
	binary.LittleEndian.PutUint16(msg[10:12], uint16(count))
	for _, property := range properties {
		if err := appendViewProperty(&msg, property); err != nil {
			return nil, fmt.Errorf("view %d object %d property %d: %w", viewID, objectIndex, property.index, err)
		}
	}
	return msg, nil
}

type serverViewNest struct {
	subIndex uint16
	text     *string
}

func viewNest(subIndex uint16, value string) serverViewProperty {
	return serverViewProperty{index: 0x00B5, nest: &serverViewNest{subIndex: subIndex, text: &value}}
}

type mysqlBagStore struct{ db *sql.DB }
type mysqlCurrencyStore struct{ db *sql.DB }
type mysqlEquipStore struct{ db *sql.DB }
type mysqlFacultyStore struct{ db *sql.DB }
type mysqlJingMaiStore struct{ db *sql.DB }
type mysqlQingGongStore struct{ db *sql.DB }
type mysqlShortcutStore struct{ db *sql.DB }

func openGameDataDB() (*sql.DB, error) {
	dsn := os.Getenv("NINEYIN_MYSQL_DSN")
	if dsn == "" {
		return nil, nil
	}
	normalized, err := role.ProductionDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("game data: normalize DSN: %w", err)
	}
	db, err := sql.Open("mysql", normalized)
	if err != nil {
		return nil, fmt.Errorf("game data: open MySQL: %w", err)
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("game data: ping MySQL: %w", err)
	}
	return db, nil
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func nullableInt32(value int32) any {
	if value == 0 {
		return nil
	}
	return value
}
func (store *mysqlBagStore) Load(roleID role.RoleID) ([]bagItem, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, false
	}
	rows, err := store.db.QueryContext(context.Background(), `
SELECT config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness, COALESCE(slot, 0)
FROM role_bag_items WHERE role_id = ? ORDER BY seq ASC`, roleID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var items []bagItem
	for rows.Next() {
		var item bagItem
		var name, equipType sql.NullString
		var artPack, hardiness, maxHardiness, slot sql.NullInt64
		if err := rows.Scan(&item.ConfigID, &item.ItemType, &item.Amount, &item.ViewID, &name, &equipType, &artPack, &hardiness, &maxHardiness, &slot); err != nil {
			return nil, false
		}
		item.Name = name.String
		item.EquipType = equipType.String
		item.ArtPack = int32(artPack.Int64)
		item.Hardiness = int32(hardiness.Int64)
		item.MaxHardiness = int32(maxHardiness.Int64)
		item.Slot = int32(slot.Int64)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false
	}
	return items, len(items) != 0
}
func (store *mysqlBagStore) Save(roleID role.RoleID, items []bagItem) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("bag store: zero role id")
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {
		return err
	}
	for seq, item := range items {
		slot := item.Slot
		if slot <= 0 {
			slot = int32(seq + 1)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO role_bag_items(role_id, seq, slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, roleID, seq, slot, item.ConfigID, item.ItemType, item.Amount, item.ViewID, nullableString(item.Name), nullableString(item.EquipType), nullableInt32(item.ArtPack), nullableInt32(item.Hardiness), nullableInt32(item.MaxHardiness)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (store *mysqlEquipStore) Load(roleID role.RoleID) ([]wornEquipItem, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, false
	}
	rows, err := store.db.QueryContext(context.Background(), `
SELECT config_id, item_type, equip_type, hardiness, max_hardiness, slot, art_pack
FROM role_equip_items WHERE role_id = ? ORDER BY seq ASC`, roleID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var items []wornEquipItem
	for rows.Next() {
		var item wornEquipItem
		var artPack sql.NullInt64
		if err := rows.Scan(&item.ConfigID, &item.ItemType, &item.EquipType, &item.Hardiness, &item.MaxHardiness, &item.Slot, &artPack); err != nil {
			return nil, false
		}
		item.ArtPack = int32(artPack.Int64)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false
	}
	return items, len(items) != 0
}
func (store *mysqlEquipStore) Save(roleID role.RoleID, items []wornEquipItem) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("equip store: zero role id")
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_equip_items WHERE role_id = ?", roleID); err != nil {
		return err
	}
	for seq, item := range items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO role_equip_items(role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, art_pack)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, roleID, seq, item.ConfigID, item.ItemType, item.EquipType, item.Hardiness, item.MaxHardiness, item.Slot, nullableInt32(item.ArtPack)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (store *mysqlFacultyStore) Load(roleID role.RoleID) (facultyProgressSnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return facultyProgressSnapshot{}, false
	}
	var encoded []byte
	err := store.db.QueryRowContext(context.Background(), "SELECT snapshot FROM role_faculty WHERE role_id = ?", roleID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return facultyProgressSnapshot{}, false
	}
	var value facultyProgressSnapshot
	if err := json.Unmarshal(encoded, &value); err != nil {
		return facultyProgressSnapshot{}, false
	}
	return value, true
}
func (store *mysqlFacultyStore) Save(roleID role.RoleID, value facultyProgressSnapshot) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("faculty store: zero role id")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_faculty WHERE role_id = ?", roleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO role_faculty(role_id, snapshot) VALUES (?, ?)", roleID, encoded); err != nil {
		return err
	}
	return tx.Commit()
}
func (store *mysqlJingMaiStore) Load(roleID role.RoleID) (jingMaiProgressSnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return jingMaiProgressSnapshot{}, false
	}
	var encoded []byte
	err := store.db.QueryRowContext(context.Background(), "SELECT snapshot FROM role_jingmai WHERE role_id = ?", roleID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return jingMaiProgressSnapshot{}, false
	}
	var value jingMaiProgressSnapshot
	if err := json.Unmarshal(encoded, &value); err != nil {
		return jingMaiProgressSnapshot{}, false
	}
	return value, true
}
func (store *mysqlJingMaiStore) Save(roleID role.RoleID, value jingMaiProgressSnapshot) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("jingmai store: zero role id")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_jingmai WHERE role_id = ?", roleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO role_jingmai(role_id, snapshot) VALUES (?, ?)", roleID, encoded); err != nil {
		return err
	}
	return tx.Commit()
}
func (store *mysqlShortcutStore) Load(roleID role.RoleID) ([]playerShortcut, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, false
	}
	rows, err := store.db.QueryContext(context.Background(), "SELECT shortcut_index, kind, id FROM role_shortcuts WHERE role_id = ? ORDER BY seq ASC", roleID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var shortcuts []playerShortcut
	for rows.Next() {
		var shortcut playerShortcut
		if err := rows.Scan(&shortcut.index, &shortcut.kind, &shortcut.id); err != nil {
			return nil, false
		}
		shortcuts = append(shortcuts, shortcut)
	}
	if err := rows.Err(); err != nil {
		return nil, false
	}
	return shortcuts, len(shortcuts) != 0
}
func (store *mysqlShortcutStore) Save(roleID role.RoleID, shortcuts []playerShortcut) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_shortcuts WHERE role_id = ?", roleID); err != nil {
		return err
	}
	for seq, shortcut := range shortcuts {
		if _, err := tx.ExecContext(ctx, "INSERT INTO role_shortcuts(role_id, seq, shortcut_index, kind, id) VALUES (?, ?, ?, ?, ?)", roleID, seq, shortcut.index, shortcut.kind, shortcut.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (store *mysqlShortcutStore) LoadKeyBind(roleID role.RoleID) (string, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return "", false
	}
	var value string
	err := store.db.QueryRowContext(context.Background(), "SELECT bind_value FROM role_keybinds WHERE role_id = ?", roleID).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return "", false
	}
	return value, true
}
func (store *mysqlShortcutStore) SaveKeyBind(roleID role.RoleID, keybind string) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_keybinds WHERE role_id = ?", roleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO role_keybinds(role_id, bind_value) VALUES (?, ?)", roleID, keybind); err != nil {
		return err
	}
	return tx.Commit()
}
func (store *mysqlShortcutStore) LoadCustomizing(roleID role.RoleID) (string, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return "", false
	}
	var value string
	err := store.db.QueryRowContext(context.Background(), "SELECT customizing_value FROM role_customizing WHERE role_id = ?", roleID).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return "", false
	}
	return value, true
}
func (store *mysqlShortcutStore) SaveCustomizing(roleID role.RoleID, customizing string) error {
	if store == nil || store.db == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_customizing WHERE role_id = ?", roleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO role_customizing(role_id, customizing_value) VALUES (?, ?)", roleID, customizing); err != nil {
		return err
	}
	return tx.Commit()
}

var wuxueWuxingOnce sync.Once
var wuxueWuxingData map[string]int32

func loadWuxueWuxing() map[string]int32 {
	wuxueWuxingOnce.Do(func() {
		wuxueWuxingData = make(map[string]int32)
		_ = parseModernINI(modernWuxueWuxingPath, func(section, key, value string) error {
			if key != "WuXing" {
				return nil
			}
			parsed, err := parseModernInt32(value)
			if err != nil {
				return nil
			}
			wuxueWuxingData[section] = parsed
			return nil
		})
	})
	return wuxueWuxingData
}
func neiGongDisplayBuffID(buffID string) string {
	if len(buffID) < len("mind_buf_ng_") || !strings.HasPrefix(buffID, "mind_buf_ng_") {
		return buffID
	}
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return buffID
	}
	alternative := "buf_ng_" + strings.TrimPrefix(buffID, "mind_buf_ng_")
	if _, ok := catalog.buffStatic[alternative]; ok {
		return alternative
	}
	return buffID
}
func buffStaticDataForID(buffID string) uint32 {
	if buffID == "" {
		return 0
	}
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return 0
	}
	return catalog.buffStatic[buffID]
}
func neiGongMaxLevel(staticData int32) int32 {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return 20
	}
	rangeValue, ok := catalog.ranges[staticData]
	if !ok || rangeValue.min <= 0 || rangeValue.min > rangeValue.max {
		return 20
	}
	maxLevel := int32(1)
	for id := rangeValue.min; id <= rangeValue.max; id++ {
		record, ok := catalog.varProps[id]
		if ok && record.level > maxLevel {
			maxLevel = record.level
		}
	}
	return maxLevel
}
func innerPowerAttributeName(belongType, configID string) string {
	switch belongType {
	case "ng_mp_tj":
		return "太极"
	case "ng_mp_yg":
		return "阳刚"
	case "ng_mp_yr", "ng_wmp":
		return "阴柔"
	}
	if hasAnyPrefix(configID, []string{"ng_yh_"}) {
		return "阴柔"
	}
	return ""
}
func ambientBoxGatherSpawn(scriptClass string) bool {
	class := strings.ToLower(strings.TrimSpace(scriptClass))
	return class == "boxnpc" || class == "minenpc"
}
func ambientBoxGatherKey(instance clientdata.NPCCreatorInstance) string {
	if before, _, found := strings.Cut(instance.No, ":"); found {
		return before
	}
	return instance.No
}

type patrolWaypoint struct {
	X float32
	Y float32
	Z float32
}
type patrolRoute struct {
	Name   string
	Points []patrolWaypoint
}

func parsePatrolPath(data []byte) ([]patrolWaypoint, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("patrol path: file too small (%d bytes)", len(data))
	}
	count := int(binary.LittleEndian.Uint32(data[:4]))
	if count <= 0 || count > 1_000_000 {
		return nil, fmt.Errorf("patrol path: implausible waypoint count %d", count)
	}
	plausible := func(v float32) bool {
		av := math.Abs(float64(v))
		return !math.IsNaN(av) && av >= 2 && av <= 5000
	}
	if points, ok := parseNavGraph(data, count, plausible); ok {
		return points, nil
	}
	if len(data) >= 4 && (len(data)-4)%24 == 0 {
		if points, ok := parseStride24(data, count, plausible); ok {
			return points, nil
		}
	}
	points, ok := scanWaypoints(data, count, plausible)
	if !ok {
		return nil, fmt.Errorf("patrol path: waypoint count mismatch header=%d after scan", count)
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("patrol path: no waypoints")
	}
	return points, nil
}
func parseNavGraph(data []byte, count int, plausible func(float32) bool) ([]patrolWaypoint, bool) {
	off := 4
	points := make([]patrolWaypoint, 0, count)
	for i := 0; i < count; i++ {
		if off+16 > len(data) {
			return nil, false
		}
		x := math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
		y := math.Float32frombits(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		z := math.Float32frombits(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		edgeCount := int(binary.LittleEndian.Uint32(data[off+12 : off+16]))
		if !plausible(x) || !plausible(y) || !plausible(z) {
			return nil, false
		}
		if edgeCount > 256 {
			return nil, false
		}
		points = append(points, patrolWaypoint{X: x, Y: y, Z: z})
		off += 16 + edgeCount*8
		if off > len(data) {
			return nil, false
		}
	}
	if off != len(data) {
		return nil, false
	}
	return points, true
}
func parseStride24(data []byte, count int, plausible func(float32) bool) ([]patrolWaypoint, bool) {
	points := make([]patrolWaypoint, 0, count)
	for i := 0; i < count; i++ {
		off := 4 + i*24
		if off+12 > len(data) {
			return nil, false
		}
		x := math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
		y := math.Float32frombits(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		z := math.Float32frombits(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		if !plausible(x) || !plausible(y) || !plausible(z) {
			return nil, false
		}
		points = append(points, patrolWaypoint{X: x, Y: y, Z: z})
	}
	return points, true
}
func scanWaypoints(data []byte, count int, plausible func(float32) bool) ([]patrolWaypoint, bool) {
	var points []patrolWaypoint
	groupStart := -1
	for off := 4; off+12 <= len(data); off += 4 {
		x := math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
		y := math.Float32frombits(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		z := math.Float32frombits(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		if !plausible(x) || !plausible(y) || !plausible(z) {
			continue
		}
		if groupStart >= 0 && off-groupStart < 16 {
			continue
		}
		groupStart = off
		points = append(points, patrolWaypoint{X: x, Y: y, Z: z})
	}
	return points, len(points) >= count-128 && len(points) <= count+128
}
func loadPatrolPathFile(path string) ([]patrolWaypoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parsePatrolPath(data)
}
func loadScenePatrolRoutes(dir, resource string) ([]patrolRoute, error) {
	if strings.TrimSpace(dir) == "" || strings.TrimSpace(resource) == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var routes []patrolRoute
	for _, entry := range entries {
		name := entry.Name()
		lower := strings.ToLower(name)
		if entry.IsDir() || !strings.HasSuffix(lower, ".path") {
			continue
		}
		if lower != strings.ToLower(resource) {
			continue
		}
		points, loadErr := loadPatrolPathFile(filepath.Join(dir, name))
		if loadErr != nil {
			return nil, fmt.Errorf("load %s: %w", name, loadErr)
		}
		if len(points) < 2 {
			continue
		}
		routes = append(routes, patrolRoute{Name: name, Points: points})
	}
	return routes, nil
}

var defaultTransFuncPackagePath = runtimeProjectPath("resources", "modern", "share", "npc", "transobj", "transfuncpackage.ini")
var defaultTransPathRecPath = runtimeProjectPath("resources", "modern", "share", "npc", "transobj", "transpathrec.ini")
var defaultScenesPath = runtimeProjectPath("resources", "modern", "share", "rule", "scenes.ini")

type transPathRec struct {
	ID        string
	FromTo    string
	OriginX   float32
	OriginZ   float32
	OriginO   float32
	TargetX   float32
	TargetZ   float32
	TextID    string
	ToolType  string
	TransTool string
}
type npcTransportCatalog struct {
	funcPaths map[string][]string
	pathRecs  map[string]transPathRec
	sceneByID map[string]string
}
type transINIFields struct {
	values map[string]string
	order  []string
}

func loadNPCCatalogTransportData() (*npcTransportCatalog, error) {
	catalog := &npcTransportCatalog{funcPaths: make(map[string][]string), pathRecs: make(map[string]transPathRec), sceneByID: make(map[string]string)}
	if err := catalog.loadFuncPackage(defaultTransFuncPackagePath); err != nil {
		return nil, err
	}
	if err := catalog.loadPathRecs(defaultTransPathRecPath); err != nil {
		return nil, err
	}
	if err := catalog.loadScenes(defaultScenesPath); err != nil {
		return nil, err
	}
	return catalog, nil
}
func (c *npcTransportCatalog) loadFuncPackage(path string) error {
	ordered, err := parseTransINIOrdered(path)
	if err != nil {
		return fmt.Errorf("transfuncpackage: %w", err)
	}
	for id, fields := range ordered {
		paths := make([]string, 0)
		for _, key := range fields.order {
			if !strings.HasPrefix(key, "r") {
				continue
			}
			value := strings.TrimSpace(fields.values[key])
			if value != "" {
				paths = append(paths, value)
			}
		}
		if len(paths) != 0 {
			c.funcPaths[id] = paths
		}
	}
	return nil
}
func (c *npcTransportCatalog) loadPathRecs(path string) error {
	sections, err := parseTransINI(path)
	if err != nil {
		return fmt.Errorf("transpathrec: %w", err)
	}
	for id, fields := range sections {
		rec := transPathRec{ID: id}
		if v, ok := fields["FromTo"]; ok {
			rec.FromTo = strings.TrimSpace(v)
		}
		rec.OriginX = transFloat(fields, "OriginX")
		rec.OriginZ = transFloat(fields, "OriginZ")
		rec.OriginO = transFloat(fields, "OriginO")
		rec.TargetX = transFloat(fields, "TargetX")
		rec.TargetZ = transFloat(fields, "TargetZ")
		rec.TextID = strings.TrimSpace(fields["TextID"])
		rec.ToolType = strings.TrimSpace(fields["ToolType"])
		rec.TransTool = strings.TrimSpace(fields["TransToolID"])
		c.pathRecs[id] = rec
	}
	return nil
}
func (c *npcTransportCatalog) loadScenes(path string) error {
	sections, err := parseTransINI(path)
	if err != nil {
		return fmt.Errorf("scenes.ini: %w", err)
	}
	for id, fields := range sections {
		config := strings.TrimSpace(fields["Config"])
		if config != "" {
			c.sceneByID[id] = config
		}
	}
	return nil
}
func (c *npcTransportCatalog) destinations(transFuncID string) []transPathRec {
	var out []transPathRec
	for _, pathID := range c.funcPaths[transFuncID] {
		rec, ok := c.pathRecs[pathID]
		if ok {
			out = append(out, rec)
		}
	}
	return out
}
func (c *npcTransportCatalog) sceneConfigForID(sceneID string) string {
	if c == nil {
		return sceneID
	}
	if config := c.sceneByID[sceneID]; config != "" {
		return config
	}
	return sceneID
}
func parseFromTo(fromTo string) (from, to string) {
	fromTo = strings.TrimSpace(fromTo)
	if fromTo == "" {
		return "", ""
	}
	parts := strings.Split(fromTo, ">")
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
func parseTransINIOrdered(path string) (map[string]*transINIFields, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sections := make(map[string]*transINIFields)
	current := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			if current != "" {
				if _, ok := sections[current]; !ok {
					sections[current] = &transINIFields{values: make(map[string]string)}
				}
			}
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || current == "" {
			continue
		}
		key = strings.TrimSpace(key)
		fields := sections[current]
		if _, exists := fields.values[key]; exists {
			key += fmt.Sprintf("_%d", len(fields.values))
		}
		value = strings.TrimSpace(value)
		fields.values[key] = value
		fields.order = append(fields.order, key)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sections, nil
}
func parseTransINI(path string) (map[string]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sections := make(map[string]map[string]string)
	current := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			if current != "" {
				if _, ok := sections[current]; !ok {
					sections[current] = make(map[string]string)
				}
			}
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || current == "" {
			continue
		}
		key = strings.TrimSpace(key)
		fields := sections[current]
		if _, exists := fields[key]; exists {
			key += fmt.Sprintf("_%d", len(fields))
		}
		value = strings.TrimSpace(value)
		fields[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sections, nil
}
func transFloat(fields map[string]string, key string) float32 {
	raw := strings.TrimSpace(fields[key])
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 32)
	if err != nil {
		return 0
	}
	return float32(value)
}

const zhenQiDailyCap int32 = 500

type jingMaiBookProgress struct {
	level    int32
	fill     int32
	total    int32
	maxLevel int32
}
type healOverTime struct {
	mp        bool
	amount    int32
	remaining int
	lastTick  time.Time
}
type bagItem struct {
	ConfigID       string `json:"config_id"`
	ItemType       int32  `json:"item_type"`
	Amount         int32  `json:"amount"`
	ViewID         int32  `json:"view_id"`
	Slot           int32  `json:"slot,omitempty"`
	Name           string `json:"name,omitempty"`
	EquipType      string `json:"equip_type,omitempty"`
	ColorLevel     int32  `json:"color_level,omitempty"`
	ArtPack        int32  `json:"art_pack,omitempty"`
	Hardiness      int32  `json:"hardiness,omitempty"`
	MaxHardiness   int32  `json:"max_hardiness,omitempty"`
	MaxAmount      int32  `json:"max_amount,omitempty"`
	FuncPack       int32  `json:"func_pack,omitempty"`
	LogicPack      int32  `json:"logic_pack,omitempty"`
	PropModifyPack int32  `json:"prop_modify_pack,omitempty"`
	TextureType    int32  `json:"texture_type,omitempty"`
	CardID         int32  `json:"card_id,omitempty"`
	ToolUseEffect  string `json:"tool_use_effect,omitempty"`
	FuncBuffer     string `json:"func_buffer,omitempty"`
	MinMeleeDamage int32  `json:"min_melee_damage,omitempty"`
	MaxMeleeDamage int32  `json:"max_melee_damage,omitempty"`
}
type wornEquipItem struct {
	ConfigID     string `json:"config_id"`
	ItemType     int32  `json:"item_type"`
	EquipType    string `json:"equip_type"`
	Hardiness    int32  `json:"hardiness"`
	MaxHardiness int32  `json:"max_hardiness"`
	Slot         int32  `json:"slot"`
	ArtPack      int32  `json:"art_pack,omitempty"`
}
type bagArrangeResult struct {
	removes []int32
	adds    map[int32]bagItem
}

func (p *playerActor) setKeybind(keybind string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keybind = keybind
}
func (p *playerActor) keybindSnapshot() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.keybind
}
func (p *playerActor) applyFullUnlock(store *skillGrantStore, roleID role.RoleID) {
	if store == nil {
		return
	}
	wuxue := playerWuxueSkillIDs()
	p.mu.Lock()
	for _, id := range wuxue {
		maxLevel := skillMaxLevelFor(id)
		if maxLevel <= 0 {
			maxLevel = 1
		}
		p.learnedSkills[id] = maxLevel
	}
	p.mu.Unlock()
	allDefs := modernNeiGongBookDefs()
	neiGongIDs := make([]string, 0, len(allDefs))
	for _, def := range allDefs {
		neiGongIDs = append(neiGongIDs, def.configID)
	}
	if len(neiGongIDs) > 0 {
		have := make(map[string]struct{}, len(neiGongIDs))
		for _, id := range neiGongIDs {
			have[id] = struct{}{}
		}
		filtered := make([]neiGongBookDef, 0, len(neiGongIDs))
		for _, def := range modernNeiGongBookDefs() {
			if _, ok := have[def.configID]; ok {
				filtered = append(filtered, def)
			}
		}
		p.progress.setFullUnlockBooks(filtered)
	}
	jingMai := allJingMaiIDs()
	for _, id := range jingMai {
		p.activateJingMai(id)
	}
	if len(jingMai) > 0 {
		p.setCurJingMai(jingMai[0])
	}
	qgIDs, _ := allJianghuQingGongSkillIDs()
	for _, id := range qgIDs {
		static, ok := qgStaticData[id]
		if !ok {
			continue
		}
		maxLevel := static.maxLevel
		if maxLevel <= 1 {
			maxLevel = 1
		}
		if p.progress.qgLevels == nil {
			p.progress.qgLevels = make(map[string]int32)
		}
		p.progress.qgLevels[id] = maxLevel
	}
	p.progress.xiulian = 2147483647
	p.mu.Lock()
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.mu.Unlock()
	log.Printf("full-unlock applied role=%d wuxue=%d neigong=%d jingmai=%d xiulian=%d", roleID, len(wuxue), len(neiGongIDs), len(jingMai), p.progress.xiulian)
}
func (p *playerActor) learnedSkillCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.learnedSkills)
}
func (p *playerActor) learnedSkillSnapshot() map[string]int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := make(map[string]int32, len(p.learnedSkills))
	for configID, level := range p.learnedSkills {
		snapshot[configID] = level
	}
	return snapshot
}
func (p *playerActor) collectFwzCard(configID string) bool {
	if configID == "" {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.fwzCollected[configID]; exists {
		return false
	}
	p.fwzCollected[configID] = struct{}{}
	return true
}
func (p *playerActor) setPlayerBlock(holding bool, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.parrying == holding {
		return false
	}
	p.parrying = holding
	if holding {
		p.buffs[11] = activePlayerBuff{staticData: 4558, expiresUTC: now.UTC().Add(24 * time.Hour), level: 1}
	} else {
		delete(p.buffs, 11)
	}
	return true
}
func (p *playerActor) updatePosition(x, y, z float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.positionX = x
	p.positionY = y
	p.positionZ = z
	p.lastY = y
	if !p.groundYSet || y <= p.groundY+0.5 {
		p.groundY = y
		p.groundYSet = true
	}
}
func (p *playerActor) authoritativePosition() world.Transform {
	p.mu.Lock()
	defer p.mu.Unlock()
	return world.Transform{X: p.positionX, Y: p.positionY, Z: p.positionZ}
}
func (p *playerActor) setLeapOrigin(origin world.Transform) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.leapOrigin = origin
	p.leapActive = true
}
func (p *playerActor) takeLeapOrigin() (world.Transform, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.leapActive {
		return world.Transform{}, false
	}
	p.leapActive = false
	return p.leapOrigin, true
}
func (p *playerActor) parryingSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.parrying
}
func (p *playerActor) syncEquippedResourceCaps() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.syncEquippedNeiGongResourcesLocked()
}
func (p *playerActor) progressBookUpdateFrame(slot uint16) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.bookFrameUpdate(slot)
}
func (p *playerActor) qingGongViewUpdateFrame() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := p.progress.facultyName
	if _, ok := qgStaticData[id]; !ok {
		return nil, fmt.Errorf("not a qinggong faculty target %q", id)
	}
	level := p.progress.qgLevels[id]
	if level <= 0 {
		level = 1
	}
	slot := qingGongViewSlot(id)
	if slot == 0 {
		return nil, fmt.Errorf("no qinggong view slot for %q", id)
	}
	levelByte := uint8(level)
	return serverObjectProperty(46, slot, []serverViewProperty{{index: 1465, byte1: &levelByte}})
}
func (p *playerActor) restoreCurNeiGong(id string) {
	if id == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.progress.book(id) == nil {
		return
	}
	p.progress.curNeiGong = id
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.syncEquippedNeiGongResourcesLocked()
}
func (p *playerActor) qingGongLevelsSnapshot() map[string]int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.progress.qgLevels) == 0 {
		return nil
	}
	snapshot := make(map[string]int32, len(p.progress.qgLevels))
	for id, level := range p.progress.qgLevels {
		snapshot[id] = level
	}
	return snapshot
}
func (p *playerActor) deactivateQingGong(id string) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.activeQingGong[id]; !exists {
		return 0, false
	}
	delete(p.activeQingGong, id)
	for index, activeID := range p.activeQingGongOrder {
		if activeID != id {
			continue
		}
		p.activeQingGongOrder = append(p.activeQingGongOrder[:index], p.activeQingGongOrder[index+1:]...)
		return index, true
	}
	return 0, false
}
func clampCapital(balance, delta, max int32) int32 {
	next := int64(balance) + int64(delta)
	if next < 0 {
		return 0
	}
	if next > int64(max) {
		return max
	}
	return int32(next)
}
func (p *playerActor) buffConfigID(slot uint16) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok {
		return ""
	}
	if buff.configID != "" {
		return buff.configID
	}
	id, _ := activeBuffConfigID(buff.staticData)
	return id
}
func (p *playerActor) takeBuffSlot(slot uint16, now time.Time) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || !buff.expiresUTC.After(now.UTC()) {
		return "", false
	}
	delete(p.buffs, slot)
	if buff.configID != "" {
		return buff.configID, true
	}
	id, _ := activeBuffConfigID(buff.staticData)
	return id, true
}
func (p *playerActor) qingGongTick() bool {
	return p.actor.RestoreQingGong(time.Now())
}
func (p *playerActor) blockStateFrame(parrying bool, now time.Time) ([]byte, error) {
	blockValue := uint8(0)
	if parrying {
		blockValue = 1
	}
	props := []clientdata.IndexedProperty{{Index: 471, Name: "InParry", Value: clientdata.ByteValue(blockValue)}}
	buffValue := ""
	if parrying {
		buffValue = p.bufferInfo(11, now)
	}
	props = append(props, clientdata.IndexedProperty{Index: 639, Name: bufferInfoPropertyName(11), Value: clientdata.StringValue(buffValue)})
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, props)
}
func (p *playerActor) currentNeiGongAttribute() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.currentNeiGongAttribute()
}
func (p *playerActor) skillElementMult(definition combatSkillDefinition) float64 {
	if definition.attribute == "" {
		return 1
	}
	if definition.attribute == p.currentNeiGongAttribute() {
		return 1.2
	}
	return 1
}
func (p *playerActor) setGold(value int32) {
	p.mu.Lock()
	p.gold = clampCapital(p.gold, value-p.gold, 999999999)
	p.mu.Unlock()
}
func (p *playerActor) setSilverCard(value int32) {
	p.mu.Lock()
	p.silverCard = clampCapital(p.silverCard, value-p.silverCard, 30000000)
	p.mu.Unlock()
}
func (p *playerActor) setSilverTicket(value int32) {
	p.mu.Lock()
	p.silverTicket = clampCapital(p.silverTicket, value-p.silverTicket, 1000000000)
	p.mu.Unlock()
}
func (p *playerActor) addGold(delta int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gold = clampCapital(p.gold, delta, 999999999)
	return p.gold
}
func (p *playerActor) addSilverCard(delta int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.silverCard = clampCapital(p.silverCard, delta, 30000000)
	return p.silverCard
}
func (p *playerActor) currencySnapshot() (silver, gold, silverCard, silverTicket int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.silver, p.gold, p.silverCard, p.silverTicket
}
func (p *playerActor) currencyProperties() []clientdata.IndexedProperty {
	p.mu.Lock()
	silver, gold, silverCard, silverTicket := p.silver, p.gold, p.silverCard, p.silverTicket
	p.mu.Unlock()
	return []clientdata.IndexedProperty{{Index: 417, Name: "CapitalType1", Value: clientdata.Int64Value(int64(silver))}, {Index: 416, Name: "CapitalType0", Value: clientdata.Int64Value(int64(gold))}, {Index: 418, Name: "CapitalType2", Value: clientdata.Int64Value(int64(silverCard))}, {Index: 420, Name: "CapitalType4", Value: clientdata.Int64Value(int64(silverTicket))}}
}
func (p *playerActor) currenciesUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, p.currencyProperties())
}
func (p *playerActor) goldUpdate() ([]byte, error) {
	p.mu.Lock()
	gold := p.gold
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 416, Name: "CapitalType0", Value: clientdata.Int64Value(int64(gold))}})
}
func (p *playerActor) silverCardUpdate() ([]byte, error) {
	p.mu.Lock()
	silverCard := p.silverCard
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 418, Name: "CapitalType2", Value: clientdata.Int64Value(int64(silverCard))}})
}
func (p *playerActor) silverTicketUpdate() ([]byte, error) {
	p.mu.Lock()
	silverTicket := p.silverTicket
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 420, Name: "CapitalType4", Value: clientdata.Int64Value(int64(silverTicket))}})
}
func (p *playerActor) restoreBag(saved []bagItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bagItems = append([]bagItem(nil), saved...)
	p.assignMissingBagSlotsLocked()
}
func (p *playerActor) assignMissingBagSlotsLocked() {
	used := make(map[uint16]map[int32]struct{})
	for _, item := range p.bagItems {
		view := bagViewForViewID(item.ViewID)
		if item.Slot <= 0 {
			continue
		}
		if used[view] == nil {
			used[view] = make(map[int32]struct{})
		}
		used[view][item.Slot] = struct{}{}
	}
	for index := range p.bagItems {
		if p.bagItems[index].Slot > 0 {
			continue
		}
		view := bagViewForViewID(p.bagItems[index].ViewID)
		slots := used[view]
		slot := int32(1)
		for {
			if _, exists := slots[slot]; !exists {
				break
			}
			slot++
		}
		p.bagItems[index].Slot = slot
		if used[view] == nil {
			used[view] = make(map[int32]struct{})
		}
		used[view][slot] = struct{}{}
	}
}
func (p *playerActor) addBagItem(item bagItem) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	view := bagViewForViewID(item.ViewID)
	if item.Slot <= 0 {
		used := make(map[int32]struct{})
		for _, existing := range p.bagItems {
			if bagViewForViewID(existing.ViewID) != view || existing.Slot <= 0 {
				continue
			}
			used[existing.Slot] = struct{}{}
		}
		item.Slot = 1
		for {
			if _, exists := used[item.Slot]; !exists {
				break
			}
			item.Slot++
		}
	}
	p.bagItems = append(p.bagItems, item)
	return int(item.Slot)
}
func (p *playerActor) decrementBagItem(view uint16, slot int32, amount int32) (bagItem, int32, bool, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index, item := range p.bagItems {
		if bagViewForViewID(item.ViewID) != view || item.Slot != slot {
			continue
		}
		if amount < item.Amount {
			item.Amount -= amount
			p.bagItems[index] = item
			return item, item.Amount, false, true
		}
		p.bagItems = append(p.bagItems[:index], p.bagItems[index+1:]...)
		return item, 0, true, true
	}
	return bagItem{}, 0, false, false
}
func (p *playerActor) bagSnapshot() []bagItem {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]bagItem(nil), p.bagItems...)
}
func (p *playerActor) arrangeBagView(view uint16) (bagArrangeResult, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	items := make([]bagItem, 0)
	oldSlots := make(map[int32]struct{})
	for _, item := range p.bagItems {
		if bagViewForViewID(item.ViewID) != view {
			continue
		}
		items = append(items, item)
		if item.Slot > 0 {
			oldSlots[item.Slot] = struct{}{}
		}
	}
	if len(items) == 0 {
		return bagArrangeResult{}, false
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Slot < items[j].Slot
	})
	merged := make([]bagItem, 0, len(items))
	for _, item := range items {
		cap := item.MaxAmount
		if cap <= 0 {
			cap = 1
		}
		rest := item.Amount
		for i := range merged {
			if rest <= 0 {
				break
			}
			if merged[i].ConfigID != item.ConfigID || merged[i].Amount >= cap {
				continue
			}
			room := cap - merged[i].Amount
			take := rest
			if take > room {
				take = room
			}
			merged[i].Amount += take
			rest -= take
		}
		if rest > 0 {
			item.Amount = rest
			merged = append(merged, item)
		}
	}
	result := bagArrangeResult{adds: make(map[int32]bagItem)}
	next := int32(1)
	oldBySlot := make(map[int32]bagItem, len(items))
	for _, item := range items {
		oldBySlot[item.Slot] = item
	}
	for i := range merged {
		item := &merged[i]
		old, hadOld := oldBySlot[next]
		if hadOld && old.ConfigID == item.ConfigID && old.Amount == item.Amount {
			item.Slot = next
			next++
			continue
		}
		item.Slot = next
		result.adds[next] = *item
		next++
	}
	newSlots := make(map[int32]struct{})
	for _, item := range merged {
		newSlots[item.Slot] = struct{}{}
	}
	for slot := range oldSlots {
		_, still := newSlots[slot]
		if still {
			continue
		}
		result.removes = append(result.removes, slot)
	}
	if len(result.adds) == 0 && len(result.removes) == 0 {
		return bagArrangeResult{}, false
	}
	newItems := make([]bagItem, 0, len(p.bagItems))
	for _, item := range p.bagItems {
		if bagViewForViewID(item.ViewID) == view {
			continue
		}
		newItems = append(newItems, item)
	}
	newItems = append(newItems, merged...)
	p.bagItems = newItems
	return result, true
}
func (p *playerActor) restoreEquip(saved []wornEquipItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	items := make([]wornEquipItem, 0, len(saved))
	for _, item := range saved {
		if slot := equipBodySlot(item.EquipType); slot > 0 {
			item.Slot = slot
		}
		items = append(items, item)
	}
	p.equipItems = items
}
func (p *playerActor) equipSnapshot() []wornEquipItem {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]wornEquipItem(nil), p.equipItems...)
}
func (p *playerActor) applyWeapon(weapon string) ([]byte, error) {
	p.mu.Lock()
	p.weapon = weapon
	p.mu.Unlock()
	log.Printf("applyWeapon Weapon@396=%q", weapon)
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 396, Name: "Weapon", Value: clientdata.StringValue(weapon)}})
}
func (p *playerActor) weaponEffectFrame(value string) ([]byte, error) {
	log.Printf("weaponEffectFrame ActionSet=%q (dual 405+406)", value)
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 405, Name: "ActionSet", Value: clientdata.StringValue(value)}, {Index: 406, Name: "ActionSet", Value: clientdata.StringValue(value)}})
}
func (p *playerActor) weaponMountFrames(weapon string) ([][]byte, error) {
	clearEffect, err := p.weaponEffectFrame("")
	if err != nil {
		return nil, err
	}
	clearWeapon, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 396, Name: "Weapon", Value: clientdata.StringValue("")}})
	if err != nil {
		return nil, err
	}
	if weapon == "" {
		return [][]byte{clearEffect, clearWeapon}, nil
	}
	setEffect, err := p.weaponEffectFrame(p.mountedWeaponHeldMode())
	if err != nil {
		return nil, err
	}
	setWeapon, err := p.applyWeapon(weapon)
	if err != nil {
		return nil, err
	}
	setWeaponInfo, err := p.weaponInfoFrame(weapon)
	if err != nil {
		return nil, err
	}
	return [][]byte{clearEffect, clearWeapon, setEffect, setWeapon, setWeaponInfo}, nil
}
func (p *playerActor) weaponInfoFrame(weapon string) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 1598, Name: "WeaponName", Value: clientdata.StringValue(weapon)}, {Index: 1599, Name: "WeaponMode", Value: clientdata.StringValue(p.mountedWeaponMode())}, {Index: 1600, Name: "WeaponType", Value: clientdata.ByteValue(p.mountedWeaponItemType())}})
}
func (p *playerActor) applyShortcutKey(keybind string) ([]byte, error) {
	if keybind == "" {
		return nil, nil
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 198, Name: "ShortcutKey", Value: clientdata.StringValue(keybind)}})
}
func (p *playerActor) setWeaponItemType(itemType uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.weaponItemType = itemType
}
func (p *playerActor) setWeaponMode(mode string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.weaponMode = mode
}
func (p *playerActor) setWeaponHeldMode(mode string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.weaponHeldMode = mode
}
func (p *playerActor) mountedWeaponMode() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.weaponMode
}
func (p *playerActor) mountedWeaponHeldMode() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.weaponHeldMode
}
func (p *playerActor) mountedWeapon() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.weapon
}
func (p *playerActor) mountedWeaponItemType() uint8 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.weaponItemType
}
func (p *playerActor) applyEquipmentStats(catalog *equipCatalog) {
	var minDmg, maxDmg, hpAdd, mpAdd, phyDef, parry int32
	if catalog != nil {
		p.mu.Lock()
		for _, eq := range p.equipItems {
			cat, ok := catalog.byID[eq.ConfigID]
			if !ok {
				continue
			}
			minDmg += cat.MinMeleeDamage
			maxDmg += cat.MaxMeleeDamage
			hpAdd += cat.MaxHPAdd
			mpAdd += cat.MaxMPAdd
			phyDef += cat.PhyDef
			parry += cat.MaxParry
		}
		p.mu.Unlock()
	}
	p.mu.Lock()
	p.progress.attributes.equipMinMeleeDamage = minDmg
	p.progress.attributes.equipMaxMeleeDamage = maxDmg
	p.progress.attributes.equipMaxHPAdd = hpAdd
	p.progress.attributes.equipMaxMPAdd = mpAdd
	p.progress.attributes.equipPhyDef = phyDef
	p.progress.attributes.equipMaxParry = parry
	p.progress.attributes.equipMeleePower = minDmg
	p.mu.Unlock()
}
func (p *playerActor) equipmentStatsUpdate() ([]byte, error) {
	p.mu.Lock()
	props := p.progress.properties()
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, props)
}
func (p *playerActor) applyConsumableHeal(itemType int32) (bool, []byte, error) {
	state := p.actor.Snapshot()
	changed := false
	switch itemType {
	case 1, 11:
		if state.MaxHP <= state.HP {
			break
		}
		amount := state.MaxHP - state.HP
		if itemType == 1 {
			perTick := sitcrossRecoveryPerSecond(state.MaxHP)
			if perTick < amount {
				amount = perTick
			}
			p.mu.Lock()
			p.healOT = healOverTime{amount: perTick, remaining: 4, lastTick: time.Now()}
			p.mu.Unlock()
		}
		changed = p.actor.ApplyHeal(amount)
	case 2, 12:
		if state.MaxMP <= state.MP {
			break
		}
		amount := state.MaxMP - state.MP
		if itemType == 2 {
			perTick := sitcrossRecoveryPerSecond(state.MaxMP)
			if perTick < amount {
				amount = perTick
			}
			p.mu.Lock()
			p.healOT = healOverTime{mp: true, amount: perTick, remaining: 4, lastTick: time.Now()}
			p.mu.Unlock()
		}
		changed = p.actor.ApplyMPRestore(amount)
	}
	if !changed {
		return false, nil, nil
	}
	frame, err := p.vitalUpdate()
	return true, frame, err
}
func (p *playerActor) yufengTick(now time.Time) bool {
	p.mu.Lock()
	if p.yufengMode == 0 {
		p.mu.Unlock()
		return false
	}
	if now.Sub(p.yufengLastRunAt) < time.Second {
		p.yufengRunAccum += time.Second
	}
	if p.yufengMode == 1 {
		if p.yufengRunAccum >= 6*time.Second {
			p.yufengMode = 2
			p.yufengStageAt = now
			p.yufengLastMp = now
			p.buffs[16] = activePlayerBuff{staticData: 10293, expiresUTC: now.UTC().Add(24 * time.Hour), level: 1}
			p.mu.Unlock()
			return true
		}
		if now.Sub(p.yufengStageAt) >= 30*time.Second {
			delete(p.buffs, 16)
			p.yufengMode = 0
			p.mu.Unlock()
			return true
		}
		p.mu.Unlock()
		return false
	}
	if now.Sub(p.yufengLastMp) < 10*time.Second {
		p.mu.Unlock()
		return false
	}
	p.yufengLastMp = now
	p.mu.Unlock()
	state := p.actor.Snapshot()
	if state.LogicState == 1 {
		p.mu.Lock()
		delete(p.buffs, 16)
		p.yufengMode = 0
		p.mu.Unlock()
		return true
	}
	cost := state.MaxMP / 100
	if cost < 1 {
		cost = 1
	}
	if !p.actor.SpendMP(cost) {
		p.mu.Lock()
		delete(p.buffs, 16)
		p.yufengMode = 0
		p.mu.Unlock()
		return true
	}
	return true
}
func (p *playerActor) healOverTimeTick(now time.Time) bool {
	p.mu.Lock()
	ot := p.healOT
	if ot.remaining <= 0 || now.Sub(ot.lastTick) < 2*time.Second {
		p.mu.Unlock()
		return false
	}
	ot.lastTick = now
	ot.remaining--
	p.healOT = ot
	p.mu.Unlock()
	state := p.actor.Snapshot()
	if ot.mp {
		if state.MP >= state.MaxMP {
			return false
		}
		return p.actor.ApplyMPRestore(ot.amount)
	}
	if state.HP >= state.MaxHP {
		return false
	}
	return p.actor.ApplyHeal(ot.amount)
}
func (p *playerActor) startYufengReady(now time.Time) []byte {
	p.mu.Lock()
	p.yufengMode = 1
	p.yufengStageAt = now
	p.yufengRunAccum = 0
	p.yufengLastRunAt = now
	p.buffs[16] = activePlayerBuff{staticData: 10532, expiresUTC: now.Add(30 * time.Second), level: 1}
	p.mu.Unlock()
	frame, err := p.vitalUpdate()
	if err != nil {
		return nil
	}
	return frame
}
func (p *playerActor) recordYufengMotion(x, z float32, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.yufengMode == 0 {
		return
	}
	if x == p.yufengLastX && z == p.yufengLastZ {
		return
	}
	p.yufengLastX = x
	p.yufengLastZ = z
	p.yufengLastRunAt = now
}
func (p *playerActor) clearYufeng() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.yufengMode = 0
	delete(p.buffs, 16)
}
func (p *playerActor) enterCombatPresence() bool {
	p.mu.Lock()
	p.combatPresence = true
	p.mu.Unlock()
	state := p.actor.Snapshot()
	if state.HP <= 0 || state.LogicState == 1 {
		return false
	}
	p.actor.SetLogicState(1)
	return true
}
func (p *playerActor) playerBirthProperties(location role.Position, visual roleVisual) []clientdata.IndexedProperty {
	p.mu.Lock()
	if p.appearance.cloth == "" && p.appearance.pants == "" && p.appearance.shoes == "" {
		p.appearance = visual
		p.defaultAppearance = visual
	}
	current := p.appearance
	p.mu.Unlock()
	state := p.actor.Snapshot()
	motion := p.motionSnapshot()
	weapon := p.mountedWeapon()
	actionSet := p.mountedWeaponHeldMode()
	if actionSet == "" {
		actionSet = current.actionSet
	}
	return []clientdata.IndexedProperty{{Index: 181, Name: "Type", Value: clientdata.ByteValue(2)}, {Index: 182, Name: "Sex", Value: clientdata.ByteValue(visual.sex)}, {Index: 184, Name: "Photo", Value: clientdata.StringValue(visual.photo)}, {Index: 174, Name: "Camp", Value: clientdata.ByteValue(0)}, {Index: 1588, Name: "Job", Value: clientdata.StringValue("")}, {Index: 1465, Name: "Level", Value: clientdata.ByteValue(1)}, {Index: 188, Name: "Hair", Value: clientdata.StringValue(current.hair)}, {Index: 185, Name: "Face", Value: clientdata.StringValue(current.face)}, {Index: 393, Name: "Cloth", Value: clientdata.StringValue(current.cloth)}, {Index: 394, Name: "Pants", Value: clientdata.StringValue(current.pants)}, {Index: 395, Name: "Shoes", Value: clientdata.StringValue(current.shoes)}, {Index: 396, Name: "Weapon", Value: clientdata.StringValue(weapon)}, {Index: 405, Name: "ActionSet", Value: clientdata.StringValue(actionSet)}, {Index: 406, Name: "ActionSet", Value: clientdata.StringValue(actionSet)}, {Index: 183, Name: "State", Value: clientdata.StringValue("stand")}, {Index: 463, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)}, {Index: 384, Name: "MoveSpeed", Value: clientdata.Float32Value(motion.moveSpeed)}, {Index: 385, Name: "WalkSpeed", Value: clientdata.Float32Value(1.25)}, {Index: 386, Name: "RunSpeed", Value: clientdata.Float32Value(motion.runSpeed)}, {Index: 469, Name: "HPRatio", Value: clientdata.Int32Value(int32(int64(state.HP) * 100 / int64(state.MaxHP)))}, {Index: 470, Name: "MPRatio", Value: clientdata.Int32Value(int32(int64(state.MP) * 100 / int64(state.MaxMP)))}, {Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)}, {Index: 579, Name: "MaxMP", Value: clientdata.Int32Value(state.BaseMaxMP)}, {Index: 465, Name: "HP", Value: clientdata.Int32Value(state.HP)}, {Index: 466, Name: "MP", Value: clientdata.Int32Value(state.MP)}, {Index: 391, Name: "Hat", Value: clientdata.StringValue(current.hair)}, {Index: 178, Name: "Name", Value: clientdata.WideStringValue(state.Name)}, {Index: 1559, Name: "NpcType", Value: clientdata.Int32Value(0)}, {Index: 203, Name: "CantMove", Value: clientdata.ByteValue(0)}, {Index: 204, Name: "CantAttack", Value: clientdata.ByteValue(0)}}
}
func (p *playerActor) setAppearanceModel(index uint16, name string, model string) ([]byte, error) {
	p.mu.Lock()
	switch index {
	case 391:
		p.appearance.hair = model
	case 393:
		p.appearance.cloth = model
	case 394:
		p.appearance.pants = model
	case 395:
		p.appearance.shoes = model
	}
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: index, Name: name, Value: clientdata.StringValue(model)}})
}
func (p *playerActor) resetAppearanceModel(index uint16, name string) ([]byte, error) {
	p.mu.Lock()
	var model string
	switch index {
	case 391:
		model = p.defaultAppearance.hair
		p.appearance.hair = model
	case 393:
		model = p.defaultAppearance.cloth
		p.appearance.cloth = model
	case 394:
		model = p.defaultAppearance.pants
		p.appearance.pants = model
	case 395:
		model = p.defaultAppearance.shoes
		p.appearance.shoes = model
	}
	p.mu.Unlock()
	if model == "" {
		return nil, nil
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: index, Name: name, Value: clientdata.StringValue(model)}})
}
func (p *playerActor) playerAppearanceFrame(visual roleVisual) ([]byte, error) {
	p.mu.Lock()
	if p.appearance.cloth == "" && p.appearance.pants == "" && p.appearance.shoes == "" {
		p.appearance = visual
		p.defaultAppearance = visual
	}
	current := p.appearance
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 182, Name: "Sex", Value: clientdata.ByteValue(visual.sex)}, {Index: 184, Name: "Photo", Value: clientdata.StringValue(visual.photo)}, {Index: 188, Name: "Hair", Value: clientdata.StringValue(current.hair)}, {Index: 185, Name: "Face", Value: clientdata.StringValue(current.face)}, {Index: 393, Name: "Cloth", Value: clientdata.StringValue(current.cloth)}, {Index: 394, Name: "Pants", Value: clientdata.StringValue(current.pants)}, {Index: 395, Name: "Shoes", Value: clientdata.StringValue(current.shoes)}, {Index: 406, Name: "ActionSet", Value: clientdata.StringValue(current.actionSet)}, {Index: 391, Name: "Hat", Value: clientdata.StringValue(current.hair)}})
}
func (p *playerActor) playerSnapshot608(location role.Position, visual roleVisual) ([]byte, error) {
	properties := latestClientPlayerWireProperties(p.playerBirthProperties(location, visual))
	return buildSnapshot608Frame(snapshot608PlayerSnapshot{EntityID: uint64(playerObjectID) | uint64(playerOwnerID)<<32, X: location.X, Y: location.Y, Z: location.Z, Orient: location.Orient, Properties: properties})
}
func appearanceIndexFor(equipType string) (uint16, string) {
	switch equipType {
	case "Hat", "InnerHat", "FashionHat":
		return 391, "Hat"
	case "Mask", "Suit", "Cloth", "NewSuit", "InnerCloth", "FashionCoat":
		return 393, "Cloth"
	case "Pants", "InnerPants":
		return 394, "Pants"
	case "Shoes", "InnerShoes", "FashionShoes":
		return 395, "Shoes"
	default:
		return 0, ""
	}
}
func (p *playerProgress) totalResourceAdds() (maxHPAdd int32, maxMPAdd int32) {
	a := p.attributes
	maxHPAdd = a.equipMaxHPAdd
	maxMPAdd = a.equipMaxMPAdd
	if equipped := p.book(p.curNeiGong); equipped != nil {
		a.str += equipped.strAdd
		a.sta += equipped.staAdd
		a.ing += equipped.ingAdd
		a.spi += equipped.spiAdd
		maxHPAdd += equipped.maxHPAdd
		maxMPAdd += equipped.maxMPAdd
	}
	maxHPAdd += a.str*2 + a.sta*7
	maxMPAdd += a.ing*4 + a.spi
	return maxHPAdd, maxMPAdd
}
func (p *playerProgress) setFullUnlockBooks(defs []neiGongBookDef) {
	if len(defs) == 0 {
		return
	}
	books := make([]learnedNeiGong, 0, len(defs))
	for _, def := range defs {
		maxLevel := neiGongMaxLevel(def.staticData)
		if maxLevel <= 0 {
			maxLevel = 1
		}
		books = append(books, learnedNeiGong{configID: def.configID, staticData: def.staticData, itemType: def.itemType, level: maxLevel, maxLevel: maxLevel, total: 750, power: 120, maxPower: 120})
	}
	p.books = books
	if p.book(p.curNeiGong) == nil && len(books) > 0 {
		p.curNeiGong = books[0].configID
	}
	if p.book(p.facultyName) == nil {
		p.facultyName = p.curNeiGong
	}
	p.refreshAllModernEffects()
}
func (p *playerProgress) currentNeiGongAttribute() string {
	equipped := p.book(p.curNeiGong)
	if equipped == nil {
		return ""
	}
	return equipped.attribute
}
func (p *playerProgress) bookFrameUpdate(slot uint16) ([]byte, error) {
	if slot == 0 || int(slot) > len(p.books) {
		return nil, fmt.Errorf("invalid neigong slot %d", slot)
	}
	book := p.books[slot-1]
	return serverObjectProperty(viewportNeiGong, slot, []serverViewProperty{viewByte(0x05B9, byte(book.level)), viewInt(0x0823, book.maxLevel), viewInt(0x08EC, book.neiGongLevel), viewInt(0x02FD, book.total), viewInt(0x08C3, book.wuXing), viewString(0x08ED, neiGongDisplayBuffID(book.buffID))})
}

type qgSkillStatic struct {
	staticData int32
	itemType   int32
	maxLevel   int32
}

var qgPropOnce sync.Once
var qgPropTable iniTable
var qgPropLoadErr error
var qgSkillOnce sync.Once
var qgSkillLoadErr error
var allJianghuQingGongCategoryIDs = []string{"qinggong_1", "qinggong_2", "qinggong_3", "qinggong_4", "qinggong_5", "qinggong_6", "qinggong_7", "qinggong_8", "qinggong_9", "qinggong_10", "qinggong_11", "qinggong_gaojie"}
var allJianghuQingGongCache []string
var qgStaticData = map[string]qgSkillStatic{}

func loadQGSkillProp() (iniTable, error) {
	qgPropOnce.Do(func() {
		qgPropTable, qgPropLoadErr = loadINISections(filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgskillprop.ini"))
	})
	return qgPropTable, qgPropLoadErr
}
func qgSkillMaxLevel(staticData int32) int32 {
	if staticData <= 0 {
		return 1
	}
	table, err := loadQGSkillProp()
	if err != nil {
		return 1
	}
	prefix := strconv.FormatInt(int64(staticData), 10)
	maxLevel := int32(1)
	for section, fields := range table {
		if !strings.HasPrefix(section, prefix) {
			continue
		}
		if level := iniInt(fields, "Level"); level > maxLevel {
			maxLevel = level
		}
	}
	return maxLevel
}
func qingGongSkillIDs() ([]string, error) {
	qgSkillOnce.Do(func() {
		qgSkillLoadErr = loadQGStaticData(starterQingGongSkillIDs)
	})
	return starterQingGongSkillIDs, qgSkillLoadErr
}
func allJianghuQingGongSkillIDs() ([]string, error) {
	qgSkillOnce.Do(func() {
		table, err := loadINISections(filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgskill.ini"))
		if err != nil {
			qgSkillLoadErr = err
			return
		}
		var ids []string
		for id, fields := range table {
			if !strings.HasPrefix(id, "QG_JH_") {
				continue
			}
			ids = append(ids, id)
			staticData := iniInt(fields, "StaticData")
			itemType := iniInt(fields, "ItemType")
			qgStaticData[id] = qgSkillStatic{staticData: staticData, itemType: itemType, maxLevel: qgSkillMaxLevel(staticData)}
		}
		sort.Strings(ids)
		allJianghuQingGongCache = ids
	})
	return allJianghuQingGongCache, qgSkillLoadErr
}
func qingGongViewSlot(id string) uint16 {
	ids, err := allJianghuQingGongSkillIDs()
	if err != nil {
		return 0
	}
	categories := qingGongCategoriesForSkills(ids)
	for i, skillID := range ids {
		if skillID == id {
			return uint16(i + len(categories) + 1)
		}
	}
	return 0
}
func loadQGStaticData(ids []string) error {
	table, err := loadINISections(filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgskill.ini"))
	if err != nil {
		return err
	}
	for _, id := range ids {
		fields, ok := table[id]
		if !ok {
			continue
		}
		staticData := iniInt(fields, "StaticData")
		qgStaticData[id] = qgSkillStatic{staticData: staticData, itemType: iniInt(fields, "ItemType"), maxLevel: qgSkillMaxLevel(staticData)}
	}
	return nil
}
func qingGongCategoriesForSkills(skillIDs []string) []string {
	if len(skillIDs) == 0 {
		return nil
	}
	table, err := loadINISections(filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgskillstatic.ini"))
	if err != nil {
		return nil
	}
	names := make(map[string]struct{})
	for _, id := range skillIDs {
		static, ok := qgStaticData[id]
		if !ok || static.staticData <= 0 {
			continue
		}
		fields, ok := table[fmt.Sprintf("%d", static.staticData)]
		if !ok {
			continue
		}
		name := iniValue(fields, "QingGongName")
		if name == "" {
			continue
		}
		names[name] = struct{}{}
	}
	categories := make([]string, 0, len(names))
	for name := range names {
		categories = append(categories, name)
	}
	sort.Strings(categories)
	return categories
}
func deactivateQingGongRecord(conn sceneMessageConnection, player *playerActor, id string) error {
	if player == nil {
		return fmt.Errorf("character has not entered a scene")
	}
	if id == "" {
		return fmt.Errorf("active qinggong id is empty")
	}
	row, ok := player.deactivateQingGong(id)
	if !ok {
		return nil
	}
	if err := conn.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, 1, uint16(row))); err != nil {
		return fmt.Errorf("delete ActiveQGSkillRec %s: %w", id, err)
	}
	return nil
}
func isJianghuQingGongRole(roleName string) bool {
	return strings.Contains(roleName, "龙傲天")
}

type qingGongSnapshot struct {
	Categories []string `json:"categories"`
	SkillIDs   []string `json:"skill_ids"`
}
type qingGongStoreIface interface {
	Load(role.RoleID) (qingGongSnapshot, bool)
}

func newMySQLQingGongStore(db *sql.DB) *mysqlQingGongStore {
	return &mysqlQingGongStore{db: db}
}
func (store *mysqlQingGongStore) Load(roleID role.RoleID) (qingGongSnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return qingGongSnapshot{}, false
	}
	rows, err := store.db.QueryContext(context.Background(), `
SELECT skill_id FROM role_skills WHERE role_id = ? AND skill_type = 'qinggong'
ORDER BY skill_id`, roleID)
	if err != nil {
		return qingGongSnapshot{}, false
	}
	defer rows.Close()
	var skillIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return qingGongSnapshot{}, false
		}
		skillIDs = append(skillIDs, id)
	}
	if len(skillIDs) == 0 {
		return qingGongSnapshot{}, false
	}
	return qingGongSnapshot{SkillIDs: skillIDs}, true
}
func grantRoleQingGong(conn sceneMessageConnection, store qingGongStoreIface, roleID role.RoleID, roleName string, qgLevels map[string]int32) error {
	if store != nil && roleID != 0 {
		_, granted, err := grantQingGongFromStore(conn, store, roleID, roleName, qgLevels)
		if err != nil {
			return err
		}
		if granted {
			return nil
		}
	}
	return grantStarterQingGong(conn, roleName, qgLevels)
}

const qingGongStoredIDProperty uint16 = 0x05A0
const qingGongStoredItemTypeProperty uint16 = 0x0761
const qingGongStoredStaticProperty uint16 = 0x08BD
const qingGongStoredLevelProperty uint16 = 0x05B9
const qingGongStoredMaxLevelProperty uint16 = 0x0823
const qingGongStoredItemType int32 = 1005
const qingGongStoredViewCapacity uint16 = 512

func grantQingGongFromStore(conn sceneMessageConnection, store qingGongStoreIface, roleID role.RoleID, roleName string, qgLevels map[string]int32) (qingGongSnapshot, bool, error) {
	if store == nil {
		return qingGongSnapshot{}, false, errors.New("qinggong store unavailable")
	}
	snapshot, ok := store.Load(roleID)
	if !ok {
		return qingGongSnapshot{}, false, nil
	}
	if len(snapshot.Categories) == 0 && len(snapshot.SkillIDs) == 0 {
		return snapshot, false, nil
	}
	if err := loadQGStaticData(snapshot.SkillIDs); err != nil {
		return snapshot, true, err
	}
	if len(snapshot.Categories) == 0 {
		snapshot.Categories = qingGongCategoriesForSkills(snapshot.SkillIDs)
	}
	for _, id := range snapshot.Categories {
		if err := conn.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordQingGong, id)); err != nil {
			return snapshot, true, fmt.Errorf("add QingGongRec %s: %w", id, err)
		}
	}
	viewItems := append(append([]string(nil), snapshot.Categories...), snapshot.SkillIDs...)
	frame := serverCreateView(serverViewSpec{ID: viewportQingGong, Capacity: qingGongStoredViewCapacity})
	if err := conn.WriteFrame(frame); err != nil {
		return snapshot, true, fmt.Errorf("create QingGongContainer: %w", err)
	}
	for slot, id := range viewItems {
		props := []serverViewProperty{viewString(qingGongStoredIDProperty, id), viewInt(qingGongStoredItemTypeProperty, qingGongStoredItemType)}
		if static, ok := qgStaticData[id]; ok {
			level := qgLevels[id]
			if level <= 0 {
				level = 1
			}
			maxLevel := static.maxLevel
			if maxLevel <= 1 {
				maxLevel = 1
			}
			if level > maxLevel {
				level = maxLevel
			}
			props = append(props, viewInt(qingGongStoredStaticProperty, static.staticData), viewByte(qingGongStoredLevelProperty, uint8(level)), viewInt(qingGongStoredMaxLevelProperty, maxLevel))
		}
		frame, err := serverViewAdd(viewportQingGong, uint16(slot+1), props)
		if err != nil {
			return snapshot, true, fmt.Errorf("encode QingGongContainer %s: %w", id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return snapshot, true, fmt.Errorf("add QingGongContainer %s: %w", id, err)
		}
	}
	if err := grantFunctionUnlock(conn); err != nil {
		return snapshot, true, err
	}
	return snapshot, true, nil
}

var defaultQuestTaskRoot = runtimeProjectPath("resources", "modern", "share", "task")

type huntTarget struct {
	npcID string
	count int32
}

func loadQuestCatalogFromTables(taskRoot string) (map[uint32]*questTemplate, error) {
	taskDir := filepath.Join(taskRoot, "task")
	taskFilesDir := filepath.Join(taskDir, "task")
	huntDir := filepath.Join(taskDir, "huntlist")
	huntByTask, err := loadHuntIndex(huntDir)
	if err != nil {
		return nil, err
	}
	taskFiles, err := readTaskFileIndex(filepath.Join(taskFilesDir, "task_index_file.ini"))
	if err != nil {
		taskFiles = scanTaskFiles(taskFilesDir)
	}
	catalog := make(map[uint32]*questTemplate, 4096)
	if len(taskFiles) == 0 {
		return nil, fmt.Errorf("no task files found under %s", taskFilesDir)
	}
	for _, fileName := range taskFiles {
		hdrs, rows, err := parseTabTable(filepath.Join(taskFilesDir, fileName))
		if err != nil {
			continue
		}
		idx := headerIndex(hdrs)
		if idx["ID"] < 0 || idx["AcceptNpc"] < 0 || idx["SubmitNpc"] < 0 {
			continue
		}
		for _, row := range rows {
			id := parseCellUint(row, idx["ID"])
			if id == 0 {
				continue
			}
			submitSceneID := uint32(0)
			if submitSceneIndex, exists := idx["SubmitSceneID"]; exists {
				submitSceneID = parseCellUint(row, submitSceneIndex)
			}
			q := &questTemplate{ID: id, Line: int32(parseCellUint(row, idx["Line"])), SubmitSceneID: submitSceneID, AcceptNPC: cell(row, idx["AcceptNpc"]), SubmitNPC: cell(row, idx["SubmitNpc"]), TitleID: cell(row, idx["TitleId"]), ContextID: cell(row, idx["ContextId"]), CompleteID: firstDialogToken(cell(row, idx["CompleteDialogId"])), TargetID: cell(row, idx["TaskTargetId"]), MenuID: menuToken(cell(row, idx["AcceptDialogId"])), TaskSubID: strconv.FormatUint(uint64(id), 10)}
			if targets := huntByTask[id]; len(targets) != 0 {
				q.KillNPC = targets[0].npcID
				q.KillCount = targets[0].count
			}
			if q.AcceptNPC == "" && q.SubmitNPC == "" {
				continue
			}
			catalog[id] = q
		}
	}
	return catalog, nil
}
func loadHuntIndex(dir string) (map[uint32][]huntTarget, error) {
	hunt := make(map[uint32][]huntTarget, 4096)
	entries, err := filepath.Glob(filepath.Join(dir, "huntlist_*.txt"))
	if err != nil {
		return hunt, nil
	}
	for _, entry := range entries {
		hdrs, rows, err := parseTabTable(entry)
		if err != nil {
			continue
		}
		idx := headerIndex(hdrs)
		if idx["TaskID"] < 0 || idx["NpcId"] < 0 || idx["Count"] < 0 {
			continue
		}
		for _, row := range rows {
			taskID := parseCellUint(row, idx["TaskID"])
			if taskID == 0 {
				continue
			}
			npc := cell(row, idx["NpcId"])
			if npc == "" {
				continue
			}
			count := int32(parseCellUint(row, idx["Count"]))
			hunt[taskID] = append(hunt[taskID], huntTarget{npcID: npc, count: count})
		}
	}
	return hunt, nil
}
func readTaskFileIndex(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	files := make([]string, 0, 256)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '[' {
			continue
		}
		_, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name := strings.TrimSpace(value)
		if name != "" {
			files = append(files, name)
		}
	}
	return files, scanner.Err()
}
func scanTaskFiles(dir string) []string {
	entries, _ := filepath.Glob(filepath.Join(dir, "task_*.txt"))
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		files = append(files, filepath.Base(entry))
	}
	return files
}
func parseTabTable(path string) ([]string, [][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	lines := make([]string, 0, 512)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	if len(lines) < 2 {
		return nil, nil, fmt.Errorf("table %s has no header", path)
	}
	hdr := strings.Split(lines[1], "\t")
	rows := make([][]string, 0, len(lines)-2)
	for _, line := range lines[2:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}
	return hdr, rows, nil
}
func headerIndex(hdr []string) map[string]int {
	idx := make(map[string]int)
	for i, column := range hdr {
		idx[strings.TrimSpace(column)] = i
	}
	return idx
}
func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}
func parseCellUint(row []string, i int) uint32 {
	value := strings.TrimSpace(cell(row, i))
	n, _ := strconv.ParseUint(value, 10, 32)
	return uint32(n)
}
func firstDialogToken(value string) string {
	before, _, found := strings.Cut(value, ";")
	if found {
		value = before
	}
	return strings.TrimSpace(value)
}
func menuToken(value string) string {
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "menu_") {
			return part
		}
	}
	return ""
}

type questTemplate struct {
	ID            uint32
	TitleID       string
	ContextID     string
	TargetID      string
	MenuID        string
	AcceptMenuID  string
	CompleteID    string
	CompleteMenu  string
	Line          int32
	SubmitSceneID uint32
	AcceptNPC     string
	SubmitNPC     string
	KillNPC       string
	KillCount     int32
	TaskSubID     string
	RewardSilver  int32
}
type playerQuest struct {
	template  *questTemplate
	kills     int32
	done      bool
	submitted bool
}

func (q *playerQuest) meetsKillGoal() bool {
	return q.kills >= q.template.KillCount
}

type questRuntime struct {
	mu      sync.Mutex
	active  map[uint32]*playerQuest
	doneIDs map[uint32]bool
}

func newQuestRuntime() *questRuntime {
	return &questRuntime{active: make(map[uint32]*playerQuest), doneIDs: make(map[uint32]bool)}
}

type questMenuActionKind uint64

const (
	questActionAccept questMenuActionKind = iota
	questActionSubmit
)

type questMenuAction struct {
	kind   questMenuActionKind
	taskID uint32
}

const taskMenuAcceptBase int32 = 100000000
const taskMenuSubmitBase int32 = 200000000

var questCatalog = map[uint32]*questTemplate{82512: {ID: 82512, TitleID: "title_82512", ContextID: "context_82512", TargetID: "target_82512", CompleteID: "complete_82512", Line: 7, AcceptNPC: "WorldNpc02164", SubmitNPC: "WorldNpc02164", KillNPC: "AttackNPC_tm5nei_dog", KillCount: 2, TaskSubID: "82512", RewardSilver: 500}, 10210: {ID: 10210, TitleID: "title_10210", ContextID: "context_10210", MenuID: "menu_10210", AcceptMenuID: "task_menu_accept_10210", CompleteID: "complete_10210", CompleteMenu: "task_menu_complete_10210", Line: 8, KillNPC: "monrc053600", KillCount: 1, TaskSubID: "10210", RewardSilver: 200}}

func (s *sceneLifecycle) taskSpawnIndexesFor(taskSubID string) []npcSpawn {
	if s.registry == nil || taskSubID == "" {
		return nil
	}
	byTask := s.registry.taskSpawnsFor(role.Scene{Config: s.sceneConfig, Resource: s.sceneResource})
	if byTask == nil {
		return nil
	}
	return byTask[taskSubID]
}
func (s *sceneLifecycle) acceptQuest(taskID uint32) error {
	template, ok := questCatalog[taskID]
	if !ok {
		return fmt.Errorf("accept unknown quest %d", taskID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quests == nil {
		s.quests = &questRuntime{active: make(map[uint32]*playerQuest), doneIDs: make(map[uint32]bool)}
	}
	s.quests.mu.Lock()
	if _, held := s.quests.active[taskID]; held {
		s.quests.mu.Unlock()
		return fmt.Errorf("quest %d already active", taskID)
	}
	if s.quests.doneIDs[taskID] {
		s.quests.mu.Unlock()
		return fmt.Errorf("quest %d already completed", taskID)
	}
	s.quests.active[taskID] = &playerQuest{template: template}
	s.quests.mu.Unlock()
	frame, err := serverRecordAddCells(0x11000001, 0x007A5C0B, 2, []recordCell{recordInt(taskID), recordInt(uint32(template.Line)), recordInt(0), recordInt(template.SubmitSceneID), recordString(template.SubmitNPC), recordString(template.TitleID), recordInt(0)})
	if err != nil {
		return fmt.Errorf("encode Task_Accepted quest %d: %w", taskID, err)
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return err
	}
	addFrame, err := serverCustomStringMessageWithOpcode(0x1e, "task_msg", customInt(9), customInt(int32(taskID)))
	if err != nil {
		return fmt.Errorf("encode task_msg add quest %d: %w", taskID, err)
	}
	if err := s.conn.WriteFrame(addFrame); err != nil {
		return err
	}
	spawnCount := 0
	for _, spawn := range s.taskSpawnIndexesFor(template.TaskSubID) {
		s.taskNPCCount++
		id := s.taskNPCCount | 0x11100000
		if err := s.registerNPC(id, spawn); err != nil {
			log.Printf("%s: refresh quest %d spawn %s: %v", s.remote, taskID, spawn.resolved.ConfigID, err)
			continue
		}
		spawnCount++
	}
	log.Printf("%s: accepted quest %d title=%s line=%d refreshed_spawns=%d", s.remote, taskID, template.TitleID, template.Line, spawnCount)
	return nil
}
func (s *sceneLifecycle) recordQuestKill(configID string) {
	if s.quests == nil || configID == "" {
		return
	}
	s.quests.mu.Lock()
	defer s.quests.mu.Unlock()
	for taskID, quest := range s.quests.active {
		if quest.submitted {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(quest.template.KillNPC), configID) {
			continue
		}
		if quest.kills < quest.template.KillCount {
			quest.kills++
		}
		if !quest.meetsKillGoal() || quest.done {
			continue
		}
		quest.done = true
		log.Printf("%s: quest %d kill goal met npc=%s kills=%d", s.remote, taskID, configID, quest.kills)
	}
}
func (s *sceneLifecycle) submitQuest(taskID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quests == nil {
		return fmt.Errorf("no quest runtime")
	}
	s.quests.mu.Lock()
	quest, held := s.quests.active[taskID]
	if !held {
		s.quests.mu.Unlock()
		return fmt.Errorf("quest %d not active", taskID)
	}
	if !quest.done {
		s.quests.mu.Unlock()
		return fmt.Errorf("quest %d objectives not met", taskID)
	}
	template := quest.template
	delete(s.quests.active, taskID)
	s.quests.doneIDs[taskID] = true
	s.quests.mu.Unlock()
	if template.RewardSilver > 0 {
		if err := s.grantQuestReward(template); err != nil {
			return err
		}
	}
	rowValues := make(map[uint32]uint32)
	finalRow := taskID / 32
	rowValues[finalRow] |= uint32(1) << (taskID % 32)
	for currentRow := uint32(0); currentRow <= finalRow; currentRow++ {
		value := rowValues[currentRow]
		frame := make([]byte, 20)
		binary.LittleEndian.PutUint16(frame[0:], 0x0011)
		binary.LittleEndian.PutUint32(frame[2:], 0x11000001)
		binary.LittleEndian.PutUint32(frame[6:], 0x007A5C0B)
		binary.LittleEndian.PutUint16(frame[10:], 3)
		binary.LittleEndian.PutUint16(frame[12:], 0)
		binary.LittleEndian.PutUint16(frame[14:], 1)
		binary.LittleEndian.PutUint32(frame[16:], value)
		if err := s.conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	submitFrame, err := serverCustomStringMessageWithOpcode(0x1e, "task_msg", customInt(8), customInt(int32(taskID)))
	if err != nil {
		return fmt.Errorf("encode task_msg submit quest %d: %w", taskID, err)
	}
	if err := s.conn.WriteFrame(submitFrame); err != nil {
		return err
	}
	log.Printf("%s: submitted quest %d title=%s reward_silver=%d", s.remote, taskID, template.TitleID, template.RewardSilver)
	return nil
}
func (s *sceneLifecycle) grantQuestReward(template *questTemplate) error {
	player := s.player
	if player == nil {
		return nil
	}
	player.addSilver(template.RewardSilver)
	frame, err := player.silverUpdate()
	if err != nil {
		return err
	}
	return s.conn.WriteFrame(frame)
}
func (s *sceneLifecycle) availableQuestIDs() []uint32 {
	if s.quests == nil {
		ids := make([]uint32, 0, len(questCatalog))
		for taskID := range questCatalog {
			ids = append(ids, taskID)
		}
		return ids
	}
	s.quests.mu.Lock()
	defer s.quests.mu.Unlock()
	ids := make([]uint32, 0, len(questCatalog))
	for taskID := range questCatalog {
		if _, active := s.quests.active[taskID]; active {
			continue
		}
		if s.quests.doneIDs[taskID] {
			continue
		}
		ids = append(ids, taskID)
	}
	return ids
}
func (store *roleStore) register(ctx context.Context, key role.AccountKey, verifier []byte) (role.Account, error) {
	account, err := store.accounts.CreateAccount(ctx, key)
	if err != nil {
		return role.Account{}, err
	}
	if len(verifier) != 0 {
		if err := store.accounts.SetPassword(ctx, key, verifier); err != nil {
			return role.Account{}, err
		}
	}
	return account, nil
}
func (store *roleStore) setAccountPassword(ctx context.Context, key role.AccountKey, verifier []byte) error {
	return store.accounts.SetPassword(ctx, key, verifier)
}
func (r *sceneNPCRegistry) taskSpawnsFor(scene role.Scene) map[string][]npcSpawn {
	folder, err := r.resolveFolder(scene)
	if err != nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.catalogs[folder]; !ok {
		catalog, byTask, stats, err := loadNPCSceneSpawnsFromTables(r.tables, filepath.Join(r.root, folder), r.schema)
		if err != nil {
			return nil
		}
		r.catalogs[folder] = catalog
		r.stats[folder] = stats
		if byTask != nil {
			r.taskSpawns[folder] = byTask
		}
	}
	return r.taskSpawns[folder]
}
func questsForNPC(configID string) (accept, submit []uint32) {
	for id, q := range questCatalog {
		if strings.EqualFold(strings.TrimSpace(q.AcceptNPC), configID) {
			accept = append(accept, id)
		}
		if strings.EqualFold(strings.TrimSpace(q.SubmitNPC), configID) {
			submit = append(submit, id)
		}
	}
	return accept, submit
}
func appendQuestMenuItems(items []talkMenuItem, accept, submit []uint32) []talkMenuItem {
	if len(items) > 0 && items[len(items)-1].funcID == 600000000 {
		items = items[:len(items)-1]
	}
	for _, taskID := range accept {
		_ = questCatalog[taskID]
		items = append(items, talkMenuItem{funcID: taskMenuAcceptBase + int32(taskID), textID: "ui_form_talk_receive_task"})
	}
	for _, taskID := range submit {
		_ = questCatalog[taskID]
		items = append(items, talkMenuItem{funcID: taskMenuSubmitBase + int32(taskID), textID: "ui_form_talk_complete_task"})
	}
	items = append(items, talkMenuItem{funcID: 600000000, textID: "ui_form_talk_leave"})
	return items
}
func npcPatrolPathID(npc npcSpawn) string {
	for _, key := range []string{"template.PathID", "creator.PathId", "creator.PathID"} {
		if v := strings.TrimSpace(npc.resolved.Extensions[key]); v != "" {
			return v
		}
	}
	if v, ok := npc.resolved.Properties["PathID"]; ok && v.Value.Text != "" {
		return strings.TrimSpace(v.Value.Text)
	}
	return ""
}
func isFixedRoleNPC(npc npcSpawn) bool {
	desc := npc.resolved.Extensions["creator.desc"]
	name := npc.resolved.Properties["Name"].Value.Text
	for _, k := range []string{"店", "老板", "掌柜", "管家", "郎中", "货郎", "更夫", "伙夫", "伙计", "账房", "店主", "掌柜的"} {
		if strings.Contains(desc, k) || strings.Contains(name, k) {
			return true
		}
	}
	return false
}
func fixedNPCPrefix(configID string) bool {
	lower := strings.ToLower(configID)
	fixed := []string{"funcnpc", "funnpc", "guild", "hireshop", "shopnpc", "shop", "farmmanage", "tablenpc", "doornpc", "minenpc", "gather", "cannpc", "tasknpc", "springmachine", "watertask", "burntask", "eventtrigger", "leitai", "guildbuilding", "guildrecruit", "guildaward", "teamleader", "drive", "trigger", "blocknpc", "machine", "skinnable", "escort", "abduct", "transcity", "tgnpc", "weiginpc", "booknpc", "cloneshop", "activity", "sworn", "operation", "relive", "chushi", "yingzhan", "beimg", "weapon", "door", "eventnpc"}
	for _, p := range fixed {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}
func (entity sceneEntity) bornActionIsWalk() bool {
	for _, property := range entity.properties {
		if property.Name == "BornAction" && strings.Contains(strings.ToLower(property.Value.Text), "walk") {
			return true
		}
	}
	return false
}
func (entity sceneEntity) hasBornAction() bool {
	for _, property := range entity.properties {
		if property.Name == "BornAction" {
			return true
		}
	}
	return false
}
func patrolEligibleNPC(meta sceneEntity) bool {
	switch strings.ToLower(meta.scriptClass) {
	case "bossnpc", "attacknpc":
	case "commonnpc":
		if meta.pathID == "" {
			return false
		}
	default:
		return false
	}
	if fixedNPCPrefix(meta.configID) || meta.fixedRole {
		return false
	}
	return true
}
func patrolCandidateMeta(meta sceneEntity) bool {
	if meta.interaction == npcInteractionCombat {
		return false
	}
	if meta.hasBornAction() && !meta.bornActionIsWalk() {
		return false
	}
	return patrolEligibleNPC(meta)
}

type patrolSegment struct {
	name   string
	points []patrolWaypoint
}
type patrolCandidate struct {
	id        uint32
	transform worldcore.Transform
	pathID    string
}

func patrolDistance(a worldcore.Transform, b patrolWaypoint) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}
func wpDist(a, b patrolWaypoint) float32 {
	dx := a.X - b.X
	dz := a.Z - b.Z
	return float32(math.Sqrt(float64(dx*dx + dz*dz)))
}
func nearestWaypointWindow(routes []patrolRoute, at worldcore.Transform) (patrolSegment, bool) {
	bestDist := float32(3.4028234663852886e+38)
	bestRoute := patrolRoute{}
	bestIdx := -1
	for _, route := range routes {
		for idx, point := range route.Points {
			d := patrolDistance(at, point)
			if d < bestDist {
				bestRoute = route
				bestIdx = idx
				bestDist = d
			}
		}
	}
	if bestIdx < 0 || len(bestRoute.Points) == 0 {
		return patrolSegment{}, false
	}
	spawn := patrolWaypoint{X: at.X, Y: at.Y, Z: at.Z}
	build := func(startIdx, dir int, window []patrolWaypoint) []patrolWaypoint {
		prev := patrolWaypoint{}
		pts := bestRoute.Points
		for k := startIdx; k >= 0 && k < len(pts) && len(window) < 12; k += dir {
			p := pts[k]
			if len(window) != 0 {
				if wpDist(prev, p) > 30 {
					break
				}
				if wpDist(window[len(window)-1], p) < 8 {
					continue
				}
			}
			if wpDist(spawn, p) < 4 {
				continue
			}
			window = append(window, p)
			prev = p
		}
		return window
	}
	window := build(bestIdx, 1, nil)
	if len(window) < 2 {
		window = build(bestIdx, -1, nil)
	}
	if len(window) < 2 {
		far := patrolWaypoint{}
		farDist := float32(-1)
		for k := bestIdx; k >= 0 && k < len(bestRoute.Points); k++ {
			p := bestRoute.Points[k]
			d := wpDist(spawn, p)
			if d > farDist && d <= 30 {
				far = p
				farDist = d
			}
		}
		if farDist > 0 {
			window = []patrolWaypoint{far}
		}
	}
	if len(window) < 2 {
		return patrolSegment{}, false
	}
	return patrolSegment{name: bestRoute.Name, points: window}, true
}
func mappathRouteUsable(points []mapPathPoint, at worldcore.Transform) (int, float32, bool) {
	best := 0
	bestDist := float32(3.4028234663852886e+38)
	for i, p := range points {
		dx := at.X - p.X
		dz := at.Z - p.Z
		d := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		if d < bestDist {
			best = i
			bestDist = d
		}
	}
	return best, bestDist, bestDist <= 30
}
func (s *sceneLifecycle) setSceneRegistry(registry *sceneNPCRegistry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registry = registry
}
func (s *sceneLifecycle) setPlayer(player *playerActor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.player = player
}
func (s *sceneLifecycle) setPatrolPathDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pathDir = strings.TrimSpace(dir)
	s.mapPathDir = strings.TrimSpace(filepath.Join(runtimeProjectRoot, "resources", "modern", "share", "creator", "mappath"))
}
func (s *sceneLifecycle) clearSelectedTarget() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.selectedObject = 0
	s.selectedOwner = 0
	return s.conn.WriteFrame(sceneObjectObjectProperty(playerObjectID, playerOwnerID, uint32(propLastObject), 0, 0))
}
func (s *sceneLifecycle) selectSceneObject(objectID uint32, ownerID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	entity, exists := s.entities[worldcore.EntityID(objectID)]
	if !exists || (ownerID != 0 && ownerID != entity.ownerID) {
		return fmt.Errorf("unknown scene object id=%d owner=%d", objectID, ownerID)
	}
	if err := s.conn.WriteFrame(sceneObjectObjectProperty(playerObjectID, playerOwnerID, uint32(propLastObject), entity.id, entity.ownerID)); err != nil {
		return err
	}
	s.selectedObject = entity.id
	s.selectedOwner = entity.ownerID
	return nil
}
func questMenuActionFromFuncID(funcID int32) (questMenuAction, bool) {
	if delta := uint32(funcID - taskMenuAcceptBase); delta < 1_000_000 {
		return questMenuAction{kind: questActionAccept, taskID: delta}, true
	}
	if delta := uint32(funcID - taskMenuSubmitBase); delta < 1_000_000 {
		return questMenuAction{kind: questActionSubmit, taskID: delta}, true
	}
	return questMenuAction{}, false
}
func transportMenuIndex(funcID int32) (int, bool) {
	index := int(funcID - 860001000)
	if index < 0 || index > 512 {
		return 0, false
	}
	return index, true
}
func isTalkAction(funcID int32) bool {
	return funcID == 860000001 || funcID == 860000007
}
func (s *sceneLifecycle) performQuestMenuAction(action questMenuAction) error {
	switch action.kind {
	case questActionAccept:
		return s.acceptQuest(action.taskID)
	case questActionSubmit:
		return s.submitQuest(action.taskID)
	default:
		return fmt.Errorf("unknown quest menu action kind=%d task=%d", action.kind, action.taskID)
	}
}
func npcHasCombatSkill(npc npcSpawn) bool {
	raw := strings.Trim(strings.TrimSpace(npc.resolved.Properties["template.table@SkillRec"].Value.Text), "\"")
	return raw != "" && raw != "0"
}
func (s *sceneLifecycle) scheduleMapPath(id uint32, pathID string, points []mapPathPoint, index int, direction int) {
	s.mu.Lock()
	if s.closed || len(points) == 0 || index < 0 || index >= len(points) {
		s.mu.Unlock()
		return
	}
	epoch := s.epoch
	meta, exists := s.entities[worldcore.EntityID(id)]
	if !exists {
		s.mu.Unlock()
		return
	}
	waypoint := points[index]
	s.mu.Unlock()
	target := worldcore.Transform{X: waypoint.X, Y: waypoint.Y, Z: waypoint.Z, Orient: meta.transform.Orient}
	const speed float32 = 1.77
	heading := headingTowards(meta.transform, target)
	distance := horizontalDistanceXZ(meta.transform, target)
	if distance < 0.25 {
		next := index + direction
		if next < 0 || next >= len(points) {
			direction = -direction
			next = index + direction
		}
		s.scheduleMapPath(id, pathID, points, next, direction)
		return
	}
	duration := time.Duration(distance/speed*1000) * time.Millisecond
	if duration < 300*time.Millisecond {
		duration = 300 * time.Millisecond
	}
	frame := serverEntityMove([]entityMove{{id: id, position: meta.transform, movement: makeMovementData(speed, heading)}})
	if err := s.conn.WriteFrame(frame); err != nil {
		log.Printf("%s: mappath walk id=%d: %v", s.remote, id, err)
		return
	}
	log.Printf("%s: mappath id=%d -> (%.1f,%.1f,%.1f) d=%.1f spd=%.2f", s.remote, id, target.X, target.Y, target.Z, distance, speed)
	stop := time.Duration(waypoint.stopMillis) * time.Millisecond
	if stop <= 0 {
		stop = 400 * time.Millisecond
	}
	timer := time.AfterFunc(duration+stop, func() {
		s.mu.Lock()
		if s.closed || s.epoch != epoch {
			s.mu.Unlock()
			return
		}
		if err := s.setNPCTransformLocked(worldcore.EntityID(id), target); err != nil {
			s.mu.Unlock()
			return
		}
		meta := s.entities[worldcore.EntityID(id)]
		frame := serverEntityMove([]entityMove{{id: id, position: meta.transform}})
		s.mu.Unlock()
		if err := s.conn.WriteFrame(frame); err != nil {
			log.Printf("%s: mappath stop id=%d: %v", s.remote, id, err)
			return
		}
		next := index + direction
		if next < 0 || next >= len(points) {
			direction = -direction
			next = index + direction
		}
		s.scheduleMapPath(id, pathID, points, next, direction)
	})
	s.mu.Lock()
	if s.closed || s.epoch != epoch {
		s.mu.Unlock()
		return
	}
	s.timers = append(s.timers, timer)
	s.mu.Unlock()
}
func (s *sceneLifecycle) schedulePingPong(id uint32, segment patrolSegment, index int, direction int) {
	s.mu.Lock()
	if s.closed || len(segment.points) == 0 || index < 0 || index >= len(segment.points) {
		s.mu.Unlock()
		return
	}
	epoch := s.epoch
	meta, exists := s.entities[worldcore.EntityID(id)]
	if !exists {
		s.mu.Unlock()
		return
	}
	point := segment.points[index]
	s.mu.Unlock()
	target := worldcore.Transform{X: point.X, Y: point.Y, Z: point.Z, Orient: meta.transform.Orient}
	const speed float32 = 1.77
	heading := headingTowards(meta.transform, target)
	distance := horizontalDistanceXZ(meta.transform, target)
	if distance < 0.25 {
		next := index + direction
		if next < 0 || next >= len(segment.points) {
			direction = -direction
			next = index + direction
		}
		s.schedulePingPong(id, segment, next, direction)
		return
	}
	duration := time.Duration(distance/speed*1000) * time.Millisecond
	if duration < 300*time.Millisecond {
		duration = 300 * time.Millisecond
	}
	frame := serverEntityMove([]entityMove{{id: id, position: meta.transform, movement: makeMovementData(speed, heading)}})
	if err := s.conn.WriteFrame(frame); err != nil {
		log.Printf("%s: patrol walk id=%d: %v", s.remote, id, err)
		return
	}
	log.Printf("%s: patrol id=%d -> (%.1f,%.1f,%.1f) d=%.1f spd=%.2f", s.remote, id, target.X, target.Y, target.Z, distance, speed)
	timer := time.AfterFunc(duration, func() {
		s.mu.Lock()
		if s.closed || s.epoch != epoch {
			s.mu.Unlock()
			return
		}
		if err := s.setNPCTransformLocked(worldcore.EntityID(id), target); err != nil {
			s.mu.Unlock()
			return
		}
		meta := s.entities[worldcore.EntityID(id)]
		frame := serverEntityMove([]entityMove{{id: id, position: meta.transform}})
		s.mu.Unlock()
		if err := s.conn.WriteFrame(frame); err != nil {
			log.Printf("%s: patrol stop id=%d: %v", s.remote, id, err)
			return
		}
		next := index + direction
		if next < 0 || next >= len(segment.points) {
			direction = -direction
			next = index + direction
		}
		s.schedulePingPong(id, segment, next, direction)
	})
	s.mu.Lock()
	if s.closed || s.epoch != epoch {
		s.mu.Unlock()
		return
	}
	s.timers = append(s.timers, timer)
	s.mu.Unlock()
}
func (s *sceneLifecycle) assignPatrolForEntity(id worldcore.EntityID) bool {
	s.mu.Lock()
	meta, exists := s.entities[id]
	if !exists || meta.patrolAssigned || !patrolCandidateMeta(meta) {
		s.mu.Unlock()
		return false
	}
	meta.patrolAssigned = true
	s.entities[id] = meta
	candidate := patrolCandidate{id: uint32(id), transform: meta.transform, pathID: meta.pathID}
	routes := append([]patrolRoute(nil), s.patrolRoutes...)
	mapRoutes := s.mapPathRoutes
	s.mu.Unlock()
	if len(mapRoutes) > 0 {
		points, ok := mapRoutes[candidate.pathID]
		if ok && len(points) >= 2 {
			idx, dist, usable := mappathRouteUsable(points, candidate.transform)
			if usable {
				s.scheduleMapPath(candidate.id, candidate.pathID, points, idx, 1)
				return true
			}
			log.Printf("%s: %s(%d) mappath %s nearest waypoint %.0fm away (>%.0f), falling back to nav graph", s.remote, meta.configID, candidate.id, candidate.pathID, dist, float32(30))
		}
	}
	if len(routes) > 0 {
		segment, ok := nearestWaypointWindow(routes, candidate.transform)
		if ok {
			dist := patrolDistance(candidate.transform, segment.points[0])
			if dist <= 300 {
				s.schedulePingPong(candidate.id, segment, 0, 1)
				return true
			}
		}
	}
	s.schedulePatrolStep(candidate.id, 0)
	return true
}

type shortcutStoreIface interface {
	Load(roleID role.RoleID) ([]playerShortcut, bool)
	LoadCustomizing(roleID role.RoleID) (string, bool)
	LoadKeyBind(roleID role.RoleID) (string, bool)
	Save(roleID role.RoleID, shortcuts []playerShortcut) error
	SaveCustomizing(roleID role.RoleID, customizing string) error
	SaveKeyBind(roleID role.RoleID, keybind string) error
}

var loginDefaultCustomizing = "\x01\x65\x01\x01\x01\x10\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x01\x01\x01\x01\x02\x02\x02\x02\x02\x01\x02\x02\x65\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x01\x10\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x02\x01\x01\x02\x02\x01\x02\x02\x01\x01\x0b\x01\x65\x01\x01\x01\x02\x01\x01\x01\x02\x02\x02\x02\x01\x02\x02\x01\x02\x01\x01\x02\x01\x02\x01\x01\x01\x02\x03\x02\x01\x01\x02\x02\x02\x01\x02\x02\x02\x65\x01\x15\x01\x01\x02\x02\x02\x01\x01\x01\x01\x01\x01\x01\x02\x01\x01\x01\x02"

func handleCustomizingSave(store shortcutStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, string, error) {
	if len(custom.Values) < 2 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0x6B || custom.Values[1].Type != 6 {
		return false, "", nil
	}
	customizing := custom.Values[1].Text
	if len(customizing) > 1<<20 {
		return true, "", fmt.Errorf("customizing too large: %d", len(customizing))
	}
	if customizing == loginDefaultCustomizing {
		log.Printf("%s: customizing is login default template, skipping persist", remote)
		return true, "", nil
	}
	if len(customizing) >= 85 && customizing[84] != 2 {
		log.Printf("%s: customizing is login default UI state (byte84=%02x), skipping persist", remote, customizing[84])
		return true, "", nil
	}
	if store != nil {
		if err := store.SaveCustomizing(roleID, customizing); err != nil {
			log.Printf("%s: persist customizing: %v", remote, err)
		}
	}
	log.Printf("%s: customizing accepted bytes=%d", remote, len(customizing))
	return true, customizing, nil
}
func handleSelectTargetCustom(world *sceneLifecycle, caster worldcore.Transform, custom clientCustomMessage, remote string) (bool, error) {
	if world == nil || len(custom.Values) != 2 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0x1AF || custom.Values[1].Type != 8 {
		return false, nil
	}
	raw := custom.Values[1].Raw
	objectID := binary.LittleEndian.Uint32(raw[0:4])
	ownerID := binary.LittleEndian.Uint32(raw[4:8])
	if objectID == 0 {
		if err := world.clearSelectedTarget(); err != nil {
			return true, err
		}
		if err := updateSkillRangeHints(world.conn, caster, worldcore.Transform{}, false); err != nil {
			log.Printf("%s: restore skill range hints: %v", remote, err)
		}
		log.Printf("%s: cleared selected target via 431 object=0", remote)
		return true, nil
	}
	if err := world.selectSceneObject(objectID, ownerID); err != nil {
		log.Printf("%s: 431 ignore unknown object id=%d owner=%d: %v", remote, objectID, ownerID, err)
		return true, nil
	}
	if target, ok := world.selectedCombatTargetTransform(); ok {
		if err := updateSkillRangeHints(world.conn, caster, target, true); err != nil {
			log.Printf("%s: update skill range hints: %v", remote, err)
		}
	}
	log.Printf("%s: selected target via 431 id=%d owner=%d", remote, objectID, ownerID)
	return true, nil
}
func handleKeyBindCustom(store shortcutStoreIface, roleID role.RoleID, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 3 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0xC3 || custom.Values[2].Type != 6 {
		return false, nil
	}
	keybind := custom.Values[2].Text
	if len(keybind) > 0x1000 {
		return true, fmt.Errorf("keybind string too long: %d", len(keybind))
	}
	if store != nil {
		if err := store.SaveKeyBind(roleID, keybind); err != nil {
			log.Printf("%s: persist keybind: %v", remote, err)
		}
	}
	if player != nil {
		player.setKeybind(keybind)
	}
	log.Printf("%s: keybind accepted bytes=%d", remote, len(keybind))
	return true, nil
}

type entityMove struct {
	id       uint32
	position worldcore.Transform
	movement [20]byte
}

func makeMovementData(speed, turn float32) [20]byte {
	var data [20]byte
	rate := float32(2 * math.Pi)
	if turn < 0 {
		rate = -rate
	}
	binary.LittleEndian.PutUint32(data[0:4], math.Float32bits(speed))
	binary.LittleEndian.PutUint32(data[4:8], math.Float32bits(rate))
	data[16] = 1
	return data
}
func headingTowards(a, b worldcore.Transform) float32 {
	return float32(math.Atan2(float64(b.Z-a.Z), float64(b.X-a.X)))
}
func horizontalDistanceXZ(a, b worldcore.Transform) float32 {
	dx := a.X - b.X
	dz := a.Z - b.Z
	return float32(math.Sqrt(float64(dx*dx + dz*dz)))
}
func serverEntityMove(entries []entityMove) []byte {
	msg := make([]byte, 3+44*len(entries))
	msg[0] = 0x21
	binary.LittleEndian.PutUint16(msg[1:3], uint16(len(entries)))
	offset := 3
	for _, entry := range entries {
		binary.LittleEndian.PutUint32(msg[offset:], entry.id)
		binary.LittleEndian.PutUint32(msg[offset+4:], 1)
		binary.LittleEndian.PutUint32(msg[offset+8:], math.Float32bits(entry.position.X))
		binary.LittleEndian.PutUint32(msg[offset+12:], math.Float32bits(entry.position.Y))
		binary.LittleEndian.PutUint32(msg[offset+16:], math.Float32bits(entry.position.Z))
		binary.LittleEndian.PutUint32(msg[offset+20:], math.Float32bits(entry.position.Orient))
		copy(msg[offset+24:offset+44], entry.movement[:])
		offset += 44
	}
	return msg
}
func handleShopBuyCustom(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, bagStore bagStoreIface, currencyStore currencyStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 5 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0x46 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject shop buy before player spawn", remote)
		return true, nil
	}
	if custom.Values[1].Type != 6 && custom.Values[1].Type != 7 {
		log.Printf("%s: shop buy: shopid must be a string", remote)
		return true, nil
	}
	shopID := custom.Values[1].Text
	if custom.Values[2].Type != 2 || custom.Values[3].Type != 2 || custom.Values[4].Type != 2 {
		log.Printf("%s: shop buy %s: page/pos/amount must be int32", remote, shopID)
		return true, nil
	}
	page := custom.Values[2].Int32
	pos := custom.Values[3].Int32
	amount := custom.Values[4].Int32
	if page < 0 || pos <= 0 || amount < 1 || amount > 99 {
		log.Printf("%s: shop buy %s malformed current-client page=%d pos=%d amount=%d", remote, shopID, page, pos, amount)
		return true, nil
	}
	items, _, _, err := shopCatalogItems(defaultShopINIPath, shopID)
	if err != nil {
		log.Printf("%s: shop buy %s: %v", remote, shopID, err)
		return true, nil
	}
	listing := currentShopListing(items, page, pos)
	if listing == nil {
		log.Printf("%s: shop buy %s no listing at page=%d pos=%d", remote, shopID, page, pos)
		return true, nil
	}
	item := *listing
	if item.priceMode < 0 || item.priceMode > 2 {
		log.Printf("%s: shop buy %s item %s unsupported capital type %d", remote, shopID, item.configID, item.priceMode)
		return true, nil
	}
	total := int64(item.price) * int64(amount)
	switch item.priceMode {
	case 0:
		_, gold, _, _ := player.currencySnapshot()
		if int64(gold) < total {
			log.Printf("%s: shop buy %s item %s needs gold %d, has %d", remote, shopID, item.configID, total, gold)
			return true, nil
		}
		player.addGold(-int32(total))
	case 1:
		silver, _, _, _ := player.currencySnapshot()
		if int64(silver) < total {
			log.Printf("%s: shop buy %s item %s needs silver %d, has %d", remote, shopID, item.configID, total, silver)
			return true, nil
		}
		player.addSilver(-int32(total))
	case 2:
		_, _, silverCard, _ := player.currencySnapshot()
		if int64(silverCard) < total {
			log.Printf("%s: shop buy %s item %s needs silverCard %d, has %d", remote, shopID, item.configID, total, silverCard)
			return true, nil
		}
		player.addSilverCard(-int32(total))
	}
	currencyFrame, err := player.currenciesUpdate()
	if err != nil {
		return true, err
	}
	reward := bagItem{ConfigID: item.configID, Amount: amount}
	reward = enrichBagItem(reward, itemCatalog, nil)
	slot := player.addBagItem(reward)
	view := bagViewForViewID(reward.ViewID)
	itemFrame, err := serverViewAdd(view, uint16(slot), bagItemProps(view, reward))
	if err != nil {
		return true, err
	}
	frames := [][]byte{currencyFrame, itemFrame}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, nil, roleID, player)
	if currencyStore != nil {
		silver, gold, silverCard, silverTicket := player.currencySnapshot()
		if err := currencyStore.Save(roleID, currencySnapshot{Silver: silver, Gold: gold, SilverCard: silverCard, SilverTicket: silverTicket}); err != nil {
			log.Printf("%s: persist currency after shop buy: %v", remote, err)
		}
	}
	log.Printf("%s: shop buy %s item %s x%d capital=%d price=%d -> bag view=%d slot=%d", remote, shopID, item.configID, amount, item.priceMode, item.price, view, slot)
	return true, nil
}

func shopCatalogItems(path, shopID string) ([]shopCatalogItem, int32, int32, error) {
	// The current client derives its buy-back UI page locally for ordinary
	// shops. Return only authored shop.ini rows; do not append synthetic
	// products or inflate PageCount beyond the resource-defined pages.
	return loadShopCatalogSection(path, shopID)
}

var starterShortcutIDs = []string{"CS_light_rad_81", "CS_jh_cqgf01", "CS_jh_cqgf04", "CS_jh_cqgf06"}

func grantCollectSkillRec(conn sceneMessageConnection, player *playerActor) error {
	if player == nil {
		return nil
	}
	learned := player.learnedSkillSnapshot()
	ids := make([]string, 0, len(learned))
	for id := range learned {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if id == "zs_default_01" || id == "taolu_zhenfa_wzyx" || id == "taolu_zhenfa_cfxz" {
			continue
		}
		level := learned[id]
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, 7, []recordCell{recordString(id), recordInt(uint32(level))})
		if err != nil {
			return fmt.Errorf("encode CollectSkillRec %s: %w", id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("write CollectSkillRec %s: %w", id, err)
		}
	}
	return nil
}
func (p *playerActor) resyncShortcutRecord(conn sceneMessageConnection) error {
	if p == nil {
		return nil
	}
	current := p.shortcutSnapshot()
	for row := len(p.lastShortcutRows) - 1; row >= 0; row-- {
		if err := conn.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordShortcut, uint16(row))); err != nil {
			return err
		}
	}
	for _, shortcut := range current {
		frame, err := shortcutAddFrame(shortcut)
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	p.lastShortcutRows = current
	return nil
}
func (p *playerActor) removeShortcutByIndex(index int32) (playerShortcut, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for row, shortcut := range p.shortcuts {
		if shortcut.index != index {
			continue
		}
		p.shortcuts = append(p.shortcuts[:row], p.shortcuts[row+1:]...)
		return shortcut, true
	}
	return playerShortcut{}, false
}
func grantStarterShortcuts(conn sceneMessageConnection, player *playerActor) (int, error) {
	if player == nil {
		return 0, nil
	}
	granted := 0
	for offset, id := range starterShortcutIDs {
		index := int32(offset + 1)
		if _, occupied := player.shortcutAt(index); occupied {
			continue
		}
		if _, has := player.shortcutWithID("skill", id); has {
			continue
		}
		_, unchanged := player.setShortcut(index, "skill", id)
		if unchanged {
			continue
		}
		granted++
	}
	log.Printf("grantStarterShortcuts: granted=%d bar=%v", granted, player.shortcutSnapshot())
	if granted > 0 {
		if err := player.resyncShortcutRecord(conn); err != nil {
			return granted, err
		}
	}
	return granted, nil
}
func (p *playerActor) shortcutAt(index int32) (playerShortcut, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, shortcut := range p.shortcuts {
		if shortcut.index == index {
			return shortcut, true
		}
	}
	return playerShortcut{}, false
}
func (p *playerActor) shortcutWithID(kind, id string) (playerShortcut, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, shortcut := range p.shortcuts {
		if shortcut.kind == kind && shortcut.id == id {
			return shortcut, true
		}
	}
	return playerShortcut{}, false
}
func (p *playerActor) restoreShortcuts(saved []playerShortcut) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, shortcut := range saved {
		found := false
		for _, row := range p.shortcuts {
			if row.index == shortcut.index {
				found = true
				break
			}
		}
		if found {
			continue
		}
		p.shortcuts = append(p.shortcuts, shortcut)
	}
	log.Printf("restoreShortcuts: saved=%d -> bar=%v", len(saved), p.shortcuts)
}
func persistShortcuts(store shortcutStoreIface, roleID role.RoleID, player *playerActor, remote, action string) {
	if store == nil || roleID == 0 || player == nil {
		return
	}
	if err := store.Save(roleID, player.shortcutSnapshot()); err != nil {
		log.Printf("%s: persist shortcuts after %s: %v", remote, action, err)
	}
}

const shortcutStoreVersion = 1

type shortcutPersisted struct {
	Index int32  `json:"index"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
}
type shortcutStore struct {
	mu          sync.Mutex
	path        string
	roles       map[string][]shortcutPersisted
	keybinds    map[string]string
	customizing map[string]string
}
type shortcutStoreFile struct {
	Version     int                            `json:"version"`
	Roles       map[string][]shortcutPersisted `json:"roles"`
	Keybinds    map[string]string              `json:"keybinds,omitempty"`
	Customizing map[string]string              `json:"customizing,omitempty"`
}

func openShortcutStore() (*shortcutStore, error) {
	return openShortcutStoreAt(filepath.Join(runtimeProjectRoot, "data", "shortcuts.json"))
}
func openShortcutStoreAt(path string) (*shortcutStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("shortcut store: empty path")
	}
	path = filepath.Clean(path)
	store := &shortcutStore{path: path, roles: make(map[string][]shortcutPersisted), keybinds: make(map[string]string), customizing: make(map[string]string)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read shortcut store: %w", err)
	}
	var document shortcutStoreFile
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode shortcut store: %w", err)
	}
	if document.Version != shortcutStoreVersion {
		return nil, fmt.Errorf("decode shortcut store: unsupported version %d", document.Version)
	}
	if document.Roles != nil {
		store.roles = document.Roles
	}
	if document.Keybinds != nil {
		store.keybinds = document.Keybinds
	}
	if document.Customizing != nil {
		store.customizing = document.Customizing
	}
	return store, nil
}
func (store *shortcutStore) SaveKeyBind(roleID role.RoleID, keybind string) error {
	if store == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := strconv.FormatUint(uint64(roleID), 10)
	previous, existed := store.keybinds[key]
	store.keybinds[key] = keybind
	if err := store.persistLocked(); err != nil {
		if existed {
			store.keybinds[key] = previous
		} else {
			delete(store.keybinds, key)
		}
		return err
	}
	return nil
}
func (store *shortcutStore) LoadKeyBind(roleID role.RoleID) (string, bool) {
	if store == nil || roleID == 0 {
		return "", false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	value, ok := store.keybinds[strconv.FormatUint(uint64(roleID), 10)]
	return value, ok
}
func (store *shortcutStore) SaveCustomizing(roleID role.RoleID, customizing string) error {
	if store == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := strconv.FormatUint(uint64(roleID), 10)
	previous, existed := store.customizing[key]
	store.customizing[key] = customizing
	if err := store.persistLocked(); err != nil {
		if existed {
			store.customizing[key] = previous
		} else {
			delete(store.customizing, key)
		}
		return err
	}
	return nil
}
func (store *shortcutStore) LoadCustomizing(roleID role.RoleID) (string, bool) {
	if store == nil || roleID == 0 {
		return "", false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	value, ok := store.customizing[strconv.FormatUint(uint64(roleID), 10)]
	return value, ok
}
func (store *shortcutStore) Load(roleID role.RoleID) ([]playerShortcut, bool) {
	if store == nil || roleID == 0 {
		return nil, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	rows, ok := store.roles[strconv.FormatUint(uint64(roleID), 10)]
	if !ok {
		return nil, false
	}
	out := make([]playerShortcut, 0, len(rows))
	for _, row := range rows {
		out = append(out, playerShortcut{index: row.Index, kind: row.Kind, id: row.ID})
	}
	return out, true
}
func (store *shortcutStore) Save(roleID role.RoleID, shortcuts []playerShortcut) error {
	if store == nil {
		return nil
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := strconv.FormatUint(uint64(roleID), 10)
	rows := make([]shortcutPersisted, 0, len(shortcuts))
	for _, shortcut := range shortcuts {
		rows = append(rows, shortcutPersisted{Index: shortcut.index, Kind: shortcut.kind, ID: shortcut.id})
	}
	previous, existed := store.roles[key]
	store.roles[key] = rows
	if err := store.persistLocked(); err != nil {
		if existed {
			store.roles[key] = previous
		} else {
			delete(store.roles, key)
		}
		return err
	}
	return nil
}
func (store *shortcutStore) persistLocked() error {
	document := shortcutStoreFile{Version: shortcutStoreVersion, Roles: store.roles, Keybinds: store.keybinds, Customizing: store.customizing}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode shortcut store: %w", err)
	}
	data = append(data, '\n')
	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create shortcut store directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".shortcut-*.tmp")
	if err != nil {
		return fmt.Errorf("create shortcut store temporary: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if temporary != nil {
			_ = temporary.Close()
		}
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set shortcut store temporary permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write shortcut store temporary: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync shortcut store temporary: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close shortcut store temporary: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return fmt.Errorf("replace shortcut store: %w", err)
	}
	committed = true
	return nil
}

var skillMaxOnce sync.Once
var skillMaxTable iniTable
var skillMaxLoadErr error

func applyUseSkillBook(link sceneMessageConnection, player *playerActor, bagStore bagStoreIface, roleID role.RoleID, srcView uint16, srcPos int32, item bagItem, remote string) (bool, error) {
	skillID := strings.TrimPrefix(item.ConfigID, "book_")
	if skillID == item.ConfigID {
		log.Printf("%s: USEITEM %s is not a skill book (view=%d pos=%d)", remote, item.ConfigID, srcView, srcPos)
		return true, nil
	}
	updated, remaining, consumed, ok := player.decrementBagItem(srcView, srcPos, 1)
	if !ok {
		log.Printf("%s: USEITEM skill book %s source vanished view=%d pos=%d", remote, item.ConfigID, srcView, srcPos)
		return true, nil
	}
	frames := make([][]byte, 0, 3)
	if consumed {
		frames = append(frames, serverViewRemove(srcView, uint16(srcPos)))
	} else {
		frame, err := serverViewAdd(srcView, uint16(srcPos), bagItemProps(srcView, updated))
		if err != nil {
			return true, err
		}
		frames = append(frames, frame)
	}
	var (
		learnFrames [][]byte
		learnErr    error
	)
	switch {
	case strings.HasPrefix(skillID, "fwz_"):
		collected := player.collectFwzCard(skillID)
		log.Printf("%s: USEITEM fwz card %s collected=%t (CardRec/wear protocol pending official capture)", remote, skillID, collected)
	case strings.HasPrefix(skillID, "qinggong_"):
		learnFrames, learnErr = learnQingGongBookFrames(player, skillID)
	default:
		learnFrames, learnErr = learnWuxueBookFrames(player, skillID)
	}
	if learnErr != nil {
		log.Printf("%s: USEITEM skill book %s learn %s: %v", remote, item.ConfigID, skillID, learnErr)
	} else {
		frames = append(frames, learnFrames...)
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, err
	}
	persistBagEquip(bagStore, nil, roleID, player)
	log.Printf("%s: USEITEM skill book %s -> learn %s (remaining=%d)", remote, item.ConfigID, skillID, remaining)
	return true, nil
}
func learnWuxueBookFrames(player *playerActor, skillID string) ([][]byte, error) {
	if _, exists := player.learnedSkillLevel(skillID); exists {
		return nil, fmt.Errorf("skill %s already learned", skillID)
	}
	player.learnSkill(skillID, 1)
	staticData, maxLevel := skillStaticFor(skillID)
	slot := player.learnedSkillCount()
	viewFrame, err := serverViewAdd(viewportSkill, uint16(slot), []serverViewProperty{viewInt(0x0761, 1000), viewString(0x05A0, skillID), viewByte(0x08BC, 1), viewInt(0x08BD, staticData), viewByte(0x05B9, 1), viewInt(0x0823, maxLevel), viewInt(0x07F2, 0)})
	if err != nil {
		return nil, fmt.Errorf("encode SkillContainer %s: %w", skillID, err)
	}
	recordFrame, err := serverRecordAddCells(playerObjectID, playerOwnerID, 7, []recordCell{recordString(skillID), recordInt(1)})
	if err != nil {
		return nil, fmt.Errorf("encode CollectSkillRec %s: %w", skillID, err)
	}
	return [][]byte{viewFrame, recordFrame}, nil
}
func learnQingGongBookFrames(player *playerActor, qinggongID string) ([][]byte, error) {
	recordFrame := serverRecordAddString(playerObjectID, playerOwnerID, recordQingGong, qinggongID)
	slot := len(starterQingGongIDs) + len(starterQingGongSkillIDs) + 1
	properties := []serverViewProperty{viewString(0x05A0, qinggongID), viewInt(0x0761, 1005)}
	if static, ok := qgStaticData[qinggongID]; ok && static.staticData > 0 {
		properties = append(properties, viewInt(0x08BD, static.staticData))
	}
	viewFrame, err := serverViewAdd(46, uint16(slot), properties)
	if err != nil {
		return nil, fmt.Errorf("encode QingGongContainer %s: %w", qinggongID, err)
	}
	return [][]byte{recordFrame, viewFrame}, nil
}
func skillStaticFor(skillID string) (int32, int32) {
	for _, skill := range starterSkillViews {
		if skill.configID == skillID {
			return skill.staticData, skillMaxLevelFor(skillID)
		}
	}
	if definition, ok := installedCombatSkillCatalog.definition(skillID, 1); ok {
		return definition.staticData, skillMaxLevelFor(skillID)
	}
	return 0, 1
}
func loadSkillMaxLevelTable() (iniTable, error) {
	skillMaxOnce.Do(func() {
		skillMaxTable, skillMaxLoadErr = loadINISections(filepath.Join(defaultModernShareRoot, "skill", "skill_maxlevel.ini"))
	})
	return skillMaxTable, skillMaxLoadErr
}
func skillMaxLevelFor(skillID string) int32 {
	table, err := loadSkillMaxLevelTable()
	if err == nil {
		if maxLevel := skillMaxLevelFromTable(table, skillID); maxLevel > 1 {
			return maxLevel
		}
		if strings.HasSuffix(skillID, "_hide") {
			base := strings.TrimSuffix(skillID, "_hide")
			if base != skillID {
				if maxLevel := skillMaxLevelFromTable(table, base); maxLevel > 1 {
					return maxLevel
				}
			}
		}
	}
	for _, skill := range starterSkillViews {
		if skill.configID == skillID {
			return skill.maxLevel
		}
	}
	return 1
}
func skillMaxLevelFromTable(table iniTable, skillID string) int32 {
	if fields := table["book_"+skillID]; len(fields) > 0 {
		return maxCultivationLevel(fields)
	}
	maxLevel := int32(1)
	for i := 1; i <= 99; i++ {
		fields := table[fmt.Sprintf("book_%s%02d", skillID, i)]
		if len(fields) == 0 {
			continue
		}
		if level := maxCultivationLevel(fields); level > maxLevel {
			maxLevel = level
		}
	}
	for _, suffix := range []string{"_1st", "_2nd", "_3rd", "_4th", "_5th", "_6th", "_7th", "_8th", "_9th"} {
		fields := table["book_"+skillID+suffix]
		if len(fields) == 0 {
			continue
		}
		if level := maxCultivationLevel(fields); level > maxLevel {
			maxLevel = level
		}
	}
	return maxLevel
}
func maxCultivationLevel(fields []iniField) int32 {
	maxLevel := 1
	for _, field := range fields {
		for _, part := range strings.Split(field.value, ",") {
			level, err := strconv.Atoi(strings.TrimSpace(part))
			if err == nil && level > maxLevel {
				maxLevel = level
			}
		}
	}
	return int32(maxLevel)
}

type skillBuffCandidate struct {
	configID   string
	staticData uint32
	immunity   int32
	isDamage   bool
	lifetime   time.Duration
}

func isAuxBuffSuffix(suffix string) bool {
	for _, aux := range []string{"_mark", "_cd", "_npc", "_mianyi", "_hide", "_sky"} {
		if strings.Contains(suffix, aux) {
			return true
		}
	}
	return false
}
func buffLifetimeFor(configID string, level int32, tables skillResourceTables) time.Duration {
	fields := tables.buffNew[configID]
	lifetime := time.Duration(iniInt(fields, "LifeTime")) * time.Millisecond
	staticData := iniInt(fields, "StaticData")
	static := tables.buffStatic[strconv.Itoa(int(staticData))]
	minimum := iniInt(static, "MinVarPropNo")
	maximum := iniInt(static, "MaxVarPropNo")
	for id := minimum; id <= maximum && id > 0; id++ {
		vp := tables.buffVar[strconv.Itoa(int(id))]
		if iniInt(vp, "Level") != level {
			continue
		}
		milliseconds := iniInt(vp, "LifeTime")
		if milliseconds > 0 {
			lifetime = time.Duration(milliseconds) * time.Millisecond
		}
		break
	}
	if lifetime <= 0 {
		lifetime = 24 * time.Hour
	}
	return lifetime
}
func collectSkillBuffCandidates(skillID string, level int32, tables skillResourceTables) []skillBuffCandidate {
	var out []skillBuffCandidate
	prefix := "buf_" + skillID
	for section, fields := range tables.buffNew {
		if !strings.HasPrefix(section, prefix) {
			continue
		}
		if isAuxBuffSuffix(section[len(prefix):]) {
			continue
		}
		staticData := iniInt(fields, "StaticData")
		if staticData <= 0 {
			continue
		}
		static := tables.buffStatic[strconv.Itoa(int(staticData))]
		out = append(out, skillBuffCandidate{configID: section, staticData: uint32(staticData), immunity: iniInt(fields, "Immunity"), isDamage: iniInt(static, "IsDamage") != 0, lifetime: buffLifetimeFor(section, level, tables)})
	}
	return out
}
func skillElementAttribute(skillID string) string {
	for _, prefix := range []string{"CS_wd_", "CS_jl_", "CS_mj_", "CS_ts_", "CS_bg_"} {
		if strings.HasPrefix(skillID, prefix) {
			return "太极"
		}
	}
	for _, prefix := range []string{"CS_sl_", "CS_gb_", "CS_jy_", "CS_xt_"} {
		if strings.HasPrefix(skillID, prefix) {
			return "阳刚"
		}
	}
	for _, prefix := range []string{"CS_em_", "CS_tm_", "CS_jz_", "CS_kl_", "CS_yhwq_"} {
		if strings.HasPrefix(skillID, prefix) {
			return "阴柔"
		}
	}
	return ""
}
func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

const taoLuSwitchCooldown = 7 * time.Second

var jumpLeapSkillIDs = map[string]struct{}{"CS_jh_myjf02": {}, "CS_jh_chz03": {}}
var parryStanceBuffSlots = []uint16{6, 7, 9}

func skillCooldownFrameWithDuration(definition combatSkillDefinition, now time.Time, duration time.Duration) ([]byte, error) {
	start := now.UnixMilli()
	end := start + duration.Milliseconds()
	return serverRecordAddCells(playerObjectID, playerOwnerID, recordCooldown, []recordCell{recordInt(uint32(definition.cooldownCategory)), recordInt64(start), recordInt64(end), recordInt(uint32(definition.cooldownTeam))})
}
func (p *playerActor) taoLuSwitchCooldownFrames(active int32, now time.Time) ([][]byte, error) {
	frames := make([][]byte, 0)
	for id, level := range p.learnedSkills {
		definition, ok := installedCombatSkillCatalog.definition(id, level)
		if !ok {
			continue
		}
		key := definition.teamKey()
		if key == active || key == 0 {
			continue
		}
		frame, err := skillCooldownFrameWithDuration(definition, now, taoLuSwitchCooldown)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
func nativeSkillActionFrameAt(skillID string, targetID, targetOwnerID uint32, hasTargetPos bool, tx, ty, tz float32) ([]byte, error) {
	if skillID == "" {
		return nil, fmt.Errorf("native skill action ID is empty")
	}
	values := []serverCustomValue{customObject(playerObjectID, playerOwnerID), customString(skillID), customObject(targetID, targetOwnerID), customInt(1), customFloat32(1)}
	if hasTargetPos {
		values = append(values, customFloat32(tx), customFloat32(ty), customFloat32(tz))
	}
	return serverCustomIntMessage(serverNativeSkillActionMessage, values...)
}
func skillBufferFrame(mode int32, buffID, name string, uniqueID uint64, category int32) ([]byte, error) {
	if buffID == "" {
		return nil, fmt.Errorf("skill buffer ID is empty")
	}
	return serverCustomIntMessage(0xA9, customInt(mode), customString(buffID), customObject(playerObjectID, playerOwnerID), customObject(playerObjectID, playerOwnerID), customString(name), customString(name), customInt64(int64(uniqueID)), customInt(category))
}
func skillCurEffectFrame(staticData uint32, level int32, uniqueID uint64) ([]byte, error) {
	value := fmt.Sprintf("%X,%X,%X,%d,%X,0", staticData, playerObjectID, playerOwnerID, level, uniqueID)
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 0x027B, Name: "CurSkillEffect", Value: clientdata.StringValue(value)}})
}
func skillCurEffectClearFrame() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{Index: 0x027B, Name: "CurSkillEffect", Value: clientdata.StringValue("")}})
}
func skillEffectFrame(skillID string, targetID, targetOwnerID uint32) ([]byte, error) {
	if skillID == "" {
		return nil, fmt.Errorf("skill effect ID is empty")
	}
	return serverCustomIntMessage(0xCD, customObject(playerObjectID, playerOwnerID), customString(skillID), customObject(targetID, targetOwnerID))
}
func skillCastPreparationFrame(transform worldcore.Transform) ([]byte, error) {
	return serverCustomIntMessage(0x151, customObject(playerObjectID, playerOwnerID), customFloat32(transform.X), customFloat32(transform.Y), customFloat32(transform.Z), customFloat32(transform.Orient), customInt(0))
}
func skillHitFrame(targetID, targetOwnerID uint32, damage int32, skillID, hitKind, victimName, attackerName string) ([]byte, error) {
	return serverCustomIntMessageWithOpcode(0x1E, 0x18, customObject(targetID, targetOwnerID), customInt(17), customInt(damage), customInt(0), customInt(0), customString(skillID), customString(hitKind), customString(victimName), customString(attackerName), customInt(0))
}
func skillAnimationFrame(casterID, casterOwner uint32, skillID string, targetID, targetOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(serverNativeSkillActionMessage, customObject(casterID, casterOwner), customString(skillID), customObject(targetID, targetOwner), customInt(0), customFloat32(0))
}
func skillNPCCastPreparationFrame(casterID, casterOwner uint32, transform worldcore.Transform) ([]byte, error) {
	return serverCustomIntMessage(0x151, customObject(casterID, casterOwner), customFloat32(transform.X), customFloat32(transform.Y), customFloat32(transform.Z), customFloat32(transform.Orient), customInt(0))
}
func skillNPCAttackHitFrame(targetID, targetOwnerID, attackerID, attackerOwner uint32, skillID, victimName, attackerName string, damage int32) ([]byte, error) {
	return serverCustomIntMessageWithOpcode(0x1E, 0x18, customObject(targetID, targetOwnerID), customInt(7), customInt(1), customObject(attackerID, attackerOwner), customString(victimName), customString(attackerName), customInt(damage), customString(skillID))
}
func skillHurtFrame(targetID, targetOwnerID, attackerID, attackerOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(0x12, customObject(targetID, targetOwnerID), customString("hurt_half"), customInt(0), customObject(attackerID, attackerOwner))
}
func skillResultFrame(casterID, casterOwner uint32, skillID string, targetID, targetOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(0xCD, customObject(casterID, casterOwner), customString(skillID), customObject(targetID, targetOwner))
}
func useSkillTargetObject(custom clientCustomMessage) (objectID uint32, ownerID uint32, ok bool) {
	if len(custom.Values) <= 10 || custom.Values[10].Type != 8 {
		return 0, 0, false
	}
	raw := custom.Values[10].Raw
	objectID = binary.LittleEndian.Uint32(raw[:4])
	ownerID = binary.LittleEndian.Uint32(raw[4:])
	if objectID == 0 {
		return 0, 0, false
	}
	return objectID, ownerID, true
}
func isJumpLeapSkill(skillID string) bool {
	_, ok := jumpLeapSkillIDs[skillID]
	return ok
}
func currentParryStanceSlot(_ combatSkillDefinition, applied []appliedSkillBuff) uint16 {
	for _, buff := range applied {
		for _, slot := range parryStanceBuffSlots {
			if buff.slot == slot {
				return slot
			}
		}
	}
	return 0
}
func playerSkillAttackBonus(player *playerActor) int32 {
	attrs := player.progress.attributes
	return int32(float64(attrs.meleePower) * 0.30)
}
func useSkillDragTarget(custom clientCustomMessage) (float32, float32, float32, bool) {
	if len(custom.Values) >= 9 && custom.Values[6].Type == 4 && custom.Values[7].Type == 4 && custom.Values[8].Type == 4 {
		tx, ty, tz := custom.Values[6].Float32, custom.Values[7].Float32, custom.Values[8].Float32
		if tx != 0 || ty != 0 || tz != 0 {
			return tx, ty, tz, true
		}
	}
	if len(custom.Values) >= 14 && custom.Values[11].Type == 4 && custom.Values[12].Type == 4 && custom.Values[13].Type == 4 {
		tx, ty, tz := custom.Values[11].Float32, custom.Values[12].Float32, custom.Values[13].Float32
		if tx != 0 || ty != 0 || tz != 0 {
			return tx, ty, tz, true
		}
	}
	return 0, 0, 0, false
}
func skillTargetOutOfRange(caster worldcore.Transform, world *sceneLifecycle, maxRange float32) bool {
	if world == nil || maxRange <= 0 {
		return false
	}
	target, targetOK := world.selectedCombatTargetTransform()
	if !targetOK {
		return false
	}
	dx := caster.X - target.X
	dz := caster.Z - target.Z
	distance := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	out := distance > maxRange
	log.Printf("skill range gate caster=(%.1f,%.1f) target=(%.1f,%.1f) dist=%.1f max=%.1f %s", caster.X, caster.Z, target.X, target.Z, distance, maxRange, map[bool]string{true: "REJECT", false: "ok"}[out])
	return out
}
func (definition combatSkillDefinition) teamKey() int32 {
	teamKey := definition.cooldownTeam
	if teamKey == 0 {
		teamKey = -definition.cooldownCategory
	}
	return teamKey
}
func (p *playerActor) lockOtherSkillTeams(active int32, now time.Time) {
	for id, level := range p.learnedSkills {
		definition, ok := installedCombatSkillCatalog.definition(id, level)
		if !ok {
			continue
		}
		key := definition.teamKey()
		if key == active || key == 0 {
			continue
		}
		p.skillTeamCooldowns[key] = now.Add(taoLuSwitchCooldown)
	}
}
func (s *sceneLifecycle) entityInternalName(id uint32) string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, exists := s.entities[worldcore.EntityID(id)]
	if !exists {
		return ""
	}
	if meta.configID != "" {
		return meta.configID
	}
	return meta.description
}
func (s *sceneLifecycle) entityHealthPercent(id uint32) (int32, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	actor := s.combatActors[worldcore.EntityID(id)]
	if actor == nil {
		return 0, false
	}
	state := actor.Snapshot()
	if state.MaxHP <= 0 {
		return 0, false
	}
	return int32(int64(state.HP) * 100 / int64(state.MaxHP)), true
}

type entityRatioProperty struct {
	id    uint16
	value int32
}

func entityPropertyBatch(entityID uint64, properties []entityRatioProperty) []byte {
	msg := []byte{0x2D}
	msg = binary.LittleEndian.AppendUint16(msg, 1)
	msg = binary.LittleEndian.AppendUint64(msg, entityID)
	msg = binary.LittleEndian.AppendUint16(msg, uint16(len(properties)))
	for _, property := range properties {
		msg = binary.LittleEndian.AppendUint16(msg, property.id)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(property.value))
	}
	return msg
}
func writeSkillHitEvents(conn sceneMessageConnection, player *playerActor, world *sceneLifecycle, definition combatSkillDefinition, targetID, targetOwnerID uint32, damage int32, includeAnimation bool) error {
	victimName := ""
	healthPercent := int32(-1)
	if world != nil {
		victimName = world.entityInternalName(targetID)
		if percent, ok := world.entityHealthPercent(targetID); ok {
			healthPercent = percent
		}
	}
	hitKind := ""
	if definition.requiresTarget {
		hitKind = "SkillLock"
	}
	attackerName := player.actor.Snapshot().Name
	if includeAnimation {
		animation, err := skillAnimationFrame(playerObjectID, playerOwnerID, definition.id, targetID, targetOwnerID)
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(animation); err != nil {
			return err
		}
	}
	hit, err := skillHitFrame(targetID, targetOwnerID, damage, definition.id, hitKind, victimName, attackerName)
	if err != nil {
		return err
	}
	if err := conn.WriteFrame(hit); err != nil {
		return err
	}
	entityID := uint64(targetID) | uint64(targetOwnerID)<<32
	log.Printf("skill hit event sent target=%d owner=%d entity=%016X damage=%d skill=%s script=%s victim=%q frame_len=%d", targetID, targetOwnerID, entityID, damage, definition.id, definition.script, victimName, len(hit))
	if healthPercent >= 0 {
		ratio := entityPropertyBatch(entityID, []entityRatioProperty{{id: 0x1C, value: healthPercent}, {id: 0x71, value: healthPercent}})
		if err := conn.WriteFrame(ratio); err != nil {
			return err
		}
	}
	hurt, err := skillHurtFrame(targetID, targetOwnerID, playerObjectID, playerOwnerID)
	if err != nil {
		return err
	}
	return conn.WriteFrame(hurt)
}
func (s *sceneLifecycle) applySkillDamageTo(id, ownerID uint32, amount int32, now time.Time) ([]byte, int32, bool, error) {
	if s == nil || amount <= 0 {
		return nil, 0, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entityID := worldcore.EntityID(id)
	actor := s.combatActors[entityID]
	meta, exists := s.entities[entityID]
	if actor == nil || !exists || meta.ownerID != ownerID || meta.interaction != npcInteractionCombat || actor.Snapshot().HP <= 0 {
		return nil, 0, false, nil
	}
	if !actor.ApplyDamage(amount) {
		return nil, 0, false, nil
	}
	state := actor.Snapshot()
	if combat, isCombat := s.combatStates[entityID]; isCombat {
		combat.threat += amount
		if state.HP <= 0 {
			combat.threat = 0
			combat.respawnAt = now.Add(combat.respawnAfter)
			combat.leaveCombatAt = now.Add(5 * time.Second)
		}
		s.combatStates[entityID] = combat
	}
	frame, err := sceneObjectProperties(id, ownerID, 0, npcCombatProperties(state))
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
	return frame, loot, state.HP <= 0, err
}
func (s *sceneLifecycle) targetIfCombat(objectID, ownerID uint32) (uint32, uint32, bool) {
	if s == nil {
		return 0, 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := worldcore.EntityID(objectID)
	actor := s.combatActors[id]
	if actor == nil || actor.Snapshot().HP <= 0 {
		return 0, 0, false
	}
	meta, exists := s.entities[id]
	if !exists || meta.interaction != npcInteractionCombat {
		return 0, 0, false
	}
	if ownerID != 0 && meta.ownerID != ownerID {
		return 0, 0, false
	}
	return objectID, meta.ownerID, true
}
func skillAreaCastCenter(custom clientCustomMessage, definition combatSkillDefinition, world *sceneLifecycle) (worldcore.Transform, bool) {
	switch definition.targetMode {
	case skillTargetCasterCircle, skillTargetCasterSector, skillTargetCasterRectangle:
		return useSkillCastTransform(custom)
	case skillTargetSelectedArea:
		tx, ty, tz, hasPos := useSkillDragTarget(custom)
		if hasPos {
			return worldcore.Transform{X: tx, Y: ty, Z: tz}, true
		}
		return world.selectedCombatTargetTransform()
	default:
		return worldcore.Transform{}, false
	}
}
func skillTargetBufferFrame(mode int32, buffID string, targetID, targetOwnerID uint32, casterName string, uniqueID uint64) ([]byte, error) {
	if buffID == "" {
		return nil, fmt.Errorf("target buffer ID is empty")
	}
	return serverCustomIntMessage(0xA9, customInt(mode), customString(buffID), customObject(targetID, targetOwnerID), customObject(playerObjectID, playerOwnerID), customString(casterName), customString(casterName), customInt64(int64(uniqueID)), customInt(0))
}
func (s *sceneLifecycle) npcOwnerID(id uint32) (uint32, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, exists := s.entities[worldcore.EntityID(id)]
	if !exists {
		return 0, false
	}
	return meta.ownerID, true
}
func settleOpeningSkillHit(conn sceneMessageConnection, player *playerActor, world *sceneLifecycle, definition combatSkillDefinition, targetID, targetOwnerID uint32, damage int32, center worldcore.Transform, hasCenter bool, appliedBuffs []appliedSkillBuff, now time.Time) {
	if world == nil || player == nil || damage <= 0 {
		return
	}
	writeVictimEffects := func(victimID, victimOwner uint32) {
		uniqueID := uint64(now.UnixMilli())
		for _, buff := range appliedBuffs {
			if buff.staticData == 8682 {
				continue
			}
			if effectFrame, effectErr := skillCurEffectFrame(buff.staticData, definition.level, uniqueID); effectErr == nil {
				if err := conn.WriteFrame(effectFrame); err != nil {
					return
				}
			}
			buffID := player.buffConfigID(buff.slot)
			if buffID == "" {
				break
			}
			name := player.actor.Snapshot().Name
			if buffFrame, buffErr := skillBufferFrame(0, buffID, name, uniqueID, 0); buffErr == nil {
				if err := conn.WriteFrame(buffFrame); err != nil {
					return
				}
			}
			break
		}
		if effectFrame, err := skillEffectFrame(definition.id, victimID, victimOwner); err == nil {
			_ = conn.WriteFrame(effectFrame)
		}
	}
	if definition.targetMode == skillTargetSelected {
		if targetID == 0 || targetID == playerObjectID {
			return
		}
		targetFrame, loot, killed, err := world.applySkillDamageTo(targetID, targetOwnerID, damage, now)
		if err != nil || targetFrame == nil {
			return
		}
		player.enterCombatPresence()
		if err := conn.WriteFrame(targetFrame); err != nil {
			return
		}
		if err := writeSkillHitEvents(conn, player, world, definition, targetID, targetOwnerID, damage, false); err != nil {
			return
		}
		writeVictimEffects(targetID, targetOwnerID)
		if killed {
			player.addSilver(loot)
			if silverFrame, err := player.silverUpdate(); err == nil {
				_ = conn.WriteFrame(silverFrame)
			}
		}
		return
	}
	if !hasCenter {
		return
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
		log.Printf("skill opening area damage id=%s: %v", definition.id, err)
		return
	}
	for _, result := range results {
		player.enterCombatPresence()
		if err := conn.WriteFrame(result.frame); err != nil {
			return
		}
		if err := writeSkillHitEvents(conn, player, world, definition, result.id, result.ownerID, damage, false); err != nil {
			continue
		}
		writeVictimEffects(result.id, result.ownerID)
		if result.killed {
			player.addSilver(result.loot)
			if silverFrame, err := player.silverUpdate(); err == nil {
				_ = conn.WriteFrame(silverFrame)
			}
		}
	}
}
func applyOffensiveSegmentDamage(conn sceneMessageConnection, player *playerActor, world *sceneLifecycle, definition combatSkillDefinition, targetID, targetOwnerID uint32, damage int32, at time.Duration, center worldcore.Transform, hasCenter bool) {
	if world == nil || player == nil || damage <= 0 {
		return
	}
	now := time.Now()
	if definition.targetMode == skillTargetSelected {
		if targetID == 0 || targetID == playerObjectID {
			return
		}
		targetFrame, loot, killed, err := world.applySkillDamageTo(targetID, targetOwnerID, damage, now)
		if err != nil {
			log.Printf("skill segment damage id=%s target=%d at=%s: %v", definition.id, targetID, at, err)
			return
		}
		if targetFrame == nil {
			return
		}
		player.enterCombatPresence()
		if err := conn.WriteFrame(targetFrame); err != nil {
			return
		}
		if err := writeSkillHitEvents(conn, player, world, definition, targetID, targetOwnerID, damage, false); err != nil {
			log.Printf("skill segment hit events id=%s target=%d: %v", definition.id, targetID, err)
			return
		}
		if resultFrame, err := skillResultFrame(playerObjectID, playerOwnerID, definition.id, targetID, targetOwnerID); err == nil {
			if err := conn.WriteFrame(resultFrame); err != nil {
				return
			}
		}
		if killed {
			player.addSilver(loot)
			if silverFrame, err := player.silverUpdate(); err == nil {
				_ = conn.WriteFrame(silverFrame)
			}
			log.Printf("skill segment defeated NPC id=%s target=%d at=%s loot=%d", definition.id, targetID, at, loot)
		}
		return
	}
	if !hasCenter {
		return
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
		log.Printf("skill segment area damage id=%s at=%s: %v", definition.id, at, err)
		return
	}
	for _, result := range results {
		player.enterCombatPresence()
		if err := conn.WriteFrame(result.frame); err != nil {
			return
		}
		if err := writeSkillHitEvents(conn, player, world, definition, result.id, result.ownerID, damage, false); err != nil {
			continue
		}
		if resultFrame, err := skillResultFrame(playerObjectID, playerOwnerID, definition.id, result.id, result.ownerID); err == nil {
			if err := conn.WriteFrame(resultFrame); err != nil {
				return
			}
		}
		if result.killed {
			player.addSilver(result.loot)
			if silverFrame, err := player.silverUpdate(); err == nil {
				_ = conn.WriteFrame(silverFrame)
			}
			log.Printf("skill segment defeated NPC id=%s target=%d at=%s loot=%d", definition.id, result.id, at, result.loot)
		}
	}
}
func scheduleSkillHitFrames(conn sceneMessageConnection, player *playerActor, world *sceneLifecycle, definition combatSkillDefinition, targetID, targetOwnerID uint32, damage int32, center worldcore.Transform, hasCenter bool, appliedBuffs []appliedSkillBuff) {
	for index, at := range definition.hitFrames {
		at := at
		time.AfterFunc(at, func() {
			if index == 0 {
				log.Printf("skill opening hit-frame firing id=%s target=%d at=%s damage=%d", definition.id, targetID, at, damage)
				settleOpeningSkillHit(conn, player, world, definition, targetID, targetOwnerID, damage, center, hasCenter, appliedBuffs, time.Now())
				return
			}
			log.Printf("skill hit-frame firing id=%s target=%d at=%s damage=%d", definition.id, targetID, at, damage)
			applyOffensiveSegmentDamage(conn, player, world, definition, targetID, targetOwnerID, damage, at, center, hasCenter)
		})
	}
}

const viewportNormalAttack uint16 = 41

func updateSkillRangeHints(conn sceneMessageConnection, caster worldcore.Transform, target worldcore.Transform, hasTarget bool) error {
	for index, skill := range starterSkillViews {
		definition, ok := combatSkills[skill.configID]
		canUse := uint8(1)
		if ok && hasTarget && definition.range_ > 0 && definition.requiresTarget {
			dx := caster.X - target.X
			dz := caster.Z - target.Z
			if float32(math.Sqrt(float64(dx*dx+dz*dz))) > definition.range_ {
				canUse = 0
			}
		}
		frame, err := serverObjectProperty(viewportSkill, uint16(index+1), []serverViewProperty{viewByte(0x08BC, canUse)})
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	return nil
}
func refreshSkillRangeHints(link sceneMessageConnection, world *sceneLifecycle, pos role.Position) {
	if link == nil || world == nil {
		return
	}
	target, hasTarget := world.selectedCombatTargetTransform()
	caster := worldcore.Transform{X: pos.X, Y: pos.Y, Z: pos.Z}
	if err := updateSkillRangeHints(link, caster, target, hasTarget); err != nil {
		log.Printf("refresh skill range hints: %v", err)
	}
}
func skillViewFrames(player *playerActor) ([][]byte, error) {
	learned := player.learnedSkillSnapshot()
	capacity := len(learned)
	if capacity <= 512 {
		capacity = 512
	}
	frames := [][]byte{serverCreateView(serverViewSpec{ID: viewportSkill, Capacity: uint16(capacity)})}
	ids := make([]string, 0, len(learned))
	for id := range learned {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	slot := 0
	for _, id := range ids {
		level := learned[id]
		if level <= 0 {
			continue
		}
		staticData, maxLevel := skillStaticFor(id)
		slot++
		frame, err := serverViewAdd(viewportSkill, uint16(slot), []serverViewProperty{viewInt(propItemType, 1000), viewString(0x05A0, id), viewByte(0x08BC, 1), viewInt(propStaticData, staticData), viewByte(0x05B9, byte(level)), viewInt(propMaxLevel, maxLevel), viewInt(0x07F2, 0)})
		if err != nil {
			return nil, fmt.Errorf("encode SkillContainer %s: %w", id, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
func starterNormalAttackViewFrames() ([][]byte, error) {
	normal := starterSkillViews[1]
	frames := [][]byte{serverCreateView(serverViewSpec{ID: viewportNormalAttack, Capacity: 1})}
	frame, err := serverViewAdd(viewportNormalAttack, 1, []serverViewProperty{viewInt(propItemType, normal.itemType), viewString(0x05A0, normal.configID), viewByte(0x08BC, 1), viewInt(propStaticData, normal.staticData), viewByte(0x05B9, byte(normal.level)), viewInt(propMaxLevel, normal.maxLevel), viewInt(0x07F2, int32(normal.pauseTime))})
	if err != nil {
		return nil, fmt.Errorf("encode NormalAttackContainer: %w", err)
	}
	frames = append(frames, frame)
	return frames, nil
}

type snapshot608PlayerSnapshot struct {
	EntityID   uint64
	X          float32
	Y          float32
	Z          float32
	Orient     float32
	Properties []clientdata.IndexedProperty
}

func appendSnapshot608Transform(dst []byte, s snapshot608PlayerSnapshot) []byte {
	dst = binary.LittleEndian.AppendUint32(dst, math.Float32bits(s.X))
	dst = binary.LittleEndian.AppendUint32(dst, math.Float32bits(s.Y))
	dst = binary.LittleEndian.AppendUint32(dst, math.Float32bits(s.Z))
	dst = binary.LittleEndian.AppendUint32(dst, math.Float32bits(s.Orient))
	return dst
}
func buildSnapshot608Frame(s snapshot608PlayerSnapshot) ([]byte, error) {
	raw := make([]byte, 8, 62+len(s.Properties)*16)
	binary.LittleEndian.PutUint64(raw, s.EntityID)
	raw = appendSnapshot608Transform(raw, s)
	raw = appendSnapshot608Transform(raw, s)
	raw = append(raw, make([]byte, 20)...)
	raw = binary.LittleEndian.AppendUint16(raw, uint16(len(s.Properties)))
	for _, property := range s.Properties {
		raw = appendNPCProperty(raw, property)
	}
	encoder, err := quicklz.New(quicklz.COMPRESSION_LEVEL_1, quicklz.STREAMING_BUFFER_0)
	if err != nil {
		return nil, err
	}
	compressed := make([]byte, len(raw)+400)
	n, err := encoder.Compress(&raw, &compressed)
	if err != nil {
		return nil, err
	}
	pkt := make([]byte, 1+int(n))
	pkt[0] = 0x28
	copy(pkt[1:], compressed[:n])
	return pkt, nil
}

type facultyStoreIface interface {
	Load(role.RoleID) (facultyProgressSnapshot, bool)
	Save(role.RoleID, facultyProgressSnapshot) error
}
type bagStoreIface interface {
	Load(role.RoleID) ([]bagItem, bool)
	Save(role.RoleID, []bagItem) error
}
type equipStoreIface interface {
	Load(role.RoleID) ([]wornEquipItem, bool)
	Save(role.RoleID, []wornEquipItem) error
}
type jingMaiStoreIface interface {
	Load(role.RoleID) (jingMaiProgressSnapshot, bool)
	Save(role.RoleID, jingMaiProgressSnapshot) error
}
type currencyStoreIface interface {
	Load(role.RoleID) (currencySnapshot, bool)
	Save(role.RoleID, currencySnapshot) error
}

var defaultStringNameINI = filepath.Join(defaultModernTextRoot, "stringname.idres")

func loadStringNames(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	names := make(map[string]string, 130000)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "\ufeff")
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		names[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return names, nil
}
