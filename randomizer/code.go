package randomizer

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v2"
)

// loaded from yaml, then converted to asm.
type asmData struct {
	filename string
	Common   yaml.MapSlice
	Floating yaml.MapSlice
	Seasons  yaml.MapSlice
	Ages     yaml.MapSlice
}

var articleRegexp = regexp.MustCompile("^(an?|the) ")

// return a map of internal item names to text that should be displayed for the
// item in shops.
func loadShopNames(game string) map[string]string {
	m := make(map[string]string)

	// load names used for owl hints
	itemFiles := []string{
		"/hints/common_items.yaml",
		fmt.Sprintf("/hints/%s_items.yaml", game),
	}
	for _, filename := range itemFiles {
		if err := yaml.Unmarshal(
			FSMustByte(false, filename), m); err != nil {
			panic(err)
		}
	}

	// remove articles
	for k, v := range m {
		m[k] = articleRegexp.ReplaceAllString(v, "")
	}

	return m
}
