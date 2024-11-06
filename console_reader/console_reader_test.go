package console_reader

import (
	"fmt"
	"github.com/ipfs/go-cid"
	"testing"
	"time"

	"github.com/streamingfast/logging"
	"github.com/stretchr/testify/require"
)

var zlogTest, tracerTest = logging.PackageLogger("test", "github.com/streamingfast/firehose-core/firecore")

// Example log:
// FIRE BLOCK 2118596 1 1730884260 bafy2bzacebfhqge6ulu6hlgqv6xhnimrhgdxdzg4lccyztitpfpdpx7oe7hzk bafy2bzacebc2l6w2kzfr752pijk4xyhdg7ptiags34s35buim4ilyrmcvchmg

func Test_Ctx_readBlock(t *testing.T) {
	reader := &ConsoleReader{
		logger: zlogTest,
		tracer: tracerTest,

		readerProtocolVersion: "1.0",
		protoMessageType:      "type.googleapis.com/sf.ethereum.type.v2.Block",
	}

	height := uint64(2118551)
	blockCount := 1
	tipsetCID, _ := cid.Decode("bafy2bzacedv2decyusd7n2kpknsgpiaopxh7ve3grzyfkrxouaza5ohypoi2c")
	parentTipsetCID, _ := cid.Decode("bafy2bzacebgnvekynqmddtbkdw77ygogoc4omoy5fsgtrhbd7kxwfsbgcaphm")

	//pbBlock := test.Block{
	//	Hash:   tipsetCID.Bytes(),
	//	Number: height,
	//}

	//anypbBlock, err := anypb.New(&pbBlock)

	//require.NoError(t, err)
	nowNano := time.Now().UnixNano()
	line := fmt.Sprintf(
		"%d %d %d %s %s",
		height,
		blockCount,
		nowNano,
		tipsetCID.String(),
		parentTipsetCID.String(),
	)

	block, err := reader.readBlock(line)
	require.NoError(t, err)

	require.Equal(t, height, block.Number)
	require.Equal(t, tipsetCID.String(), block.Id)
	require.Equal(t, parentTipsetCID.String(), block.ParentId)
	require.Equal(t, uint64(1), block.LibNum)
	require.Equal(t, int32(time.Unix(0, nowNano).Nanosecond()), block.Timestamp.Nanos)

	require.NoError(t, err)
	//require.Equal(t, anypbBlock.GetValue(), block.Payload.Value)

}

func Test_GetNext(t *testing.T) {
	lines := make(chan string, 2)
	reader := newConsoleReader(lines, zlogTest, tracerTest)

	initLine := "FIRE INIT 1.0 sf.filecoin.type.v1.Tipset"
	blockLine := "FIRE BLOCK 2118596 1 1730884260 bafy2bzacebfhqge6ulu6hlgqv6xhnimrhgdxdzg4lccyztitpfpdpx7oe7hzk bafy2bzacebc2l6w2kzfr752pijk4xyhdg7ptiags34s35buim4ilyrmcvchmg"

	lines <- initLine
	lines <- blockLine
	close(lines)

	block, err := reader.ReadBlock()
	require.NoError(t, err)

	require.Equal(t, uint64(2118596), block.Number)
	require.Equal(t, "bafy2bzacebfhqge6ulu6hlgqv6xhnimrhgdxdzg4lccyztitpfpdpx7oe7hzk", block.Id)
	require.Equal(t, "bafy2bzacebc2l6w2kzfr752pijk4xyhdg7ptiags34s35buim4ilyrmcvchmg", block.ParentId)
	require.Equal(t, uint64(1), block.LibNum)
	require.Equal(t, int32(time.Unix(0, 1730884260).Nanosecond()), block.Timestamp.Nanos)
}
