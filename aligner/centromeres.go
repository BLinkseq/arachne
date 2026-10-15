package aligner

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Region struct {
	start int
	end   int
}

func loadCentromeres(filename *string) (map[string]Region, error) {
	toRet := map[string]Region{}
	if *filename == "" {
		return toRet, nil
	}
	file, err := os.Open(*filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open centromeres file: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "track"):
			continue
		case strings.HasPrefix(line, "browser"):
			continue
		case strings.HasPrefix(line, "#"):
			continue
		default:
			tokens := strings.Split(line, "\t")
			if len(tokens) < 3 {
				continue
			}
			chrom := tokens[0]
			start, err := strconv.Atoi(tokens[1])
			if err != nil {
				continue
			}
			end, err := strconv.Atoi(tokens[2])
			if err != nil {
				continue
			}
			if start > end {
				return nil, fmt.Errorf("a row in the centromeres file has start > end: %v\t%v\t%v", chrom, start, end)
			}
			toRet[chrom] = Region{start: start, end: end}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to read centromeres file: %w", err)
	}
	return toRet, nil
}

// inCentromere reports whether pos falls within contig's known centromere
// region. Regions use standard BED half-open semantics: inclusive start,
// exclusive end. Returns false if contig has no centromere entry.
func inCentromere(centromeres map[string]Region, contig string, pos int64) bool {
	region, ok := centromeres[contig]
	if !ok {
		return false
	}
	return pos >= int64(region.start) && pos < int64(region.end)
}
