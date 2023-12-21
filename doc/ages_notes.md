# Oracle of Ages randomization notes

These notes apply specifically to Ages, alongside the game-neutral ones in
[README.md](https://github.com/jangler/oracles-randomizer/blob/master/README.md).


## Randomization

- The harp is progressive, starting with the Tune of Echoes, then Currents,
  then Ages.
- The trading sequence is removed. The second sword is in the item pool, and
  the Poe in Yoll Graveyard gives a randomized item.
- The red soldier who takes you to Ambi's palace for bombs waits in Deku Forest
  instead. Talk to him to trade mystery seeds for an item.
- The first **and** second prizes for Target Carts are randomized.
- The first **and** second prizes for Goron Dancing are randomized. The first
  prize (corresponding to the goron bracelet) can be obtained in either the
  present or in the past at any time. The second prize (corresponding to the
  mermaid key) can be obtained only in the past, after obtaining the Letter of
  Recommendation in addition to the first prize.
- Other than Goron Dancing and Target Carts, only the first item for each other
  minigame is randomized, with the exception of the Lynna Village Shooting
  Gallery, which has no randomized prize since its normal prize is a strange
  flute.
- Past and present versions of the same mystical seed tree grow the same type
  of seed.
- The secret shop items are not randomized.


## Other notable changes

- The intro sequence is removed. Your first item is found on the screen where
  Impa normally gives you the sword.
- The time portals on the screens adjacent to the Maku Tree are active
  permanently.
- The Tokays on Crescent Island do not steal your items, and the raft does not
  encounter a storm in the Sea of Storms.
- Playing the Tune of Currents triggers reentry into a return portal Link is
  standing on. This is useful if you warp into a patch of bushes without a
  bush-breaking item.
- Ambi's palace courtyard is open from the beginning.


## Logic

"Logic" means the set of plays that the randomizer may require you to make in
order to progress in the game. Anything the vanilla game requires you to do is,
of course, in logic. Beyond that and some easy alternate tactics, the rules in
this document apply.


### Normal logic

In logic:

- Jumping over 2 tiles with feather, 3 tiles with feather + pegasus seeds, 4
  tiles with cape, or 6 tiles with cape + pegasus seeds. Subtract 1 tile if the
  jump is over water or lava instead of a pit. These rules apply to jumps
  directly up/down/left/right, while diagonal jumps depend on the situation.
- Using only ember seeds from the seed satchel as a weapon
- Using only ember, scent, or gale seeds from the seed shooter as a weapon
- Using expert's ring as a weapon
- Using thrown objects (bushes, pots) as weapons
- Flipping spiked beetles using the shovel
- Farming rupees for minigames

Out of logic:

- Required damage (damage boosts, falling in pits)
- Farming rupees for shop purchases
- Getting initial scent seeds from the plants in D3
- Getting initial bombs from Goron Shooting Gallery or Head Thwomp
- Lighting torches using mystery seeds
- Using only mystery seeds as a weapon
- Using only scent or gale seeds from the seed satchel as a weapon
- Using only bombs as a weapon, except versus enemies immune to sword
- Using fist ring as a weapon
- Carrying bushes or pots between rooms
- Doing Patch's restoration ceremony without sword
- Getting a potion for King Zora from Maple instead of Syrup
- Guard skip
- Text warps
- Magic rings from non-randomized locations
- Linked secrets


### Hard logic

Choosing hard difficulty enables things that are out of normal logic, with the
exception of:

- Farming rupees for shop purchases, except by shovel RNG manips
- Mystery seeds as a weapon
- Bombs as a weapon, except versus enemies immune to sword
- Double damage boosts
- Text warps
- Clipping into fairies' woods by switch hooking the Octorok
- Trading scent seeds for a shield without having a seed item
- Magic rings from non-randomized locations
- Linked secrets

See
[ages_hard_guide.md](https://github.com/jangler/oracles-randomizer/blob/master/doc/ages_hard_guide.md)
for more information on specific tricks in hard logic.


### Crossitems

When porting Ages items into Seasons, it was sometimes necessary to arbitrarily
decide which items can damage which enemies. This is mainly the case with bosses
which tend to have their own unique damage tables.

Fool's Ore: Should work on anything that a normal sword slash works on,
including Blue Stalfos (D8 miniboss).

Magic Boomerang: Can damage gibdos as in Seasons. It should not damage any
bosses beyond what the L-1 boomerang already does. (It works on Angler Fish for
example.)

Rod of Seasons:
- Damages the same undead enemies as in seasons (stalfos, ghini, gibdos).
- Also damages D1 bosses Giant Ghini and Pumpkin Head, as they're also undead.
