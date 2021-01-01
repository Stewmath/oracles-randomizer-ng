package randomizer

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v2"
)

// data associated with a particular item ID and sub ID.
type treasure struct {
	displayName string // this can change based on ring replacement etc
	id, subid   byte
	addr        address

	// in order, starting at addr
	mode   byte // collection mode (this usually has no effect since it's overridden)
	param  byte // parameter value to use for giveTreasure
	text   byte
	sprite byte
}

// returns a slice of consecutive bytes of treasure data, as they would appear
// in the ROM.
func (t treasure) bytes() []byte {
	return []byte{t.mode, t.param, t.text, t.sprite}
}

// implements `mutate()` from the `mutable` interface.
func (t treasure) mutate(b []byte) {
	// fake treasure
	if t.addr.offset == 0 {
		return
	}

	addr, data := t.addr.fullOffset(), t.bytes()
	for i := 0; i < 4; i++ {
		b[addr+i] = data[i]
	}
}

// implements `check()` from the `mutable` interface.
func (t treasure) check(b []byte) error {
	// fake treasure
	if t.addr.offset == 0 {
		return nil
	}

	addr, data := t.addr.fullOffset(), t.bytes()
	if bytes.Compare(b[addr:addr+4], data) != 0 {
		return fmt.Errorf("expected %x at %x; found %x",
			data, addr, b[addr:addr+4])
	}
	return nil
}

// returns the full offset of the treasure's four-byte entry in the rom.
func getTreasureAddr(b []byte, tableAddr address, id, subid byte) address {
	ptr := tableAddr

	ptr.offset += uint16(id) * 4
	if b[ptr.fullOffset()]&0x80 != 0 {
		ptr.offset = uint16(b[ptr.fullOffset()+1]) +
			uint16(b[ptr.fullOffset()+2])*0x100
	}
	ptr.offset += uint16(subid) * 4

	return ptr
}

// return a map of treasure names to treasure data. if b is nil, only "static"
// data is loaded.
func loadTreasures(b []byte, tableAddr address, game int) map[string]*treasure {
	allRawIds := make(map[string]map[string]uint16)
	if err := yaml.Unmarshal(
		FSMustByte(false, "/romdata/treasures.yaml"), allRawIds); err != nil {
		panic(err)
	}

	rawIds := make(map[string]uint16)
	for k, v := range allRawIds["common"] {
		rawIds[k] = v
	}
	for k, v := range allRawIds[gameNames[game]] {
		rawIds[k] = v
	}

	m := make(map[string]*treasure)
	for name, rawId := range rawIds {
		t := &treasure{
			displayName: name,
			id:          byte(rawId >> 8),
			subid:       byte(rawId),
		}

		if b != nil {
			t.addr = getTreasureAddr(b, tableAddr, t.id, t.subid)
			t.mode = b[t.addr.fullOffset()]
			t.param = b[t.addr.fullOffset()+1]
			t.text = b[t.addr.fullOffset()+2]
			t.sprite = b[t.addr.fullOffset()+3]
		}

		m[name] = t
	}

	// add dummy treasures for seed trees
	m["ember tree seeds"] = &treasure{id: 0x00}
	m["scent tree seeds"] = &treasure{id: 0x01}
	m["pegasus tree seeds"] = &treasure{id: 0x02}
	m["gale tree seeds"] = &treasure{id: 0x03}
	m["mystery tree seeds"] = &treasure{id: 0x04}

	return m
}
