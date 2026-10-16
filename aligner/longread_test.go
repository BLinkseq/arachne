package aligner

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"arachne/fastqreader"
	"arachne/gominibwa"

	sam "github.com/biogo/hts/sam"
)

// Single-end (long) reads are not part of a pair, so they carry no improper
// pair penalty and are unmapped only when they have no position.
func TestSingleEndScoringAndUnmapped(t *testing.T) {
	setEMGlobals()
	paired := &Alignment{mismatches: 3, pos: 100, score: 500}
	single := &Alignment{mismatches: 3, pos: 100, score: 500, single_end: true}
	if got := scoreAlignment(paired, nil, 0); got != -6-4 {
		t.Errorf("paired read without a mate scored %v, want -6 mismatches and -4 improper pair penalty", got)
	}
	if got := scoreAlignment(single, nil, 0); got != -6 {
		t.Errorf("single-end read scored %v, want -6 with no improper pair penalty", got)
	}
	finishAlignment(single)
	if single.log_alignment_probability != -6 {
		t.Errorf("single-end log_alignment_probability = %v, want -6", single.log_alignment_probability)
	}
	// short-read rule: a lone low-scoring read is unmapped; a long read is not
	paired.score, single.score = 20, 20
	if !paired.IsUnmapped() || single.IsUnmapped() {
		t.Errorf("IsUnmapped: paired %v (want true), single-end %v (want false)", paired.IsUnmapped(), single.IsUnmapped())
	}
	single.pos = -1
	if !single.IsUnmapped() {
		t.Error("a single-end read without a position must be unmapped")
	}
}

// Scores over a long read fall far below -308 (log10), where exp underflows.
// MAPQ estimation must stay a number in 0..60.
func TestEstimateMapQualitiesExtremeScores(t *testing.T) {
	setEMGlobals()
	noDebug := false
	debugPrintMove = &noDebug
	for _, c := range []struct {
		name       string
		mmA, mmB   int
		wantMapq   int
		wantActive int
	}{
		{"clear winner", 3000, 3010, 60, 0},
		{"tie", 3000, 3000, 3, 0},
	} {
		f := &emFixture{}
		for _, mm := range []int{c.mmA, c.mmB} {
			a := f.add(0, "chr1", int64(mm), false)
			a.single_end, a.mismatches = true, mm
			// a 20 kb read: the score of leaving it unplaced scales with length
			long := make([]byte, 20000)
			a.read_seq = &long
		}
		ph := f.add(1, "", -1, false)
		ph.single_end = true
		f.alignments[0][c.wantActive].active = true
		ph.active = true
		estimateMapQualities(f.alignments, nil, -4)
		if got := f.alignments[0][c.wantActive].mapq; got != c.wantMapq {
			t.Errorf("%s: MAPQ = %d, want %d", c.name, got, c.wantMapq)
		}
	}
}

// A long read always has many mismatches; the short-read MAPQ ceiling would
// pin it at 0.
func TestEMCeilingDoesNotApplyToSingleEnd(t *testing.T) {
	c := DefaultEMConfig(-4)
	aln := &Alignment{mismatches: 500, single_end: true}
	if got := c.scoreMapq(aln); got != 60 {
		t.Errorf("single-end ceiling = %v, want 60", got)
	}
	aln.single_end = false
	if got := c.scoreMapq(aln); got >= 0 {
		t.Errorf("short-read ceiling with 500 mismatches = %v, want it below 0", got)
	}
}

func TestBuildRecordSingleEnd(t *testing.T) {
	addComments := false
	AddComments = &addComments
	debugTags := false
	contigs := specTestContigs(t, "chr1", "chr2")

	primary := newSAMSpecTestAlignment("lr1", "chr1", 100, 110, 40)
	primary.single_end, primary.mate_id, primary.duplicate = true, 1, true
	split := newSAMSpecTestAlignment("lr1", "chr2", 500, 510, 30)
	split.single_end, split.mate_id = true, 1
	primary.secondary = split

	rec := buildRecord(primary, primary, &debugTags, contigs)
	for _, bad := range []sam.Flags{sam.Paired, sam.ProperPair, sam.MateUnmapped, sam.MateReverse, sam.Read1, sam.Read2} {
		if rec.Flags&bad != 0 {
			t.Errorf("single-end record has paired flag %v set (flags %#x)", bad, int(rec.Flags))
		}
	}
	if rec.Flags&sam.Duplicate == 0 {
		t.Error("duplicate flag was not written on a single-end record")
	}
	if rec.MatePos != -1 || rec.MateRef != nil || rec.TempLen != 0 {
		t.Errorf("mate fields set on a single-end record: pos %d ref %v tlen %d", rec.MatePos, rec.MateRef, rec.TempLen)
	}

	rec = buildRecord(split, primary, &debugTags, contigs)
	if rec.Flags&sam.Supplementary == 0 || rec.Flags&sam.Secondary != 0 {
		t.Errorf("long-read split record flags %#x, want supplementary and not secondary", int(rec.Flags))
	}
	if rec.Flags&sam.Duplicate == 0 {
		t.Error("split record should share its primary's duplicate flag")
	}
}

