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
	mapTile               byte     // overworld map coords, yx
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

	// required if not == low byte of room or in dungeon.
	MapTile *byte

	// pick one
	Label string
	Dummy bool
	Tree  bool

	// optional additional rooms
	MoreRooms []uint16

	Local bool // dummy implies true
}

// data that can be inferred from a room's music.
type musicData struct {
	MapTile byte
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

	/*
	allMusic := make(map[string](map[byte]musicData))
	if err := yaml.Unmarshal(
		FSMustByte(false, "/romdata/music.yaml"), allMusic); err != nil {
		panic(err)
	}
	musicMap := allMusic[gameNames[rom.game]]
	*/

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

		// TODO: mapTile
		/*
		// unspecified map tile = assume overworld
		if raw.MapTile == nil && rom.data != nil {
			musicIndex := getMusicIndex(rom.data, rom.game, slot.group, slot.room)
			if music, ok := musicMap[musicIndex]; ok {
				slot.mapTile = music.MapTile
			} else {
				// nope, definitely not overworld.
				if slot.group > 2 || (slot.group == 2 &&
					(slot.room&0x0f > 0x0d || slot.room&0xf0 > 0xd0)) {
					panic(fmt.Sprintf("invalid room for %s: %04x",
						name, raw.Room))
				}
				slot.mapTile = slot.room
			}
		} else if raw.MapTile != nil {
			slot.mapTile = *raw.MapTile
		}
		*/

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

// returns the music index for the given room.
func getMusicIndex(b []byte, game int, group, room byte) byte {
	// TODO
	return 0
	/*
	ptr := sora(game, address{0x04, 0x483c}, address{0x04, 0x495c}).(address)

	ptr.offset += uint16(group) * 2
	ptr.offset = uint16(b[ptr.fullOffset()]) +
		uint16(b[ptr.fullOffset()+1])*0x100
	ptr.offset += uint16(room)

	return b[ptr.fullOffset()]
	*/
}
