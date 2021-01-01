package randomizer

import (
	"fmt"
	"log"
	"os"
	"bufio"
	"strings"
	"strconv"
)

func readSymbolFile(filename string) map[string]address {
	symbolPanic := func(line string) {
		panic(fmt.Sprintf("Error parsing symbol file \"%s\": \"%s\"", filename, line))
	}

	retval := make(map[string]address)

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(fmt.Sprintf("Couldn't open symbol file \"%s\".", filename))
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.Trim(scanner.Text(), " \t")
		if strings.HasPrefix(s, ";") || strings.HasPrefix(s, "[") || len(s) == 0 {
			continue
		}
		list := strings.Split(s, " ")
		if len(list) != 2 {
			symbolPanic(s)
		}
		addr := strings.Split(list[0], ":")
		if len(addr) != 2 {
			// Ignore non-ROM addresses
			continue
		}
		bank, _ := strconv.ParseUint(addr[0], 16, 8)
		offset, _ := strconv.ParseUint(addr[1], 16, 16)
		retval[list[1]] = address{uint8(bank), uint16(offset)}
	}

	return retval
}
