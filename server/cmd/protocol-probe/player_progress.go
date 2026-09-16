package main

import (
	"fmt"
	"github.com/local/9yin-go-server/internal/clientdata"
	"time"
)

const (
	propCurNeiGong     uint16 = 653
	propFaculty        uint16 = 798
	propFacultyState   uint16 = 758
	propFacultyStyle   uint16 = 759
	propFacultyName    uint16 = 760
	propCurFillValue   uint16 = 764
	propTotalFillValue uint16 = 765
	propCurLevel       uint16 = 763
	propFillSpeed      uint16 = 773
	propPowerValue     uint16 = 2287
	propMaxPowerValue  uint16 = 2288
	propSP             uint16 = 467
	propStr            uint16 = 485
	propSta            uint16 = 489
	propDex            uint16 = 487
	propIng            uint16 = 486
	propSpi            uint16 = 488
	propMeleePower     uint16 = 500
	propMagicPower     uint16 = 502
	propPhyHit         uint16 = 556
	propMagicHit       uint16 = 559
	propDodge          uint16 = 562
	propMaxParry       uint16 = 476
	propPhyDef         uint16 = 584
	propStiDef         uint16 = 586
	propNegDef         uint16 = 588
	propMasDef         uint16 = 589
	propJujDef         uint16 = 587
	propMinMeleeDamage uint16 = 510
	propMaxMeleeDamage uint16 = 509
	propMaxHPAdd       uint16 = 581
	propMaxMPAdd       uint16 = 582
	propMaxLevel       uint16 = 2083
	propStaticData     uint16 = 2237
	propItemType       uint16 = 1889
	propMaxParryAdd    uint16 = 477
	propMinMagicDefAdd uint16 = 597
)
const activeFacultyCardReward int32 = 20000
const (
	viewportNeiGong            uint16 = 43
	facultyStateNone           int32  = 0
	facultyStateConvert        int32  = 2
	facultyStyleNormal         int32  = 1
	facultyStyleAct            int32  = 2
	logicStateFaculty          int32  = 108
	normalFacultyFillPerMinute int32  = 200
)

