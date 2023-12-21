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
var dungeonEntranceNameRegexp = regexp.MustCompile(`^d[1-8] entrance$`)

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
	labels       map[string]*address
	definitions  map[string]uint32
}

func newRomState(data []byte, labels map[string]*address, definitions map[string]uint32, game, player int, crossitems bool) *romState {
	rom := &romState{
		game:        game,
		player:      player,
		data:        data,
		labels:      labels,
		definitions: definitions,
		treasures:   loadTreasures(data, labels["treasureObjectData"], game),
	}
	rom.itemSlots = rom.loadSlots(crossitems)
	rom.initializeMutables(game)
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

func (rom *romState) lookupLabel(name string) *address {
	val, ok := rom.labels[name]
	if !ok {
		panic(fmt.Sprintf("Label \"%s\" not found!", name))
	}
	return val
}

func (rom *romState) lookupDefinition(name string) uint32 {
	val, ok := rom.definitions[name]
	if !ok {
		panic(fmt.Sprintf("Definition \"%s\" not found!", name))
	}
	return val
}

// set the initial satchel and slingshot seeds (and selections) based on what
// grows on the horon village tree, and set the map icon for each tree to match
// the seed type.
func (rom *romState) setSeedData() {
	initialTreeName := sora(rom.game, "horon village tree", "south lynna tree").(string)
	initialSeedType := rom.itemSlots[initialTreeName].treasure.id
	rom.data[rom.lookupLabel("randovar_initialSeedType").fullOffset()] = initialSeedType

	if rom.game == gameSeasons {
		for i, treeData := range [][]string{
			{"horon village tree",  "5"},
			{"woods of winter tree","4"},
			{"north horon tree",    "2"},
			{"spool swamp tree",    "3"},
			{"sunken city tree",    "1"},
			{"tarm ruins tree",     "0"},
		} {
			// Set seed type
			id := rom.itemSlots[treeData[0]].treasure.id
			addr := rom.lookupLabel("enemyCode5a@treeDataTable")
			rom.data[addr.fullOffset() + i * 3] = id

			// Set map popup (order is different)
			popupIndex, _ := strconv.ParseInt(treeData[1], 10, 64)
			addr = rom.lookupLabel("treeWarps")
			rom.data[addr.fullOffset() + int(popupIndex) * 3 + 2] = 0x15 + id
		}
	} else {
		for _, treeData := range [][]string{
			{"crescent island tree",    "0ac", "present", "0"},
			{"symmetry city tree",      "013", "present", "1"},
			{"south lynna tree",        "078", "present", "2"},
			{"zora village tree",       "0c1", "present", "3"},
			{"rolling ridge west tree", "108", "past",    "0"},
			{"ambi's palace tree",      "125", "past",    "1"},
			{"rolling ridge east tree", "12d", "past",    "2"},
			{"south lynna tree",        "178", "past",    "3"},
			{"deku forest tree",        "180", "past",    "4"},
			{"zora village tree",       "1c1", "past",    "5"},
		} {
			// Set seed type
			id := rom.itemSlots[treeData[0]].treasure.id
			group := treeData[1][0:1]
			room := treeData[1][1:3]
			labelName := fmt.Sprintf("group%sMap%sObjectData", group, room)
			addr := rom.lookupLabel(labelName).fullOffset()

			for {
				// Check for seed object to change the seed type. I was too lazy
				// to write code to parse object data properly; so this is
				// simply looking for byte 0xf7 (obj_SpecificEnemyA)
				// corresponding to the seed object. I mean, hey, it works.
				if rom.data[addr] == 0xf7 && rom.data[addr + 2] == 0x5a {
					subid := rom.data[addr + 3]
					subid &= 0x0f // lower nybble is something else, must preserve
					subid |= id << 4
					rom.data[addr + 3] = subid
					break
				} else {
					addr += 1
					continue
				}
			}

			// Set map popup
			addr = rom.lookupLabel(treeData[2] + "TreeWarps").fullOffset()
			popupIndex, _ := strconv.ParseInt(treeData[3], 10, 64)
			rom.data[addr + int(popupIndex) * 3 + 2] = 0x15 + id
		}
	}
}

// set the locations of the sparkles for the jewels on the treasure map.
func (rom *romState) setTreasureMapData() {
	for i, name := range []string{"round", "pyramid", "square", "x-shaped"} {
		addr := rom.lookupLabel("mapMenu_drawJewelLocations@jewelLocations").fullOffset() + i * 2
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
	baseAddr := rom.lookupLabel("dungeonRoomPropertiesGroup4Data")
	offset := uint16(room)
	offset += baseAddr.offset
	if group%2 != 0 {
		offset += 0x100
	}
	return &address{baseAddr.bank, offset}
}

// randomizes the types of rings in the item pool, returning a map of vanilla
// ring names to the randomized ones.
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
	Label     string
	Index     byte
	MapTile   uint16

	// set after loading
	vanillaMapTile uint16
	len, romOffset int

	vanillaData []byte // read from rom
}

func (rom *romState) setWarps(warpMap map[string]string, dungeons bool) {
	lookupWarpSource := func(label string, index byte) int {
		return rom.lookupLabel(label).fullOffset() + int(index) * 4
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
			warp.romOffset = lookupWarpSource(warp.Label, warp.Index)
		} else {
			warp.len = 2
			warp.romOffset = lookupWarpSource(warp.Label, warp.Index) + 2
		}

		warp.vanillaData = make([]byte, warp.len)
		copy(warp.vanillaData, rom.data[warp.romOffset:warp.romOffset+warp.len])

		if warp.len == 2 {
			// Unset special-case "alt entrance" bit used for newly added
			// present d2 warp
			warp.vanillaData[1] &= 0x7f
		}

		warp.vanillaMapTile = warp.MapTile
	}

	// Ages needs essence warp data to d2 & d6 present entrances, even though it
	// doesn't exist in vanilla.
	if rom.game == gameAges {
		warps["d2 present essence"] = &warpData{
			vanillaData: []byte{0x80, 0x83, 0x25, 0x01},
		}
		warps["d6 present essence"] = &warpData{
			vanillaData: []byte{0x81, 0x0e, 0x16, 0x01},
		}
	}

	// set randomized data
	exitCount := make(map[string]int)
	addedExtraExit := false
	for srcName, destName := range warpMap {
		enterWarp           := warps[srcName  + " entrance"]
		exitWarp            := warps[destName + " exit"]
		origEssenceWarp     := warps[srcName  + " essence"]
		changingEssenceWarp := warps[destName + " essence"]

		origEnter, origExit := destName + " entrance", srcName + " exit"

		var entranceWarpData* []byte
		var exitWarpData*     []byte

		// Ages D2 has multiple entrances but only 1 exit. Must do some
		// shenanigans to make it work properly.
		if rom.game == gameAges {
			if origEnter == "d2 entrance" {
				// Read the warp data for entering d2 from "d2 past entrance"
				// (either is fine)
				origEnter = "d2 past entrance"
				// Change the d2 past essence warp (d2 present essence doesn't exist)
				changingEssenceWarp = warps["d2 past essence"]
			}

			if origExit == "d2 past exit" {
				origExit = "d2 exit" // Original exit warp leads to past overworld
			} else if origExit == "d2 present exit" {
				// 0x4d = Dest warp index added in disassembly for d2 present
				// exit warp. 0x03 = group (0) + transition type (3).
				exitWarpData = &[]byte { 0x4d, 0x03 }
			}
		}

		if entranceWarpData == nil {
			entranceWarpData = &[]byte { 0, 0 }
			copy(*entranceWarpData, warps[origEnter].vanillaData)
		}
		if exitWarpData == nil {
			exitWarpData = &[]byte { 0, 0 }
			copy(*exitWarpData, warps[origExit].vanillaData)
		}

		exitCount[destName] += 1
		numExits, _ := exitCount[destName];
		if rom.game == gameAges && numExits > 1 {
			if addedExtraExit {
				fatalErr("More than 1 extra dungeon exit created")
			}
			// This is the 2nd entrance linked to this dungeon. Use the essence
			// warp data to tell the disassembly's special-case code where to
			// send Link when it detects that he's exiting from this entrance.
			duplicateExitOffset := rom.lookupLabel("randoAltDungeonEntranceRoom").fullOffset()
			rom.data[duplicateExitOffset + 0] = origEssenceWarp.vanillaData[0]
			rom.data[duplicateExitOffset + 1] = origEssenceWarp.vanillaData[1]
			rom.data[duplicateExitOffset + 2] = origEssenceWarp.vanillaData[2]
			rom.data[duplicateExitOffset + 3] = origEssenceWarp.vanillaData[3]

			// Also set a bit in the entrance warp's "group" variable which will
			// trigger the special-case
			(*entranceWarpData)[1] |= 0x80

			addedExtraExit = true
		} else if rom.game == gameSeasons && numExits > 1 {
			fatalErr("More than 1 entrance leads to same dungeon")
		}

		// Write the warp data for entering/exiting the dungeon
		for i := 0; i < enterWarp.len; i++ {
			rom.data[enterWarp.romOffset+i] = (*entranceWarpData)[i]

			// If this is the 2nd warp to this dungeon, don't write the exit
			// warp data here, rely on the special-case code to handle exiting
			// the dungeon instead
			if numExits == 1 {
				rom.data[exitWarp.romOffset+i]  = (*exitWarpData)[i]
			}
		}

		// Make the dungeon's essence warp to the same destination
		if numExits == 1 && !strings.HasSuffix(destName, "portal") && destName != "d6 present" {
			if changingEssenceWarp.romOffset == 0 {
				fatalErr("Bad rom offset for essence warp '" + destName + "'.")
			}
			for i := 0; i < changingEssenceWarp.len; i++ {
				rom.data[changingEssenceWarp.romOffset+i] = origEssenceWarp.vanillaData[i]
			}
		}

		// Update maptile for jewels
		if rom.game == gameSeasons {
			enterWarp.MapTile = warps[origEnter].vanillaMapTile
		}
	}

	if rom.game == gameSeasons {
		// shuffle treasure map data along with dungeons.
		changeTreasureMapTiles(rom.itemSlots, func(c chan tileChange) {
			for name, warp := range warps {
				if dungeonEntranceNameRegexp.MatchString(name) {
					c <- tileChange{warp.vanillaMapTile, warp.MapTile}
				}
			}
			close(c)
		})

		if dungeons {
			// Connect d2 stairs exits directly to each other
			enter1 := warps["d2 alt left entrance"]
			enter2 := warps["d2 alt right entrance"]
			exit1 := warps["d2 alt left exit"]
			exit2 := warps["d2 alt right exit"]

			rom.data[exit1.romOffset+0] = enter2.vanillaData[0]
			rom.data[exit1.romOffset+1] = enter2.vanillaData[1]
			rom.data[exit2.romOffset+0] = enter1.vanillaData[0]
			rom.data[exit2.romOffset+1] = enter1.vanillaData[1]

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
	fileSelectRow1 := stringToTiles(strings.ToUpper(ternary(len(version) <= 16,
		fmt.Sprintf("%s", version),
		fmt.Sprintf("%16s", version)[:16]).(string)))
	fileSelectRow2 := stringToTiles(
		strings.ToUpper(strings.ReplaceAll(row2, "-", " ")))

	writeLine := func(row []byte, addr int) {
		padding := 16 - len(row)
		copy(rom.data[addr+padding/2:addr+padding/2+32], row)
	}

	tileAddr := rom.lookupLabel("randoFileSelectStringTiles").fullOffset() + 2
	writeLine(fileSelectRow1, tileAddr)
	writeLine(fileSelectRow2, tileAddr+32)
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
	} else { // Ages
		shopMap["randoText_wildTokayGame"] = "wild tokay game"
		shopMap["randoText_goronShootingGallery"] = "goron shooting gallery"
	}
	for codeName, slotName := range shopMap {
		addr := rom.lookupLabel(codeName).fullOffset()
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
		config |= 1 << rom.lookupDefinition("RANDO_CONFIG_KEYSANITY")
	}
	if ropts.dungeons {
		config |= 1 << rom.lookupDefinition("RANDO_CONFIG_DUNGEON_ENTRANCES")
	}
	if ropts.autoMermaid {
		config |= 1 << rom.lookupDefinition("RANDO_CONFIG_MERMAID_AUTO")
	}

	addr := rom.lookupLabel("randoConfig").fullOffset()
	rom.data[addr] = config
}
