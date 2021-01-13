package randomizer

import (
	"crypto/sha1"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"strconv"

	"gopkg.in/yaml.v2"
)

const bankSize = 0x4000

var rings []string

// only applies to seasons! used for warps
var dungeonNameRegexp = regexp.MustCompile(`^d[1-8]$`)

// a fully-specified memory address. "offset" isn't true offset from the start
// of the bank (except for bank 0); it's bus address.
type address struct {
	bank   uint8
	offset uint16
}

// fullOffset returns the actual offset of the address in the ROM, based on
// bank number and relative address.
func (a *address) fullOffset() int {
	var bankOffset int
	if a.bank >= 2 {
		bankOffset = bankSize * (int(a.bank) - 1)
	}
	return bankOffset + int(a.offset)
}

func romIsAges(b []byte) bool {
	return string(b[0x134:0x13f]) == "ZELDA NAYRU"
}

func romIsSeasons(b []byte) bool {
	return string(b[0x134:0x13d]) == "ZELDA DIN"
}

func romIsJp(b []byte) bool {
	return b[0x014a] == 0
}

func romIsVanilla(b []byte) bool {
	knownSum := ternary(romIsSeasons(b),
		"\xba\x12\x68\x29\x0f\xb2\xb1\xb7\x05\x05\xd2\xd7\xb5\x82\x5f\xc8\xa4"+
			"\x81\x6a\x4b",
		"\x88\x03\x74\xfb\x97\x8b\x18\xaf\x4a\xa5\x29\xe2\xe3\x2f\x7f\xfb\x4d"+
			"\x7d\xd2\xf4").(string)
	sum := sha1.Sum(b)
	return string(sum[:]) == knownSum
}

// returns a 16-bit checksum of the rom data, for placing in the rom header.
// this is calculated by summing the non-global-checksum bytes in the rom.
// not to be confused with the header checksum, which is the byte before.
func makeRomChecksum(data []byte) [2]byte {
	var sum uint16
	for _, c := range data[:0x14e] {
		sum += uint16(c)
	}
	for _, c := range data[0x150:] {
		sum += uint16(c)
	}
	return [2]byte{byte(sum >> 8), byte(sum)}
}

type romState struct {
	game         int
	player       int
	data         []byte // actual contents of the file
	treasures    map[string]*treasure
	itemSlots    map[string]*itemSlot
	codeMutables map[string]*mutableRange
	bankEnds     []uint16 // bus offset of free space in each bank
	symbols      map[string]*address
}

func newRomState(data []byte, symbols map[string]*address, game, player int) *romState {
	rom := &romState{
		game:      game,
		player:    player,
		data:      data,
		symbols:   symbols,
		treasures: loadTreasures(data, *symbols["treasureObjectData"], game),
	}
	rom.itemSlots = rom.loadSlots()
	rom.initializeMutables()
	return rom
}

// changes the contents of loaded ROM bytes in place. returns a checksum of the
// result or an error.
func (rom *romState) mutate(warpMap map[string]string, seed uint32,
	ropts *randomizerOptions) ([]byte, error) {
	// need to set this *before* treasure map data
	if len(warpMap) != 0 {
		rom.setWarps(warpMap, ropts.dungeons)
	}

	if rom.game == gameSeasons {
		rom.setTreasureMapData()
	}

	rom.setSeedData()
	rom.setFileSelectText(optString(seed, ropts, "+"))
	rom.attachText()
	//rom.codeMutables["multiPlayerNumber"].new[0] = byte(rom.player) // TODO: Multiworld

	mutables := rom.getAllMutables()
	for _, k := range orderedKeys(mutables) {
		mutables[k].mutate(rom.data)
	}

	rom.setConfigData(ropts)

	sum := makeRomChecksum(rom.data)
	rom.data[0x14e] = sum[0]
	rom.data[0x14f] = sum[1]

	outSum := sha1.Sum(rom.data)
	return outSum[:], nil
}

