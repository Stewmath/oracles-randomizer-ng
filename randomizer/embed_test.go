package randomizer

import (
	"strings"
	"testing"
)

func TestLoadYaml(t *testing.T) {
	// make sure all yaml files are well-formed.
	dirnames := []string{"hints", "logic", "romdata"}
	for _, dirname := range dirnames {
		// get list of files in directory
		files, err := ReadEmbeddedDir(dirname)
		if err != nil {
			t.Fatal(err)
		}

		for _, file := range files {
			// ignore non-yaml files like readmes
			if !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}

			path := dirname + "/" + file.Name()

			// either a slice or a map should work
			m := make(map[interface{}]interface{})
			mapErr := ReadEmbeddedYaml(path, &m)
			a := make([]interface{}, 0)
			sliceErr := ReadEmbeddedYaml(path, &a)

			if mapErr != nil && sliceErr != nil {
				t.Errorf("failed to unmarshal %s into map or slice", path)
				t.Error(mapErr)
				t.Error(sliceErr)
			}
		}
	}
}
