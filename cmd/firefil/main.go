package main

import (
	pbfilecoin "github.com/chainsafe/firehose-filecoin/pb/sf/filecoin/type/v1"
	firecore "github.com/streamingfast/firehose-core"
	fhCmd "github.com/streamingfast/firehose-core/cmd"
)

func main() {
	fhCmd.Main(&firecore.Chain[*pbfilecoin.Tipset]{
		ShortName:            "filecoin",
		LongName:             "Filecoin",
		ExecutableName:       "forest",
    FullyQualifiedModule: "github.com/chainsafe/firehose-filecoin",
		Version:              version,

		FirstStreamableBlock: 1,

		BlockFactory:         func() firecore.Block { return new(pbfilecoin.Tipset) },
		ConsoleReaderFactory: firecore.NewConsoleReader,

		Tools: &firecore.ToolsConfig[*pbfilecoin.Tipset]{},
	})
}

// Version value, injected via go build `ldflags` at build time, **must** not be removed or inlined
var version = "dev"
