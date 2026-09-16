package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	modernInnerPowerDescriptionPath = filepath.Join(defaultModernTextRoot, "chineses", "desc2.idres")
	modernBuffStaticPath            = filepath.Join(defaultModernShareRoot, "skill", "buff_static.ini")
	modernBuffVarPropPath           = filepath.Join(defaultModernShareRoot, "skill", "buff_varprop.ini")
)

const baseLocalDodgeProbability = int32(10)

type innerPowerMode uint8

const (
	innerPowerModeNone              innerPowerMode = 0
	innerPowerModeIncomingDodgeHeal innerPowerMode = 1
)

type innerPowerPassiveProfile struct {
	mode         innerPowerMode
	buffID       string
	buffLevel    int32
	procChance   int32
	dodgeAdd     int32
	procLifetime time.Duration
	procCooldown time.Duration
	healPercent  int32
	healCooldown time.Duration
	procBuffID   string
	procStatic   uint32
}
type innerPowerDescription struct {
	buffID string
	level  int32
	text   string
}
type nativeBuffRange struct {
	min int32
	max int32
}
type nativeBuffLevel struct {
	lifetime  time.Duration
	modifiers map[string]int32
}
type nativeBuffDefinition struct {
	configID   string
	staticData uint32
	rangeData  nativeBuffRange
	levels     map[int32]nativeBuffLevel
}
type innerPowerPassiveCatalog struct {
	descriptions map[string]map[int32]innerPowerDescription
	profiles     map[string]map[int32]innerPowerPassiveProfile
}

var (
	innerPowerPassiveOnce    sync.Once
	innerPowerPassives       *innerPowerPassiveCatalog
	innerPowerPassiveErr     error
	innerPowerDescriptionKey = regexp.MustCompile(`^desc_mind_(buf_ng_.+)_(\d+)_ngtips$`)
	incomingDodgeProcPattern = regexp.MustCompile(`被攻击时，有(\d+)%几率提升自身闪避率(\d+)%.*持续(\d+)秒，(\d+)秒冷却。`)
	incomingDodgeHealPattern = regexp.MustCompile(`成功闪避时，恢复自身(\d+)%最大气血.*?，(\d+)秒冷却。`)
)

