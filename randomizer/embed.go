package randomizer

import (
	"embed"
	"io/fs"

	"gopkg.in/yaml.v2"
)

//go:embed hints/* logic/* romdata/*
var embeddedFS embed.FS

func ReadEmbeddedYaml(filename string, out interface{}) error {
	b, err := embeddedFS.ReadFile(filename)
	if err != nil {
		f, _ := ReadEmbeddedDir("")
		print('a')
		print(f[0])
		print('b')
		panic(err)
	}
	return yaml.Unmarshal(b, out)
}

func ReadEmbeddedDir(filename string) ([]fs.DirEntry, error) {
	return embeddedFS.ReadDir(filename)
}
