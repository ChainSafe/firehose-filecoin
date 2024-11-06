package firecore

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/streamingfast/bstream"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/dmetrics"
	"github.com/streamingfast/firehose-core/node-manager/mindreader"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const FirePrefix = "FIRE "
const FirePrefixLen = len(FirePrefix)
const InitLogPrefix = "INIT "
const InitLogPrefixLen = len(InitLogPrefix)
const BlockLogPrefix = "BLOCK "
const BlockLogPrefixLen = len(BlockLogPrefix)

// Requried by NewConsoleReader but never used
type BlockEncoder interface {
	Encode(block Block) (blk *pbbstream.Block, err error)
}

// Block represents the chain-specific Protobuf block. Chain specific's block
// model must implement this interface so that Firehose core is able to properly
// marshal/unmarshal your block into/to the Firehose block envelope binary format.
//
// All the methods are prefixed with `GetFirehoseBlock` to avoid any potential
// conflicts with the fields/getters of your chain's block model that would
// prevent you from implementing this interface.
//
// Consumer of your chain's protobuf block model don't need to be aware of those
// details, they are internal Firehose core information that are required to function
// properly.
//
// The value you return for each of those methods must be done respecting Firehose rules
// which are enumarated in the documentation of each method.
type Block interface {
	proto.Message

	// GetFirehoseBlockID returns the block ID as a string, usually in the representation
	// used by your chain (hex, base58, base64, etc.). The block ID must be unique across
	// all blocks that will ever exist on your chain.
	GetFirehoseBlockID() string

	// GetFirehoseBlockNumber returns the block number as an unsigned integer. The block
	// number could be shared by multiple blocks in which case one is the canonical one
	// and the others are forks (resolution of forks is handled by Firehose core later in the
	// block processing pipeline).
	//
	// The value should be sequentially ordered which means that a block with block number 10
	// has come before block 11. Firehose core will deal with block skips without problem though
	// (e.g. block 1, is produced then block 3 where block 3's parent is block 1).
	GetFirehoseBlockNumber() uint64

	// GetFirehoseBlockParentID returns the block ID of the parent block as a string. All blocks
	// ever produced must have a parent block ID except for the genesis block which is the first
	// one. The value must be the same as the one returned by GetFirehoseBlockID() of the parent.
	//
	// If it's the genesis block, return an empty string.
	GetFirehoseBlockParentID() string

	// GetFirehoseBlockParentNumber returns the block number of the parent block as a uint64.
	// The value must be the same as the one returned by GetFirehoseBlockNumber() of the parent
	// or `0` if the block has no parent
	//
	// This is useful on chains that have holes. On other chains, this is as simple as "BlockNumber - 1".
	GetFirehoseBlockParentNumber() uint64

	// GetFirehoseBlockTime returns the block timestamp as a time.Time of when the block was
	// produced. This should the consensus agreed time of the block.
	GetFirehoseBlockTime() time.Time
}

type ParsingStats struct {
}

type ConsoleReader struct {
	lines     chan string
	done      chan interface{}
	closeOnce sync.Once
	logger    *zap.Logger
	tracer    logging.Tracer

	// Parsing context
	readerProtocolVersion string
	protoMessageType      string
	lastBlock             bstream.BlockRef
	lastParentBlock       bstream.BlockRef
	lastBlockTimestamp    time.Time

	lib uint64

	blockRate *dmetrics.AvgRatePromCounter
}

func NewConsoleReader(lines chan string, blockEncoder BlockEncoder, logger *zap.Logger, tracer logging.Tracer) (mindreader.ConsolerReader, error) {
	reader := newConsoleReader(lines, logger, tracer)

	delayBetweenStats := 30 * time.Second
	if tracer.Enabled() {
		delayBetweenStats = 5 * time.Second
	}

	go func() {
		defer reader.blockRate.Stop()

		for {
			select {
			case <-reader.done:
				return
			case <-time.After(delayBetweenStats):
				reader.printStats()
			}
		}
	}()

	return reader, nil
}

func newConsoleReader(lines chan string, logger *zap.Logger, tracer logging.Tracer) *ConsoleReader {
	return &ConsoleReader{
		lines:  lines,
		done:   make(chan interface{}),
		logger: logger,
		tracer: tracer,

		blockRate: dmetrics.MustNewAvgRateFromPromCounter(ConsoleReaderBlockReadCount, 1*time.Second, 30*time.Second, "blocks"),
	}
}

func (r *ConsoleReader) Done() <-chan interface{} {
	return r.done
}

func (r *ConsoleReader) Close() error {
	r.closeOnce.Do(func() {
		r.blockRate.SyncNow()
		r.printStats()

		r.logger.Info("console reader done")
		close(r.done)
	})

	return nil
}

type blockRefView struct {
	ref bstream.BlockRef
}

func (v blockRefView) String() string {
	if v.ref == nil {
		return "<unset>"
	}

	return v.ref.String()
}

type blockRefViewTimestamp struct {
	ref       bstream.BlockRef
	timestamp time.Time
}

func (v blockRefViewTimestamp) String() string {
	return fmt.Sprintf("%s @ %s", blockRefView{v.ref}, v.timestamp.Local().Format(time.RFC822Z))
}

