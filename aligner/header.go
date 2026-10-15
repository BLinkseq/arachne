package aligner

import (
	"fmt"
	"os"
	"strings"
	"time"

	"arachne/gominibwa"

	"github.com/biogo/hts/sam"
)

// Return a SAM header built from the reference, sample ID, and arachne version
func buildHeader(idx *gominibwa.Index, sampleid, version string) (*sam.Header, map[string]*sam.Reference, error) {
	contigs := make(map[string]*sam.Reference)
	references := make([]*sam.Reference, 0)

	names, lengths := idx.Contigs()
	for i, name := range names {
		r, err := sam.NewReference(name, name, "NA", int(lengths[i]), nil, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid reference contig %q: %w", name, err)
		}
		references = append(references, r)
		contigs[name] = r
	}

	h, err := sam.NewHeader([]byte(""), references)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to build SAM header: %w", err)
	}

	rg, err := sam.NewReadGroup(
		sampleid,   //ID
		"",         //CN
		"",         //DS
		"",         //LB
		"",         //PG
		"ILLUMINA", //PL
		"",         //PU: just make same as ID?
		sampleid,   //SM
		"",
		"",
		time.Now(),
		0)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid sample ID %q: %w", sampleid, err)
	}
	h.AddReadGroup(rg)

	// Add a program line for arachne
	prog := sam.NewProgram(
		"arachne",                  // ID
		"arachne",                  // PN
		strings.Join(os.Args, " "), // CL
		"",                         // PP - empty bc Arachne produces the initial SAM
		version)                    // VN
	h.AddProgram(prog)
	return h, contigs, nil
}

// Sum of all contig lengths in the reference.
func totalReferenceLength(contigs map[string]*sam.Reference) float64 {
	total := 0.0
	for _, c := range contigs {
		total += float64(c.Len())
	}
	return total
}
