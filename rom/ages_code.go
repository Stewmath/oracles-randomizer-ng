package rom

func newAgesRomBanks() *romBanks {
	r := romBanks{
		endOfBank: make([]uint16, 0x40),
	}

	r.endOfBank[0x00] = 0x3ef8
	r.endOfBank[0x02] = 0x7e93
	r.endOfBank[0x03] = 0x7ebd
	r.endOfBank[0x04] = 0x7edb
	r.endOfBank[0x05] = 0x7d9d
	r.endOfBank[0x10] = 0x7ef4
	r.endOfBank[0x12] = 0x7e8f

	return &r
}

func initAgesEOB() {
	r := newAgesRomBanks()

	// bank 00

	// don't play any music if the -nomusic flag is given. because of this,
	// this *must* be the first function at the end of bank zero (for now).
	r.appendToBank(0x00, "no music func",
		"\x67\xfe\x40\x30\x03\x3e\x08\xc9\xf0\xb7\xc9")
	r.replace(0x00, 0x0c9a, "no music call",
		"\x67\xf0\xb7", "\x67\xf0\xb7") // modified only by SetNoMusic()

	// bank 02

	// warp to ember tree if holding start when closing the map screen.
	treeWarp := r.appendToBank(0x02, "tree warp",
		"\xfa\x81\xc4\xe6\x08\x28\x1b"+ // close as normal if start not held
			"\xfa\x2d\xcc\xfe\x02\x38\x06"+ // check if indoors
			"\x3e\x5a\xcd\x98\x0c\xc9"+ // play error sound and ret
			"\x21\xb7\xcb\x36\x02\xb7\x28\x02\x36\x03"+ // set tree based on age
			"\xaf\xcd\xac\x5f\xc3\xba\x4f") // close + warp
	r.replaceMultiple([]Addr{{0x02, 0x6133}, {0x02, 0x618b}}, "tree warp jump",
		"\xc2\xba\x4f", "\xc4"+treeWarp)

	// warp to room under cursor if wearing developer ring.
	devWarp := r.appendToBank(0x02, "dev ring warp func",
		"\xfa\xcb\xc6\xfe\x40\x20\x12\xfa\x2d\xcc\xfe\x02\x30\x0b\xf6\x80"+
			"\xea\x47\xcc\xfa\xb6\xcb\xea\x48\xcc\x3e\x03\xcd\xad\x0c\xc9")
	r.replace(0x02, 0x5fcc, "dev ring warp call", "\xad\x0c", devWarp)

	// bank 03

	// allow skipping the capcom screen after one second by pressing start
	skipCapcom := r.appendToBank(0x03, "skip capcom func",
		"\xe5\xfa\xb3\xcb\xfe\x94\x30\x03\xcd\x86\x08\xe1\xcd\x37\x02\xc9")
	r.replace(0x03, 0x4d6c, "skip capcom call", "\x37\x02", skipCapcom)

	// set intro flags to skip opening
	skipOpening := r.appendToBank(0x03, "skip opening",
		"\xcd\xf9\x31\x3e\x0a\xcd\xf9\x31"+ // set global flags
			"\xe5\x21\x7a\xc7\xcb\xf6\x2e\x6a\xcb\xf6"+ // set room flags
			"\x2e\x59\xcb\xf6\x2e\x39\x36\xc8\xe1\xc9") // set more room flags
	r.replace(0x03, 0x6e97, "call skip opening",
		"\xc3\xf9\x31", "\xc3"+skipOpening)

	// bank 04

	// look up tiles in custom replacement table after loading a room. the
	// format is (group, room, YX, tile ID), with ff ending the table. if bit 0
	// of the room is set, no replacements are made.
	tileReplaceTable := r.appendToBank(0x04, "tile replace table",
		"\x01\x48\x45\xd7"+ // portal south of past maku tree
			"\x03\x0f\x66\xf9"+ // water in d6 past entrance
			"\x04\x1b\x03\x78"+ // key door in D1
			"\xff")
	tileReplaceFunc := r.appendToBank(0x04, "tile replace body",
		"\xcd\x7d\x19\xe6\x01\x20\x28"+
			"\xc5\x21"+tileReplaceTable+"\xfa\x2d\xcc\x47\xfa\x30\xcc\x4f"+
			"\x2a\xfe\xff\x28\x16\xb8\x20\x0e\x2a\xb9\x20\x0b"+
			"\xd5\x16\xcf\x2a\x5f\x2a\x12\xd1\x18\xea"+
			"\x23\x23\x23\x18\xe5\xc1\xcd\xef\x5f\xc9")
	r.replace(0x00, 0x38c0, "tile replace call",
		"\xcd\xef\x5f", "\xcd"+tileReplaceFunc)

	// bank 05

	// if wearing dev ring, jump over any tile like a ledge by pressing B with
	// no B item equipped.
	cliffLookup := r.appendToBank(0x05, "cliff lookup",
		"\xf5\xfa\xcb\xc6\xfe\x40\x20\x13"+ // check ring
			"\xfa\x88\xc6\xb7\x20\x0d"+ // check B item
			"\xfa\x81\xc4\xe6\x02\x28\x06"+ // check input
			"\xf1\xfa\x09\xd0\x37\xc9"+ // jump over ledge
			"\xf1\xc3\x1f\x1e") // jp to normal lookup
	r.replace(0x05, 0x6083, "call cliff lookup",
		"\xcd\x1f\x1e", "\xcd"+cliffLookup)

	// bank 10

	// override heart piece with starting item on starting item screen.
	foundItemLookup := r.appendToBank(0x10, "found item lookup",
		"\x01\x00\x2b\xfa\x2d\xcc\xb7\xc0\xfa\x30\xcc\xfe\x39\xc0"+
			"\x01\x00\x05\xc9")
	r.replace(0x10, 0x74bd, "call found item lookup",
		"\x01\x00\x2b", "\xcd"+foundItemLookup)

	// bank 12

	// replace nayru intro screen interaction with starting item.
	startingItem := r.appendToBank(0x12, "starting item",
		"\xf2\xdc\x07\x28\x78\xfe")
	r.replace(0x12, 0x5b06, "starting item pointer",
		"\xf1\x6b\x01", "\xf3"+startingItem)
}
