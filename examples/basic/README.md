# Basic Example

This example demonstrates basic usage of the go-windows-eventlog library.

## What it Does

- Reads events from a specified Windows Event Log channel
- Prints event summaries to stdout
- Persists read position to a checkpoint file
- Handles graceful shutdown on SIGINT/SIGTERM

## Usage

```bash
# Read from Application log
go run main.go

# Read from Security log
go run main.go -channel Security

# Filter events (only errors and warnings)
go run main.go -query "*[System[(Level=1 or Level=2)]]"

# Custom checkpoint file
go run main.go -checkpoint /path/to/checkpoint.yml

# Adjust batch size
go run main.go -batch 500
```

## Command-Line Flags

- `-channel` - Event log channel name (default: "Application")
- `-checkpoint` - Path to checkpoint file (default: ".checkpoint.yml")
- `-batch` - Batch size for reading events (default: 100)
- `-query` - XPath query for filtering (default: "*")

## Common XPath Queries

**Only Errors:**
```
*[System[(Level=2)]]
```

**Errors and Warnings:**
```
*[System[(Level=1 or Level=2)]]
```

**Specific Event IDs:**
```
*[System[(EventID=4624 or EventID=4625)]]
```

**Time Range (last 24 hours):**
```
*[System[TimeCreated[timediff(@SystemTime) <= 86400000]]]
```

**Specific Provider:**
```
*[System[Provider[@Name='Microsoft-Windows-Security-Auditing']]]
```

## Output Example

```
[1] Event ID: 1000, Provider: Application Error, Time: 2025-01-15T10:30:45Z
    Message: Faulting application name: example.exe, version: 1.0.0.0...

[2] Event ID: 1001, Provider: Application, Time: 2025-01-15T10:31:12Z
    Message: The application started successfully.

Read 2 events in 1.234s (1.62 events/sec)
```

## Notes

- Requires Windows OS to run
- Requires appropriate permissions for the channel being read
- Security log requires administrator privileges
