package randomizer

import (
	"fmt"
)

// an instance of ROM data that can be changed by the randomizer.
type mutable interface {
	mutate([]byte)      // change ROM bytes
	check([]byte) error // verify that the mutable matches the ROM
}

// a length of mutable bytes starting at a given address.
type mutableRange struct {
	addr     address
	old, new []byte
}

// implements `mutate()` from the `mutable` interface.
func (mut *mutableRange) mutate(b []byte) {
	offset := mut.addr.fullOffset()
	for i, value := range mut.new {
		b[offset+i] = value
	}
}

// implements `check()` from the `mutable` interface.
func (mut *mutableRange) check(b []byte) error {
	offset := mut.addr.fullOffset()
	for i, value := range mut.old {
		if b[offset+i] != value {
			return fmt.Errorf("expected %x at %x; found %x",
				mut.old[i], offset+i, b[offset+i])
		}
	}
	return nil
}

// add a ranged mutable to the rom's codeMutables list
func (rom *romState) addMutableRange(symbol string, old []byte, new []byte) {
	if rom.data == nil {
		return
	}
	addr := rom.lookupLabel(symbol)
	if _, ok := rom.codeMutables[symbol]; ok {
		panic(fmt.Sprintf("Tried to add mutable \"%s\" to rom multiple times", symbol))
	}
	rom.codeMutables[symbol] = &mutableRange{*addr, old, new}
}

func (rom *romState) addMutableByte(symbol string, old byte, new byte) {
	rom.addMutableRange(symbol, []byte{old}, []byte{new})
}

// add mutables to "rom.codeMutables" (generally the "new" values will be
// overwritten later on)
func (rom *romState) initializeMutables() {
	rom.codeMutables = make(map[string]*mutableRange)

	rom.addMutableByte("randovar_animalCompanion", 0x0b, 0x00)

	defaultSeasonTable := []byte{
		0x00,0x00,0x00,0x00,0x00,0x00,0x03,0x00,0x00,0x01,0x00,0x00,0x00,0x00,0x00,0x00,
		0x03,0x02,0x01,0x02,0x00,0x01,0x03,0x02,0x00,0x01,0x00,0x03,0x03,0x03}
	rom.addMutableRange("roomPackSeasonTable", defaultSeasonTable, defaultSeasonTable)
}

// sets the natzu region based on a companion number 1 to 3.
func (rom *romState) setAnimal(companion int) {
	rom.codeMutables["randovar_animalCompanion"].new = []byte{byte(companion + 0x0a)}
}

// key = area name (as in asm/vars.yaml), id = season index (spring -> winter).
func (rom *romState) setSeason(area string, id byte) {
	roomPack, ok := seasonAreaRoomPacks[area]
	if !ok {
		panic(fmt.Sprintf("Room pack for area \"%s\" not defined?", area))
	}
	rom.codeMutables["roomPackSeasonTable"].new[roomPack] = id
}

// get a collated map of all mutables.
func (rom *romState) getAllMutables() map[string]mutable {
	allMutables := make(map[string]mutable)
	for k, v := range rom.itemSlots {
		if v.treasure == nil {
			panic(fmt.Sprintf("treasure for %s is nil", k))
		}
		addMutOrPanic(allMutables, k, v)
	}
	for k, v := range rom.treasures {
		addMutOrPanic(allMutables, k, v)
	}
	for k, v := range rom.codeMutables {
		addMutOrPanic(allMutables, k, v)
	}
	return allMutables
}

// if the mutable does not exist in the map, add it. if it already exists,
// panic.
func addMutOrPanic(m map[string]mutable, k string, v mutable) {
	if _, ok := m[k]; ok {
		panic("duplicate mutable key: " + k)
	}
	m[k] = v
}

// returns the name of a mutable that covers the given address, or an empty
// string if none is found.
func (rom *romState) findAddr(bank byte, addr uint16) string {
	muts := rom.getAllMutables()
	offset := (&address{bank, addr}).fullOffset()

	for name, mut := range muts {
		switch mut := mut.(type) {
		case *mutableRange:
			if offset >= mut.addr.fullOffset() &&
				offset < mut.addr.fullOffset()+len(mut.new) {
				return name
			}
		case *itemSlot:
			if mut.hasAddr && offset == mut.addr.fullOffset() {
				return name
			}
		case *treasure:
			if offset >= mut.addr.fullOffset() &&
				offset < mut.addr.fullOffset()+4 {
				return name
			}
		default:
			panic("unknown type for mutable: " + name)
		}
	}

	return ""
}
