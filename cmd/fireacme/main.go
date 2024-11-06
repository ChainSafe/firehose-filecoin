package main

import (
	pbacme "github.com/chainsafe/firehose-filecoin/pb/sf/acme/type/v1"
	firecore "github.com/streamingfast/firehose-core"
	fhCmd "github.com/streamingfast/firehose-core/cmd"
)

func main() {
	fhCmd.Main(&firecore.Chain[*pbacme.Block]{
		ShortName:            "acme",
		LongName:             "Acme",
		ExecutableName:       "dummy-blockchain",
		FullyQualifiedModule: "github.com/chainsafe/firehose-filecoin",
		Version:              version,

		FirstStreamableBlock: 1,

		BlockFactory:         func() firecore.Block { return new(pbacme.Block) },
		ConsoleReaderFactory: firecore.NewConsoleReader,

		Tools: &firecore.ToolsConfig[*pbacme.Block]{},
	})
}

// Version value, injected via go build `ldflags` at build time, **must** not be removed or inlined
var version = "dev"