func facultyBool(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

type learnedNeiGong struct {
	configID             string
	staticData           int32
	itemType             int32
	level                int32
	maxLevel             int32
	fill                 int32
	total                int32
	power                int32
	maxPower             int32
	neiGongLevel         int32
	buffID               string
	buffStaticData       uint32
	buffLevel            int32
	strAdd               int32
	staAdd               int32
	dexAdd               int32
	ingAdd               int32
	spiAdd               int32
	maxHPAdd             int32
	maxMPAdd             int32
	maxParryAdd          int32
	minMagicDefAdd       int32
	minMeleeDamageAdd    int32
	maxMeleeDamageAdd    int32
	meleePowerAdd        int32
	magicPowerAdd        int32
	phyHitAdd            int32
	magicHitAdd          int32
	dodgeAdd             int32
	phyDefAdd            int32
	stiDefAdd            int32
	negDefAdd            int32
	masDefAdd            int32
	jujDefAdd            int32
	vaDefAdd             int32
	magicVaAdd           int32
	phyFinalDamageAdd    int32
	stiFinalDamageAdd    int32
	jujFinalDamageAdd    int32
	negFinalDamageAdd    int32
	masFinalDamageAdd    int32
	phyFinalDamageReduce int32
	stiFinalDamageReduce int32
	jujFinalDamageReduce int32
	negFinalDamageReduce int32
	masFinalDamageReduce int32
	attribute            string
	wuXing               int32
	skillModifiers       map[string][]modernNeiGongModifier
}
type playerProgress struct {
	books        []learnedNeiGong
	curNeiGong   string
	faculty      int32
	facultyState int32
	facultyStyle int32
	facultyName  string
	fillSpeed    int32
	lastAdvance  time.Time
	attributes   playerAttributes
	qgLevels     map[string]int32
	xiulian      int32
}
type playerAttributes struct {
	sp                  int32
	str                 int32
	sta                 int32
	dex                 int32
	ing                 int32
	spi                 int32
	meleePower          int32
	magicPower          int32
	phyHit              int32
	magicHit            int32
	dodge               int32
	maxParry            int32
	phyDef              int32
	stiDef              int32
	negDef              int32
	masDef              int32
	jujDef              int32
	minMeleeDamage      int32
	maxMeleeDamage      int32
	maxHPAdd            int32
	maxMPAdd            int32
	equipMinMeleeDamage int32
	equipMaxMeleeDamage int32
	equipMaxHPAdd       int32
	equipMaxMPAdd       int32
	equipPhyDef         int32
	equipMaxParry       int32
	equipMeleePower     int32
}

func (p *playerProgress) equippedResourceAdds() (maxHPAdd, maxMPAddRaw int32) {
	if equipped := p.book(p.curNeiGong); equipped != nil {
		return equipped.maxHPAdd, equipped.maxMPAdd
	}
	return 0, 0
}
func newPlayerProgress() playerProgress {
	books := []learnedNeiGong{{configID: "ng_jh_001", staticData: 1, itemType: 1002, level: 1, maxLevel: 20, total: 750, power: 120, maxPower: 120}}
	progress := playerProgress{books: books, curNeiGong: books[0].configID, faculty: 9_999_999, facultyState: facultyStateConvert, facultyStyle: facultyStyleNormal, facultyName: books[0].configID, fillSpeed: normalFacultyFillPerMinute, lastAdvance: time.Now(), attributes: playerAttributes{sp: 100, str: 30, sta: 30, dex: 30, ing: 30, spi: 30, meleePower: 80, magicPower: 80, phyHit: 50, magicHit: 50, dodge: 35, maxParry: 100, phyDef: 60, stiDef: 50, negDef: 50, masDef: 50, jujDef: 50, minMeleeDamage: 20, maxMeleeDamage: 35}}
	progress.refreshAllModernEffects()
	return progress
}
func (book *learnedNeiGong) applyModernEffect() error {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return err
	}
	effect, err := catalog.effect(book.staticData, book.level)
	if err != nil {
		return err
	}
	book.buffID = effect.buffID
	book.buffStaticData = effect.staticData
	book.buffLevel = effect.buffLevel
	book.attribute = innerPowerAttributeName(effect.attribute, book.configID)
	book.neiGongLevel = effect.neiGongLevel
	book.wuXing = loadWuxueWuxing()[book.configID]
	book.strAdd = effect.stats["StrAdd"]
	book.staAdd = effect.stats["StaAdd"]
	book.dexAdd = effect.stats["DexAdd"]
	book.ingAdd = effect.stats["IngAdd"]
	book.spiAdd = effect.stats["SpiAdd"]
	book.maxHPAdd = effect.stats["MaxHPAdd"]
	book.maxMPAdd = effect.stats["MaxMPAdd"]
	book.maxParryAdd = effect.stats["MaxParryAdd"]
	book.minMagicDefAdd = effect.stats["MinMagicDefAdd"]
	book.minMeleeDamageAdd = effect.stats["MinMeleeDamage"]
	book.maxMeleeDamageAdd = effect.stats["MaxMeleeDamage"]
	book.meleePowerAdd = effect.stats["MeleePowerAdd"]
	book.magicPowerAdd = effect.stats["MagicPowerAdd"]
	book.phyHitAdd = effect.stats["PhyHitAdd"]
	book.magicHitAdd = effect.stats["MagicHitAdd"]
	book.dodgeAdd = effect.stats["DodgeAdd"]
	book.phyDefAdd = effect.stats["PhyDefAdd"]
	book.stiDefAdd = effect.stats["StiDefAdd"]
	book.negDefAdd = effect.stats["NegDefAdd"]
	book.masDefAdd = effect.stats["MasDefAdd"]
	book.jujDefAdd = effect.stats["JujDefAdd"]
	book.vaDefAdd = effect.stats["VaDefAdd"]
	book.magicVaAdd = effect.stats["MagicVaAdd"]
	book.phyFinalDamageAdd = effect.stats["PhyFinalDamageAdd"]
	book.stiFinalDamageAdd = effect.stats["StiFinalDamageAdd"]
	book.jujFinalDamageAdd = effect.stats["JujFinalDamageAdd"]
	book.negFinalDamageAdd = effect.stats["NegFinalDamageAdd"]
	book.masFinalDamageAdd = effect.stats["MasFinalDamageAdd"]
	book.phyFinalDamageReduce = effect.stats["PhyFinalDamageReduce"]
	book.stiFinalDamageReduce = effect.stats["StiFinalDamageReduce"]
	book.jujFinalDamageReduce = effect.stats["JujFinalDamageReduce"]
	book.negFinalDamageReduce = effect.stats["NegFinalDamageReduce"]
	book.masFinalDamageReduce = effect.stats["MasFinalDamageReduce"]
	book.skillModifiers = make(map[string][]modernNeiGongModifier, len(effect.skillModifiers))
	for skillID, modifiers := range effect.skillModifiers {
		book.skillModifiers[skillID] = append([]modernNeiGongModifier(nil), modifiers...)
	}
	return nil
}
func (p *playerProgress) refreshAllModernEffects() {
	for index := range p.books {
		if err := p.books[index].applyModernEffect(); err != nil {
			continue
		}
	}
}
func playerProgressFields() []clientdata.FieldSpec {
	return []clientdata.FieldSpec{{Index: propCurNeiGong, Name: "CurNeiGong", Type: clientdata.WireString}, {Index: propFaculty, Name: "Faculty", Type: clientdata.WireInt32}, {Index: propFacultyState, Name: "FacultyState", Type: clientdata.WireInt32}, {Index: propFacultyStyle, Name: "FacultyStyle", Type: clientdata.WireInt32}, {Index: propFacultyName, Name: "FacultyName", Type: clientdata.WireString}, {Index: propCurFillValue, Name: "CurFillValue", Type: clientdata.WireInt32}, {Index: propTotalFillValue, Name: "TotalFillValue", Type: clientdata.WireInt32}, {Index: propCurLevel, Name: "CurLevel", Type: clientdata.WireInt32}, {Index: propFillSpeed, Name: "FillSpeed", Type: clientdata.WireInt32}, {Index: propPowerValue, Name: "PowerValue", Type: clientdata.WireInt32}, {Index: propMaxPowerValue, Name: "MaxPowerValue", Type: clientdata.WireInt32}, {Index: propSP, Name: "SP", Type: clientdata.WireInt32}, {Index: propStr, Name: "Str", Type: clientdata.WireInt32}, {Index: propSta, Name: "Sta", Type: clientdata.WireInt32}, {Index: propDex, Name: "Dex", Type: clientdata.WireInt32}, {Index: propIng, Name: "Ing", Type: clientdata.WireInt32}, {Index: propSpi, Name: "Spi", Type: clientdata.WireInt32}, {Index: propMeleePower, Name: "MeleePower", Type: clientdata.WireInt32}, {Index: propMagicPower, Name: "MagicPower", Type: clientdata.WireInt32}, {Index: propPhyHit, Name: "PhyHit", Type: clientdata.WireInt32}, {Index: propMagicHit, Name: "MagicHit", Type: clientdata.WireInt32}, {Index: propDodge, Name: "Dodge", Type: clientdata.WireInt32}, {Index: propMaxParry, Name: "MaxParry", Type: clientdata.WireInt32}, {Index: propPhyDef, Name: "PhyDef", Type: clientdata.WireInt32}, {Index: propStiDef, Name: "StiDef", Type: clientdata.WireInt32}, {Index: propNegDef, Name: "NegDef", Type: clientdata.WireInt32}, {Index: propMasDef, Name: "MasDef", Type: clientdata.WireInt32}, {Index: propJujDef, Name: "JujDef", Type: clientdata.WireInt32}, {Index: propMinMeleeDamage, Name: "MinMeleeDamage", Type: clientdata.WireInt32}, {Index: propMaxMeleeDamage, Name: "MaxMeleeDamage", Type: clientdata.WireInt32}, {Index: propMaxHPAdd, Name: "MaxHPAdd", Type: clientdata.WireInt32}, {Index: propMaxMPAdd, Name: "MaxMPAdd", Type: clientdata.WireInt32}, {Index: propMaxLevel, Name: "MaxLevel", Type: clientdata.WireInt32}, {Index: propStaticData, Name: "StaticData", Type: clientdata.WireInt32}, {Index: propItemType, Name: "ItemType", Type: clientdata.WireInt32}, {Index: propMaxParryAdd, Name: "MaxParryAdd", Type: clientdata.WireInt32}, {Index: propMinMagicDefAdd, Name: "MinMagicDefAdd", Type: clientdata.WireInt32}}
}
func (p *playerProgress) book(id string) *learnedNeiGong {
	for index := range p.books {
		if p.books[index].configID == id {
			return &p.books[index]
		}
	}
	return nil
}
func (p *playerProgress) targetBook() *learnedNeiGong {
	if p.facultyName != "" {
		if book := p.book(p.facultyName); book != nil {
			return book
		}
	}
	return p.book(p.curNeiGong)
}
func (p *playerProgress) properties() []clientdata.IndexedProperty {
	book := p.targetBook()
	var level, maxLevel, fill, total, power, maxPower int32
	if book != nil {
		level = book.level
		maxLevel = book.maxLevel
		fill = book.fill
		total = book.total
		power = book.power
		maxPower = book.maxPower
	}
	a := p.attributes
	var maxParryAdd, minMagicDefAdd int32
	var phyFinalDamageAdd, stiFinalDamageAdd, jujFinalDamageAdd, negFinalDamageAdd, masFinalDamageAdd int32
	var phyFinalDamageReduce, stiFinalDamageReduce, jujFinalDamageReduce, negFinalDamageReduce, masFinalDamageReduce int32
	a.meleePower += a.equipMeleePower
	a.minMeleeDamage += a.equipMinMeleeDamage
	a.maxMeleeDamage += a.equipMaxMeleeDamage
	a.maxHPAdd += a.equipMaxHPAdd
	a.maxMPAdd += a.equipMaxMPAdd
	a.phyDef += a.equipPhyDef
	a.maxParry += a.equipMaxParry
	equipped := p.book(p.curNeiGong)
	if equipped != nil {
		a.str += equipped.strAdd
		a.sta += equipped.staAdd
		a.dex += equipped.dexAdd
		a.ing += equipped.ingAdd
		a.spi += equipped.spiAdd
		a.maxHPAdd += equipped.maxHPAdd
		a.maxMPAdd += equipped.maxMPAdd
		a.meleePower += equipped.meleePowerAdd
		a.magicPower += equipped.magicPowerAdd
		a.minMeleeDamage += equipped.minMeleeDamageAdd
		a.maxMeleeDamage += equipped.maxMeleeDamageAdd
		a.phyHit += equipped.phyHitAdd
		a.magicHit += equipped.magicHitAdd
		a.dodge += equipped.dodgeAdd
		a.phyDef += equipped.phyDefAdd
		a.stiDef += equipped.stiDefAdd
		a.negDef += equipped.negDefAdd
		a.masDef += equipped.masDefAdd
		a.jujDef += equipped.jujDefAdd
		a.phyDef += equipped.vaDefAdd
		maxParryAdd = equipped.maxParryAdd
		minMagicDefAdd = equipped.minMagicDefAdd
		phyFinalDamageAdd = equipped.phyFinalDamageAdd
		stiFinalDamageAdd = equipped.stiFinalDamageAdd
		jujFinalDamageAdd = equipped.jujFinalDamageAdd
		negFinalDamageAdd = equipped.negFinalDamageAdd
		masFinalDamageAdd = equipped.masFinalDamageAdd
		phyFinalDamageReduce = equipped.phyFinalDamageReduce
		stiFinalDamageReduce = equipped.stiFinalDamageReduce
		jujFinalDamageReduce = equipped.jujFinalDamageReduce
		negFinalDamageReduce = equipped.negFinalDamageReduce
		masFinalDamageReduce = equipped.masFinalDamageReduce
	}
	attrPhyHitAdd := a.dex
	attrHpUpSpeedAdd := int32(float32(a.sta) * 0.07)
	attrMpUpSpeedAdd := int32(float32(a.spi) * 0.2)
	attrMaxHPAdd := a.str*2 + a.sta*7
	attrMaxMPAdd := a.ing*4 + a.spi
	attrMaxParryAdd := int32(float32(a.sta) * 0.5)
	a.maxHPAdd += attrMaxHPAdd
	a.maxMPAdd += attrMaxMPAdd
	maxParryAdd += attrMaxParryAdd
	minMagicDefAdd += attrMpUpSpeedAdd
	props := []clientdata.IndexedProperty{{Index: 653, Name: "CurNeiGong", Value: clientdata.StringValue(p.curNeiGong)}, {Index: 798, Name: "Faculty", Value: clientdata.Int32Value(p.faculty)}, {Index: 758, Name: "FacultyState", Value: clientdata.Int32Value(p.facultyState)}, {Index: 759, Name: "FacultyStyle", Value: clientdata.Int32Value(p.facultyStyle)}, {Index: 760, Name: "FacultyName", Value: clientdata.StringValue(p.facultyName)}, {Index: 764, Name: "CurFillValue", Value: clientdata.Int32Value(fill)}, {Index: 765, Name: "TotalFillValue", Value: clientdata.Int32Value(total)}, {Index: 763, Name: "CurLevel", Value: clientdata.Int32Value(level)}, {Index: 773, Name: "FillSpeed", Value: clientdata.Int32Value(p.fillSpeed)}, {Index: 2287, Name: "PowerValue", Value: clientdata.Int32Value(power)}, {Index: 2288, Name: "MaxPowerValue", Value: clientdata.Int32Value(maxPower)}, {Index: 467, Name: "SP", Value: clientdata.Int32Value(a.sp)}, {Index: 485, Name: "Str", Value: clientdata.Int32Value(a.str)}, {Index: 489, Name: "Sta", Value: clientdata.Int32Value(a.sta)}, {Index: 487, Name: "Dex", Value: clientdata.Int32Value(a.dex)}, {Index: 486, Name: "Ing", Value: clientdata.Int32Value(a.ing)}, {Index: 488, Name: "Spi", Value: clientdata.Int32Value(a.spi)}, {Index: 500, Name: "MeleePower", Value: clientdata.Int32Value(a.meleePower)}, {Index: 502, Name: "MagicPower", Value: clientdata.Int32Value(a.magicPower)}, {Index: 556, Name: "PhyHit", Value: clientdata.Int32Value(a.phyHit)}, {Index: 559, Name: "MagicHit", Value: clientdata.Int32Value(a.magicHit)}, {Index: 562, Name: "Dodge", Value: clientdata.Int32Value(a.dodge)}, {Index: 476, Name: "MaxParry", Value: clientdata.Int32Value(a.maxParry)}, {Index: 584, Name: "PhyDef", Value: clientdata.Int32Value(a.phyDef)}, {Index: 586, Name: "StiDef", Value: clientdata.Int32Value(a.stiDef)}, {Index: 588, Name: "NegDef", Value: clientdata.Int32Value(a.negDef)}, {Index: 589, Name: "MasDef", Value: clientdata.Int32Value(a.masDef)}, {Index: 587, Name: "JujDef", Value: clientdata.Int32Value(a.jujDef)}, {Index: 510, Name: "MinMeleeDamage", Value: clientdata.Int32Value(a.minMeleeDamage)}, {Index: 509, Name: "MaxMeleeDamage", Value: clientdata.Int32Value(a.maxMeleeDamage)}, {Index: 581, Name: "MaxHPAdd", Value: clientdata.Int32Value(a.maxHPAdd)}, {Index: 582, Name: "MaxMPAdd", Value: clientdata.Int32Value(a.maxMPAdd)}, {Index: 477, Name: "MaxParryAdd", Value: clientdata.Int32Value(maxParryAdd)}, {Index: 597, Name: "MinMagicDefAdd", Value: clientdata.Int32Value(minMagicDefAdd)}, {Index: 2083, Name: "MaxLevel", Value: clientdata.Int32Value(maxLevel)}, {Index: 475, Name: "ParryValue", Value: clientdata.Int32Value(a.maxParry + maxParryAdd)}, {Index: 503, Name: "MeleePowerAdd", Value: clientdata.Int32Value(a.str)}, {Index: 505, Name: "MagicPowerAdd", Value: clientdata.Int32Value(a.ing)}, {Index: 501, Name: "ShotPower", Value: clientdata.Int32Value(80)}, {Index: 504, Name: "ShotPowerAdd", Value: clientdata.Int32Value(attrPhyHitAdd)}, {Index: 565, Name: "PhyVa", Value: clientdata.Int32Value(0)}, {Index: 566, Name: "PhyVaAdd", Value: clientdata.Int32Value(a.str)}, {Index: 569, Name: "MagicVa", Value: clientdata.Int32Value(0)}, {Index: 570, Name: "MagicVaAdd", Value: clientdata.Int32Value(a.spi)}, {Index: 557, Name: "PhyHitAdd", Value: clientdata.Int32Value(attrPhyHitAdd)}, {Index: 560, Name: "MagicHitAdd", Value: clientdata.Int32Value(a.spi)}, {Index: 563, Name: "DodgeAdd", Value: clientdata.Int32Value(attrPhyHitAdd)}, {Index: 619, Name: "HPUpSpeedAdd", Value: clientdata.Int32Value(attrHpUpSpeedAdd)}, {Index: 620, Name: "MPUpSpeedAdd", Value: clientdata.Int32Value(attrMpUpSpeedAdd)}, {Index: 543, Name: "PhyParryDamRes", Value: clientdata.Int32Value(0)}, {Index: 548, Name: "PhyParryDamResAdd", Value: clientdata.Int32Value(0)}, {Index: 545, Name: "MagicParryDamRes", Value: clientdata.Int32Value(0)}, {Index: 549, Name: "MagicParryDamResAdd", Value: clientdata.Int32Value(0)}, {Index: 523, Name: "PhyFinalDamageAdd", Value: clientdata.Int32Value(phyFinalDamageAdd)}, {Index: 524, Name: "StiFinalDamageAdd", Value: clientdata.Int32Value(stiFinalDamageAdd)}, {Index: 525, Name: "JujFinalDamageAdd", Value: clientdata.Int32Value(jujFinalDamageAdd)}, {Index: 526, Name: "NegFinalDamageAdd", Value: clientdata.Int32Value(negFinalDamageAdd)}, {Index: 527, Name: "MasFinalDamageAdd", Value: clientdata.Int32Value(masFinalDamageAdd)}, {Index: 528, Name: "PhyFinalDamageReduce", Value: clientdata.Int32Value(phyFinalDamageReduce)}, {Index: 529, Name: "StiFinalDamageReduce", Value: clientdata.Int32Value(stiFinalDamageReduce)}, {Index: 530, Name: "JujFinalDamageReduce", Value: clientdata.Int32Value(jujFinalDamageReduce)}, {Index: 531, Name: "NegFinalDamageReduce", Value: clientdata.Int32Value(negFinalDamageReduce)}, {Index: 532, Name: "MasFinalDamageReduce", Value: clientdata.Int32Value(masFinalDamageReduce)}}
	props = append(props, facultyTipSnapshot(p.books, p.targetBook())...)
	if p.xiulian > 0 {
		props = append(props, clientdata.IndexedProperty{Index: 818, Name: "SkillUseValue", Value: clientdata.Int32Value(p.xiulian)}, clientdata.IndexedProperty{Index: 817, Name: "NeigongUseValue", Value: clientdata.Int32Value(p.xiulian)}, clientdata.IndexedProperty{Index: 2287, Name: "PowerValue", Value: clientdata.Int32Value(p.xiulian)})
	}
	return props
}
func (p *playerProgress) viewFrames() ([][]byte, error) {
	frames := [][]byte{serverCreateView(serverViewSpec{ID: viewportNeiGong, Capacity: uint16(len(p.books))})}
	for index := range p.books {
		frame, err := p.bookFrame(uint16(index + 1))
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
func (p *playerProgress) bookFrame(slot uint16) ([]byte, error) {
	if slot == 0 || int(slot) > len(p.books) {
		return nil, fmt.Errorf("invalid neigong slot %d", slot)
	}
	book := p.books[slot-1]
	return serverViewAdd(viewportNeiGong, slot, []serverViewProperty{viewString(0x05A0, book.configID), viewInt(0x0761, book.itemType), viewInt(0x08BD, book.staticData), viewByte(0x05B9, byte(book.level)), viewInt(0x0823, book.maxLevel), viewInt(0x08EC, book.neiGongLevel), viewInt(0x02FD, book.total), viewInt(0x08C3, book.wuXing), viewString(0x08ED, neiGongDisplayBuffID(book.buffID))})
}
func (p *playerProgress) equip(id string) error {
	if p.book(id) == nil {
		return fmt.Errorf("not learned neigong %q", id)
	}
	p.curNeiGong = id
	return nil
}
func (p *playerProgress) facultyReady(id string) (uint16, error) {
	_, isQingGong := qgStaticData[id]
	if isQingGong {
		if p.qgLevels == nil {
			p.qgLevels = make(map[string]int32)
		}
		if p.qgLevels[id] <= 0 {
			p.qgLevels[id] = 1
		}
		p.facultyName = id
		p.facultyState = facultyStateNone
		p.facultyStyle = facultyStyleNormal
		p.fillSpeed = 0
		p.lastAdvance = time.Now().UTC()
		return qingGongViewSlot(id), nil
	}
	slot := uint16(0)
	for index := range p.books {
		if p.books[index].configID == id {
			slot = uint16(index + 1)
			break
		}
	}
	if slot == 0 {
		return 0, fmt.Errorf("not learned neigong %q", id)
	}
	p.facultyName = id
	p.facultyState = facultyStateNone
	p.facultyStyle = facultyStyleNormal
	p.fillSpeed = 0
	p.lastAdvance = time.Now().UTC()
	return slot, nil
}
func (p *playerProgress) facultyBegin(style int32, now time.Time) error {
	_, isQingGong := qgStaticData[p.facultyName]
	if !isQingGong && p.book(p.facultyName) == nil {
		return fmt.Errorf("no selected learned neigong")
	}
	p.facultyState = facultyStateConvert
	if style != facultyStyleAct {
		style = facultyStyleNormal
	}
	p.facultyStyle = style
	p.fillSpeed = normalFacultyFillPerMinute
	if style == facultyStyleAct {
		p.fillSpeed = 0
	}
	p.lastAdvance = now.UTC()
	return nil
}
func (p *playerProgress) facultyExit() {
	p.facultyState = facultyStateNone
	p.facultyStyle = facultyStyleNormal
	p.fillSpeed = 0
	p.lastAdvance = time.Now().UTC()
}
func (p *playerProgress) advanceFaculty(now time.Time) (uint16, bool) {
	if p.facultyState != facultyStateConvert || p.facultyStyle != facultyStyleNormal || p.fillSpeed <= 0 {
		p.lastAdvance = now.UTC()
		return 0, false
	}
	current := now.UTC()
	if p.lastAdvance.IsZero() || current.Before(p.lastAdvance) {
		p.lastAdvance = current
		return 0, false
	}
	elapsed := int32(current.Sub(p.lastAdvance) / time.Minute)
	if elapsed <= 0 {
		return 0, false
	}
	p.lastAdvance = p.lastAdvance.Add(time.Duration(elapsed) * time.Minute)
	static, isQingGong := qgStaticData[p.facultyName]
	if isQingGong {
		if p.qgLevels == nil {
			p.qgLevels = make(map[string]int32)
		}
		fill := int64(p.fillSpeed) * int64(elapsed)
		resource := p.xiulian
		if resource <= 0 {
			resource = p.faculty
		}
		cost := int64(resource)
		if cost > fill {
			cost = fill
		}
		if cost <= 0 {
			p.facultyState = facultyStateNone
			p.fillSpeed = 0
			return 0, false
		}
		if p.xiulian > 0 {
			p.xiulian -= int32(cost)
		} else {
			p.faculty -= int32(cost)
		}
		cur := p.qgLevels[p.facultyName]
		if cur <= 0 {
			cur = 1
		}
		maxLevel := static.maxLevel
		if maxLevel <= 0 {
			maxLevel = 1
		}
		if maxLevel > cur {
			cur++
			p.qgLevels[p.facultyName] = cur
		}
		if cur >= maxLevel {
			p.facultyState = facultyStateNone
			p.fillSpeed = 0
		}
		return qingGongViewSlot(p.facultyName), true
	}
	index := -1
	for i := range p.books {
		if p.books[i].configID == p.facultyName {
			index = i
			break
		}
	}
	if index < 0 {
		return 0, false
	}
	book := &p.books[index]
	fill := int64(p.fillSpeed) * int64(elapsed)
	cost := int64(p.faculty)
	if cost > fill {
		cost = fill
	}
	if cost <= 0 {
		p.facultyState = facultyStateNone
		p.fillSpeed = 0
		return 0, false
	}
	p.faculty -= int32(cost)
	book.fill += int32(cost)
	for book.fill >= book.total && book.level < book.maxLevel {
		book.fill -= book.total
		book.level++
		book.total += 200
		book.power += 20
		book.maxPower = book.power
	}
	_ = book.applyModernEffect()
	if book.level >= book.maxLevel {
		book.fill = 0
		p.facultyState = facultyStateNone
		p.fillSpeed = 0
	}
	return uint16(index + 1), true
}
func (p *playerProgress) awardActiveFaculty() (slot uint16, awarded bool, levelUp bool, maxLevel bool, err error) {
	if p.facultyState != facultyStateConvert || p.facultyStyle != facultyStyleAct {
		return 0, false, false, false, nil
	}
	for index := range p.books {
		book := &p.books[index]
		if book.configID != p.facultyName {
			continue
		}
		if book.level >= book.maxLevel {
			return uint16(index + 1), false, false, true, nil
		}
		oldLevel := book.level
		book.fill += activeFacultyCardReward
		for book.fill >= book.total && book.level < book.maxLevel {
			book.fill -= book.total
			book.level++
			book.total += 200
			book.power += 20
			book.maxPower = book.power
		}
		_ = book.applyModernEffect()
		if book.level >= book.maxLevel {
			book.fill = 0
		}
		return uint16(index + 1), true, book.level != oldLevel, book.level >= book.maxLevel, nil
	}
	return 0, false, false, false, fmt.Errorf("active faculty selected book %q is not learned", p.facultyName)
}
func (p *playerProgress) snapshot() facultyProgressSnapshot {
	value := facultyProgressSnapshot{CurNeiGong: p.curNeiGong, Faculty: p.faculty, FacultyState: p.facultyState, FacultyStyle: p.facultyStyle, FacultyName: p.facultyName, FillSpeed: p.fillSpeed, LastAdvanceUTC: p.lastAdvance.UTC(), Books: make([]facultyBookSnapshot, 0, len(p.books))}
	for _, book := range p.books {
		value.Books = append(value.Books, facultyBookSnapshot{ConfigID: book.configID, Level: book.level, Fill: book.fill, Total: book.total, Power: book.power, MaxPower: book.maxPower})
	}
	if len(p.qgLevels) > 0 {
		value.QGLevels = make(map[string]int32, len(p.qgLevels))
		for id, level := range p.qgLevels {
			value.QGLevels[id] = level
		}
	}
	return value
}
func (p *playerProgress) restore(value facultyProgressSnapshot, now time.Time) {
	if value.CurNeiGong != "" && p.book(value.CurNeiGong) != nil {
		p.curNeiGong = value.CurNeiGong
	}
	if value.Faculty >= 0 {
		p.faculty = value.Faculty
	}
	if value.FacultyName != "" && p.book(value.FacultyName) != nil {
		p.facultyName = value.FacultyName
	}
	p.facultyState = value.FacultyState
	if value.FacultyStyle == facultyStyleAct {
		p.facultyStyle = facultyStyleAct
	} else {
		p.facultyStyle = facultyStyleNormal
	}
	p.fillSpeed = value.FillSpeed
	p.lastAdvance = value.LastAdvanceUTC.UTC()
	for _, saved := range value.Books {
		book := p.book(saved.ConfigID)
		if book == nil || saved.Level <= 0 || book.maxLevel < saved.Level || saved.Total <= 0 || saved.Fill < 0 {
			continue
		}
		book.level = saved.Level
		book.fill = saved.Fill
		book.total = saved.Total
		book.power = saved.Power
		book.maxPower = saved.MaxPower
	}
	for id, level := range value.QGLevels {
		static, ok := qgStaticData[id]
		if !ok || level <= 0 {
			continue
		}
		maxLevel := static.maxLevel
		if maxLevel <= 0 {
			maxLevel = 1
		}
		if level > maxLevel {
			level = maxLevel
		}
		if p.qgLevels == nil {
			p.qgLevels = make(map[string]int32)
		}
		p.qgLevels[id] = level
	}
	if p.facultyState == facultyStateConvert && p.facultyStyle == facultyStyleNormal {
		p.advanceFaculty(now)
	} else {
		p.lastAdvance = now.UTC()
	}
	p.refreshAllModernEffects()
}

const S2CFacultyMessage int32 = 191

func encodeFacultyMessage(subcommand int32, arg2 int32) ([]byte, error) {
	return serverCustomIntMessageWithOpcode(0x1E, S2CFacultyMessage, customInt(subcommand), customInt(arg2))
}
func encodeFacultyActiveBegin() ([]byte, error) {
	return encodeFacultyMessage(22, 0)
}
func handlePlayerProgressCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 {
		return false, nil
	}
	command := custom.Values[0].Int32
	if command != 0xD7 && command != 0x3EE {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("player progress request before scene player exists")
	}
	writeVital := func() error {
		frame, err := player.vitalUpdate()
		if err != nil {
			return err
		}
		return link.WriteFrame(frame)
	}
	if command == 0xD7 {
		if len(custom.Values) < 2 || custom.Values[1].Type != 6 || custom.Values[1].Text == "" {
			return true, fmt.Errorf("215 requires a learned neigong ConfigID string")
		}
		if err := player.equipNeiGong(custom.Values[1].Text); err != nil {
			return true, err
		}
		return true, writeVital()
	}
	if len(custom.Values) < 2 || custom.Values[1].Type != 2 {
		return true, fmt.Errorf("1006 requires an int subcommand")
	}
	subcommand := custom.Values[1].Int32
	switch subcommand {
	case 1:
		if len(custom.Values) < 3 || custom.Values[2].Type != 6 || custom.Values[2].Text == "" {
			return true, fmt.Errorf("1006 sub=1 requires a learned neigong ConfigID string")
		}
		id := custom.Values[2].Text
		slot, err := player.facultyReady(id)
		if err != nil {
			return true, err
		}
		_, isQingGong := qgStaticData[id]
		if isQingGong {
			frame, err := player.qingGongViewUpdateFrame()
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
			if err := writeVital(); err != nil {
				return true, err
			}
			return true, nil
		}
		frame, err := player.progressBookFrame(slot)
		if err != nil {
			return true, err
		}
		if err := link.WriteFrame(frame); err != nil {
			return true, err
		}
		if err := writeVital(); err != nil {
			return true, err
		}
		return true, nil
	case 11:
		slot, advanced, err := player.facultyBegin(facultyStyleNormal)
		if err != nil {
			return true, err
		}
		if err := writeVital(); err != nil {
			return true, err
		}
		if advanced {
			frame, err := player.progressBookFrame(slot)
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
		}
		return true, nil
	case 12:
		player.facultyExit()
		if err := writeVital(); err != nil {
			return true, err
		}
		return true, nil
	case 21:
		slot, advanced, err := player.facultyBegin(facultyStyleAct)
		if err != nil {
			return true, err
		}
		if err := writeVital(); err != nil {
			return true, err
		}
		if advanced {
			frame, err := player.progressBookFrame(slot)
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
		}
		frame, err := encodeFacultyActiveBegin()
		if err != nil {
			return true, err
		}
		if err := link.WriteFrame(frame); err != nil {
			return true, err
		}
		return true, nil
	case 22:
		player.facultyExit()
		return true, writeVital()
	case 23:
		if len(custom.Values) < 4 {
			return true, nil
		}
		card, cardOK := custom.Values[2].exactInt32()
		payment, paymentOK := custom.Values[3].exactInt32()
		if !cardOK || !paymentOK {
			return true, nil
		}
		if card < 0 || card > 2 {
			return true, nil
		}
		if payment != 1 && payment != 2 && payment != 4 {
			return true, nil
		}
		slot, awarded, levelUp, maxLevel, err := player.progress.awardActiveFaculty()
		if err != nil {
			return true, err
		}
		if awarded {
			frame, err := player.progressBookFrame(slot)
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
		}
		if err := writeVital(); err != nil {
			return true, err
		}
		if levelUp {
			maxFlag := int32(0)
			if maxLevel {
				maxFlag = 1
			}
			levelFrame, err := encodeFacultyMessage(14, maxFlag)
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(levelFrame); err != nil {
				return true, err
			}
		}
		if levelUp || maxLevel {
			exitFrame, err := encodeFacultyMessage(23, 0)
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(exitFrame); err != nil {
				return true, err
			}
			return true, nil
		}
		frame, err := encodeFacultyMessage(24, 0)
		if err != nil {
			return true, err
		}
		return true, link.WriteFrame(frame)
	case 24:
		frame, needed, err := player.facultyNameResetFrame()
		if err != nil {
			return true, err
		}
		if needed {
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
		}
		if err := writeVital(); err != nil {
			return true, err
		}
		return true, nil
	case 31, 32:
		if err := writeVital(); err != nil {
			return true, err
		}
		return true, nil
	default:
		return true, fmt.Errorf("unsupported 1006 subcommand %d", subcommand)
	}
}