// checks all the package's data against the ROM to see if it matches. It
// returns a slice of errors describing each mismatch.
func (rom *romState) verify() []error {
	errors := make([]error, 0)
	for k, m := range rom.getAllMutables() {
		switch k {
		default:
			if err := m.check(rom.data); err != nil {
				errors = append(errors, fmt.Errorf("%s: %v", k, err))
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (rom *romState) lookupSymbol(name string) *address {
	val, ok := rom.symbols[name]
	if !ok {
		panic(fmt.Sprintf("Symbol \"%s\" not found!", name))
	}
	return val
}

// set the initial satchel and slingshot seeds (and selections) based on what
// grows on the horon village tree, and set the map icon for each tree to match
// the seed type.
func (rom *romState) setSeedData() {
	treeName := sora(rom.game, "horon village tree", "south lynna tree").(string)
	initialSeedType := rom.itemSlots[treeName].treasure.id

	if rom.game == gameSeasons {
		rom.data[rom.symbols["randovar_initialSeedType"].fullOffset()] = initialSeedType

		for i, names := range [][]string{
			{"horon village tree",  "5"},
			{"woods of winter tree","4"},
			{"north horon tree",    "2"},
			{"spool swamp tree",    "3"},
			{"sunken city tree",    "1"},
			{"tarm ruins tree",     "0"},
		} {
			// Set seed type
			id := rom.itemSlots[names[0]].treasure.id
			addr := rom.lookupSymbol("enemyCode5a@treeDataTable")
			rom.data[addr.fullOffset() + i * 3] = id

			// Set map popup (order is different)
			popupIndex, _ := strconv.ParseInt(names[1], 10, 64)
			addr = rom.lookupSymbol("treeWarps")
			rom.data[addr.fullOffset() + int(popupIndex) * 3 + 2] = 0x15 + id
		}
	} else {
		// set high nybbles (seed types) of seed tree interactions
		setTreeNybble(rom.codeMutables["symmetryCityTreeSubId"],
			rom.itemSlots["symmetry city tree"])
		setTreeNybble(rom.codeMutables["southLynnaPresentTreeSubId"],
			rom.itemSlots["south lynna tree"])
		setTreeNybble(rom.codeMutables["crescentIslandTreeSubId"],
			rom.itemSlots["crescent island tree"])
		setTreeNybble(rom.codeMutables["zoraVillagePresentTreeSubId"],
			rom.itemSlots["zora village tree"])
		setTreeNybble(rom.codeMutables["rollingRidgeWestTreeSubId"],
			rom.itemSlots["rolling ridge west tree"])
		setTreeNybble(rom.codeMutables["ambisPalaceTreeSubId"],
			rom.itemSlots["ambi's palace tree"])
		setTreeNybble(rom.codeMutables["rollingRidgeEastTreeSubId"],
			rom.itemSlots["rolling ridge east tree"])
		setTreeNybble(rom.codeMutables["southLynnaPastTreeSubId"],
			rom.itemSlots["south lynna tree"])
		setTreeNybble(rom.codeMutables["dekuForestTreeSubId"],
			rom.itemSlots["deku forest tree"])
		setTreeNybble(rom.codeMutables["zoraVillagePastTreeSubId"],
			rom.itemSlots["zora village tree"])

		// satchel and shooter come with south lynna tree seeds
		rom.codeMutables["satchelInitialSeeds"].new[0] = 0x20 + initialSeedType
		rom.codeMutables["seedShooterGiveSeeds"].new[6] = 0x20 + initialSeedType
		for _, name := range []string{"satchelInitialSelection",
			"shooterInitialSelection"} {
			rom.codeMutables[name].new[1] = initialSeedType
		}

		// set map icons
		for _, name := range []string{"crescent island tree",
			"symmetry city tree", "south lynna tree", "zora village tree",
			"rolling ridge west tree", "ambi's palace tree",
			"rolling ridge east tree", "deku forest tree"} {
			codeName := inflictCamelCase(name) + "MapIcon"
			if name == "south lynna tree" || name == "zora village tree" {
				for _, n := range []string{"1", "2"} {
					rom.codeMutables[codeName+n].new[0] =
						0x15 + rom.itemSlots[name].treasure.id
				}
			} else {
				rom.codeMutables[codeName].new[0] =
					0x15 + rom.itemSlots[name].treasure.id
			}
		}
	}
}

// converts e.g. "hello world" to "helloWorld". disgusting tbh
func inflictCamelCase(s string) string {
	return fmt.Sprintf("%c%s", s[0], strings.ReplaceAll(
		strings.Title(strings.ReplaceAll(s, "'", "")), " ", "")[1:])
}

// sets the high nybble (seed type) of a seed tree interaction in ages.
func setTreeNybble(subid *mutableRange, slot *itemSlot) {
	subid.new[0] = (subid.new[0] & 0x0f) | (slot.treasure.id << 4)
}

// set the locations of the sparkles for the jewels on the treasure map.
func (rom *romState) setTreasureMapData() {
	for i, name := range []string{"round", "pyramid", "square", "x-shaped"} {
		addr := rom.symbols["_mapMenu_drawJewelLocations@jewelLocations"].fullOffset() + i * 2
		rom.data[addr+0] = 0x00
		rom.data[addr+1] = 0x63 // default to tarm gate
		for _, slot := range rom.lookupAllItemSlots(name + " jewel") {
			if int(slot.player) == 0 || int(slot.player) == rom.player {
				rom.data[addr+0] = byte(slot.mapTile >> 8)
				rom.data[addr+1] = byte(slot.mapTile & 0xff)
			}
		}
	}
}

// returns the slot where the named item was placed. this only works for unique
// items, of course.
func (rom *romState) lookupItemSlot(itemName string) *itemSlot {
	if slots := rom.lookupAllItemSlots(itemName); len(slots) > 0 {
		return slots[0]
	}
	return nil
}

// returns all slots where the named item was placed.
func (rom *romState) lookupAllItemSlots(itemName string) []*itemSlot {
	t := rom.treasures[itemName]
	slots := make([]*itemSlot, 0)
	for _, slot := range rom.itemSlots {
		if slot.treasure == t {
			slots = append(slots, slot)
		}
	}
	return slots
}

// get the location of the dungeon properties byte for a specific room.
func (rom *romState) getDungeonPropertiesAddr(group, room byte) *address {
	baseAddr := rom.symbols["dungeonRoomPropertiesGroup4Data"]
	offset := uint16(room)
	offset += baseAddr.offset
	if group%2 != 0 {
		offset += 0x100
	}
	return &address{baseAddr.bank, offset}
}

// randomizes the types of rings in the item pool, returning a map of vanilla
// ring names to the randomized ones.
// TODO: Make sure this works
func (rom *romState) randomizeRingPool(src *rand.Rand,
	planValues []string) (map[string]string, error) {
	nameMap := make(map[string]string)
	usedRings := make([]bool, 0x40)

	originalKeys := orderedKeys(rom.itemSlots)

	nRings := 0
	for _, slot := range rom.itemSlots {
		if slot.treasure.id == 0x2d {
			nRings++
		}
	}
	ringValues, i := make([]int, nRings), 0

	// load planned values if present
	for _, v := range planValues {
		if id := getStringIndex(rings, v); id != -1 {
			if i >= len(ringValues) {
				return nil, fmt.Errorf("too many rings in plan")
			}
			ringValues[i] = id
			i++
		} else {
			return nil, fmt.Errorf("no such ring: %s", v)
		}
	}

	// then roll random ones for the rest
	for i < len(ringValues) {
		// loop until we get a random ring that's not literally useless, and
		// which we haven't used before.
		done := false
		for !done {
			param := src.Intn(0x40)
			switch rings[param] {
			case "friendship ring", "GBA time ring", "GBA nature ring",
				"slayer's ring", "rupee ring", "victory ring", "sign ring",
				"100th ring":
				break
			case "rang ring L-1", "rang ring L-2", "green joy ring":
				// these rings are literally useless in ages.
				if rom.game == gameAges {
					break
				}
				fallthrough
			default:
				if !usedRings[param] {
					usedRings[param] = true
					ringValues[i] = param
					done = true
					i++
				}
			}
		}
	}
	sort.Ints(ringValues)

	i = 0
	for _, key := range originalKeys {
		slot := rom.itemSlots[key]
		if slot.treasure.id == 0x2d {
			oldName, _ := reverseLookup(rom.treasures, slot.treasure)
			slot.treasure.param = byte(ringValues[i])
			slot.treasure.displayName = rings[ringValues[i]]
			nameMap[oldName.(string)] = slot.treasure.displayName
			i++
		}
	}

	return nameMap, nil
}

// -- dungeon entrance / subrosia portal connections --

type warpData struct {
	// loaded from yaml
	EntryLabel, ExitLabel string
	EntryIndex, ExitIndex byte
	MapTile               uint16

	// set after loading
	vanillaMapTile               uint16
	len, entryOffset, exitOffset int

	vanillaEntryData, vanillaExitData []byte // read from rom
}

func (rom *romState) setWarps(warpMap map[string]string, dungeons bool) {
	lookupWarpSource := func(label string, index byte) int {
		return rom.lookupSymbol(label).fullOffset() + int(index) * 4
	}

	// load yaml data
	wd := make(map[string](map[string]*warpData))
	if err := yaml.Unmarshal(
		FSMustByte(false, "/romdata/warps.yaml"), wd); err != nil {
		panic(err)
	}
	warps := make(map[string]*warpData)
	for k, v := range wd["common"] {
		warps[k] = v
	}
	for k, v := range wd[sora(rom.game, "seasons", "ages").(string)] {
		warps[k] = v
	}

	// read vanilla data
	for name, warp := range warps {
		if strings.HasSuffix(name, "essence") {
			warp.len = 4
			warp.entryOffset = -1
			warp.exitOffset = lookupWarpSource(warp.ExitLabel, warp.ExitIndex)
		} else {
			warp.len = 2
			warp.entryOffset = lookupWarpSource(warp.EntryLabel, warp.EntryIndex) + 2
			warp.exitOffset = lookupWarpSource(warp.ExitLabel, warp.ExitIndex) + 2

			warp.vanillaEntryData = make([]byte, warp.len)
			copy(warp.vanillaEntryData,
				rom.data[warp.entryOffset:warp.entryOffset+warp.len])
		}

		warp.vanillaExitData = make([]byte, warp.len)
		copy(warp.vanillaExitData,
			rom.data[warp.exitOffset:warp.exitOffset+warp.len])

		warp.vanillaMapTile = warp.MapTile
	}

	// ages needs essence warp data to d6 present entrance, even though it
	// doesn't exist in vanilla. (TODO)
	/*
	if rom.game == gameAges {
		warps["d6 present essence"] = &warpData{
			vanillaExitData: []byte{0x81, 0x0e, 0x16, 0x01},
		}
	}
	*/

	// set randomized data
	for srcName, destName := range warpMap {
		src, dest := warps[srcName], warps[destName]
		for i := 0; i < src.len; i++ {
			rom.data[src.entryOffset+i] = dest.vanillaEntryData[i]
			rom.data[dest.exitOffset+i] = src.vanillaExitData[i]
		}
		dest.MapTile = src.vanillaMapTile

		destEssence := warps[destName+" essence"]
		if destEssence != nil && destEssence.exitOffset != 0 {
			srcEssence := warps[srcName+" essence"]
			for i := 0; i < destEssence.len; i++ {
				rom.data[destEssence.exitOffset+i] = srcEssence.vanillaExitData[i]
			}
		}
	}

	if rom.game == gameSeasons {
		// shuffle treasure map data along with dungeons.
		changeTreasureMapTiles(rom.itemSlots, func(c chan tileChange) {
			for name, warp := range warps {
				if dungeonNameRegexp.MatchString(name) {
					c <- tileChange{warp.vanillaMapTile, warp.MapTile}
				}
			}
			close(c)
		})

		if dungeons {
			// remove alternate d2 entrances and connect d2 stairs exits
			// directly to each other
			src, dest := warps["d2 alt left"], warps["d2 alt right"]
			rom.data[src.exitOffset] = dest.vanillaEntryData[0]
			rom.data[src.exitOffset+1] = dest.vanillaEntryData[1]
			rom.data[dest.exitOffset] = src.vanillaEntryData[0]
			rom.data[dest.exitOffset+1] = src.vanillaEntryData[1]

			// the alternate entrance stair tiles will also be removed (handled
			// in the disassembly)
		}
	}
}

type tileChange struct {
	old, new uint16
}

// process a set of treasure map tile changes in a way that ensures each tile
// is substituted only once (per call to this function).
func changeTreasureMapTiles(slots map[string]*itemSlot,
	generate func(chan tileChange)) {
	pendingTiles := make(map[*itemSlot]uint16)
	c := make(chan tileChange)
	go generate(c)

	for change := range c {
		for _, slot := range slots {
			// diving spot outside d4 would be mistaken for a d4 check
			if slot.mapTile == change.old &&
				slot != slots["diving spot outside D4"] {
				pendingTiles[slot] = change.new
			}
		}
	}

	for slot, tile := range pendingTiles {
		slot.mapTile = tile
	}
}

// set the string to display on the file select screen.
func (rom *romState) setFileSelectText(row2 string) {
	// construct tiles from strings
	version := strings.Replace(version, "beta", "bet", 1) // full won't fit
	fileSelectRow1 := stringToTiles(strings.ToUpper(ternary(len(version) == 5,
		fmt.Sprintf("rando-ng %s", version),
		fmt.Sprintf("ng %13s", version)[:16]).(string)))
	fileSelectRow2 := stringToTiles(
		strings.ToUpper(strings.ReplaceAll(row2, "-", " ")))

	tileAddr := rom.lookupSymbol("randoFileSelectStringTiles").fullOffset() + 2
	copy(rom.data[tileAddr:tileAddr+len(fileSelectRow1)], fileSelectRow1)
	padding := 16 - len(fileSelectRow2) // bias toward right padding
	copy(rom.data[tileAddr+32+padding/2:tileAddr+32+padding/2+32], fileSelectRow2)
}

// returns a conversion of the string to file select screen tile indexes, using
// the custom font.
func stringToTiles(s string) []byte {
	b := make([]byte, len(s))
	for i, c := range []byte(s) {
		b[i] = func() byte {
			switch {
			case c >= '0' && c <= '9':
				return c - 0x20
			case c >= 'A' && c <= 'Z':
				return c + 0xa1
			case c == ' ':
				return '\xfc'
			case c == '+':
				return '\xfd'
			case c == '-':
				return '\xfe'
			case c == '.':
				return '\xff'
			default:
				return '\xfc' // leave other characters blank
			}
		}()
	}
	return b
}

// attaches text for shop items to matching labels.
func (rom *romState) attachText() {
	// insert randomized item names into shop text
	shopNames := loadShopNames(gameNames[rom.game])
	shopMap := map[string]string{
		"randoText_shop150Rupees": "shop, 150 rupees",
	}
	if rom.game == gameSeasons {
		shopMap["randoText_membersShop1"] = "member's shop 1"
		shopMap["randoText_membersShop2"] = "member's shop 2"
		shopMap["randoText_membersShop3"] = "member's shop 3"
		shopMap["randoText_subrosiaMarket1stItem"] = "subrosia market, 1st item"
		shopMap["randoText_subrosiaMarket2ndItem"] = "subrosia market, 2nd item"
		shopMap["randoText_subrosiaMarket5thItem"] = "subrosia market, 5th item"
	}
	for codeName, slotName := range shopMap {
		addr := rom.lookupSymbol(codeName).fullOffset()
		itemName := shopNames[rom.itemSlots[slotName].treasure.displayName]
		for _, c := range []byte(itemName) {
			rom.data[addr] = c
			addr += 1
		}
		rom.data[addr] = 0
	}
}

var articleRegexp = regexp.MustCompile("^(an?|the) ")

// return a map of internal item names to text that should be displayed for the
// item in shops.
func loadShopNames(game string) map[string]string {
	m := make(map[string]string)

	// load names used for owl hints
	itemFiles := []string{
		"/hints/common_items.yaml",
		fmt.Sprintf("/hints/%s_items.yaml", game),
	}
	for _, filename := range itemFiles {
		if err := yaml.Unmarshal(
			FSMustByte(false, filename), m); err != nil {
			panic(err)
		}
	}

	// remove articles
	for k, v := range m {
		m[k] = articleRegexp.ReplaceAllString(v, "")
	}

	return m
}

// writes boolean configuration parameters to the rom
func (rom *romState) setConfigData(ropts *randomizerOptions) {
	var config byte = 0

	if ropts.keysanity {
		config |= 1
	}
	if ropts.treewarp {
		config |= 2
	}
	if ropts.dungeons {
		config |= 4
	}

	addr := rom.lookupSymbol("randoConfig").fullOffset()
	rom.data[addr] = config
}
