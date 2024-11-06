package pbfilecoin

import (
	"time"

	firecore "github.com/streamingfast/firehose-core"
)

var _ firecore.Block = (*Tipset)(nil)

func (b *Tipset) GetFirehoseBlockID() string {
	return b.TipsetKey
}

func (b *Tipset) GetFirehoseBlockNumber() uint64 {
	return b.Height
}

func (b *Tipset) GetFirehoseBlockParentID() string {
	if b.ParentTipsetKey == nil {
		return ""
	}

	return *b.ParentTipsetKey
}

func (b *Tipset) GetFirehoseBlockParentNumber() uint64 {
	if b.Height == 0 {
		return 0
	}

  return b.Height - 1
}

func (b *Tipset) GetFirehoseBlockTime() time.Time {
	return time.Unix(0, int64(b.Timestamp)).UTC()
}

func (b *Tipset) GetFirehoseBlockLIBNum() uint64 {
	return 1;
}
