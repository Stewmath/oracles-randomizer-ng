# Zelda Oracles Randomizer NG

This program reads a Zelda: Oracle of Seasons or Oracle of Ages ROM, shuffles
the locations of (most) items and mystical seeds, and writes the modified ROM to
a new file. In Seasons, the default seasons for each area are also randomized.
Most arbitrary overworld checks for essences and other game flags are removed,
so the dungeons and other checks can be done in any order that the randomized
items facilitate. However, you do need to collect all 8 essences to get the Maku
Seed and finish the game.

This is a fork of jangler's [original
randomizer](https://github.com/jangler/oracles-randomizer). See
[ng\_differences.md](doc/ng_differences.md) for a list of differences.


## Usage

At this time, this version of the randomizer only works on the commandline, and
you must be able to build the disassembly first. Non-technical users can use the
[web interface](http://oosarando.zeldahacking.net/) instead.

To run it locally: You must first build oracles-disasm from the submodule in
this project, which produces modified oracles ROMs. Then, run
`./oracles-randomizer-ng oracles-disasm/seasons.gbc <output rom>`. Only Seasons
is supported at the moment.

When building the disassembly, there will be a file named `seasons.sym` created
along with `seasons.gbc`. It is very important that this file is in the same
directory as `seasons.gbc`, otherwise this won't work.


## Download

Unavailable at the moment.


## Randomization notes

General details common to both games:

- Items and chests are randomized, with these exceptions:
    - Renewable shop and business scrub items (bombs, shield, hearts, etc.)
	- Gasha seeds and pieces of heart outside of chests
	- NPCs that give non-progression items in the vanilla game
	- Gasha nut contents
	- Fixed drops (from bushes, pots, etc.)
	- Maple drops
	- Linked secrets
- Mystical seed trees are randomized, with no more than two trees of each type.
  Items that use seeds for ammunition start with the type of seed that's on the
  Horon Village or Lynna City tree.
- For items that have two levels, the first you obtain will be L-1, and the
  second will be L-2, regardless of the order in which you obtain them. The L-2
  shield is an exception.
- There is one flute in the game for a random animal companion, and it's
  identified and usable as soon as you get it. Only the 150-rupee item in the
  shop is randomized; the other two usual means of getting a strange flute
  don't give anything special. The animal companion regions (Natzu in Seasons
  and Nuun in Ages) match whatever flute is in the seed.
- Rings are instantly appraised when you get them, and the ring list can be
  accessed from the inventory ring box icon. For convenience, the L-3 ring box
  is given at the start. The punch rings can be used with only one equip slot
  empty.
- If tree warp is enabled, holding start while closing the map screen outdoors
  warps to the seed tree in Horon Village or Lynna City. Tree warp comes with
  no warranty and is not supported as a "feature", so think carefully before
  using it.

For game-specific notes on randomization and logic, see
[seasons_notes.md](https://github.com/jangler/oracles-randomizer/blob/master/doc/seasons_notes.md)
and
[ages_notes.md](https://github.com/jangler/oracles-randomizer/blob/master/doc/ages_notes.md).

See
[multiworld.md](https://github.com/jangler/oracles-randomizer/blob/master/doc/multiworld.md)
for information on multiworld seeds.

See
[plan.md](https://github.com/jangler/oracles-randomizer/blob/master/doc/plan.md)
for information on plando generation.


## FAQ

**Q: Is there a place to discuss the randomizer?**

A: Yes, the [Oracles Discord server](https://discord.gg/pyBEbz5). The server is
mainly focused on speedrunning, but randomizer-specific channels exist as well.

**Q: I found a problem. What do I do?**

A: Open an issue about it on GitHub or bring it up in a randomizer channel in
the Oracles discord. Provide your seed's log file either way.

**Q: Can I do a linked game?**

A: Linked games are currently disabled. They will probably be re-enabled later
after being tested properly but it's low-priority.


## Thanks to:

- Jangler for the [original oracles-randomizer](https://github.com/jangler/oracles-randomizer).
- Herreteman, dragonc0, Phoenomenom714, and jaysee87 for help with logic,
  playtesting, design, and "customer support".
- Everyone who helped playtest prerelease versions of the randomizer.