func loadInnerPowerPassiveCatalog() (*innerPowerPassiveCatalog, error) {
	innerPowerPassiveOnce.Do(func() {
		entries, err := loadInnerPowerDescriptions()
		if err != nil {
			innerPowerPassiveErr = err
			return
		}
		nativeBuffs, err := loadInnerPowerNativeBuffs(entries)
		if err != nil {
			innerPowerPassiveErr = err
			return
		}
		catalog := &innerPowerPassiveCatalog{descriptions: entries, profiles: make(map[string]map[int32]innerPowerPassiveProfile)}
		for buffID, levels := range entries {
			for level, entry := range levels {
				profile, ok, err := compileIncomingDodgeHealProfile(entry, nativeBuffs)
				if err != nil {
					innerPowerPassiveErr = err
					return
				}
				if !ok {
					continue
				}
				if catalog.profiles[buffID] == nil {
					catalog.profiles[buffID] = make(map[int32]innerPowerPassiveProfile)
				}
				catalog.profiles[buffID][level] = profile
			}
		}
		innerPowerPassives = catalog
	})
	return innerPowerPassives, innerPowerPassiveErr
}
func loadInnerPowerDescriptions() (map[string]map[int32]innerPowerDescription, error) {
	content, err := os.ReadFile(modernInnerPowerDescriptionPath)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]map[int32]innerPowerDescription)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		match := innerPowerDescriptionKey.FindStringSubmatch(key)
		if len(match) != 3 {
			continue
		}
		level64, err := strconv.ParseInt(match[2], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parse inner-power description key %q: %w", key, err)
		}
		buffID := match[1]
		if entries[buffID] == nil {
			entries[buffID] = make(map[int32]innerPowerDescription)
		}
		level := int32(level64)
		entries[buffID][level] = innerPowerDescription{buffID: buffID, level: level, text: stripInnerPowerDescriptionMarkup(value)}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no inner-power passive descriptions in %s", modernInnerPowerDescriptionPath)
	}
	return entries, nil
}
func loadInnerPowerNativeBuffs(descriptions map[string]map[int32]innerPowerDescription) (map[string]nativeBuffDefinition, error) {
	modernCatalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return nil, err
	}
	wanted := make(map[uint32][]string)
	for baseID := range descriptions {
		prefix := baseID + "_"
		for configID, staticData := range modernCatalog.buffStatic {
			if !strings.HasPrefix(configID, prefix) {
				continue
			}
			wanted[staticData] = append(wanted[staticData], configID)
		}
	}
	definitions := make(map[string]nativeBuffDefinition)
	if len(wanted) == 0 {
		return nil, nil
	}
	ranges := make(map[uint32]nativeBuffRange)
	if err := parseModernINI(modernBuffStaticPath, func(section, key, value string) error {
		staticData64, err := strconv.ParseUint(section, 10, 32)
		if err != nil {
			return nil
		}
		staticData := uint32(staticData64)
		if len(wanted[staticData]) == 0 {
			return nil
		}
		current := ranges[staticData]
		switch key {
		case "MaxVarPropNo":
			parsed, err := parseModernInt32(value)
			if err != nil {
				return fmt.Errorf("buff static [%s] %s=%q: %w", section, key, value, err)
			}
			current.max = parsed
		case "MinVarPropNo":
			parsed, err := parseModernInt32(value)
			if err != nil {
				return fmt.Errorf("buff static [%s] %s=%q: %w", section, key, value, err)
			}
			current.min = parsed
		}
		ranges[staticData] = current
		return nil
	}); err != nil {
		return nil, err
	}
	for staticData, configIDs := range wanted {
		rangeData := ranges[staticData]
		if rangeData.min <= 0 || rangeData.max < rangeData.min {
			continue
		}
		for _, configID := range configIDs {
			definitions[configID] = nativeBuffDefinition{configID: configID, staticData: staticData, rangeData: rangeData, levels: make(map[int32]nativeBuffLevel)}
		}
	}
	if len(definitions) == 0 {
		return nil, nil
	}
	return loadInnerPowerNativeBuffLevels(definitions, modernCatalog)
}
func loadInnerPowerNativeBuffLevels(definitions map[string]nativeBuffDefinition, modernCatalog *modernNeiGongCatalog) (map[string]nativeBuffDefinition, error) {
	byVarProp := make(map[int32][]string)
	for configID, definition := range definitions {
		for id := definition.rangeData.min; id <= definition.rangeData.max; id++ {
			byVarProp[id] = append(byVarProp[id], configID)
		}
		definition.levels = make(map[int32]nativeBuffLevel)
		definitions[configID] = definition
	}
	section := int32(-1)
	level, lifetime := int32(0), int32(0)
	packs := make([]int32, 0, 1)
	flush := func() {
		if level <= 0 || len(byVarProp[section]) == 0 {
			return
		}
		for _, configID := range byVarProp[section] {
			definition := definitions[configID]
			modifiers := make(map[string]int32)
			for _, pack := range packs {
				for _, modifier := range modernCatalog.propPacks[pack] {
					if modifier.Target != "" || modifier.Mode != 0 {
						continue
					}
					modifiers[modifier.Name] += int32(modifier.Value)
				}
			}
			definition.levels[level] = nativeBuffLevel{lifetime: time.Duration(lifetime) * time.Millisecond, modifiers: modifiers}
			definitions[configID] = definition
		}
	}
	file, err := os.Open(modernBuffVarPropPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			parsed, parseErr := parseModernInt32(line[1 : len(line)-1])
			if parseErr != nil {
				section = -1
			} else {
				section = parsed
			}
			level, lifetime = 0, 0
			packs = packs[:0]
			continue
		}
		if len(byVarProp[section]) == 0 {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "Level":
			level, _ = parseModernInt32(value)
		case "LifeTime":
			lifetime, _ = parseModernInt32(value)
		case "@PropModifyPackRec":
			pack, err := parseModernInt32(value)
			if err == nil {
				packs = append(packs, pack)
			}
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return definitions, nil
}
func compileIncomingDodgeHealProfile(entry innerPowerDescription, nativeBuffs map[string]nativeBuffDefinition) (innerPowerPassiveProfile, bool, error) {
	proc := incomingDodgeProcPattern.FindStringSubmatch(entry.text)
	heal := incomingDodgeHealPattern.FindStringSubmatch(entry.text)
	if len(proc) != 5 || len(heal) != 3 {
		return innerPowerPassiveProfile{}, false, nil
	}
	var values [6]int32
	for i := 0; i < len(values); i++ {
		text := ""
		if i < 4 {
			text = proc[i+1]
		} else {
			text = heal[i-3]
		}
		parsed, err := parseModernInt32(text)
		if err != nil {
			return innerPowerPassiveProfile{}, false, err
		}
		values[i] = parsed
	}
	for configID, native := range nativeBuffs {
		if !strings.HasPrefix(configID, entry.buffID+"_") {
			continue
		}
		nativeLevel, exists := native.levels[entry.level]
		if !exists || nativeLevel.lifetime != time.Duration(values[2])*time.Second || nativeLevel.modifiers["DodgeProbAdd"] != values[1] {
			continue
		}
		return innerPowerPassiveProfile{mode: innerPowerModeIncomingDodgeHeal, buffID: entry.buffID, buffLevel: entry.level, procChance: values[0], dodgeAdd: values[1], procLifetime: time.Duration(values[2]) * time.Second, procCooldown: time.Duration(values[3]) * time.Second, healPercent: values[4], healCooldown: time.Duration(values[5]) * time.Second, procBuffID: configID, procStatic: native.staticData}, true, nil
	}
	return innerPowerPassiveProfile{}, false, nil
}
func stripInnerPowerDescriptionMarkup(value string) string {
	for {
		start := strings.IndexByte(value, '<')
		if start < 0 {
			return value
		}
		end := strings.IndexByte(value[start:], '>')
		if end < 0 {
			return value
		}
		value = value[:start] + value[start+end+1:]
	}
}
func (p *playerActor) equippedInnerPowerPassive() (innerPowerPassiveProfile, bool) {
	p.mu.Lock()
	book := p.progress.book(p.progress.curNeiGong)
	if book == nil || book.buffID == "" || book.buffLevel <= 0 {
		p.mu.Unlock()
		return innerPowerPassiveProfile{}, false
	}
	buffID := book.buffID
	buffLevel := book.buffLevel
	p.mu.Unlock()
	catalog, err := loadInnerPowerPassiveCatalog()
	if err != nil {
		log.Printf("load inner-power passive catalog: %v", err)
		return innerPowerPassiveProfile{}, false
	}
	profile, ok := catalog.profiles[buffID][buffLevel]
	return profile, ok
}
