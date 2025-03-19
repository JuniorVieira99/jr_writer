package tests

import (
	"context"
	writer "github.com/JuniorVieira99/jr_writer"
	"os"
	"strings"
	"testing"
	"time"
)

// Fixtures

var (
	file1, _ = os.CreateTemp("", "test-writer-*.txt")
	files    = []*os.File{
		file1,
	}
	filesPtr    = &files
	appendModeA = "a"
	appendModeW = "w"
	modeA, _    = writer.NewMode(&appendModeA)
	message     = "test message"
	fixRetries  = uint64(3)
	fixBackoff  = uint64(100)
	fixPool     = uint64(10)
	mapWriter   = map[string]interface{}{
		"files":   filesPtr,
		"mode":    modeA,
		"message": &message,
		"retries": fixRetries,
		"backoff": fixBackoff,
		"maxPool": fixPool,
	}
)

// Basic initialization test
func TestNewWriter(t *testing.T) {
	myWriter := writer.NewWriter(nil, nil, nil, 0, 0, 0)
	if myWriter == nil {
		t.Error("NewWriter returned nil")
	}

	err := myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

func TestNewWriterFromMap(t *testing.T) {
	myWriter, err := writer.NewWriterFromMap(mapWriter)
	if err != nil {
		t.Errorf("NewWriterFromMap returned error: %v", err)
	}
	if myWriter == nil {
		t.Error("NewWriterFromMap returned nil")
	}

	if myWriter.GetFiles() == nil {
		t.Error("Files is nil")
	}
	if myWriter.GetMode() == nil {
		t.Error("Mode is nil")
	}
	if myWriter.GetMessage() == nil {
		t.Error("Message is nil")
	}
	if myWriter.GetRetries() != 3 {
		t.Errorf("Expected retries to be 3, got %d", myWriter.GetRetries())
	}
	if myWriter.GetBackoff() != 100 {
		t.Errorf("Expected backoff to be 100, got %d", myWriter.GetBackoff())
	}
	if myWriter.GetMaxPool() != 10 {
		t.Errorf("Expected maxPool to be 10, got %d", myWriter.GetMaxPool())
	}
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test Mode creation
func TestNewMode(t *testing.T) {
	appendMode := "a"
	mode, err := writer.NewMode(&appendMode)
	if err != nil {
		t.Errorf("NewMode returned error: %v", err)
	}
	if mode == nil {
		t.Error("NewMode returned nil")
	}
}

// Test Writer creation with values
func TestNewWriterWithValues(t *testing.T) {
	// Create temporary files
	defer os.Remove(file1.Name())
	defer file1.Close()

	myWriter := writer.NewWriter(&files, modeA, &message, 10, 3, 100)

	if myWriter == nil {
		t.Error("NewWriter returned nil")
	}
	if myWriter.GetFiles() == nil {
		t.Error("Files is nil")
	}
	if myWriter.GetMode() == nil {
		t.Error("Mode is nil")
	}
	if myWriter.GetMessage() == nil {
		t.Error("Message is nil")
	}
	if myWriter.GetRetries() != 3 {
		t.Errorf("Expected retries to be 3, got %d", myWriter.GetRetries())
	}
	if myWriter.GetBackoff() != 100 {
		t.Errorf("Expected backoff to be 100, got %d", myWriter.GetBackoff())
	}
	err := myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test WriterConfig creation
func TestNewWriterFromStruct(t *testing.T) {
	defer os.Remove(file1.Name())
	defer file1.Close()

	config := writer.WriterConfig{
		Files:   &files,
		Mode:    modeA,
		Message: &message,
		MaxPool: fixPool,
		Retries: fixRetries,
		Backoff: fixBackoff,
	}

	myWriter, err := writer.NewWriterFromStruct(&config)
	if err != nil {
		t.Errorf("NewWriterFromStruct returned error: %v", err)
	}

	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}

	if err != nil {
		t.Errorf("NewWriterFromStruct returned error: %v", err)
	}
	if myWriter == nil {
		t.Error("NewWriterFromStruct returned nil")
	}

	if myWriter.GetMode() == nil {
		t.Error("Mode is nil")
	}

	if myWriter.GetMessage() == nil {
		t.Error("Message is nil")
	}

	if myWriter.GetMessage() != &message {
		t.Errorf("Expected message to be '%s', got '%s'", message, *myWriter.GetMessage())
	}

	if myWriter.GetRetries() != 3 {
		t.Errorf("Expected retries to be 3, got %d", myWriter.GetRetries())
	}

	if myWriter.GetBackoff() != 100 {
		t.Errorf("Expected backoff to be 100, got %d", myWriter.GetBackoff())
	}
}

// Test Writer SetXXX methods
func TestWriterSetters(t *testing.T) {
	defer os.Remove(file1.Name())
	defer file1.Close()

	myWriter := writer.NewWriter(&files, modeA, &message, 10, 3, 100)

	// Initialize err
	var err error

	// Test SetFiles
	newFiles := []*os.File{file1}
	err = myWriter.SetFiles(&newFiles)
	if err != nil {
		t.Errorf("SetFiles returned error: %v", err)
	}

	// Test SetMode
	newMode, _ := writer.NewMode(&appendModeW)
	err = myWriter.SetMode(newMode)
	if err != nil {
		t.Errorf("SetMode returned error: %v", err)
	}

	// Test SetMessage
	newMessage := "new test message"
	err = myWriter.SetMessage(&newMessage)
	if err != nil {
		t.Errorf("SetMessage returned error: %v", err)
	}

	// Test SetRetries
	err = myWriter.SetRetries(5)
	if err != nil {
		t.Errorf("SetRetries returned error: %v", err)
	}
	if myWriter.GetRetries() != 5 {
		t.Errorf("Expected retries to be 5, got %d", myWriter.GetRetries())
	}

	// Test SetBackoff
	err = myWriter.SetBackoff(200)
	if err != nil {
		t.Errorf("SetBackoff returned error: %v", err)
	}
	if myWriter.GetBackoff() != 200 {
		t.Errorf("Expected backoff to be 200, got %d", myWriter.GetBackoff())
	}

	// Test SetMaxPool
	err = myWriter.SetMaxPool(20)
	if err != nil {
		t.Errorf("SetMaxPool returned error: %v", err)
	}

	if myWriter.GetMaxPool() != 20 {
		t.Errorf("Expected max pool to be 20, got %d", myWriter.GetMaxPool())
	}
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test checking connection pool
func TestConnectionPool(t *testing.T) {
	defer os.Remove(file1.Name())
	defer file1.Close()

	myWriter := writer.NewWriter(&files, modeA, &message, 10, 3, 100)

	// Test GetConn
	conn, err := myWriter.GetConn(file1)
	if err != nil {
		t.Errorf("GetConn returned error: %v", err)
	}
	if conn == nil {
		t.Error("GetConn returned nil")
	}

	// Connection should be closed now
	if myWriter.CheckConnStatus(file1) {
		t.Error("CheckConnStatus returned true after closing, expected false")
	}
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test basic writing functionality
func TestWrite(t *testing.T) {
	defer os.Remove(file1.Name())
	defer file1.Close()

	myWriter := writer.NewWriter(&files, modeA, &message, 10, 3, 100)

	results, err := myWriter.Write(2)
	if err != nil {
		t.Errorf("Write returned error: %v", err)
	}

	if results == nil {
		t.Error("Write returned nil results")
	}

	if results.Success != 1 {
		t.Errorf("Expected 1 successful write, got %d", results.Success)
	}

	if results.Failure != 0 {
		t.Errorf("Expected 0 failures, got %d", results.Failure)
	}

	if results.SuccessRate != 1.0 {
		t.Errorf("Expected success rate of 1.0, got %f", results.SuccessRate)
	}

	// Verify content was written
	file1, err = os.Open(file1.Name())
	if err != nil {
		t.Fatalf("Failed to open temp file: %v", err)
	}
	content := make([]byte, len(message))
	_, err = file1.Read(content)
	if err != nil {
		t.Fatalf("Failed to read from file: %v", err)
	}
	if string(content) != message {
		t.Errorf("Expected content '%s', got '%s'", message, string(content))
	}
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test writing with timeout
func TestWriteWithTimeout(t *testing.T) {
	defer os.Remove(file1.Name())
	defer file1.Close()

	myWriter := writer.NewWriter(&files, modeA, &message, 10, 3, 100)

	// Test with sufficient timeout
	results, err := myWriter.WriteWithTimeout(2, 1*time.Second)
	if err != nil {
		t.Errorf("WriteWithTimeout returned error: %v", err)
	}
	if results == nil {
		t.Error("WriteWithTimeout returned nil results")
		return
	}
	if results.Success != 1 && results != nil {
		t.Errorf("Expected 1 successful write, got %d", results.Success)
	}
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test writing with cancellation
func TestWriteWithCancel(t *testing.T) {
	defer os.Remove(file1.Name())
	defer file1.Close()

	myWriter := writer.NewWriter(&files, modeA, &message, 10, 3, 100)

	cancel, resultCh, errCh := writer.StartWriteWithCancel(myWriter, 2)

	// Wait for either result or error
	select {
	case results := <-resultCh:
		if results.Success != 1 {
			t.Errorf("Expected 1 successful write, got %d", results.Success)
		}
	case err := <-errCh:
		t.Errorf("Received error: %v", err)
	case <-time.After(1 * time.Second):
		cancel() // Cancel the operation
		t.Log("Write operation canceled")
	}
	err := myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test context handling
func TestContextHandling(t *testing.T) {
	defer os.Remove(file1.Name())
	defer file1.Close()

	myWriter := writer.NewWriter(&files, modeA, &message, 10, 3, 100)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	myWriter.SetContext(ctx)

	_, err := myWriter.Write(2)

	if err == nil {
		t.Error("Expected error due to canceled context, got nil")
	}
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

// Test Results creation
func TestNewResults(t *testing.T) {
	results := writer.NewResults()
	if results == nil {
		t.Error("NewResults returned nil")
	}
	if results.Total != 0 {
		t.Errorf("Expected Total to be 0, got %d", results.Total)
	}
	if results.Success != 0 {
		t.Errorf("Expected Success to be 0, got %d", results.Success)
	}
	if results.Failure != 0 {
		t.Errorf("Expected Failure to be 0, got %d", results.Failure)
	}
	if results.SuccessRate != 0 {
		t.Errorf("Expected SuccessRate to be 0, got %f", results.SuccessRate)
	}
	if results.FailureRate != 0 {
		t.Errorf("Expected FailureRate to be 0, got %f", results.FailureRate)
	}
	if len(results.ErrSlice) != 0 {
		t.Errorf("Expected ErrSlice to be empty, got %d items", len(results.ErrSlice))
	}
	if len(results.Info) != 0 {
		t.Errorf("Expected Info to be empty, got %d items", len(results.Info))
	}
}

func TestDWriter(t *testing.T) {
	err := writer.Dwriter.FactoryReset()
	if err != nil {
		t.Errorf("FactoryReset returned error: %v", err)
	}

	err = writer.Dwriter.AddFiles(files)
	if err != nil {
		t.Errorf("AddFiles returned error: %v", err)
	}
	results, wErr := writer.Dwriter.Write(2)
	if wErr != nil {
		t.Errorf("Write returned error: %v", wErr)
	}

	if results == nil {
		t.Error("Write returned nil results")
	}

	if results.Success != 1 {
		t.Errorf("Expected 1 successful write, got %d", results.Success)
	}

	err = writer.Dwriter.CloseAllConns()
	if err != nil {
		t.Errorf("CloseAllConns returned error: %v", err)
	}
}

func TestResultPrint(t *testing.T) {
	results := writer.NewResults()
	results.Total = 2
	results.Success = 1
	results.Failure = 1
	results.SuccessRate = 0.5
	results.FailureRate = 0.5
	results.Info["filename_1"] = "completed"
	results.Info["filename_2"] = "error"
	results.Print()

	if !strings.Contains(results.GetStringRepresentation(), "Total: 2") {
		t.Errorf("Expected Total to be 2, got %d", results.Total)
	}
	results.GetStringRepresentation()
}