func (r *ConsoleReader) printStats() {
	r.logger.Info("console reader stats",
		zap.Stringer("block_rate", r.blockRate),
		zap.Stringer("last_block", blockRefViewTimestamp{r.lastBlock, r.lastBlockTimestamp}),
		zap.Stringer("last_parent_block", blockRefView{r.lastParentBlock}),
		zap.Uint64("lib", r.lib),
	)
}

func (r *ConsoleReader) ReadBlock() (out *pbbstream.Block, err error) {
	out, err = r.next()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (r *ConsoleReader) next() (out *pbbstream.Block, err error) {
	for line := range r.lines {
		if !strings.HasPrefix(line, "FIRE ") {
			continue
		}

		line = line[FirePrefixLen:]

		switch {
		case strings.HasPrefix(line, BlockLogPrefix):
			out, err = r.readBlock(line[BlockLogPrefixLen:])

		case strings.HasPrefix(line, InitLogPrefix):
			err = r.readInit(line[InitLogPrefixLen:])
		default:
			if r.tracer.Enabled() {
				r.logger.Debug("skipping unknown Firehose log line", zap.String("line", line))
			}
			continue
		}

		if err != nil {
			chunks := strings.SplitN(line, " ", 2)
			return nil, fmt.Errorf("%s: %s (line %q)", chunks[0], err, line)
		}

		if out != nil {
			return out, nil
		}
	}

	r.Close()

	return nil, io.EOF
}

// Formats
// [READER_PROTOCOL_VERSION] sf.ethereum.type.v2.Block
func (r *ConsoleReader) readInit(line string) error {
	chunks, err := splitInBoundedChunks(line, 2)
	if err != nil {
		return fmt.Errorf("split: %s", err)
	}

	r.readerProtocolVersion = chunks[0]

	switch r.readerProtocolVersion {
	// Implementation of RPC poller were set to use 1.0 so we keep support for it for now
	case "1.0", "3.0":
		// Supported
	default:
		return fmt.Errorf("major version of Firehose exchange protocol is unsupported (expected: one of [1.0, 3.0], found %s), you are most probably running an incompatible version of the Firehose aware node client/node poller", r.readerProtocolVersion)
	}

	protobufFullyQualifiedName := chunks[1]
	if protobufFullyQualifiedName == "" {
		return fmt.Errorf("protobuf fully qualified name is empty, it must be set to a valid Protobuf fully qualified message type representing your block format")
	}

	r.setProtoMessageType(protobufFullyQualifiedName)

	r.logger.Info("console reader protocol version init",
		zap.String("version", r.readerProtocolVersion),
		zap.String("protobuf_fully_qualified_name", protobufFullyQualifiedName),
	)

	return nil
}

// Formats
// [block_num:342342342] [block_hash] [parent_num] [parent_hash] [lib:123123123] [timestamp:unix_nano] B64ENCODED_any
func (r *ConsoleReader) readBlock(line string) (out *pbbstream.Block, err error) {
	if r.readerProtocolVersion == "" {
		return nil, fmt.Errorf("reader protocol version not set, did you forget to send the 'FIRE INIT <reader_protocol_version> <protobuf_fully_qualified_type>' line?")
	}

	chunks, err := splitInBoundedChunks(line, 5)
	if err != nil {
		return nil, fmt.Errorf("splitting block log line: %w", err)
	}

	height, err := strconv.ParseUint(chunks[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing block num %q: %w", chunks[0], err)
	}

	blockCount, err := strconv.ParseUint(chunks[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing block num %q: %w", chunks[0], err)
	}

	timestampUnixNano, err := strconv.ParseUint(chunks[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing timestamp %q: %w", chunks[5], err)
	}

	timestamp := time.Unix(0, int64(timestampUnixNano))

	tipsetKey := chunks[3]

	parentTipsetKey := chunks[4]

	parentNum := height - 1

	libNum := uint64(1)

	blockPayload := &anypb.Any{
		TypeUrl: r.protoMessageType,
		Value:   []byte{byte(blockCount)},
	}

	block := &pbbstream.Block{
		Id:        tipsetKey,
		Number:    height,
		ParentId:  parentTipsetKey,
		ParentNum: parentNum,
		Timestamp: timestamppb.New(timestamp),
		LibNum:    1,
		Payload:   blockPayload,
	}

	ConsoleReaderBlockReadCount.Inc()
	r.lastBlock = bstream.NewBlockRef(tipsetKey, height)
	r.lastParentBlock = bstream.NewBlockRef(parentTipsetKey, parentNum)
	r.lastBlockTimestamp = timestamp
	r.lib = libNum

	return block, nil
}

func (r *ConsoleReader) setProtoMessageType(typeURL string) {
	if strings.HasPrefix(typeURL, "type.googleapis.com/") {
		r.protoMessageType = typeURL
		return
	}

	if strings.Contains(typeURL, "/") {
		panic(fmt.Sprintf("invalid type url %q, expecting type.googleapis.com/", typeURL))
	}

	r.protoMessageType = "type.googleapis.com/" + typeURL
}

// splitInBoundedChunks splits the line in `count` chunks and returns the slice `chunks[1:count]` (so exclusive end),
// but will accumulate all trailing chunks within the last (for free-form strings, or JSON objects)
func splitInBoundedChunks(line string, count int) ([]string, error) {
	chunks := strings.SplitN(line, " ", count)
	if len(chunks) != count {
		return nil, fmt.Errorf("%d fields required but found %d fields for line %q", count, len(chunks), line)
	}

	return chunks, nil
}
