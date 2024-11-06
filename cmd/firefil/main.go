package main

import (
	pbfilecoin "github.com/chainsafe/firehose-filecoin/pb/sf/filecoin/type/v1"
	firecore "github.com/streamingfast/firehose-core"
	fhCmd "github.com/streamingfast/firehose-core/cmd"
)

func main() {
	fhCmd.Main(&firecore.Chain[*pbfilecoin.Block]{
		ShortName:            "filecoin",
		LongName:             "Filecoin",
		ExecutableName:       "dummy-blockchain",
		FullyQualifiedModule: "github.com/chainsafe/firehose-filecoin",
		Version:              version,

		FirstStreamableBlock: 1,

		BlockFactory:         func() firecore.Block { return new(pbfilecoin.Block) },
		ConsoleReaderFactory: firecore.NewConsoleReader,

		Tools: &firecore.ToolsConfig[*pbfilecoin.Block]{},
	})
}

// Version value, injected via go build `ldflags` at build time, **must** not be removed or inlined
var version = "dev"
