# go-windows-eventlog

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Go library for reading Windows Event Logs, extracted from [Elastic Beats](https://github.com/elastic/beats) v9.2.1.

## Installation

```bash
go get github.com/tianlin/go-windows-eventlog
```

**Requirements:** Go 1.21+, Windows OS

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tianlin/go-windows-eventlog/pkg/eventlog"
)

func main() {
    cfg := eventlog.Config{
        Name:           "Application",
        CheckpointFile: ".checkpoint.yml",
        BatchSize:      100,
    }

    if err := cfg.Validate(); err != nil {
        log.Fatal(err)
    }

    reader, err := eventlog.NewReader(cfg, eventlog.NewStdLogger(nil))
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()

    ctx := context.Background()
    err = reader.Read(ctx, func(event eventlog.Event) error {
        fmt.Printf("Event ID: %v, Provider: %s\n",
            event.Fields["event.code"],
            event.Fields["event.provider"])
        return nil
    })

    if err != nil {
        log.Fatal(err)
    }
}
```

## Source

Extracted from [elastic/beats](https://github.com/elastic/beats) v9.2.1 winlogbeat module.

## License

Apache License 2.0 - Same as the original Elastic Beats project.
