package randomizer

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"io"

	"gopkg.in/yaml.v2"
)

// generate a bunch of seeds.
func generateSeeds(n int, filename string, ropts randomizerOptions) []*routeInfo {
	threads := runtime.NumCPU()
	dummyLogf := func(string, ...interface{}) {}

	b, labels, definitions, game, err := readGivenRom(filename)
	if err != nil {
		fatal(err, printErrf)
		return nil
	}

	// search for routes
	routeChan := make(chan *routeInfo)
	attempts := 0
	for i := 0; i < threads; i++ {
		go func() {
			for i := 0; i < n/threads; i++ {
				for {
					// i don't know if a new romState actually *needs* to be
					// created for each iteration.
					seed := uint32(rand.Int())
					src := rand.New(rand.NewSource(int64(seed)))
					rom := newRomState(b, labels, definitions, game, 1, ropts.crossitems)
					route, _ := findRoute(rom, seed, src, ropts, false, dummyLogf)
					if route != nil {
						attempts += route.attemptCount
						routeChan <- route
						break
					}
				}
			}
		}()
	}

	// receive found routes
	routes := make([]*routeInfo, n/threads*threads)
	for i := 0; i < len(routes); i++ {
		routes[i] = <-routeChan
		fmt.Fprintf(os.Stderr, "%d routes found\n", i+1)
	}
	fmt.Fprintf(os.Stderr, "%.01f%% of seeds succeeded\n",
		100*float64(n)/float64(attempts))

	return routes
}

// generate a bunch of seeds and print item configurations in YAML format.
func logStats(trials int, filename string, ropts randomizerOptions,  logf logFunc) {
	_, _, _, game, _ := readGivenRom(filename)

	// get `trials` routes
	routes := generateSeeds(trials, filename, ropts)

	// make a YAML-serializable slice of check maps
	stringChecks := make([]map[string]string, len(routes))
	for i, ri := range routes {
		stringChecks[i] = make(map[string]string)
		for k, v := range getChecks(ri.usedItems, ri.usedSlots) {
			stringChecks[i][k.name] = v.name
		}
		if game == gameSeasons {
			for area, seasonId := range ri.seasons {
				// make sure not to overwrite info about lost woods item
				if area == "lost woods" {
					area = "lost woods (season)"
				}
				stringChecks[i][area] = seasonsById[int(seasonId)]
			}
		}
		stringChecks[i]["_seed"] = fmt.Sprintf("%08x", ri.seed)
	}

	// encode to stdout
	if err := yaml.NewEncoder(os.Stdout).Encode(stringChecks); err != nil {
		panic(err)
	}
}

func printStatPercentage(w io.Writer, success, trials int, name string) {
	fmt.Printf("%.1f%%\t%s\n", 100*float64(success)/float64(trials), name)
}

// generate a bunch of seeds and print how often a node is required
func logNodeStats(trials int, filename string, nodeName string, ropts randomizerOptions) {
	// get `trials` routes
	routes := generateSeeds(trials, filename, ropts)
	printStatPercentage(os.Stdout, getNodeStats(routes, nodeName), trials, nodeName)
}

// returns count of how often a node is required
func getNodeStats(routes []*routeInfo, nodeName string) int {
	count := 0

	for _, r := range routes {
		if !r.graph["done"].reached {
			fmt.Printf("NOT REACHED")
			continue
		}
		// create a "null" node that is never true, add it as a parent to our
		// node of interest, and make that node type "and" to ensure it's not
		// reachable.
		item := r.graph[nodeName]
		oldType := item.ntype
		item.ntype = andNode

		null := newNode("null", orNode)
		r.graph["null"] = null

		item.addParent(null)
		r.graph.reset()
		r.graph["start"].explore()

		if !r.graph["done"].reached {
			count++
		}

		item.removeParent(null)
		item.ntype = oldType
		r.graph.reset()
		r.graph["start"].explore()
	}

	return count
}
