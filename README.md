# Zelda Oracles Randomizer NG

This program reads a Zelda: Oracle of Seasons or Oracle of Ages ROM, shuffles
the locations of (most) items and mystical seeds, and writes the modified ROM to
a new file.

You can get more information and generate seeds using the [Web UI](https://oosarando.zeldahacking.net/).


## Development

The simplest way to set things up is to use the instructions in the [Web UI
repository](https://github.com/Stewmath/oracles-randomizer-ng-webui). The
"build-rando" command outlined there will build both the required baseroms in
the oracles-disasm submodule, and the oracles-randomizer-ng executable file
itself.

Alternatively, to compile oracles-randomizer-ng on its own, run:

```
go generate
go build
```

Then, assuming the baseroms from the oracles-disasm submodule have already been
built, you can generate a seed from the commandline with:

```
./oracles-randomizer-ng oracles-disasm/<ages|seasons>.gbc <output>.gbc TODO
```

In the oracles-disasm folder, there should be a file named `seasons.sym` (or
`ages.sym`) created along with `seasons.gbc`. It is very important that this
file is in the same directory as `seasons.gbc`, otherwise this won't work.


## For more information

- [WebUI Documentation](https://oosarando.zeldahacking.net/info) 
- [Differences in oracles-randomizer-ng](doc/ng_differences.md)
- [Seasons notes](doc/seasons_notes.md)
- [Ages notes](doc/ages_notes.md)
