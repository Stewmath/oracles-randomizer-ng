package randomizer

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// an item slot (chest, gift, etc). it references room data and treasure data.
type itemSlot struct {
	treasure              *treasure

	hasAddr               bool
	addr                  *address // address of item slot data (only if hasAddr is true)
	label                 string   // label referencing item slot

	group, room, player   byte
	moreRooms             []uint16 // high = group, low = room
	mapTile               uint16   // overworld map coords (includes group)
	localOnly             bool     // multiworld
}

// implementes `mutate` from the `mutable` interface.
func (mut *itemSlot) mutate(b []byte) {
	if mut.hasAddr {
		b[mut.addr.fullOffset()] = mut.treasure.id
		b[mut.addr.fullOffset()+1] = mut.treasure.subid
	}
}

// helper function for itemSlot.check()
func checkByte(b []byte, addr *address, value byte) error {
	if b[addr.fullOffset()] != value {
		return fmt.Errorf("expected %x at %x; found %x",
			value, addr.fullOffset(), b[addr.fullOffset()])
	}
	return nil
}

// implements `check()` from the `mutable` interface.
func (mut *itemSlot) check(b []byte) error {
	// TODO
	return nil

	/*
	// skip zero addresses
	if len(mut.idAddrs) == 0 || mut.idAddrs[0].offset == 0 {
		return nil
	}

	// only check ID addresses, since situational variants and progressive
	// items mess with everything else.
	for _, addr := range mut.idAddrs {
		if err := checkByte(b, addr, mut.treasure.id); err != nil {
			return err
		}
	}

	return nil
	*/
}

// raw slot data loaded from yaml.
type rawSlot struct {
	// required
	Treasure string
	Room     uint16

	// required if not equal to room and not in dungeon.
	MapTile *uint16

	// pick one
	Label string
	Dummy bool
	Tree  bool

	// optional additional rooms
	MoreRooms []uint16

	Local bool // dummy implies true
}

var seasonsDungeonMapTiles = map[string]uint16 {
	"d0": 0x0d4,
	"d1": 0x096,
	"d2": 0x08d,
	"d3": 0x060,
	"d4": 0x01d,
	"d5": 0x08a,
	"d6": 0x000,
	"d7": 0x0d0,
	"d8": 0x100,
}

var agesDungeonMapTiles = map[string]uint16 {
	"d1": 0x08d,
	"d2": 0x183, // TODO: Past & present entrances
	"d3": 0x0ba,
	"d4": 0x003,
	"d5": 0x00a,
	"d6 past":    0x13c,
	"d6 present": 0x03c,
	"d7": 0x090,
	"d8": 0x15c,
}


// return a map of slot names to slot data. if romState.data is nil, only
// "static" data is loaded.
func (rom *romState) loadSlots() map[string]*itemSlot {
	raws := make(map[string]*rawSlot)

	filename := fmt.Sprintf("/romdata/%s_slots.yaml", gameNames[rom.game])
	if err := yaml.Unmarshal(
		FSMustByte(false, filename), raws); err != nil {
		panic(err)
	}

	m := make(map[string]*itemSlot)
	for name, raw := range raws {
		if raw.Room == 0 && !raw.Dummy {
			panic(name + " room is zero")
		}

		slot := &itemSlot{
			treasure:  rom.treasures[raw.Treasure],
			group:     byte(raw.Room >> 8),
			room:      byte(raw.Room),
			moreRooms: raw.MoreRooms,
			localOnly: raw.Local || raw.Dummy,
		}

		if raw.MapTile != nil {
			slot.mapTile = *raw.MapTile
		} else { // unspecified map tile = assume either overworld or dungeon
			dungeonMapTiles := sora(rom.game,
				seasonsDungeonMapTiles, agesDungeonMapTiles).(map[string]uint16)
			dungeonName := getDungeonName(name)
			if tile, ok := dungeonMapTiles[dungeonName]; ok {
				slot.mapTile = tile
			} else if slot.group < 2 { // Overworld / Subrosia
				slot.mapTile = raw.Room
			} else {
				panic(fmt.Sprintf("need a maptile for %s", name))
			}
		}

		if raw.Label != "" {
			slot.label = raw.Label
			if val, ok := rom.symbols[slot.label]; ok {
				slot.addr = val
				slot.hasAddr = true
			} else {
				panic(fmt.Sprintf("label \"%s\" not found for slot \"%s\".", slot.label, name))
			}
		} else if raw.Tree {
		} else if raw.Dummy {
		} else {
			panic(fmt.Sprintf("invalid raw slot: %s: %#v", name, raw))
		}

		m[name] = slot
	}

	return m
}