// Two copies of a segment are competing placements; a chimeric read's other
// half is a split piece, not a competitor.
func TestGetAlignmentsSingleEnd(t *testing.T) {
	bin := minibwaBinary(t)
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	setEMGlobals()
	sid := "sample"
	sample_id = &sid

	rng := rand.New(rand.NewSource(9))
	random := func(n int) []byte {
		b := make([]byte, n)
		for i := range b {
			b[i] = "ACGT"[rng.Intn(4)]
		}
		return b
	}
	repeat := random(4000)
	diverged := repeat // an identical copy, which minibwa always reports
	var ref []byte
	ref = append(ref, random(20000)...)
	ref = append(ref, repeat...)
	ref = append(ref, random(20000)...)
	ref = append(ref, diverged...)
	uniqueAt := len(ref)
	ref = append(ref, random(20000)...)
	farAt := len(ref)
	ref = append(ref, random(20000)...)

	fa := filepath.Join(t.TempDir(), "ref.fa")
	if err := os.WriteFile(fa, append(append([]byte(">chrT\n"), ref...), '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gominibwa.Build(fa, 1); err != nil {
		t.Fatal(err)
	}
	idx, err := gominibwa.LoadIndex(fa)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	opts, err := gominibwa.NewOptions("lr")
	if err != nil {
		t.Fatal(err)
	}
	mapper := idx.NewMapper(opts)
	defer mapper.Close()

	rec := func(name string, seq []byte) fastqreader.FastQRecord {
		q := make([]byte, len(seq))
		for i := range q {
			q[i] = 'I'
		}
		return fastqreader.FastQRecord{ReadInfo: name, Read1: seq, ReadQual1: q, Barcode: []byte("BX"), Valid: true}
	}
	chimera := append(append([]byte(nil), ref[uniqueAt+1000:uniqueAt+3000]...), ref[farAt+1000:farAt+3000]...)
	reads := []fastqreader.FastQRecord{
		rec("unique", ref[uniqueAt+500:uniqueAt+3500]),
		rec("repeat", repeat[200:3200]),
		rec("chimera", chimera),
	}
	cands, full, _ := GetAlignments(mapper, reads, 17)
	if len(cands) != 6 {
		t.Fatalf("got %d read slots, want 6 (two per read)", len(cands))
	}
	for i := range reads {
		ph := cands[2*i+1]
		if len(ph) != 1 || ph[0].pos != -1 || !ph[0].single_end || ph[0].read1 {
			t.Errorf("read %d: second slot should be one unmapped single-end placeholder, got %d entries", i, len(ph))
		}
		for _, a := range cands[2*i] {
			if !a.single_end {
				t.Errorf("read %d: alignment not marked single-end", i)
			}
		}
	}
	if n := len(cands[0]); n != 1 {
		t.Errorf("unique read has %d candidates, want 1", n)
	}
	if n := len(cands[2]); n != 2 {
		t.Errorf("repeat read has %d candidates, want its 2 copies", n)
	}
	if n := len(cands[4]); n != 1 {
		t.Errorf("chimeric read has %d candidates, want 1 (its other half is a split piece)", n)
	}
	if n := len(full[4]); n < 2 {
		t.Errorf("chimeric read has %d alignments in full, want both halves", n)
	}
	for _, a := range cands[4] {
		if a.score <= 0 || math.IsNaN(a.log_alignment_probability) {
			t.Errorf("chimeric candidate has score %d, log probability %v", a.score, a.log_alignment_probability)
		}
	}
}

// Hits on mostly the same part of the read compete; hits on different parts
// of it (split pieces) do not.
func TestLongReadCompetition(t *testing.T) {
	best := &gominibwa.Hit{QueryStart: 0, QueryEnd: 10000}
	for _, c := range []struct {
		name string
		h    gominibwa.Hit
		want bool
	}{
		{"same span", gominibwa.Hit{QueryStart: 0, QueryEnd: 10000}, true},
		{"mostly the same span", gominibwa.Hit{QueryStart: 500, QueryEnd: 9800}, true},
		{"a piece inside the best hit", gominibwa.Hit{QueryStart: 6000, QueryEnd: 10000}, true}, // covered by best, so it competes
		{"disjoint", gominibwa.Hit{QueryStart: 11000, QueryEnd: 12000}, false},
		{"empty", gominibwa.Hit{QueryStart: 5, QueryEnd: 5}, false},
	} {
		if got := competes(&c.h, best); got != c.want {
			t.Errorf("competes(%s) = %v, want %v", c.name, got, c.want)
		}
	}
	// a chimeric read: the best hit covers the first part, another locus the rest
	chimera := &gominibwa.Hit{QueryStart: 0, QueryEnd: 6000}
	if competes(&gominibwa.Hit{QueryStart: 6000, QueryEnd: 10000}, chimera) {
		t.Error("the adjacent other half of a chimeric read should not compete")
	}
	if competes(&gominibwa.Hit{QueryStart: 5000, QueryEnd: 10000}, chimera) {
		t.Error("a hit sharing 1/5 of its span with the best hit should not compete")
	}
}
