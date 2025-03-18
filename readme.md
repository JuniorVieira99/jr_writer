# Writer Component for Go

A high-performance file writing utility for Go applications with support for concurrent writes, connection pooling, and resilient operations.

[![Go Report Card](https://goreportcard.com/badge/github.com/JuniorVieira99/jr_writer)](https://goreportcard.com/report/github.com/JuniorVieira99/jr_writer)
[![GoDoc](https://godoc.org/github.com/JuniorVieira99/jr_writer?status.svg)](https://pkg.go.dev/github.com/JuniorVieira99/jr_writer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Race and Coverage tests](https://github.com/JuniorVieira99/jr_writer/actions/workflows/race_coverage_tests.yml/badge.svg)](https://github.com/JuniorVieira99/jr_writer/actions/workflows/race_coverage_tests.yml)
[![benchmar_tests](https://github.com/JuniorVieira99/jr_writer/actions/workflows/benchmar_tests.yml/badge.svg)](https://github.com/JuniorVieira99/jr_writer/actions/workflows/benchmar_tests.yml)
[![unit_tests](https://github.com/JuniorVieira99/jr_writer/actions/workflows/unit_tests.yml/badge.svg)](https://github.com/JuniorVieira99/jr_writer/actions/workflows/unit_tests.yml)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Advanced Usage](#advanced-usage)
  - [Initializing a Custom Writer](#initializing-a-custom-writer)
  - [Writing with Timeouts](#writing-with-timeouts)
  - [Cancellable Writes](#cancellable-writes)
  - [Creating a Writer from Configuration](#creating-a-writer-from-configuration)
- [API Reference](#api-reference)
  - [Writer](#writer)
  - [Mode](#mode)
  - [Results](#results)
  - [Logger Methods](#logger-methods)
- [Performance Considerations](#performance-considerations)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [License](#license)

## Features

- **Concurrent Writing**: Efficiently write to multiple files simultaneously using worker pools
- **Connection Pooling**: Manage file handles efficiently to avoid excessive resource usage
- **Retry Mechanism**: Built-in retry logic with configurable exponential backoff
- **Multiple Initialization Methods**: Create writers from structs, maps, or JSON
- **Flexible Writing Modes**: Support for both append and write/truncate modes
- **Context Support**: Cancel operations with timeout or explicit cancellation
- **Detailed Results**: Get comprehensive statistics about writing operations
- **Batch Processing**: Automatic batch processing for large file sets
- **Thread Safety**: Safe for concurrent use with goroutines
- **Extensive Documentation**: All usable methods are documented for easy reference
- **Default Writer**: Use the default writer for quick operations

## Installation

```bash
go get github.com/JuniorVieira99/jr_writer
```

## Quick Start

```go
package main

import (
    "fmt"
    "os"
    writer "jrlogger/logger"
)

func main() {
    // Create temporary files
    file1, _ := os.CreateTemp("", "example-*.txt")
    file2, _ := os.CreateTemp("", "example-*.txt")
    files := []*os.File{file1, file2}

    message := "Hello, World!"
    
    // Use the Default Writer for Simple Operations
    // Use it with `writer.Dwriter` for convenience
    writer.Dwriter.SetFiles(files)
    writer.Dwriter.SetMessage(message)
    // Write to all files with 4 workers
    results, err:= writer.Dwriter.Write(4)
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    // Check Results
    results.Print()
}
```

## Advanced Usage

### Initializing a Custom Writer

```go

// Create a new Writer
myWriter := writer.NewWriter(files, mode, message, 10, 3, 100)

// Write to all files with 4 workers
results, err := myWriter.Write(4)

if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
}
// Check Results
results.Print()
```

### Writing with Timeouts

```go
// Write with a 5-second timeout
results, err := myWriter.WriteWithTimeout(4, 5*time.Second)
```

### Cancellable Writes

```go
// Start a cancellable write operation
cancel, resultCh, errCh := writer.StartWriteWithCancel(myWriter, 4)

// Cancel after some condition
if someCondition {
    cancel()
}

// Wait for results or error
select {
case results := <-resultCh:
    fmt.Printf("Success: %d\n", results.Success)
case err := <-errCh:
    fmt.Printf("Error: %v\n", err)
}
```

### Creating a Writer from Configuration

```go
// From a struct
config := writer.WriterConfig{
    Files:   &files,
    Mode:    mode,
    Message: &message,
    MaxPool: 10,
    Retries: 3,
    Backoff: 100,
}

myWriter, err := writer.NewWriterFromStruct(&config)

// From a map
mapWriter := map[string]interface{}{
    "files":   &files,
    "mode":    mode,
    "message": &message,
    "retries": uint64(3),
    "backoff": uint64(100),
    "maxPool": uint64(10),
}

myWriter, err = writer.NewWriterFromMap(mapWriter)

// From JSON
jsonWriter := []byte(`{
    "files": [file1.txt, file2.txt],
    "mode": "a",
    "message": "Hello, World!",
    "retries": 3,
    "backoff": 100,
    "maxPool": 10
}`)

myWriter, err = writer.NewWriterFromJSON(jsonWriter)
```

## API Reference

### Writer

The core component that handles writing operations.

#### Writer Fields

| Field     | Type           | Description                                      |
|-----------|----------------|--------------------------------------------------|
| files     | `[]*os.File`   | List of files to write to                        |
| mode      | `Mode`         | Writing mode (append or write/truncate)          |
| message   | `string`       | Message to write to files                        |
|openFilesPool| `map[string]*os.File` | Pool of open file handles                 |
| connPoolLock | `sync.Mutex` | Mutex for connection pool                        |
| connLastUsed | `map[string]time.Time` | Last used time for connections           |
| maxConn   | `uint64`       | Maximum number of connections in the pool        |
| retries   | `uint64`       | Number of retries on failure                     |
| backoff   | `uint64`       | Exponential backoff factor in milliseconds       |
| ctx       | `context.Context` | Context for cancellation                         |
| mu        | `sync.Mutex`   | Mutex for context                                |

#### Dwriter

A default writer instance for simple operations. For quick operations, use `Dwriter` with the default settings:

- Use `GetDefaultWriter` to get the default writer.
- Use `AddFiles` to add files to the default writer and execute write operations.
- Use `SetMessage` to set the message to write to files.
- Use any of the set methods to configure
- Then, `write` to write to all files with a specified number of workers.

Default values:

| Field     | Value          |
|-----------|----------------|
| files     | Empty slice []*os.File         |
| mode      | AppendMode = 'a'   |
| maxPool   | uint64(runtime.NumCPU() * 4)            |
| retries   | 2              |
| backoff   | 100            |
| message   | "Add text to write"             |

Example:

```go
// Get default writer
writer := writer.GetDefaultWriter()
// Add files
writer.AddFiles(files)
// Set message
writer.SetMessage("Hello, World!")
// Write to all files with 4 workers
results, err := writer.Write(4)
// Check Results
results.Print()
```

#### Constructor Methods

- `NewWriter(files, mode, message, maxPool, retries, backoff)`: Create a new Writer
- `NewWriterFromStruct(config)`: Create a Writer from a WriterConfig struct
- `NewWriterFromMap(config)`: Create a Writer from a map
- `NewWriterFromJSON(config)`: Create a Writer from a JSON byte slice

#### Writer Methods

- `Write(maxWorkers)`: Write to all files using a specified number of worker goroutines

```go
func (w *Writer) Write(maxWorkers int) (*Results, error) {...}
```

- `WriteWithTimeout(maxWorkers, timeout)`: Write to all files with a timeout

```go
func (w *Writer) WriteWithTimeout(maxWorkers int, timeout time.Duration) (*Results, error) {...}
```

- `StartWriteWithCancel(maxWorkers)`: Start a cancellable write operation

```go
func (w *Writer) StartWriteWithCancel(maxWorkers int) (cancel func(), resultCh <-chan *Results, errCh <-chan error) {...}
```

#### Setting Fields

- `SetFiles(files)`: Set the files to write to
- `SetMode(mode)`: Set the writing mode
- `SetMessage(message)`: Set the message to write
- `SetMaxPool(maxPool)`: Set the maximum connection pool size
- `SetRetries(retries)`: Set the number of retries on failure
- `SetBackoff(backoff)`: Set the exponential backoff factor
- `SetContext(ctx)`: Set the context for cancellation

#### Getting Fields

- `GetFiles()`: Get the files
- `GetMode()`: Get the writing mode
- `GetMessage()`: Get the message
- `GetMaxPool()`: Get the maximum connection pool size
- `GetRetries()`: Get the number of retries on failure
- `GetBackoff()`: Get the exponential backoff factor
- `GetContext()`: Get the context for cancellation

#### Pooling Methods

- `GetOpenFilesPool()`: Get the open files pool
- `AddConn(file *os.File)`: Add a file connection to the pool
- `RemoveConn(file *os.File)`: Remove a file connection from the pool
- `GetConn(file *os.File)`: Get a file connection from the pool
- `CheckConnStatus(file *os.File)`: Check the status of a file connection

#### Cleaning Methods

- `CloseConn(file *os.File)`: Close a file connection
- `CloseAllConns()`: Close all open file connections
- `ClearAll()`: Clear all pools
- `ClearFiles()`: Clear the files slice
- `FactoryReset()`: Close all connections, clear pools, clear files, and reset the factory

### Mode

Represents the file writing mode.

- `NewMode(mode)`: Create a new Mode with 'a' for append or 'w' for write/truncate
- `SetMode()`: Set the mode
- `GetMode()`: Get the current mode

### Results

Contains statistics about writing operations.

| Field     | Type           | Description                                      |
|-----------|----------------|--------------------------------------------------|
| Total     | `int`          | Total number of files processed                  |
| Success   | `int`          | Number of successful writes                      |
| Failure   | `int`          | Number of failed writes                          |
| SuccessRate| `float64`     | Percentage of successful writes                  |
| FailureRate| `float64`     | Percentage of failed writes                      |
| ErrSlice  | `[]error`      | Slice of errors encountered                       |
| Info      | `map[string]interface{}` | Additional information                  |

### Result Methods

- `NewResults()`: Create a new Results instance
- `Print()` : Print the results to stdout
    Example Output:

    ```shell
    Total: 2
    Success: 1
    Failure: 1
    Success Rate: 0.500000
    Failure Rate: 0.500000
    Info:
    filename_1: completed
    filename_2: error
    ```

- `GetStringRepresentation()`: Get the results as a string

#### Logger Methods

- `Default()`: Creates and returns a default logger that writes to stdout
- `IsDebugMode()`: Checks if debug mode is enabled
- `SetDebugMode(enabled)`: Enables or disables debug logging
- `Debug(format, v...)`: Logs a debug message with formatting (only when debug mode is enabled)
- `GetAvailableModes()`: Returns a list of available writing modes

#### Using the Logger

```go
// Enable debug mode -> Verbose logging
writer.SetDebugMode(true)
```

## Performance Considerations

- **Worker Pool Size**: For optimal performance, set the worker pool size to match your system's CPU count
- **Connection Pool Size**: Adjust the connection pool size based on the number of files you're writing to
- **Batching**: For very large file sets (>1000 files), the writer automatically uses batching

## Error Handling

The Writer provides comprehensive error information:

- All methods return an error when they fail
- The Results struct includes an ErrSlice with all errors encountered
- Detailed error messages help identify the source of failures

## Testing

Run the tests using the following command in the test directory:

```bash
go test -v
```

## License

MIT License - See LICENSE file for details.
