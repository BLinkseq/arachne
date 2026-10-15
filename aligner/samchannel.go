package aligner

import (
	"bufio"
	"os"

	"github.com/biogo/hts/sam"
)

// NewSamWriterChannel writes records received on the returned channel to stdout.
// Sends the first write error (or nil) on doneChan when the input channel is
// closed. After an error the channel is still drained so producers never block.
func NewSamWriterChannel(head *sam.Header, cp, buff, threads int) (chan *sam.Record, chan error, error) {
	outChan := make(chan *sam.Record, cp)
	doneChan := make(chan error, 1)

	sio := bufio.NewWriterSize(os.Stdout, buff)
	sw, err := sam.NewWriter(sio, head, sam.FlagDecimal)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		var werr error
		for rec := range outChan {
			if werr == nil {
				werr = sw.Write(rec)
			}
		}
		if err := sio.Flush(); werr == nil { // always flush
			werr = err
		}
		doneChan <- werr
	}()

	return outChan, doneChan, nil
}
