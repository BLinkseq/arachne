package aligner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arachne/fastqreader"
	"arachne/gominibwa"

	sam "github.com/biogo/hts/sam"
)

// runInjected pushes one WorkUnit through WorkerThread with a process func
// that triggers the fault, and returns the run's error. It also checks the
// worker drains remaining work and returns every buffer after a failure.
func runInjected(t *testing.T, process BarcodeFunc) error {
	t.Helper()
	fail := &runFailure{}
	input := make(chan *WorkUnit, 3)
	bufChan := make(chan *[]fastqreader.FastQRecord, 3)
	for i := 1; i <= 3; i++ {
		b := []fastqreader.FastQRecord{}
		input <- &WorkUnit{barcode_num: i, poolBuf: &b}
	}
	close(input)

	done := make(chan struct{})
	go func() {
		// zero Mapper: Close is a no-op and the injected process never uses it
		WorkerThread(input, nil, bufChan, nilMapper(), process, nil, nil, fail)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("WorkerThread did not finish: deadlock after failure")
	}
	if len(bufChan) != 3 {
		t.Errorf("returned %d buffers, want 3", len(bufChan))
	}
	return fail.err
}

func nilMapper() *gominibwa.Mapper { return &gominibwa.Mapper{} }

func wantErr(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error %q does not contain %q", err, substr)
	}
}

func noopProcess(*WorkUnit, chan *sam.Record, *gominibwa.Mapper, map[string]*sam.Reference, *bool) {}

func TestWorkerRecoversPlainPanic(t *testing.T) {
	err := runInjected(t, func(w *WorkUnit, _ chan *sam.Record, _ *gominibwa.Mapper, _ map[string]*sam.Reference, _ *bool) {
		panic("minibwa failed: boom")
	})
	wantErr(t, err, "minibwa failed")
	wantErr(t, err, "barcode group 1") // first failure wins; later work is skipped
}

func TestWorkerNoFailureNoError(t *testing.T) {
	if err := runInjected(t, noopProcess); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// the original nil-deref class: a read with an empty alignment list
func TestInjectedReadWithNoAlignments(t *testing.T) {
	penalty := -4.0
	improper_pair_penalty = &penalty
	err := runInjected(t, func(*WorkUnit, chan *sam.Record, *gominibwa.Mapper, map[string]*sam.Reference, *bool) {
		tagBestAlignments([][]*Alignment{{}, {}})
	})
	wantErr(t, err, "nil pointer dereference")
}

func TestInjectedReadIDMismatch(t *testing.T) {
	penalty := -4.0
	improper_pair_penalty = &penalty
	err := runInjected(t, func(*WorkUnit, chan *sam.Record, *gominibwa.Mapper, map[string]*sam.Reference, *bool) {
		a := newTagTestAlignment(5, 1, "r", "chr1", 10, false) // sits in slot 0 with read_id 5
		b := newTagTestAlignment(1, 0, "r", "chr1", 20, false)
		tagBestAlignments([][]*Alignment{{a}, {b}})
	})
	wantErr(t, err, "does not match its position")
}

func TestInjectedMateIDMismatch(t *testing.T) {
	penalty := -4.0
	improper_pair_penalty = &penalty
	err := runInjected(t, func(*WorkUnit, chan *sam.Record, *gominibwa.Mapper, map[string]*sam.Reference, *bool) {
		a := newTagTestAlignment(0, 1, "r", "chr1", 10, false)
		stray := newTagTestAlignment(7, 0, "r", "chr1", 20, false) // slot 1, read_id 7
		tagBestAlignments([][]*Alignment{{a}, {stray}})
	})
	wantErr(t, err, "does not match mate alignment")
}

func TestInjectedFlushEmptyRead(t *testing.T) {
	err := runInjected(t, func(w *WorkUnit, out chan *sam.Record, _ *gominibwa.Mapper, c map[string]*sam.Reference, d *bool) {
		flushToChannel([][]*Alignment{{}}, out, c, d)
	})
	wantErr(t, err, "read has no alignments")
}

func TestInjectedFlushNoActiveAlignment(t *testing.T) {
	nu := false
	noUnmapped = &nu
	err := runInjected(t, func(w *WorkUnit, out chan *sam.Record, _ *gominibwa.Mapper, c map[string]*sam.Reference, d *bool) {
		a := newTagTestAlignment(0, 1, "r", "chr1", 10, false)
		a.active = false
		flushToChannel([][]*Alignment{{a}}, out, c, d)
	})
	wantErr(t, err, "none is active")
}

func TestInjectedIllegalCigar(t *testing.T) {
	err := runInjected(t, func(*WorkUnit, chan *sam.Record, *gominibwa.Mapper, map[string]*sam.Reference, *bool) {
		fixCigar([]uint32{9999, 10})
	})
	wantErr(t, err, "invalid BAM CIGAR operation")
}

func TestCentromereFileErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.bed")
	_, err := loadCentromeres(&missing)
	wantErr(t, err, "unable to open centromeres file")

	bad := filepath.Join(t.TempDir(), "bad.bed")
	if werr := os.WriteFile(bad, []byte("chr1\t500\t100\n"), 0o644); werr != nil {
		t.Fatal(werr)
	}
	_, err = loadCentromeres(&bad)
	wantErr(t, err, "start > end")
}
