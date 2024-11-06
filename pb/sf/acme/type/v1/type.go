package pbacme

import (
	"time"

	firecore "github.com/streamingfast/firehose-core"
)

var _ firecore.Block = (*Tipset)(nil)

func (b *Tipset) GetFirehoseBlockID() string {
	return "blah"
}

func (b *Tipset) GetFirehoseBlockNumber() uint64 {
	return b.Height
}

func (b *Tipset) GetFirehoseBlockParentID() string {
		return ""
}

func (b *Tipset) GetFirehoseBlockParentNumber() uint64 {
	return *&b.Height - 1
}

func (b *Tipset) GetFirehoseBlockTime() time.Time {
	return time.Unix(0, int64(b.Timestamp)).UTC()
}

func (b *Tipset) GetFirehoseBlockLIBNum() uint64 {
	return b.Height
}
