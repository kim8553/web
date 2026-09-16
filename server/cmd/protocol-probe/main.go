package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/qinggong"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/session"
	"github.com/local/9yin-go-server/internal/transport"
	"github.com/local/9yin-go-server/internal/world"
	"github.com/local/9yin-go-server/migrations"
	"io"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

func main() {
	addr := flag.String("listen", "127.0.0.1:19061", "TCP listen address")
	allChengduNPCs := flag.Bool("npc-all-chengdu", true, "compatibility flag: load NPCs dynamically from the active modern scene")
	npcRadius := flag.Float64("npc-radius", 120, "NPC viewport radius in scene units")
	npcTablePath := flag.String("npc-table", defaultNPCTablePath, "modern client NPC template table")
	npcCreatorPath := flag.String("npc-creator", defaultNPCCreatorPath, "modern client scene NPC creator")
	npcConfigID := flag.String("npc-config", "Relivecity05B", "NPC template ID to materialize")
	npcPatrolCount := flag.Int("npc-patrol-count", 3, "nearby NPCs simulated by the low-rate server patrol loop (0 disables it)")
	npcPathDir := flag.String("npc-path-dir", "", "map_path directory containing <scene>_main.path patrol routes (empty: try resources/modern/share/map/path)")
	demoDamage := flag.Int("demo-damage", 0, "test-only damage applied once after the player reaches the active scene (0 disables it)")
	starterSilver := flag.Int("starter-silver", 99999, "test-only starting CapitalType1 (碎银) for the active player")
	gmListen := flag.String("gm-listen", "127.0.0.1:19062", "local GM HTTP listen address (empty disables)")
	logFile := flag.String("log-file", runtimeProjectPath("artifacts", "local", "logs", "protocol-probe-live.log"), "append runtime log to this local file")
	npcFuncPath := flag.String("npc-func", defaultLegacyNPCFuncPath, "Npc_Func.xml used to build per-NPC service menus")
	switchScene := flag.String("switch-scene", "", "test-only live destination: config,resource,x,y,z,orient (empty disables it)")
	npcAuditAll := flag.Bool("npc-audit-all", false, "inventory every modern NPC creator scene and exit")
	npcServiceAuditScene := flag.String("npc-service-audit-scene", "", "write effective NPC service audit for config,resource,x,y,z,orient and exit")
	qingGongDefinePath := flag.String("qinggong-define", filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgdefine.ini"), "current-client QGDefine.ini used to validate and charge 216")
	flag.Parse()
	if envCount := os.Getenv("NINEYIN_NPC_PATROL_COUNT"); envCount != "" {
		if v, err := strconv.Atoi(envCount); err == nil {
			*npcPatrolCount = v
			log.Printf("npc patrol count overridden by NINEYIN_NPC_PATROL_COUNT=%d", v)
		}
	}
	if *logFile != "" {
		if err := os.MkdirAll(filepath.Dir(*logFile), 0755); err != nil {
			log.Fatalf("create log directory: %v", err)
		}
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("open runtime log %s: %v", *logFile, err)
		}
		defer file.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, file))
	}
	var switchDestination *sceneDestination
	if *switchScene != "" {
		parsed, parseErr := parseSceneDestination(*switchScene)
		if parseErr != nil {
			log.Fatalf("parse -switch-scene: %v", parseErr)
		}
		switchDestination = &parsed
	}
	npcSchema := clientdata.VisibleNPCModernV1()
	qingGongCatalog, qingGongErr := qinggong.Load(*qingGongDefinePath)
	if qingGongErr != nil {
		log.Fatalf("load current-client qinggong definitions: %v", qingGongErr)
	}
	log.Printf("qinggong catalog ready definitions=%d path=%s", qingGongCatalog.Count(), *qingGongDefinePath)
	npcFuncs, npcFuncErr := npcfunc.Load(*npcFuncPath)
	if npcFuncErr != nil {
		log.Printf("Npc_Func menu registry unavailable (%s): %v; NPCs without this table will expose no fabricated service menu", *npcFuncPath, npcFuncErr)
	} else {
		log.Printf("Npc_Func menu registry ready configured_npcs=%d path=%s", npcFuncs.Count(), *npcFuncPath)
	}
	if *npcAuditAll {
		entries, auditErr := auditNPCSceneRoot(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir), npcSchema)
		if auditErr != nil {
			log.Fatal(auditErr)
		}
		payload, marshalErr := json.MarshalIndent(entries, "", "  ")
		if marshalErr != nil {
			log.Fatal(marshalErr)
		}
		output := runtimeProjectPath("artifacts", "audit", "npc-scene-audit.json")
		if writeErr := os.MkdirAll(filepath.Dir(output), 0755); writeErr != nil {
			log.Fatal(writeErr)
		}
		if writeErr := os.WriteFile(output, append(payload, '\n'), 0644); writeErr != nil {
			log.Fatal(writeErr)
		}
		var creators, resolved, unresolved, failed int
		for _, entry := range entries {
			creators += entry.CreatorInstances
			resolved += entry.Resolved
			unresolved += entry.Unresolved
			if entry.Error != "" {
				failed++
			}
		}
		log.Printf("NPC full audit scenes=%d creator_instances=%d resolved=%d unresolved=%d failed_scenes=%d output=%s", len(entries), creators, resolved, unresolved, failed, output)
		return
	}
	var staticNPCs []npcSpawn
	var sceneRegistry *sceneNPCRegistry
	var err error
	if *allChengduNPCs {
		sceneRegistry, err = newSceneNPCRegistry(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir), npcSchema)
		if err == nil {
			log.Printf("modern scene NPC registry ready template_tables=%d creator_root=%s", len(sceneRegistry.tables), sceneRegistry.root)
		}
	} else {
		staticNPCs, err = loadNPCSpawns(*npcTablePath, *npcCreatorPath, *npcConfigID, npcSchema)
	}
	if err != nil {
		log.Fatalf("load modern NPC catalog: %v", err)
	}
	if *npcServiceAuditScene != "" {
		if sceneRegistry == nil {
			log.Fatal("npc service audit requires dynamic modern scene registry")
		}
		destination, parseErr := parseSceneDestination(*npcServiceAuditScene)
		if parseErr != nil {
			log.Fatalf("parse -npc-service-audit-scene: %v", parseErr)
		}
		scene := normalizeClientScene(destination.location.Scene)
		catalog, stats, auditErr := sceneRegistry.catalogFor(scene)
		if auditErr != nil {
			log.Fatal(auditErr)
		}
		report := buildNPCServiceAudit(scene, stats, catalog, npcFuncs)
		payload, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			log.Fatal(marshalErr)
		}
		output := runtimeProjectPath("artifacts", "audit", "npc-service-audit-"+auditFileComponent(scene.Resource)+".json")
		if writeErr := os.MkdirAll(filepath.Dir(output), 0755); writeErr != nil {
			log.Fatal(writeErr)
		}
		if writeErr := os.WriteFile(output, append(payload, '\n'), 0644); writeErr != nil {
			log.Fatal(writeErr)
		}
		log.Printf("NPC service audit scene=%s resource=%s NPCs=%d resolved=%d output=%s", scene.Config, scene.Resource, len(report.NPCs), stats.Resolved, output)
		return
	}
	for _, npc := range staticNPCs {
		_, propertyErr := npc.resolved.OrderedProperties(npcSchema)
		if propertyErr != nil {
			log.Fatalf("validate modern NPC %s: %v", npc.resolved.ConfigID, propertyErr)
		}
	}
	if sceneRegistry == nil {
		log.Printf("static modern NPC catalog ready objects=%d viewport_radius=%.1f", len(staticNPCs), *npcRadius)
	}
	gameDB, err := openGameDataDB()
	if err != nil {
		log.Fatal(err)
	}
	if gameDB != nil {
		defer gameDB.Close()
		if migrateErr := (migrations.Runner{DB: gameDB}).Up(context.Background()); migrateErr != nil {
			log.Fatalf("apply game data migrations: %v", migrateErr)
		}
		log.Printf("game data store: MySQL ready (NINEYIN_MYSQL_DSN)")
	} else {
		log.Printf("game data store: legacy JSON files (set NINEYIN_MYSQL_DSN to use MySQL)")
	}
	patrolPathDir := *npcPathDir
	if patrolPathDir == "" {
		patrolPathDir = os.Getenv("NINEYIN_NPC_PATH_DIR")
	}
	if patrolPathDir == "" {
		patrolPathDir = filepath.Join(runtimeProjectRoot, "resources", "modern", "share", "map", "path")
	}
	if _, statErr := os.Stat(patrolPathDir); statErr != nil {
		log.Printf("patrol path dir %s unavailable (%v); NPC patrol falls back to simulated movement (use -npc-path-dir or NINEYIN_NPC_PATH_DIR)", patrolPathDir, statErr)
		patrolPathDir = ""
	} else {
		log.Printf("patrol path dir: %s", patrolPathDir)
	}
	store, err := openRoleStore(gameDB)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	var facultyStore facultyStoreIface
	var shortcutStore shortcutStoreIface
	var bagStore bagStoreIface
	var equipStore equipStoreIface
	var jingmaiStore jingMaiStoreIface
	var currencyStore currencyStoreIface
	var qinggongStore qingGongStoreIface
	var skillStore *skillGrantStore
	var fwzCardStore fwzCardStoreIface
	if gameDB != nil {
		facultyStore = &mysqlFacultyStore{db: gameDB}
		shortcutStore = &mysqlShortcutStore{db: gameDB}
		bagStore = &mysqlBagStore{db: gameDB}
		equipStore = &mysqlEquipStore{db: gameDB}
		jingmaiStore = &mysqlJingMaiStore{db: gameDB}
		currencyStore = &mysqlCurrencyStore{db: gameDB}
		qinggongStore = newMySQLQingGongStore(gameDB)
		skillStore = &skillGrantStore{db: gameDB}
		fwzCardStore = &mysqlFwzCardStore{db: gameDB}
	} else {
		facultyStore, err = openFacultyProgressStore()
		if err != nil {
			log.Fatal(err)
		}
		shortcutStore, err = openShortcutStore()
		if err != nil {
			log.Fatal(err)
		}
		bagStore, err = openBagStore()
		if err != nil {
			log.Fatal(err)
		}
		equipStore, err = openEquipStore()
		if err != nil {
			log.Fatal(err)
		}
		jingmaiStore, err = openJingMaiProgressStore()
		if err != nil {
			log.Fatal(err)
		}
		currencyStore, err = openCurrencyStore()
		if err != nil {
			log.Fatal(err)
		}
	}
	itemCatalog, err := loadItemCatalog(defaultToolItemINI)
	if err != nil {
		log.Fatalf("load tool_item catalog: %v", err)
	}
	log.Printf("item catalog ready definitions=%d path=%s", itemCatalog.Count(), defaultToolItemINI)
	equipCatalog, err := loadEquipCatalog(defaultEquipmentINI)
	if err != nil {
		log.Fatalf("load equipment catalog: %v", err)
	}
	log.Printf("equip catalog ready definitions=%d path=%s", equipCatalog.Count(), defaultEquipmentINI)
	stringNames, err := loadStringNames(defaultStringNameINI)
	if err != nil {
		log.Fatalf("load string names: %v", err)
	}
	log.Printf("string names ready entries=%d path=%s", len(stringNames), defaultStringNameINI)
	loaded, questErr := loadQuestCatalogFromTables(defaultQuestTaskRoot)
	if questErr != nil {
		log.Printf("load quest tables: %v; keeping built-in quests", questErr)
	} else {
		for id, q := range loaded {
			questCatalog[id] = q
		}
		log.Printf("quest catalog ready definitions=%d path=%s", len(questCatalog), defaultQuestTaskRoot)
	}
	dropTable, err := loadDropTable(defaultDropTablePath)
	if err != nil {
		log.Fatalf("load drop table: %v", err)
	}
	log.Printf("drop table ready dropIDs=%d path=%s", len(dropTable.byDropID), defaultDropTablePath)
	gm := newGMHub()
	if *gmListen != "" {
		go serveGM(*gmListen, gm, itemCatalog, equipCatalog, store, stringNames)
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("protocol probe listening on %s", *addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handle(conn, store, facultyStore, shortcutStore, bagStore, equipStore, jingmaiStore, currencyStore, qinggongStore, skillStore, fwzCardStore, itemCatalog, equipCatalog, dropTable, npcSchema, sceneRegistry, staticNPCs, npcFuncs, qingGongCatalog, float32(*npcRadius), *npcPatrolCount, int32(*demoDamage), int32(*starterSilver), switchDestination, patrolPathDir, gm)
	}
}
func handle(conn net.Conn, store *roleStore, facultyStore facultyStoreIface, shortcutStore shortcutStoreIface, bagStore bagStoreIface, equipStore equipStoreIface, jingmaiStore jingMaiStoreIface, currencyStore currencyStoreIface, qinggongStore qingGongStoreIface, skillStore *skillGrantStore, fwzCardStore fwzCardStoreIface, itemCatalog *itemCatalog, equipCatalog *equipCatalog, dropTable *dropTable, npcSchema clientdata.Schema, sceneRegistry *sceneNPCRegistry, staticNPCs []npcSpawn, npcFuncs *npcfunc.Registry, qingGongCatalog *qinggong.Catalog, npcRadius float32, npcPatrolCount int, demoDamage int32, starterSilver int32, switchDestination *sceneDestination, patrolPathDir string, gm *gmHub) {
	config := transport.DefaultConnectionConfig()
	config.MaxFrameSize = 64 << 10
	link, err := transport.NewConnection(conn, config)
	if err != nil {
		log.Printf("%s: initialize transport: %v", conn.RemoteAddr(), err)
		_ = conn.Close()
		return
	}
	defer link.Close()
	machine := session.New()
	var sceneNPCs []npcSpawn
	log.Printf("session=%d remote=%s opened phase=%s", machine.ID(), conn.RemoteAddr(), machine.Phase())
	defer func() {
		machine.Close()
		log.Printf("session=%d remote=%s closed phase=%s", machine.ID(), conn.RemoteAddr(), machine.Phase())
	}()
	world := newSceneLifecycle(conn.RemoteAddr().String(), link, npcFuncs)
	world.setPatrolPathDir(patrolPathDir)
	defer world.close()
	ctx := context.Background()
	var accountKey role.AccountKey
	var account role.Account
	var selected *role.RoleSnapshot
	var player *playerActor
	var runtime *sceneRuntime
	var gmSession *gmSession
	defer func() {
		if player != nil {
			player.clearYufeng()
		}
	}()
	persistFaculty := func() {
		if facultyStore == nil || selected == nil || player == nil {
			return
		}
		if err := facultyStore.Save(selected.ID, player.facultySnapshot()); err != nil {
			log.Printf("%s: persist faculty progress: %v", conn.RemoteAddr(), err)
		}
	}
	defer persistFaculty()
	persistJingMai := func() {
		if jingmaiStore == nil || selected == nil || player == nil {
			return
		}
		if err := jingmaiStore.Save(selected.ID, player.jingMaiProgressSnapshotFull()); err != nil {
			log.Printf("%s: persist jingmai progress: %v", conn.RemoteAddr(), err)
		}
	}
	defer persistJingMai()
	persistCurrency := func() {
		if currencyStore == nil || selected == nil || player == nil {
			return
		}
		var snapshot currencySnapshot
		snapshot.fromActor(player)
		if err := currencyStore.Save(selected.ID, snapshot); err != nil {
			log.Printf("%s: persist currency: %v", conn.RemoteAddr(), err)
		}
	}
	defer persistCurrency()
	qingGongGranted := false
	neigongGranted := false
	customizingRestored := false
	grantStarterProgress := func() error {
		if player == nil {
			return fmt.Errorf("character has not entered a scene")
		}
		frames, err := player.progressViewFrames()
		if err != nil {
			return fmt.Errorf("encode View=43: %w", err)
		}
		for _, frame := range frames {
			if err := link.WriteFrame(frame); err != nil {
				return fmt.Errorf("write View=43: %w", err)
			}
		}
		skillFrames, err := skillViewFrames(player)
		if err != nil {
			return fmt.Errorf("encode View=40: %w", err)
		}
		for _, frame := range skillFrames {
			if err := link.WriteFrame(frame); err != nil {
				return fmt.Errorf("write View=40: %w", err)
			}
		}
		normalFrames, normalErr := starterNormalAttackViewFrames()
		if normalErr != nil {
			return fmt.Errorf("encode View=41: %w", normalErr)
		}
		for _, frame := range normalFrames {
			if err := link.WriteFrame(frame); err != nil {
				return fmt.Errorf("write View=41: %w", err)
			}
		}
		if err := grantStarterZhenFa(link, player); err != nil {
			return fmt.Errorf("grant zhenfa view: %w", err)
		}
		if err := grantStarterShouFa(link, player); err != nil {
			return fmt.Errorf("grant shoufa view: %w", err)
		}
		if err := grantStarterJingMai(link, player); err != nil {
			return fmt.Errorf("grant jingmai view: %w", err)
		}
		if err := grantJingMaiProgress(link, player); err != nil {
			return fmt.Errorf("replay jingmai progress: %w", err)
		}
		if err := player.resyncShortcutRecord(link); err != nil {
			return fmt.Errorf("restore persisted shortcut rows: %w", err)
		}
		if granted, err := grantStarterShortcuts(link, player); err != nil {
			return fmt.Errorf("grant starter shortcuts: %w", err)
		} else if granted > 0 {
			log.Printf("%s: granted %d starter shortcuts [%d..%d]", conn.RemoteAddr(), granted, 1, len(starterShortcutIDs))
		}
		if err := grantCollectSkillRec(link, player); err != nil {
			return fmt.Errorf("grant CollectSkillRec: %w", err)
		}
		if err := grantBagItems(link, player, itemCatalog, equipCatalog, conn.RemoteAddr().String()); err != nil {
			return fmt.Errorf("grant bag items: %w", err)
		}
		if err := grantEquipView(link, player, equipCatalog, conn.RemoteAddr().String()); err != nil {
			return fmt.Errorf("grant equip view: %w", err)
		}
		if fwzCardStore != nil && selected != nil {
			if err := grantFwzRecords(link, fwzCardStore, selected.ID); err != nil {
				return fmt.Errorf("grant fwz records: %w", err)
			}
		}
		state, err := player.vitalUpdate()
		if err != nil {
			return fmt.Errorf("encode inner-power/player attributes: %w", err)
		}
		if err := link.WriteFrame(state); err != nil {
			return fmt.Errorf("write inner-power/player attributes: %w", err)
		}
		if !customizingRestored && selected != nil {
			sendCustomizingRestore(link, shortcutStore, selected.ID, conn.RemoteAddr().String())
			customizingRestored = true
		}
		neigongGranted = true
		return nil
	}
	grantSceneRedSuperArmor := func() error {
		if player == nil {
			return fmt.Errorf("character has not entered a scene")
		}
		expires, err := grantRedSuperArmor(link, player)
		if err != nil {
			return err
		}
		schedulePlayerBuffExpiry(link, player, redSuperArmorBuffSlot, redSuperArmorStaticData, expires)
		return nil
	}
	defer func() {
		if gmSession != nil {
			gm.detach(gmSession.id)
		}
	}()
	explicitSceneReady := false
	activeNPCs := make(map[int]struct{})
	lastPositionSave := time.Time{}
	lastRangeHintAt := time.Time{}
	var pendingLocation *role.Location
	defer func() {
		if pendingLocation != nil && selected != nil {
			if err := store.saveLocation(ctx, selected, *pendingLocation); err != nil {
				log.Printf("%s: persist final position: %v", conn.RemoteAddr(), err)
			}
		}
	}()
	processGMCommand := func(command gmCommand) (commandErr error) {
		defer func() {
			command.complete(commandErr)
		}()
		if gmSession == nil || player == nil {
			return fmt.Errorf("character has not entered a scene")
		}
		switch command.action {
		case "set_silver":
			player.setSilver(command.silver)
			update, updateErr := player.silverUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("碎银设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setSilver(command.silver)
			log.Printf("%s: GM set CapitalType1=%d", conn.RemoteAddr(), command.silver)
			persistCurrency()
		case "set_gold":
			player.setGold(command.gold)
			update, updateErr := player.goldUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("黄金设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("黄金设为 %d", command.gold))
			log.Printf("%s: GM set CapitalType0=%d", conn.RemoteAddr(), command.gold)
			persistCurrency()
		case "set_silver_card":
			player.setSilverCard(command.silverCard)
			update, updateErr := player.silverCardUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("官银设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("官银设为 %d", command.silverCard))
			log.Printf("%s: GM set CapitalType2=%d", conn.RemoteAddr(), command.silverCard)
			persistCurrency()
		case "set_silver_ticket":
			player.setSilverTicket(command.silverTicket)
			update, updateErr := player.silverTicketUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("银票设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("银票设为 %d", command.silverTicket))
			log.Printf("%s: GM set CapitalType4=%d", conn.RemoteAddr(), command.silverTicket)
			persistCurrency()
		case "set_sp", "add_sp", "fill_sp":
			var sp int32
			var update []byte
			var updateErr error
			switch command.action {
			case "set_sp":
				sp, update, updateErr = player.setSP(command.sp)
			case "add_sp":
				sp, update, updateErr = player.addSP(command.sp)
			default:
				sp, update, updateErr = player.setSP(maxPlayerSP)
			}
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("怒气设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setSP(sp, fmt.Sprintf("怒气设为 %d/%d", sp, maxPlayerSP))
			log.Printf("%s: GM %s SP=%d", conn.RemoteAddr(), command.action, sp)
			persistCurrency()
		case "set_move_speed":
			update, updateErr := player.setMoveSpeed(command.value)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("移动速度设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("移动速度设为 %.2f", command.value))
			log.Printf("%s: GM set MoveSpeed/RunSpeed=%.2f", conn.RemoteAddr(), command.value)
		case "set_gravity":
			update, updateErr := player.setGravity(command.value)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("下落重力设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("下落重力设为 %.2f", command.value))
			log.Printf("%s: GM set Gravity=%.2f", conn.RemoteAddr(), command.value)
		case "give_item":
			item, ok := itemCatalog.Lookup(command.configID)
			if !ok {
				eq, eqOK := equipCatalog.Lookup(command.configID)
				if eqOK {
					bagItem := bagItem{ConfigID: eq.ConfigID, ItemType: eq.ItemType, Amount: command.amount, ViewID: eq.ViewID, Name: eq.Name, EquipType: eq.EquipType, ColorLevel: eq.ColorLevel, Hardiness: eq.Hardiness, MaxHardiness: eq.MaxHardiness}
					player.addBagItem(bagItem)
					if saveErr := bagStore.Save(selectedRoleID(selected), player.bagSnapshot()); saveErr != nil {
						log.Printf("%s: persist bag after give: %v", conn.RemoteAddr(), saveErr)
					}
					if err := grantBagItems(link, player, itemCatalog, equipCatalog, conn.RemoteAddr().String()); err != nil {
						gmSession.setAction("发装备失败：" + err.Error())
						return err
					}
					gmSession.setAction(fmt.Sprintf("已发装备 %s (%s) x%d 到背包装备栏", eq.ConfigID, eq.EquipType, command.amount))
					log.Printf("%s: GM give equip %s (%s/%s) x%d to bag view=%d", conn.RemoteAddr(), eq.ConfigID, eq.Name, eq.EquipType, command.amount, eq.ViewID)
					return nil
				}
				gmSession.setAction("未找到物品：" + command.configID)
				return fmt.Errorf("unknown item %q", command.configID)
			}
			if command.container != "" && command.container != "bag" {
				gmSession.setAction("暂不支持容器：" + command.container)
				return fmt.Errorf("unsupported give container %q", command.container)
			}
			bagItem := bagItem{ConfigID: item.ConfigID, ItemType: item.ItemType, Amount: command.amount, ViewID: item.ViewID, Name: item.Name, MaxAmount: item.MaxAmount, FuncPack: item.FuncPack, LogicPack: item.LogicPack, PropModifyPack: item.PropModifyPack, TextureType: item.TextureType}
			player.addBagItem(bagItem)
			if saveErr := bagStore.Save(selectedRoleID(selected), player.bagSnapshot()); saveErr != nil {
				log.Printf("%s: persist bag after give: %v", conn.RemoteAddr(), saveErr)
			}
			// Latest-client LIVE A/B: a live View 2 already exists here. Historical
			// bag lifecycle evidence refreshes a live bag with DELETE_VIEW before
			// CREATE_VIEW + authoritative replay. Delete only the normal tool bag;
			// grantBagItems below recreates it using the current ordinal-safe encoder.
			for _, bagView := range []uint16{2, 121, 123, 125} {
				if err := link.WriteFrame(serverDeleteViewCompat(bagView)); err != nil {
					gmSession.setAction("刷新背包失败：" + err.Error())
					return err
				}
			}
			log.Printf("%s: latest-client LEGACY-BAG refresh deleted canonical main Views=2,121,123,125 before authoritative replay", conn.RemoteAddr())
			if err := grantBagItems(link, player, itemCatalog, equipCatalog, conn.RemoteAddr().String()); err != nil {
				gmSession.setAction("发背包物品失败：" + err.Error())
				return err
			}
			gmSession.setAction(fmt.Sprintf("已发物品 %s x%d 到背包", item.Name, command.amount))
			log.Printf("%s: GM give item %s (%s) x%d to bag", conn.RemoteAddr(), item.ConfigID, item.Name, command.amount)
		case "set_faction":
			preset, presetErr := gmFactionPresetByKey(command.faction)
			if presetErr != nil {
				return presetErr
			}
			if saveErr := store.saveFaction(ctx, selected, preset.Faction); saveErr != nil {
				gmSession.setAction("门派保存失败：" + saveErr.Error())
				return saveErr
			}
			update, updateErr := player.setFaction(preset.Faction)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("门派同步失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setFaction(preset.Faction, preset.Label)
			log.Printf("%s: GM set School=%s (%s)", conn.RemoteAddr(), preset.Faction, preset.Label)
		case "switch_scene":
			if runtime == nil || selected == nil {
				gmSession.setAction("切图失败：角色尚未进入场景")
				return fmt.Errorf("character has not entered a scene")
			}
			if switchErr := runtime.SwitchScene(command.destination); switchErr != nil {
				gmSession.setAction("切图失败：" + switchErr.Error())
				return switchErr
			}
			sceneNPCs = runtime.npcCatalog
			pendingLocation = &runtime.activeRole.Location
			if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
				log.Printf("%s: persist GM scene switch: %v", conn.RemoteAddr(), saveErr)
			} else {
				pendingLocation = nil
			}
			gmSession.setLocation(runtime.activeRole.Location, "已发送 GM 在线切图，等待目标场景 Ready")
			log.Printf("%s: GM sent live 0x0C re-entry to scene=%s resource=%s", conn.RemoteAddr(), runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource)
		case "teleport":
			if runtime == nil || selected == nil {
				gmSession.setAction("场景内瞬移失败：角色尚未进入场景")
				return fmt.Errorf("character has not entered a scene")
			}
			if teleportErr := runtime.TeleportWithinScene(command.position); teleportErr != nil {
				gmSession.setAction("场景内瞬移失败：" + teleportErr.Error())
				return teleportErr
			}
			pendingLocation = &runtime.activeRole.Location
			if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
				log.Printf("%s: persist GM same-scene teleport: %v", conn.RemoteAddr(), saveErr)
			} else {
				pendingLocation = nil
			}
			gmSession.setLocation(runtime.activeRole.Location, "已发送场景内瞬移（0x1F）")
			log.Printf("%s: GM sent same-scene ServerLocation x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), command.position.X, command.position.Y, command.position.Z, command.position.Orient)
		case "quest_accept":
			if err := world.acceptQuest(uint32(command.value)); err != nil {
				gmSession.setAction("接任务失败：" + err.Error())
				return err
			}
			gmSession.setAction(fmt.Sprintf("已接任务 %d，任务怪已刷新", uint32(command.value)))
			log.Printf("%s: GM accepted quest %d", conn.RemoteAddr(), uint32(command.value))
		case "quest_submit":
			if err := world.submitQuest(uint32(command.value)); err != nil {
				gmSession.setAction("交任务失败：" + err.Error())
				return err
			}
			gmSession.setAction(fmt.Sprintf("已交任务 %d，获得奖励", uint32(command.value)))
			log.Printf("%s: GM submitted quest %d", conn.RemoteAddr(), uint32(command.value))
		case "quest_list":
			ids := world.availableQuestIDs()
			gmSession.setAction(fmt.Sprintf("可接任务: %v", ids))
			log.Printf("%s: GM listed quests %v", conn.RemoteAddr(), ids)
		case "npc_bubble":
			ident, bubbleErr := world.sendGMNPCBubble("ui_shop")
			if bubbleErr != nil {
				gmSession.setAction("NPC 气泡测试失败：" + bubbleErr.Error())
				return bubbleErr
			}
			gmSession.setAction("已发送 NPC 气泡：" + ident)
			log.Printf("%s: GM sent custom %d NPC bubble target=%s", conn.RemoteAddr(), customNPCTalk, ident)
		case "drama_prompt":
			if dramaErr := world.sendDramaPrompt(gmDramaProbe()); dramaErr != nil {
				gmSession.setAction("章节提示卡测试失败：" + dramaErr.Error())
				return dramaErr
			}
			gmSession.setAction("已发送章节提示卡（协议测试）")
			log.Printf("%s: GM sent custom %d drama prompt", conn.RemoteAddr(), customBeginDrama)
		case "red_super_armor":
			if selected == nil {
				gmSession.setAction("红霸体发送失败：角色尚未进入场景")
				return fmt.Errorf("character has not entered a scene")
			}
			if armorErr := grantSceneRedSuperArmor(); armorErr != nil {
				gmSession.setAction("红霸体发送失败：" + armorErr.Error())
				return armorErr
			}
			gmSession.setAction("已发送红霸体（20 秒）")
			log.Printf("%s: GM sent BufferInfo1 red super armor for %s", conn.RemoteAddr(), selected.Name)
		case "action_state_fishing":
			frame, frameErr := playerActionStateFrame("interact268")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("State 钓鱼动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 发送 interact268")
			log.Printf("%s: GM action probe route=State action=interact268", conn.RemoteAddr())
		case "action_state_hsqs05":
			frame, frameErr := playerActionStateFrame("hsqs_05_h")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("State 花神动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 发送 hsqs_05_h")
			log.Printf("%s: GM action probe route=State action=hsqs_05_h", conn.RemoteAddr())
		case "action_state_sgmd":
			frame, frameErr := playerActionStateFrame("hsqs_07")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("神鬼莫敌动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 发送神鬼莫敌 hsqs_07")
			log.Printf("%s: GM action probe route=State skill=CS_yhwq_hsqs07 action=hsqs_07 frames=147", conn.RemoteAddr())
		case "action_native_fishing":
			frame, frameErr := nativeGenericActionFrame("interact268")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("action 钓鱼动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过字符串 action 发送 interact268")
			log.Printf("%s: GM action probe route=string-action action=interact268", conn.RemoteAddr())
		case "action_state_stop":
			frame, frameErr := playerActionStateFrame("stand")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("恢复待机失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 恢复 stand")
			log.Printf("%s: GM action probe route=State action=stand", conn.RemoteAddr())
		default:
			return fmt.Errorf("unsupported GM action %q", command.action)
		}
		return nil
	}
	type inboundFrame struct {
		plain []byte
		err   error
	}
	inbound := make(chan inboundFrame, 1)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			plain, readErr := link.ReadFrame()
			select {
			case inbound <- inboundFrame{plain: plain, err: readErr}:
			case <-done:
				return
			}
			if readErr != nil {
				return
			}
		}
	}()
	facultyTicker := time.NewTicker(time.Second)
	defer facultyTicker.Stop()
	combatMoveTicker := time.NewTicker(150 * time.Millisecond)
	defer combatMoveTicker.Stop()
	for {
		var commandQueue <-chan gmCommand
		if gmSession != nil {
			commandQueue = gmSession.commands
		}
		var plain []byte
		var readErr error
		select {
		case command := <-commandQueue:
			_ = processGMCommand(command)
			continue
		case now := <-combatMoveTicker.C:
			if player == nil || runtime == nil {
				continue
			}
			if moveErr := world.combatMoveTick(player, runtime.activeRole.Location.Position, now); moveErr != nil {
				log.Printf("%s: run NPC chase move tick: %v", conn.RemoteAddr(), moveErr)
				return
			}
			continue
		case now := <-facultyTicker.C:
			if player == nil || selected == nil {
				continue
			}
			slot, advanced, qgCultivateAdvanced := player.facultyTick(now)
			innerPowerPassiveExpired := player.expireInnerPowerPassive(now)
			sitcrossAdvanced := player.sitcrossTick()
			combatBruiseAdvanced := player.combatBruiseTick()
			jingMaiID, jingMaiAdvanced := player.jingMaiCultivateTick(now)
			qingGongAdvanced := player.qingGongTick()
			healOTAdvanced := player.healOverTimeTick(now)
			yufengAdvanced := player.yufengTick(now)
			combatAdvanced := false
			if runtime != nil {
				combat, combatErr := world.combatTick(player, runtime.activeRole.Location.Position, now)
				if combatErr != nil {
					log.Printf("%s: run NPC combat tick: %v", conn.RemoteAddr(), combatErr)
					return
				}
				combatAdvanced = combat.playerStateChanged
				if combat.playerRevived {
					log.Printf("%s: auto-revived player after NPC defeat", conn.RemoteAddr())
				}
			}
			var tickErr error
			if advanced {
				var frame []byte
				if qgCultivateAdvanced {
					frame, tickErr = player.qingGongViewUpdateFrame()
				} else {
					frame, tickErr = player.progressBookUpdateFrame(slot)
				}
				if tickErr == nil {
					tickErr = link.WriteFrame(frame)
				}
			}
			if tickErr == nil && jingMaiAdvanced {
				progress := player.jingMaiProgressFor(jingMaiID)
				var jmFrame []byte
				jmFrame, tickErr = jingMaiViewUpdateFrame(jingMaiID, progress)
				if tickErr == nil {
					tickErr = link.WriteFrame(jmFrame)
				}
			}
			if tickErr == nil && (advanced || sitcrossAdvanced || combatBruiseAdvanced || combatAdvanced || innerPowerPassiveExpired || jingMaiAdvanced || qingGongAdvanced || healOTAdvanced || yufengAdvanced) {
				var state []byte
				state, tickErr = player.vitalUpdate()
				if tickErr == nil {
					tickErr = link.WriteFrame(state)
				}
			}
			if tickErr != nil {
				log.Printf("%s: publish role timer state: %v", conn.RemoteAddr(), tickErr)
				return
			}
			if advanced {
				persistFaculty()
				log.Printf("%s: settled inner cultivation role=%d slot=%d", conn.RemoteAddr(), selected.ID, slot)
			}
			if jingMaiAdvanced {
				persistJingMai()
				progress := player.jingMaiProgressFor(jingMaiID)
				log.Printf("%s: settled jingmai cultivation role=%d id=%s level=%d fill=%d/%d", conn.RemoteAddr(), selected.ID, jingMaiID, progress.level, progress.fill, progress.total)
			}
			if sitcrossAdvanced {
				state := player.actor.Snapshot()
				log.Printf("%s: settled sitcross HP=%d/%d MP=%d/%d", conn.RemoteAddr(), state.HP, state.MaxHP, state.MP, state.MaxMP)
			}
			if combatBruiseAdvanced {
				state := player.actor.Snapshot()
				log.Printf("%s: settled combat bruised blood HP=%d HitHP=%d", conn.RemoteAddr(), state.HP, state.HitHP)
			}
			if healOTAdvanced {
				state := player.actor.Snapshot()
				log.Printf("%s: settled sustained potion HP=%d/%d MP=%d/%d", conn.RemoteAddr(), state.HP, state.MaxHP, state.MP, state.MaxMP)
			}
			if yufengAdvanced {
				state := player.actor.Snapshot()
				log.Printf("%s: yufeng tick mode=%d HP=%d MP=%d/%d", conn.RemoteAddr(), player.yufengMode, state.HP, state.MP, state.MaxMP)
			}
			continue
		case received := <-inbound:
			plain, readErr = received.plain, received.err
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("%s: %v", conn.RemoteAddr(), readErr)
			}
			return
		}
		if len(plain) == 0 {
			log.Printf("%s: empty frame", conn.RemoteAddr())
			continue
		}
		if err := machine.ValidateMessage(plain); err != nil {
			log.Printf("%s: reject client message: %v", conn.RemoteAddr(), err)
			return
		}
		if plain[0] == session.ClientLogin {
			identity, err := auth.ParseLoginAccountIdentity(plain)
			if err != nil {
				log.Printf("session=%d remote=%s reject login structure: %v", machine.ID(), conn.RemoteAddr(), err)
				return
			}
			accountKey = role.AccountKey(identity.String())
			log.Printf("session=%d remote=%s opcode=0x02 decoded_len=%d account=%s", machine.ID(), conn.RemoteAddr(), len(plain), identity.Prefix())
		} else {
			fmt.Printf("remote=%s opcode=0x%02X decoded_len=%d\n%s\n", conn.RemoteAddr(), plain[0], len(plain), hex.Dump(plain))
		}
		if plain[0] == 0x02 {
			_, passwordCT, perr := auth.ParseLoginCredentials(plain)
			if perr != nil {
				log.Printf("%s: parse login credentials: %v", conn.RemoteAddr(), perr)
				return
			}
			var err error
			account, selected, err = store.login(ctx, accountKey, passwordCT)
			if err != nil {
				code := uint32(51002)
				if errors.Is(err, role.ErrNotFound) {
					code = 51001
				}
				if errors.Is(err, role.ErrPasswordMismatch) {
					code = 51002
				}
				payload := make([]byte, 5)
				payload[0] = 0x03
				binary.LittleEndian.PutUint32(payload[1:], code)
				if werr := link.WriteFrame(payload); werr != nil {
					log.Printf("%s: write login error: %v", conn.RemoteAddr(), werr)
				}
				log.Printf("%s: reject login code=%d: %v", conn.RemoteAddr(), code, err)
				return
			}
			exists := selected != nil
			payload := noRoleLoginSuccess(time.Now())
			if exists {
				payload = oneRoleLoginSuccess(time.Now(), selected.Name, selected.Appearance.Values, selected.Location.Scene)
			}
			if err := link.WriteFrame(payload); err != nil {
				log.Printf("%s: write no-role login success: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.LoginCompleted(exists); err != nil {
				log.Printf("%s: complete login transition: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent ServerPlayerRoles(opcode=0x04, role_exists=%t)", conn.RemoteAddr(), exists)
		} else if plain[0] == 0x03 {
			if err := link.WriteFrame(worldInfo(0, "0,book1,0,book2,0,book4,0,book5,0,0")); err != nil {
				log.Printf("%s: write empty world info: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent ServerWorldInfo with four open books", conn.RemoteAddr())
		} else if plain[0] == 0x05 {
			name := createRoleName(plain)
			if name == "" {
				name = "本地角色"
			}
			appearance := createRoleStrings(plain)
			var err error
			selected, err = store.create(ctx, account, name, appearance)
			if err != nil {
				log.Printf("%s: persist created role: %v", conn.RemoteAddr(), err)
				return
			}
			if skillStore != nil {
				if grantErr := skillStore.GrantAllNewRole(selected.ID); grantErr != nil {
					log.Printf("%s: grant all skills to new role=%d: %v", conn.RemoteAddr(), selected.ID, grantErr)
				} else {
					log.Printf("%s: granted full unlock set to new role=%q id=%d", conn.RemoteAddr(), name, selected.ID)
				}
			}
			if err := link.WriteFrame(oneRoleLoginSuccess(time.Now(), name, selected.Appearance.Values, selected.Location.Scene)); err != nil {
				log.Printf("%s: write created role list: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.RoleCreated(); err != nil {
				log.Printf("%s: complete create-role transition: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent ServerPlayerRoles with created role %q", conn.RemoteAddr(), name)
		} else if plain[0] == 0x04 {
			if selected == nil {
				log.Printf("%s: choose role without an account-owned role", conn.RemoteAddr())
				return
			}
			activeRole := selected
			activeRole.Location.Scene = normalizeClientScene(activeRole.Location.Scene)
			if sceneRegistry != nil {
				var stats npcCatalogStats
				sceneNPCs, stats, err = sceneRegistry.catalogFor(activeRole.Location.Scene)
				if err != nil {
					log.Printf("%s: resolve NPC scene catalog: %v", conn.RemoteAddr(), err)
					return
				}
				log.Printf("%s: active NPC scene=%s resource=%s creators=%d resolved=%d unresolved=%d ambiguous=%d", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource, stats.CreatorInstances, stats.Resolved, stats.Unresolved, stats.Ambiguous)
			} else {
				sceneNPCs = staticNPCs
			}
			visual := resolveRoleVisual(activeRole.Appearance.Values)
			player = newPlayerActor(activeRole.Name, starterSilver)
			player.faction = roleFaction(activeRole.Appearance.Values)
			player.sex = visual.sex
			if currencyStore != nil {
				if saved, exists := currencyStore.Load(activeRole.ID); exists {
					saved.toActor(player)
					log.Printf("%s: restored persisted currency role=%d silver=%d gold=%d card=%d ticket=%d", conn.RemoteAddr(), activeRole.ID, saved.Silver, saved.Gold, saved.SilverCard, saved.SilverTicket)
				}
			}
			world.setSceneRegistry(sceneRegistry)
			world.setPlayer(player)
			if shortcutStore != nil {
				if kb, exists := shortcutStore.LoadKeyBind(activeRole.ID); exists {
					player.setKeybind(kb)
					log.Printf("%s: restored persisted keybind role=%d bytes=%d", conn.RemoteAddr(), activeRole.ID, len(kb))
				}
				if saved, exists := shortcutStore.Load(activeRole.ID); exists {
					player.restoreShortcuts(saved)
					log.Printf("%s: restored %d persisted shortcuts role=%d", conn.RemoteAddr(), len(saved), activeRole.ID)
				}
			}
			if bagStore != nil {
				if saved, exists := bagStore.Load(activeRole.ID); exists {
					for i := range saved {
						if saved[i].EquipType != "" {
							continue
						}
						eq, ok := equipCatalog.Lookup(saved[i].ConfigID)
						if !ok {
							continue
						}
						if eq.EquipType != "" {
							saved[i].EquipType = eq.EquipType
						}
						if saved[i].MaxHardiness == 0 {
							saved[i].Hardiness = eq.Hardiness
							saved[i].MaxHardiness = eq.MaxHardiness
						}
					}
					player.restoreBag(saved)
					log.Printf("%s: restored %d bag items role=%d", conn.RemoteAddr(), len(saved), activeRole.ID)
				}
			}
			if equipStore != nil {
				if saved, exists := equipStore.Load(activeRole.ID); exists {
					for i := range saved {
						if saved[i].ArtPack != 0 {
							continue
						}
						if cat, ok := equipCatalog.Lookup(saved[i].ConfigID); ok {
							saved[i].ArtPack = cat.ArtPack
						}
					}
					player.restoreEquip(saved)
					player.applyEquipmentStats(equipCatalog)
					player.syncEquippedResourceCaps()
					restoreWornWeapon := func(worn wornEquipItem) bool {
						cat, ok := equipCatalog.Lookup(worn.ConfigID)
						if !ok {
							return false
						}
						model := equipCatalog.weaponModelName(cat.ArtPack)
						if model == "" {
							return false
						}
						player.weapon = model
						player.setWeaponItemType(uint8(worn.ItemType))
						player.setWeaponMode(weaponModeFor(worn.ItemType))
						player.setWeaponHeldMode(equipCatalog.actionSetForArtPack(cat.ArtPack))
						log.Printf("%s: restored worn weapon model=%s role=%d", conn.RemoteAddr(), model, activeRole.ID)
						return true
					}
					restoredWeapon := false
					for _, worn := range saved {
						if worn.EquipType == "Weapon" && restoreWornWeapon(worn) {
							restoredWeapon = true
							break
						}
					}
					if !restoredWeapon {
						for _, worn := range saved {
							switch worn.EquipType {
							case "Weapon", "ShotWeapon", "InnerWeapon", "FacultyWeapon":
								if restoreWornWeapon(worn) {
									restoredWeapon = true
								}
							}
							if restoredWeapon {
								break
							}
						}
					}
					log.Printf("%s: restored %d worn equip items role=%d", conn.RemoteAddr(), len(saved), activeRole.ID)
				}
			}
			savedFacultyCurNeiGong := ""
			if facultyStore != nil {
				if saved, exists := facultyStore.Load(activeRole.ID); exists {
					savedFacultyCurNeiGong = saved.CurNeiGong
					player.restoreFaculty(saved, time.Now())
					log.Printf("%s: restored persistent faculty role=%d state=%d style=%d name=%s", conn.RemoteAddr(), activeRole.ID, player.progress.facultyState, player.progress.facultyStyle, player.progress.facultyName)
				} else {
					persistFaculty()
					log.Printf("%s: initialized persistent normal faculty role=%d name=%s", conn.RemoteAddr(), activeRole.ID, player.progress.facultyName)
				}
			}
			if jingmaiStore != nil {
				if saved, exists := jingmaiStore.Load(activeRole.ID); exists {
					player.restoreJingMaiProgressFull(saved)
					log.Printf("%s: restored persistent jingmai role=%d active=%d cur=%s", conn.RemoteAddr(), activeRole.ID, len(saved.Active), saved.Cur)
				} else {
					persistJingMai()
					log.Printf("%s: initialized persistent jingmai role=%d", conn.RemoteAddr(), activeRole.ID)
				}
			}
			if skillStore != nil {
				player.applyFullUnlock(skillStore, activeRole.ID)
				if savedFacultyCurNeiGong != "" {
					player.restoreCurNeiGong(savedFacultyCurNeiGong)
				}
				persistFaculty()
				persistJingMai()
				log.Printf("%s: applied full unlock + max-level to role=%d", conn.RemoteAddr(), activeRole.ID)
			}
			if shortcutStore != nil {
				sendCustomizingRestore(link, shortcutStore, activeRole.ID, conn.RemoteAddr().String())
				customizingRestored = true
			}
			propertyTable := visiblePropertyTable(sceneVisiblePropertyFields(npcSchema.Fields))
			if err := link.WriteFrame(propertyTable); err != nil {
				log.Printf("%s: write property table: %v", conn.RemoteAddr(), err)
				return
			}
			recordTable, recordErr := serverRecordTable(qingGongRecordSchemas)
			if recordErr != nil {
				log.Printf("%s: encode record table: %v", conn.RemoteAddr(), recordErr)
				return
			}
			if err := link.WriteFrame(recordTable); err != nil {
				log.Printf("%s: write record table: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent property table (%d entries) and record table (%d entries, QingGongRec)", conn.RemoteAddr(), binary.LittleEndian.Uint16(propertyTable[1:]), len(qingGongRecordSchemas))
			if err := world.begin(activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource); err != nil {
				log.Printf("%s: begin scene generation: %v", conn.RemoteAddr(), err)
				return
			}
			if err := sendEntryScene(link, activeRole.Location.Scene, activeRole.Name); err != nil {
				log.Printf("%s: write entry scene: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent PlayerEntry opcode=0x0B object=%#x scene=%s resource=%s", conn.RemoteAddr(), playerObjectID, activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource)
			legacyBorn02 := activeRole.Location.Scene.Resource == "born02" &&
				activeRole.Location.Position.X > 693.907 && activeRole.Location.Position.X < 693.909 &&
				activeRole.Location.Position.Y > 24.693 && activeRole.Location.Position.Y < 24.695 &&
				activeRole.Location.Position.Z > 404.349 && activeRole.Location.Position.Z < 404.351
			if legacyBorn02 {
				old := activeRole.Location.Position
				activeRole.Location.Position.X = 905.069
				activeRole.Location.Position.Y = 10.810
				activeRole.Location.Position.Z = 196.980
				activeRole.Location.Position.Orient = 1.610
				log.Printf("%s: latest-client LEGACY-BORN02-REMAP initial spawn old=(%.3f,%.3f,%.3f,%.3f) new=(%.3f,%.3f,%.3f,%.3f) persistence=unchanged", conn.RemoteAddr(), old.X, old.Y, old.Z, old.Orient, activeRole.Location.Position.X, activeRole.Location.Position.Y, activeRole.Location.Position.Z, activeRole.Location.Position.Orient)
			}
			log.Printf("%s: latest-client player spawn diagnostic scene=%s resource=%s x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource, activeRole.Location.Position.X, activeRole.Location.Position.Y, activeRole.Location.Position.Z, activeRole.Location.Position.Orient)
			if err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {
				log.Printf("%s: write player spawn chain: %v", conn.RemoteAddr(), err)
				return
			}
			if model := player.mountedWeapon(); model != "" {
				frames, frameErr := player.weaponMountFrames(model)
				if frameErr != nil {
					log.Printf("%s: build post-spawn weapon frame: %v", conn.RemoteAddr(), frameErr)
					return
				}
				for _, frame := range frames {
					if err := link.WriteFrame(frame); err != nil {
						log.Printf("%s: write post-spawn weapon update: %v", conn.RemoteAddr(), err)
						return
					}
				}
				log.Printf("%s: sent post-spawn weapon mount frames model=%q", conn.RemoteAddr(), model)
			}
			if keybind := player.keybindSnapshot(); keybind != "" {
				frame, frameErr := player.applyShortcutKey(keybind)
				if frameErr == nil {
					frameErr = link.WriteFrame(frame)
				}
				if frameErr != nil {
					log.Printf("%s: write post-spawn ShortcutKey update: %v", conn.RemoteAddr(), frameErr)
					return
				}
				log.Printf("%s: sent post-spawn ShortcutKey update bytes=%d", conn.RemoteAddr(), len(keybind))
			}
			attributes, attrErr := player.equipmentStatsUpdate()
			if attrErr != nil {
				log.Printf("%s: build post-spawn role attribute snapshot: %v", conn.RemoteAddr(), attrErr)
				return
			}
			if err := link.WriteFrame(attributes); err != nil {
				log.Printf("%s: write post-spawn role attribute snapshot: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent post-spawn role attribute snapshot %s", conn.RemoteAddr(), facultyLevelLog(player.progress.books))
			log.Printf("%s: sent player 0x0D + 0x1F + authoritative vital 0x10", conn.RemoteAddr())
			clear(activeNPCs)
			nearbyCount, _, err := synchronizeNPCViewport(world, sceneNPCs, activeNPCs, activeRole.Location.Position, npcRadius)
			if err != nil {
				log.Printf("%s: register scene NPC viewport: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.RoleChosen(); err != nil {
				log.Printf("%s: complete choose-role transition: %v", conn.RemoteAddr(), err)
				return
			}
			runtime = &sceneRuntime{conn: link, world: world, machine: machine, registry: sceneRegistry, staticNPCs: staticNPCs, radius: npcRadius, activeNPCs: activeNPCs, player: player, activeRole: activeRole, npcCatalog: sceneNPCs}
			world.transportHandler = func(dest transPathRec) error {
				if runtime == nil || selected == nil || player == nil {
					return fmt.Errorf("transport before scene runtime")
				}
				catalog, err := loadNPCCatalogTransportData()
				if err != nil {
					return err
				}
				from, to := parseFromTo(dest.FromTo)
				targetScene := runtime.activeRole.Location.Scene.Config
				sameScene := true
				if to != "" && from != to {
					if config := catalog.sceneByID[to]; config != "" {
						targetScene = config
						sameScene = false
					}
				}
				position := role.Position{X: dest.TargetX, Z: dest.TargetZ}
				if sameScene {
					err = runtime.TeleportWithinScene(position)
				} else {
					resource, resourceErr := resourceForDoorScene(targetScene)
					if resourceErr != nil {
						return resourceErr
					}
					err = runtime.SwitchScene(sceneDestination{location: role.Location{Scene: role.Scene{Config: targetScene, Resource: resource}, Position: position}})
				}
				if err != nil {
					return fmt.Errorf("NPC transport path %s: %w", dest.ID, err)
				}
				if !sameScene {
					sceneNPCs = runtime.npcCatalog
				}
				pendingLocation = &runtime.activeRole.Location
				if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
					log.Printf("%s: persist NPC transport %s: %v", conn.RemoteAddr(), dest.ID, saveErr)
				} else {
					pendingLocation = nil
				}
				if gmSession != nil {
					action := "NPC 传送（切换场景）"
					if sameScene {
						action = "NPC 传送（场景内）"
					}
					gmSession.setLocation(runtime.activeRole.Location, action)
				}
				log.Printf("%s: NPC transport path %s (%s) completed same_scene=%t target=%s pos=(%.3f,%.3f,%.3f)", conn.RemoteAddr(), dest.ID, dest.TextID, sameScene, runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Position.X, runtime.activeRole.Location.Position.Y, runtime.activeRole.Location.Position.Z)
				return nil
			}
			gmSession = gm.attach(machine.ID(), conn.RemoteAddr().String(), activeRole, roleFaction(activeRole.Appearance.Values), player.silver, player.skillSP())
			log.Printf("%s: registered %d nearby NPC objects from catalog=%d; waiting for ClientReady", conn.RemoteAddr(), nearbyCount, len(sceneNPCs))
		} else if plain[0] == 0x09 {
			log.Printf("%s: C2S 0x09 len=%d hex=% X", conn.RemoteAddr(), len(plain), plain)
			// LATEST-CLIENT COMPATIBILITY A/B ONLY. The exact-current-EXE initial
			// entry already sent player 0x1F as part of sendPlayerSpawn. The exact
			// stable re-entry path, however, sends player 0x1F again when the target
			// explicit ClientReady arrives. The September client now reaches this
			// boundary after Actor2/NPC location replay but renders the player at an
			// invalid terrain position. Replay only the authoritative player 0x1F
			// once on the first explicit initial-entry 0x09 so LIVE A/B can decide
			// whether initial player-location delivery has the same timing contract.
			if runtime != nil && !runtime.awaitingStableReentryReady && !explicitSceneReady {
				position := runtime.activeRole.Location.Position
				if runtime.activeRole.Location.Scene.Resource == "born02" &&
					position.X > 693.907 && position.X < 693.909 &&
					position.Y > 24.693 && position.Y < 24.695 &&
					position.Z > 404.349 && position.Z < 404.351 {
					position.X = 905.069
					position.Y = 10.810
					position.Z = 196.980
					position.Orient = 1.610
					log.Printf("%s: latest-client LEGACY-BORN02-REMAP ClientReady replay new=(%.3f,%.3f,%.3f,%.3f) runtime-persistence=unchanged", conn.RemoteAddr(), position.X, position.Y, position.Z, position.Orient)
				}
				if err := link.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {
					log.Printf("%s: write latest-client initial player location replay: %v", conn.RemoteAddr(), err)
					return
				}
				log.Printf("%s: latest-client player location compatibility replay after explicit ClientReady x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), position.X, position.Y, position.Z, position.Orient)
			}
			if runtime != nil && runtime.awaitingStableReentryReady {
				if err := world.clientReady(); err != nil {
					log.Printf("%s: arm stable re-entry barrier: %v", conn.RemoteAddr(), err)
					return
				}
				if err := sendPlayerLocationAndVitals(link, runtime.player, runtime.activeRole.Location.Position); err != nil {
					log.Printf("%s: write deferred player location/vitals: %v", conn.RemoteAddr(), err)
					return
				}
				log.Printf("%s: target ClientReady; sent deferred player 0x1F + vital state", conn.RemoteAddr())
				if err := machine.Ready(); err != nil {
					log.Printf("%s: complete stable re-entry ready: %v", conn.RemoteAddr(), err)
					return
				}
				if err := world.clientActivity(); err != nil {
					log.Printf("%s: activate stable re-entry objects: %v", conn.RemoteAddr(), err)
					return
				}
				if err := machine.Activity(); err != nil {
					log.Printf("%s: complete stable re-entry activity: %v", conn.RemoteAddr(), err)
					return
				}
				if player != nil {
					levels := player.qingGongLevelsSnapshot()
					if grantErr := grantRoleQingGong(link, qinggongStore, selectedRoleID(selected), player.name, levels); grantErr != nil {
						log.Printf("%s: restore qinggong after stable re-entry: %v", conn.RemoteAddr(), grantErr)
						return
					}
					if shortcutErr := grantShortcutRows(link, player); shortcutErr != nil {
						log.Printf("%s: restore shortcuts after stable re-entry: %v", conn.RemoteAddr(), shortcutErr)
						return
					}
					qingGongGranted = true
					log.Printf("%s: restored QingGongRec/View=46 after 0x0C re-entry", conn.RemoteAddr())
					if progressErr := grantStarterProgress(); progressErr != nil {
						log.Printf("%s: restore inner-power/attributes after 0x0C re-entry: %v", conn.RemoteAddr(), progressErr)
						return
					}
					log.Printf("%s: restored View=43 inner-power and View=40 花无缺七招 after 0x0C re-entry", conn.RemoteAddr())
					if switchErr := enableSkillActionSwitch(link); switchErr != nil {
						log.Printf("%s: restore native skill-action switch after 0x0C re-entry: %v", conn.RemoteAddr(), switchErr)
						return
					}
					log.Printf("%s: enabled native skill-action switch=%d via modern S2C 0x27 message=%d after re-entry", conn.RemoteAddr(), skillActionInputSwitch, serverSwitchControlMessage)
				}
				if npcPatrolCount > 0 {
					world.startPatrol(npcPatrolCount)
				}
				runtime.awaitingStableReentryReady = false
				log.Printf("%s: target scene confirmed by 0x09; activated deferred target NPCs", conn.RemoteAddr())
				continue
			}
			explicitSceneReady = true
			log.Printf("%s: observed explicit stage_main ClientReady (0x09)", conn.RemoteAddr())
			if player != nil && !qingGongGranted {
				levels := player.qingGongLevelsSnapshot()
				if grantErr := grantRoleQingGong(link, qinggongStore, selectedRoleID(selected), player.name, levels); grantErr != nil {
					log.Printf("%s: grant stable starter qinggong: %v", conn.RemoteAddr(), grantErr)
					return
				}
				qingGongGranted = true
				log.Printf("%s: granted QingGongRec/View=46 after explicit ClientReady", conn.RemoteAddr())
			}
			if player != nil {
				if granted, shortcutErr := grantSitcrossShortcut(link, player); shortcutErr != nil {
					log.Printf("%s: grant native sitcross shortcut: %v", conn.RemoteAddr(), shortcutErr)
					return
				} else if granted {
					log.Printf("%s: granted native sitcross shortcut index=%d kind=%s id=%s", conn.RemoteAddr(), sitcrossShortcutIndex, sitcrossShortcutKind, sitcrossShortcutID)
				}
			}
			if player != nil && !neigongGranted {
				if progressErr := grantStarterProgress(); progressErr != nil {
					log.Printf("%s: grant stable inner-power/player attributes: %v", conn.RemoteAddr(), progressErr)
					return
				}
				log.Printf("%s: granted View=43 inner-power and View=40 花无缺七招 after explicit ClientReady", conn.RemoteAddr())
			}
			if player != nil {
				if switchErr := enableSkillActionSwitch(link); switchErr != nil {
					log.Printf("%s: enable native skill-action switch: %v", conn.RemoteAddr(), switchErr)
					return
				}
				log.Printf("%s: enabled native skill-action switch=%d via modern S2C 0x27 message=%d", conn.RemoteAddr(), skillActionInputSwitch, serverSwitchControlMessage)
				if weapon := player.mountedWeapon(); weapon != "" {
					weaponFrames, weaponErr := player.weaponMountFrames(weapon)
					if weaponErr != nil {
						log.Printf("%s: build ClientReady weapon frame: %v", conn.RemoteAddr(), weaponErr)
					} else {
						for _, weaponFrame := range weaponFrames {
							if err := link.WriteFrame(weaponFrame); err != nil {
								log.Printf("%s: write ClientReady weapon update: %v", conn.RemoteAddr(), err)
								return
							}
						}
						log.Printf("%s: re-mounted weapon model=%q after ClientReady", conn.RemoteAddr(), weapon)
					}
				}
			}
			if machine.Phase() == session.PhaseInWorld {
				log.Printf("%s: ignored duplicate ClientReady in active scene", conn.RemoteAddr())
				continue
			}
			if err := world.clientReady(); err != nil {
				log.Printf("%s: reconcile ready viewport: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.Ready(); err != nil {
				log.Printf("%s: complete client-ready transition: %v", conn.RemoteAddr(), err)
				return
			}
			if player != nil {
				vitals, syncErr := player.vitalUpdate()
				if syncErr != nil {
					log.Printf("%s: encode player ready vital sync: %v", conn.RemoteAddr(), syncErr)
					return
				}
				if writeErr := link.WriteFrame(vitals); writeErr != nil {
					log.Printf("%s: write player ready vital sync: %v", conn.RemoteAddr(), writeErr)
					return
				}
			}
		} else if plain[0] == 0x0A {
			phase := machine.Phase()
			if phase == session.PhaseRoleCreation || phase == session.PhaseRoleSelection {
				log.Printf("%s: ignored role-screen 0x0A CustomSend phase=%s", conn.RemoteAddr(), phase)
				continue
			}
			if runtime != nil && runtime.awaitingStableReentryReady {
				log.Printf("%s: ignored early 0x0A while target scene is loading", conn.RemoteAddr())
				continue
			}
			if phase == session.PhaseEnteringWorld {
				if err := world.clientReady(); err != nil {
					log.Printf("%s: arm compatibility ready barrier: %v", conn.RemoteAddr(), err)
					return
				}
				if err := machine.Ready(); err != nil {
					log.Printf("%s: complete compatibility ready transition: %v", conn.RemoteAddr(), err)
					return
				}
				if player != nil {
					vitals, syncErr := player.vitalUpdate()
					if syncErr != nil {
						log.Printf("%s: encode compatibility ready vital sync: %v", conn.RemoteAddr(), syncErr)
						return
					}
					if writeErr := link.WriteFrame(vitals); writeErr != nil {
						log.Printf("%s: write compatibility ready vital sync: %v", conn.RemoteAddr(), writeErr)
						return
					}
					if switchErr := enableSkillActionSwitch(link); switchErr != nil {
						log.Printf("%s: enable compatibility native skill-action switch: %v", conn.RemoteAddr(), switchErr)
						return
					}
					log.Printf("%s: enabled native skill-action switch=%d via compatibility ClientReady", conn.RemoteAddr(), skillActionInputSwitch)
					if weapon := player.mountedWeapon(); weapon != "" {
						weaponFrames, weaponErr := player.weaponMountFrames(weapon)
						if weaponErr != nil {
							log.Printf("%s: build compatibility ready weapon frame: %v", conn.RemoteAddr(), weaponErr)
						} else {
							for _, weaponFrame := range weaponFrames {
								if writeErr := link.WriteFrame(weaponFrame); writeErr != nil {
									log.Printf("%s: write compatibility ready weapon update: %v", conn.RemoteAddr(), writeErr)
									return
								}
							}
						}
					}
				}
				log.Printf("%s: promoted observed 0x0A to compatibility ClientReady", conn.RemoteAddr())
			}
			switchAfterActivity := switchDestination != nil && runtime != nil && explicitSceneReady && (machine.Phase() == session.PhaseAwaitingActivity || machine.Phase() == session.PhaseInWorld)
			if !switchAfterActivity {
				if err := world.clientActivity(); err != nil {
					log.Printf("%s: activate ready scene objects: %v", conn.RemoteAddr(), err)
					return
				}
				if npcPatrolCount > 0 {
					world.startPatrol(npcPatrolCount)
				}
			}
			if err := machine.Activity(); err != nil {
				log.Printf("%s: complete client-activity transition: %v", conn.RemoteAddr(), err)
				return
			}
			if demoDamage > 0 && player != nil && player.actor.ApplyDamage(demoDamage) {
				vitals, syncErr := player.vitalUpdate()
				if syncErr != nil {
					log.Printf("%s: encode demo damage update: %v", conn.RemoteAddr(), syncErr)
					return
				}
				if writeErr := link.WriteFrame(vitals); writeErr != nil {
					log.Printf("%s: write demo damage update: %v", conn.RemoteAddr(), writeErr)
					return
				}
				log.Printf("%s: applied authoritative demo damage=%d", conn.RemoteAddr(), demoDamage)
				demoDamage = 0
			}
			if switchAfterActivity {
				destination := *switchDestination
				switchDestination = nil
				neigongGranted = false
				if err := runtime.SwitchScene(destination); err != nil {
					log.Printf("%s: live scene switch: %v", conn.RemoteAddr(), err)
					return
				}
				sceneNPCs = runtime.npcCatalog
				pendingLocation = &runtime.activeRole.Location
				if err := store.saveLocation(ctx, selected, runtime.activeRole.Location); err != nil {
					log.Printf("%s: persist switched scene: %v", conn.RemoteAddr(), err)
				} else {
					pendingLocation = nil
				}
				log.Printf("%s: sent live 0x0C re-entry to scene=%s resource=%s; waiting for ClientReady", conn.RemoteAddr(), runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource)
			}
			if x, y, z, orient, ok := parseMotionPosition(plain); ok {
				log.Printf("%s: latest-client decoded C2S position x=%.3f y=%.3f z=%.3f orient=%.3f opcode=0x%02X len=%d", conn.RemoteAddr(), x, y, z, orient, plain[0], len(plain))
				if selected == nil {
					log.Printf("%s: position update without selected role", conn.RemoteAddr())
					return
				}
				if player != nil {
					player.updatePosition(x, y, z)
					now := time.Now()
					player.recordYufengMotion(x, z, now)
				}
				location := selected.Location
				location.Position = role.Position{X: x, Y: y, Z: z, Orient: orient}
				selected.Location = location
				nearbyCount, changed, syncErr := synchronizeNPCViewport(world, sceneNPCs, activeNPCs, location.Position, npcRadius)
				if syncErr != nil {
					log.Printf("%s: synchronize moving NPC viewport: %v", conn.RemoteAddr(), syncErr)
					return
				}
				if changed {
					if reconcileErr := world.reconcileViewport(); reconcileErr != nil {
						log.Printf("%s: reconcile moving NPC viewport: %v", conn.RemoteAddr(), reconcileErr)
						return
					}
					log.Printf("%s: NPC viewport moved center=(%.2f,%.2f) active=%d", conn.RemoteAddr(), x, z, nearbyCount)
				}
				now := time.Now()
				if now.Sub(lastRangeHintAt) >= 100*time.Millisecond {
					refreshSkillRangeHints(link, world, location.Position)
					lastRangeHintAt = now
				}
				if portal, enteredPortal := world.portalAt(location.Position, time.Now()); enteredPortal {
					if runtime == nil {
						log.Printf("%s: ignore proximity portal %s before scene runtime", conn.RemoteAddr(), portal.source)
						continue
					}
					var portalErr error
					if portal.sameScene {
						portalErr = runtime.TeleportWithinScene(portal.position)
					} else {
						portalErr = runtime.SwitchScene(portal.destination)
					}
					if portalErr != nil {
						log.Printf("%s: proximity portal %s failed: %v", conn.RemoteAddr(), portal.source, portalErr)
						continue
					}
					if !portal.sameScene {
						sceneNPCs = runtime.npcCatalog
					}
					pendingLocation = &runtime.activeRole.Location
					if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
						log.Printf("%s: persist proximity portal %s: %v", conn.RemoteAddr(), portal.source, saveErr)
					} else {
						pendingLocation = nil
					}
					if gmSession != nil {
						action := "传送门范围内移动"
						if !portal.sameScene {
							action = "传送门范围内切图"
						}
						gmSession.setLocation(runtime.activeRole.Location, action)
					}
					log.Printf("%s: entered portal %s trigger=(%.3f,%.3f) radius=%.1f same_scene=%t scene=%s resource=%s", conn.RemoteAddr(), portal.source, portal.trigger.X, portal.trigger.Z, portal.triggerRadius, portal.sameScene, runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource)
					continue
				}
				pendingLocation = &location
				if time.Since(lastPositionSave) >= 2*time.Second {
					if err := store.saveLocation(ctx, selected, location); err != nil {
						log.Printf("%s: persist movement: %v", conn.RemoteAddr(), err)
					} else {
						lastPositionSave = time.Now()
						pendingLocation = nil
					}
				}
			}
			if custom, ok, _ := parseClientActivityCustomMessage(plain); ok && len(custom.Values) > 0 && custom.Values[0].Type == 2 {
				id := custom.Values[0].Int32
				allowed := id == 200 || id == 201 || id == 240 || id == 211 || id == 215 || id == 216 || id == 217 || id == 218 || id == 54 || id == 1006 || id == 431 || id == 195 || id == 30 || id == 31 || id == 34 || id == 36 || id == 107 || id == 161 || id == 150 || id == 70
				if allowed {
					arguments := make([]string, 0, len(custom.Values)-1)
					for _, value := range custom.Values[1:] {
						arguments = append(arguments, value.String())
					}
					log.Printf("%s: client activity CustomSend opcode=0x0A msg_id=%d args=[%s]", conn.RemoteAddr(), id, strings.Join(arguments, ", "))
					switch id {
					case 211:
						if _, handleErr := handleSkillCustom(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity skill: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 200, 201:
						if _, handleErr := handleShortcutCustom(link, player, custom, conn.RemoteAddr().String(), shortcutStore, selectedRoleID(selected)); handleErr != nil {
							log.Printf("%s: handle activity shortcut: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 240:
						if _, handleErr := handleSitcrossCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity sitcross: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 216:
						if _, handleErr := handleQingGongCustom(link, player, qingGongCatalog, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity qinggong: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 217:
						if _, handleErr := handleActiveQingGongCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle active qinggong: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 218:
						if _, handleErr := handleActiveParryCustom(player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle active parry: %v", conn.RemoteAddr(), handleErr)
						}
					case 54:
						if _, handleErr := handleBlockStateCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle block state: %v", conn.RemoteAddr(), handleErr)
						}
					case 431:
						if _, handleErr := handleSelectTargetCustom(world, currentCasterPosition(runtime), custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle select target: %v", conn.RemoteAddr(), handleErr)
						}
					case 195:
						if _, handleErr := handleKeyBindCustom(shortcutStore, selectedRoleID(selected), player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle keybind: %v", conn.RemoteAddr(), handleErr)
						}
					case 107:
						accepted, payload, handleErr := handleCustomizingSave(shortcutStore, selectedRoleID(selected), custom, conn.RemoteAddr().String())
						if handleErr != nil {
							log.Printf("%s: handle customizing: %v", conn.RemoteAddr(), handleErr)
						} else if accepted && payload != "" {
							frame, frameErr := serverCustomIntMessage(705, customString(payload))
							if frameErr == nil {
								if writeErr := link.WriteFrame(frame); writeErr != nil {
									log.Printf("%s: write customizing reply: %v", conn.RemoteAddr(), writeErr)
								} else {
									log.Printf("%s: sent customizing reply bytes=%d", conn.RemoteAddr(), len(payload))
								}
							}
						}
					case 30:
						if _, handleErr := handleMoveItemCustom(link, player, itemCatalog, equipCatalog, bagStore, equipStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity equip/bag move: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 31:
						if _, handleErr := handleUseItemCustom(link, player, itemCatalog, equipCatalog, dropTable, bagStore, equipStore, fwzCardStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity equip/use: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 34:
						if _, handleErr := handleDeleteItemCustom(link, player, bagStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity bag delete: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 36:
						if _, handleErr := handleArrangeItemCustom(link, player, bagStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity bag arrange: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 70:
						if _, handleErr := handleShopBuyCustom(link, player, itemCatalog, bagStore, currencyStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity shop buy: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 215, 1006:
						if _, handleErr := handlePlayerProgressCustom(link, player, custom); handleErr != nil {
							log.Printf("%s: handle activity inner-power msg=%d: %v", conn.RemoteAddr(), id, handleErr)
							return
						}
						persistFaculty()
					case 161:
						if _, handleErr := handleJingMaiActiveCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity jingmai active: %v", conn.RemoteAddr(), handleErr)
							return
						}
						persistJingMai()
					case 150:
						if _, handleErr := handleJingMaiCultivateCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle activity jingmai cultivate: %v", conn.RemoteAddr(), handleErr)
							return
						}
						persistJingMai()
					case 972:
						if _, handleErr := handleFreshManChoice(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle fresh-man choice: %v", conn.RemoteAddr(), handleErr)
							return
						}
					case 180:
						if _, handleErr := handleFwzCardCustom(link, player, fwzCardStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
							log.Printf("%s: handle fwz card: %v", conn.RemoteAddr(), handleErr)
							return
						}
					}
				}
			}
			if custom, ok, _ := parseClientActivityCustomMessage(plain); ok && len(custom.Values) > 0 && custom.Values[0].Type == 2 {
				id := custom.Values[0].Int32
				if id != 924 && id != 15 && id != 9 {
					arguments := make([]string, 0, len(custom.Values)-1)
					for _, value := range custom.Values[1:] {
						arguments = append(arguments, value.String())
					}
					log.Printf("%s: unhandled client activity CustomSend opcode=0x0A msg_id=%d args=[%s]", conn.RemoteAddr(), id, strings.Join(arguments, ", "))
				}
			}
		} else if plain[0] == 0x07 {
			request, ok := parseClientObjectRequest(plain)
			if !ok {
				log.Printf("%s: reject malformed C2S 0x07 object request len=%d", conn.RemoteAddr(), len(plain))
				continue
			}
			if request.FuncID != 0 {
				if err := world.selectNPCMenu(request); err != nil {
					log.Printf("%s: NPC menu selection id=%d func=%d: %v", conn.RemoteAddr(), request.ObjectID, request.FuncID, err)
				}
				continue
			}
			portal, isPortal, portalErr := world.portalRoute(request)
			if portalErr != nil {
				log.Printf("%s: resolve portal id=%d owner=%d: %v", conn.RemoteAddr(), request.ObjectID, request.OwnerID, portalErr)
				continue
			}
			if isPortal {
				if runtime == nil {
					log.Printf("%s: ignore portal id=%d before scene runtime", conn.RemoteAddr(), request.ObjectID)
					continue
				}
				var routeErr error
				if portal.sameScene {
					routeErr = runtime.TeleportWithinScene(portal.position)
				} else {
					routeErr = runtime.SwitchScene(portal.destination)
				}
				if routeErr != nil {
					log.Printf("%s: portal %s id=%d failed: %v", conn.RemoteAddr(), portal.source, request.ObjectID, routeErr)
					continue
				}
				if !portal.sameScene {
					sceneNPCs = runtime.npcCatalog
				}
				pendingLocation = &runtime.activeRole.Location
				if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
					log.Printf("%s: persist portal %s: %v", conn.RemoteAddr(), portal.source, saveErr)
				} else {
					pendingLocation = nil
				}
				if gmSession != nil {
					action := "传送门场景内移动"
					if !portal.sameScene {
						action = "传送门切换场景"
					}
					gmSession.setLocation(runtime.activeRole.Location, action)
				}
				log.Printf("%s: portal %s id=%d completed same_scene=%t scene=%s resource=%s position=(%.3f,%.3f,%.3f)", conn.RemoteAddr(), portal.source, request.ObjectID, portal.sameScene, runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource, runtime.activeRole.Location.Position.X, runtime.activeRole.Location.Position.Y, runtime.activeRole.Location.Position.Z)
				continue
			}
			if _, err := world.objectRequest(request); err != nil {
				log.Printf("%s: object request id=%d owner=%d: %v", conn.RemoteAddr(), request.ObjectID, request.OwnerID, err)
			}
		} else if plain[0] == 0x1E || plain[0] == 0x27 {
			custom, err := parseClientCustomMessage(plain)
			if err != nil {
				log.Printf("%s: reject malformed C2S custom opcode=0x%02X: %v", conn.RemoteAddr(), plain[0], err)
				continue
			}
			if custom.Values[0].Type != 2 {
				log.Printf("%s: C2S custom first value must be int message ID, got %s", conn.RemoteAddr(), custom.Values[0])
				continue
			}
			arguments := make([]string, 0, len(custom.Values)-1)
			for _, value := range custom.Values[1:] {
				arguments = append(arguments, value.String())
			}
			log.Printf("%s: client CustomSend opcode=0x%02X msg_id=%d args=[%s]", conn.RemoteAddr(), custom.Opcode, custom.Values[0].Int32, strings.Join(arguments, ", "))
			switch custom.Values[0].Int32 {
			case 30:
				if _, handleErr := handleMoveItemCustom(link, player, itemCatalog, equipCatalog, bagStore, equipStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle equip/bag move: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 31:
				if _, handleErr := handleUseItemCustom(link, player, itemCatalog, equipCatalog, dropTable, bagStore, equipStore, fwzCardStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle equip/use: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 34:
				if _, handleErr := handleDeleteItemCustom(link, player, bagStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle bag delete: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 36:
				if _, handleErr := handleArrangeItemCustom(link, player, bagStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle bag arrange: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 54:
				if _, handleErr := handleBlockStateCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle block state: %v", conn.RemoteAddr(), handleErr)
				}
			case 64, 69, 79:
				if _, handleErr := handleShopExchangeContract(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle current shop exchange contract: %v", conn.RemoteAddr(), handleErr)
				}
			case 70:
				if _, handleErr := handleShopBuyCustom(link, player, itemCatalog, bagStore, currencyStore, selectedRoleID(selected), custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle shop buy: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 150:
				if _, handleErr := handleJingMaiCultivateCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle jingmai cultivate: %v", conn.RemoteAddr(), handleErr)
					return
				}
				persistJingMai()
			case 161:
				if _, handleErr := handleJingMaiActiveCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle jingmai active: %v", conn.RemoteAddr(), handleErr)
					return
				}
				persistJingMai()
			case 195:
				if _, handleErr := handleKeyBindCustom(shortcutStore, selectedRoleID(selected), player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle keybind: %v", conn.RemoteAddr(), handleErr)
				}
			case 200, 201:
				if _, handleErr := handleShortcutCustom(link, player, custom, conn.RemoteAddr().String(), shortcutStore, selectedRoleID(selected)); handleErr != nil {
					log.Printf("%s: handle shortcut: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 211:
				if _, handleErr := handleSkillCustom(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle skill: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 215, 1006:
				if _, handleErr := handlePlayerProgressCustom(link, player, custom); handleErr != nil {
					log.Printf("%s: handle inner-power msg=%d: %v", conn.RemoteAddr(), custom.Values[0].Int32, handleErr)
					return
				}
				persistFaculty()
			case 216:
				if _, handleErr := handleQingGongCustom(link, player, qingGongCatalog, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle qinggong: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 217:
				if _, handleErr := handleActiveQingGongCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle active qinggong: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 218:
				if _, handleErr := handleActiveParryCustom(player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle active parry: %v", conn.RemoteAddr(), handleErr)
				}
			case 240:
				if _, handleErr := handleSitcrossCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle sitcross: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 431:
				if _, handleErr := handleSelectTargetCustom(world, currentCasterPosition(runtime), custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle select target: %v", conn.RemoteAddr(), handleErr)
				}
			case 972:
				if _, handleErr := handleFreshManChoice(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle fresh-man choice: %v", conn.RemoteAddr(), handleErr)
					return
				}
			}
		} else {
			log.Printf("%s: unhandled client opcode=0x%02X decoded_len=%d first16=% X", conn.RemoteAddr(), plain[0], len(plain), plain[:min(16, len(plain))])
		}
	}
}
func handleQingGongCustom(link sceneMessageConnection, player *playerActor, catalog *qinggong.Catalog, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 216 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject qinggong before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 3 || custom.Values[1].Type != 6 || custom.Values[1].Text == "" || custom.Values[2].Type != 2 || (custom.Values[2].Int32 != 0 && custom.Values[2].Int32 != 1) {
		log.Printf("%s: reject malformed USE_QINGGONG values=%v", remote, custom.Values)
		return true, nil
	}
	definition, exists := catalog.Lookup(custom.Values[1].Text)
	if !exists {
		log.Printf("%s: reject unknown qinggong id=%q", remote, custom.Values[1].Text)
		return true, nil
	}
	if custom.Values[2].Int32 == 0 {
		if err := endQingGongBuff(link, player, definition); err != nil {
			return true, err
		}
		log.Printf("%s: qinggong end id=%s", remote, definition.ID)
		return true, nil
	}
	now := time.Now()
	if spec, hasBuff := qinggongBuffFor(definition); hasBuff && spec.clearOnEnd && player.hasActiveBuff(spec.slot, spec.staticData, now) {
		log.Printf("%s: ignore duplicate qinggong start id=%s", remote, definition.ID)
		return true, nil
	}
	cost, accepted := player.actor.UseQingGong(definition.Consume, now)
	if !accepted {
		state := player.actor.Snapshot()
		log.Printf("%s: reject qinggong id=%s cost=%d point=%d", remote, definition.ID, cost, state.QingGongPoint)
		return true, nil
	}
	alreadyActive, expires, err := activateQingGongBuff(player, definition, now)
	if err != nil {
		return true, err
	}
	update, err := player.vitalUpdate()
	if err != nil {
		return true, fmt.Errorf("encode qinggong update: %w", err)
	}
	if err = link.WriteFrame(update); err != nil {
		return true, fmt.Errorf("write qinggong update: %w", err)
	}
	if !alreadyActive && !expires.IsZero() {
		spec, _ := qinggongBuffFor(definition)
		schedulePlayerBuffExpiry(link, player, spec.slot, spec.staticData, expires)
	}
	state := player.actor.Snapshot()
	log.Printf("%s: accepted qinggong id=%s type=%d cost=%d point=%d/%d", remote, definition.ID, definition.Type, cost, state.QingGongPoint, state.MaxQingGongPoint+state.MaxQingGongPointAdd)
	return true, nil
}
func handleActiveQingGongCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 217 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject active qinggong before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 2 || custom.Values[1].Type != 6 || custom.Values[1].Text == "" {
		log.Printf("%s: reject malformed ACTIVE_QINGGONG values=%v", remote, custom.Values)
		return true, nil
	}
	changed, err := activateQingGongRecord(link, player, custom.Values[1].Text)
	if err != nil {
		return true, err
	}
	if changed {
		log.Printf("%s: activated qinggong id=%s via ActiveQGSkillRec", remote, custom.Values[1].Text)
	} else {
		log.Printf("%s: active qinggong already present id=%s", remote, custom.Values[1].Text)
	}
	return true, nil
}

type npcSpawn struct {
	resolved        clientdata.ResolvedNPC
	x, y, z, orient float32
}

func synchronizeNPCViewport(world *sceneLifecycle, catalog []npcSpawn, active map[int]struct{}, position role.Position, radius float32) (int, bool, error) {
	radiusSquared := radius * radius
	desired := make(map[int]struct{})
	for i := range catalog {
		if !npcVisibleInLocalViewport(catalog[i]) {
			continue
		}
		dx := catalog[i].x - position.X
		dz := catalog[i].z - position.Z
		if dx*dx+dz*dz <= radiusSquared {
			desired[i] = struct{}{}
		}
	}
	changed := false
	for i := range active {
		if _, keep := desired[i]; keep {
			continue
		}
		if err := world.unregister(uint32(i + 2)); err != nil {
			return len(active), changed, err
		}
		delete(active, i)
		changed = true
	}
	for i := range desired {
		if _, exists := active[i]; exists {
			continue
		}
		npc := catalog[i].clone()
		npc.synchronizeTransform()
		if err := world.registerNPC(uint32(i+2), npc); err != nil {
			return len(active), changed, err
		}
		active[i] = struct{}{}
		changed = true
	}
	return len(active), changed, nil
}
func countNPCsWithin(catalog []npcSpawn, position role.Position, radius float32) int {
	radiusSquared := radius * radius
	count := 0
	for i := range catalog {
		if !npcVisibleInLocalViewport(catalog[i]) {
			continue
		}
		dx := catalog[i].x - position.X
		dz := catalog[i].z - position.Z
		if dx*dx+dz*dz <= radiusSquared {
			count++
		}
	}
	return count
}
func npcVisibleInLocalViewport(npc npcSpawn) bool {
	return !strings.EqualFold(strings.TrimSpace(npc.resolved.ScriptClass), "EventNpc")
}
func npcAddObject(objectID uint32, npc npcSpawn, probeIndex int) ([]byte, error) {
	msg := make([]byte, 0x3f, 256)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], 1)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(npc.x))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(npc.y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(npc.z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(npc.orient))
	_ = probeIndex
	properties, err := npc.resolved.OrderedProperties(clientdata.VisibleNPCModernV1())
	if err != nil {
		return nil, err
	}
	for _, property := range properties {
		msg = appendNPCProperty(msg, property)
	}
	binary.LittleEndian.PutUint16(msg[0x3d:], uint16(len(properties)))
	return msg, nil
}
func sceneObjectByteProperty(objectID, ownerID uint32, propertyIndex uint16, value byte) []byte {
	msg := make([]byte, 12, 15)
	msg[0] = 0x10
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	msg = binary.LittleEndian.AppendUint16(msg, propertyIndex)
	msg = append(msg, value)
	return msg
}
func sceneObjectTransformProperties(objectID uint32, ownerID uint32, transform world.Transform) []byte {
	msg := make([]byte, 12, 36)
	binary.LittleEndian.PutUint16(msg[0:], 0x10)
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 4)
	for _, property := range []struct {
		index uint16
		value float32
	}{{9, transform.X}, {10, transform.Y}, {11, transform.Z}, {12, transform.Orient}} {
		msg = binary.LittleEndian.AppendUint16(msg, property.index)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(property.value))
	}
	return msg
}
func serverLocation(objectID uint32, ownerID uint32, transform world.Transform) []byte {
	msg := make([]byte, 25)
	msg[0] = 0x1F
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], ownerID)
	binary.LittleEndian.PutUint32(msg[9:], math.Float32bits(transform.X))
	binary.LittleEndian.PutUint32(msg[13:], math.Float32bits(transform.Y))
	binary.LittleEndian.PutUint32(msg[17:], math.Float32bits(transform.Z))
	binary.LittleEndian.PutUint32(msg[21:], math.Float32bits(transform.Orient))
	return msg
}
func sceneObjectStringProperty(objectID, ownerID uint32, propertyIndex uint16, value string) []byte {
	msg := make([]byte, 12, 18+len(value))
	msg[0] = 0x10
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	return appendStringProperty(msg, propertyIndex, value)
}
func sceneObjectObjectProperty(objectID uint32, ownerID uint32, propertyIndex uint32, targetID uint32, targetOwnerID uint32) []byte {
	msg := make([]byte, 22)
	binary.LittleEndian.PutUint16(msg[0:], 0x10)
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	binary.LittleEndian.PutUint16(msg[12:], uint16(propertyIndex))
	binary.LittleEndian.PutUint32(msg[14:], targetID)
	binary.LittleEndian.PutUint32(msg[18:], targetOwnerID)
	return msg
}
func visiblePropertyTable(properties []clientdata.FieldSpec) []byte {
	msg := make([]byte, 3, 3+len(properties)*12)
	msg[0] = 0x09
	binary.LittleEndian.PutUint16(msg[1:], uint16(len(properties)))
	for _, property := range properties {
		msg = append(msg, property.Name...)
		msg = append(msg, 0, byte(property.Type))
	}
	return msg
}
func sceneVisiblePropertyFields(npcFields []clientdata.FieldSpec) []clientdata.FieldSpec {
	fields := append([]clientdata.FieldSpec(nil), npcFields...)
	fields = append(fields, clientdata.FieldSpec{Index: 99, Name: "MaxVisuals", Type: clientdata.WireInt32})
	fields = append(fields, clientdata.FieldSpec{Index: 100, Name: "LastObject", Type: clientdata.WireObject})
	fields = append(fields, clientdata.FieldSpec{Index: 101, Name: "ShopID", Type: clientdata.WireString}, clientdata.FieldSpec{Index: 102, Name: "PageNum", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 103, Name: "PageCount", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 104, Name: "ShopType", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 105, Name: "ConfigID", Type: clientdata.WireString}, clientdata.FieldSpec{Index: 106, Name: "Amount", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 107, Name: "SellPrice0", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 108, Name: "SellPrice1", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 109, Name: "SellPrice2", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 110, Name: "MaxAmount", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 111, Name: "CapitalType1", Type: clientdata.WireInt64}, clientdata.FieldSpec{Index: 112, Name: "HitHP", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 113, Name: "HitHPRatio", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 114, Name: "QingGongPoint", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 115, Name: "MaxQingGongPoint", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 116, Name: "MaxQingGongPointAdd", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: 117, Name: "RunSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 118, Name: "JumpSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 119, Name: "DriftSpeed", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 120, Name: "DriftSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 121, Name: "DriftJumpSpeed", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 122, Name: "DriftJumpSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 123, Name: "SndJumpSpeed", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 124, Name: "SndJumpSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 125, Name: "ThdJumpSpeed", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 126, Name: "ThdJumpSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 127, Name: "AirRushSpeed", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 128, Name: "AirRushSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 129, Name: "AirRushDist", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 130, Name: "AirRushDistAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 131, Name: "LandRushSpeed", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 132, Name: "LandRushSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 133, Name: "LandRushDist", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 134, Name: "LandRushDistAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 135, Name: "ClimbSpeed", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 136, Name: "ClimbSpeedAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 137, Name: "GRushRange", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 138, Name: "GRushRangeAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 139, Name: "ARushRange", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 140, Name: "ARushRangeAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 141, Name: "WRushRange", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 142, Name: "WRushRangeAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 143, Name: "BufferListStr", Type: clientdata.WireString})
	for slot := uint16(1); slot <= bufferInfoSlotCount; slot++ {
		fields = append(fields, clientdata.FieldSpec{Index: bufferInfoPropertyIndex(slot), Name: bufferInfoPropertyName(slot), Type: clientdata.WireString})
	}
	fields = append(fields, clientdata.FieldSpec{Index: 168, Name: "Gravity", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 169, Name: "GravityAdd", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: 170, Name: "DropHeightPub", Type: clientdata.WireFloat32})
	fields = append(fields, playerProgressFields()...)
	fields = append(fields, clientdata.FieldSpec{Index: propNeigongPKStatus, Name: "NeigongPKStatus", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propDead, Name: "Dead", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propCantUseSkill, Name: "CantUseSkill", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propCurSkillID, Name: "CurSkillID", Type: clientdata.WireString}, clientdata.FieldSpec{Index: propPauseTime, Name: "PauseTime", Type: clientdata.WireFloat32}, clientdata.FieldSpec{Index: propHPHeartSpeed, Name: "HPHeartSpeed", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propHPHeartSpeedAdd, Name: "HPHeartSpeedAdd", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propMPHeartSpeed, Name: "MPHeartSpeed", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propMPHeartSpeedAdd, Name: "MPHeartSpeedAdd", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propHPUpSpeed, Name: "HPUpSpeed", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propHPUpSpeedAdd, Name: "HPUpSpeedAdd", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propMPUpSpeed, Name: "MPUpSpeed", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propMPUpSpeedAdd, Name: "MPUpSpeedAdd", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propCurSkillEffectID, Name: "CurSkillEffectID", Type: clientdata.WireString}, clientdata.FieldSpec{Index: propCurSkillLevel, Name: "CurSkillLevel", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propCurSkillTarget, Name: "CurSkillTarget", Type: clientdata.WireObject}, clientdata.FieldSpec{Index: propModifySkillLockTime, Name: "ModifySkillLockTime", Type: clientdata.WireInt32}, clientdata.FieldSpec{Index: propSkillCanUse, Name: "CanUse", Type: clientdata.WireByte}, clientdata.FieldSpec{Index: propForce, Name: "Force", Type: clientdata.WireString}, clientdata.FieldSpec{Index: propNewSchool, Name: "NewSchool", Type: clientdata.WireString})

	// Latest-client canonical bag rows require these object properties by name.
	// Append only: preserve all original negotiated ordinals 0..227.
	haveBagViewID := false
	for _, field := range fields {
		if field.Name == "ViewID" && field.Type == clientdata.WireInt32 {
			haveBagViewID = true
		}
	}
	if !haveBagViewID {
		fields = append(fields, clientdata.FieldSpec{Index: 0x0763, Name: "ViewID", Type: clientdata.WireInt32})
	}

	// Exact current FxGameLogic CapitalModule.GetCapital resolves the authored
	// capital property name and reads it through the player object's 64-bit
	// getter (vtable +0x78). The exact current share.package/rule/capital.ini
	// (SHA256 d954c54cf3c059e479aae1a5c234ae04ecd46d71be6ebe626d1a789487743b2d)
	// defines TypeCount=5 with CapitalType0..CapitalType4. CapitalType1 already
	// occupies canonical ordinal 111. Preserve canonical bag ViewID at ordinal
	// 228 and the previously appended CapitalType2/4 ordinals 229/230; append
	// the newly resource-proven CapitalType0/3 without moving existing ordinals.
	fields = append(fields,
		clientdata.FieldSpec{Index: 0x01A2, Name: "CapitalType2", Type: clientdata.WireInt64},
		clientdata.FieldSpec{Index: 0x01A4, Name: "CapitalType4", Type: clientdata.WireInt64},
		clientdata.FieldSpec{Index: 0x01A0, Name: "CapitalType0", Type: clientdata.WireInt64},
		clientdata.FieldSpec{Index: 0x01A3, Name: "CapitalType3", Type: clientdata.WireInt64},
		// Exact current FxGameLogic queries ExchangeData through its int32
		// property getter (the result is tested in EAX and passed as int32).
		// Append only so every previously negotiated ordinal remains stable.
		clientdata.FieldSpec{Index: 0x0772, Name: "ExchangeData", Type: clientdata.WireInt32},
	)
	return fields
}
func appendStringProperty(msg []byte, index uint16, value string) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, index)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(value)+1))
	msg = append(msg, value...)
	return append(msg, 0)
}
func appendObjectProperty(msg []byte, index uint16, objectID, ownerID uint32) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, index)
	return binary.LittleEndian.AppendUint64(msg, uint64(objectID)|uint64(ownerID)<<32)
}
func appendWideStringProperty(msg []byte, index uint16, value string) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, index)
	units := utf16.Encode([]rune(value))
	msg = binary.LittleEndian.AppendUint32(msg, uint32((len(units)+1)*2))
	for _, unit := range units {
		msg = binary.LittleEndian.AppendUint16(msg, unit)
	}
	return binary.LittleEndian.AppendUint16(msg, 0)
}
