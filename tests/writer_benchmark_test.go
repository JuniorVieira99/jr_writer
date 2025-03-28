package tests

import (
	"fmt"
	writer "github.com/JuniorVieira99/jr_writer"
	"os"
	"runtime"
	"testing"
	"time"
)

// Fixtures
// --------------------------------------------------------

// Make file fixtures
func makeFiles(count int) []*os.File {
	files := make([]*os.File, count)
	for i := 0; i < count; i++ {
		str := fmt.Sprintf("testfile%d", i)
		file, err := os.CreateTemp("", str)
		if err != nil {
			panic(err)
		}
		files[i] = file
	}
	return files
}

// Helper to setup basic writer for tests
func setupWriter(fileCount int) (*writer.Writer, []*os.File) {
	files := makeFiles(fileCount)
	mode, _ := writer.NewMode(&appendModeA)
	return writer.NewWriter(&files, mode, &message, 10, 3, 100), files
}

// Cleanup helper
func cleanupFiles(files []*os.File) {
	for _, file := range files {
		file.Close()
		os.Remove(file.Name())
	}
}

// Tests
// --------------------------------------------------------

func TestSimpleBenchmark(t *testing.T) {
	// Setup logger
	writer.SetDebugMode(true)
	// Setup Writer
	myWriter, myFiles := setupWriter(2)
	// Cleanup
	defer cleanupFiles(myFiles)
	// Set timer
	startTime := time.Now()
	// Do the work
	result, err := myWriter.Write(2)

	if err != nil {
		t.Errorf("Write returned error: %v", err)
		return
	}
	if result == nil {
		t.Error("Write returned nil results")
		return
	}
	if result.Success != 2 {
		t.Errorf("Expected 10 successful writes, got %d", result.Success)
	}
	if result.Failure != 0 {
		t.Errorf("Expected 0 failures, got %d", result.Failure)
	}
	if result.SuccessRate != 1.0 {
		t.Errorf("Expected success rate of 1.0, got %f", result.SuccessRate)
	}
	if len(result.ErrSlice) != 0 {
		t.Errorf("Expected ErrSlice to be empty, got %d items", len(result.ErrSlice))
		t.Errorf("Error slice: %v", result.ErrSlice)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	t.Logf("Elapsed time: %.3fs for 2 writes", elapsedTime.Seconds())
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("Error closing connections: %v", err)
	}
	t.Log("Closed all connections")
}

func Test25FilesBenchmark(t *testing.T) {
	// Setup logger
	writer.SetDebugMode(true)
	// Setup Writer
	myWriter, myFiles := setupWriter(100)
	// Cleanup
	defer cleanupFiles(myFiles)
	// Set timer
	startTime := time.Now()

	// Do the work
	result, err := myWriter.Write(25)

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	if err != nil {
		t.Errorf("Write returned error: %v", err)
	}

	if result == nil {
		t.Error("Write returned nil results")
	}

	if result.Success != 100 {
		t.Errorf("Expected 100 successful writes, got %d", result.Success)
	}

	if result.Failure != 0 {
		t.Errorf("Expected 0 failures, got %d", result.Failure)
	}

	if result.SuccessRate != 1.0 {
		t.Errorf("Expected success rate of 1.0, got %f", result.SuccessRate)
	}

	if len(result.ErrSlice) != 0 {
		t.Errorf("Expected ErrSlice to be empty, got %d items", len(result.ErrSlice))
		t.Errorf("Error slice: %v", result.ErrSlice)
	}

	t.Logf("Elapsed time: %.3fs for 25 writes", elapsedTime.Seconds())
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("Error closing connections: %v", err)
	}
	t.Log("Closed all connections")
}

func TestManyFilesBenchmark(t *testing.T) {
	// Setup logger
	writer.SetDebugMode(false)

	// File size slice
	fileSizes := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}

	// Total Time
	var totalTime time.Duration

	for i, fileSize := range fileSizes {
		// Run each file size test in a separate function
		t.Run(fmt.Sprintf("FileSize%d", fileSize), func(t *testing.T) {
			// Setup Writer
			myWriter, myFiles := setupWriter(fileSize)

			// Set timer
			startTime := time.Now()

			// Do the work
			result, err := myWriter.Write(fileSize)
			if err != nil {
				t.Errorf("Write returned error: %v", err)
			}

			// Calculate elapsed time
			elapsedTime := time.Since(startTime)
			totalTime += elapsedTime

			// Close connections first, before any cleanup
			err = myWriter.CloseAllConns()
			if err != nil {
				t.Errorf("Error closing connections: %v", err)
			}

			// Now cleanup files
			cleanupFiles(myFiles)

			if err != nil {
				t.Errorf("Write returned error: %v", err)
			}

			if result == nil {
				t.Error("Write returned nil results")
			}

			if result.Success != uint64(fileSizes[i]) {
				t.Errorf("Expected %d successful writes, got %d", fileSize, result.Success)
			}

			if result.Failure != 0 {
				t.Errorf("Expected 0 failures, got %d", result.Failure)
			}

			if result.SuccessRate != 1.0 {
				t.Errorf("Expected success rate of 1.0, got %f", result.SuccessRate)
			}

			if len(result.ErrSlice) != 0 {
				t.Errorf("Expected ErrSlice to be empty, got %d items", len(result.ErrSlice))
				t.Errorf("Error slice: %v", result.ErrSlice)
			}
			t.Logf("Elapsed time: %.3fs for %d writes", elapsedTime.Seconds(), fileSize)
			err = myWriter.CloseAllConns()
			if err != nil {
				t.Errorf("Error closing connections: %v", err)
			}
			t.Log("Closed all connections")
		})
	}

	t.Log("Test Completed\n")
	t.Logf("Total elapsed time: %.3fs", totalTime.Seconds())
}

func TestNoRetriesManyFilesBenchmark(t *testing.T) {
	// Setup logger
	writer.SetDebugMode(false)

	// File size slice
	fileSizes := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}

	// Total Time
	var totalTime time.Duration

	for i, fileSize := range fileSizes {
		t.Run(fmt.Sprintf("FileSize%d", fileSize), func(t *testing.T) {
			// Setup Writer
			myWriter, myFiles := setupWriter(fileSize)
			// Setup retries
			err := myWriter.SetRetries(0)
			if err != nil {
				t.Errorf("Error setting retries: %v", err)
			}
			// Increase pool size to match file count
			err = myWriter.SetMaxPool(uint64(fileSize))
			if err != nil {
				t.Errorf("Error setting max pool: %v", err)
			}
			// Set timer
			startTime := time.Now()

			// Do the work
			result, err := myWriter.Write(fileSize)
			if err != nil {
				t.Errorf("Write returned error: %v", err)
			}

			// Calculate elapsed time
			elapsedTime := time.Since(startTime)
			totalTime += elapsedTime

			// Close connections first, before any cleanup
			err = myWriter.CloseAllConns()
			if err != nil {
				t.Errorf("Error closing connections: %v", err)
			}

			// Now cleanup files
			cleanupFiles(myFiles)

			if err != nil {
				t.Errorf("Write returned error: %v", err)
			}

			if result == nil {
				t.Error("Write returned nil results")
			}

			if result.Success != uint64(fileSizes[i]) {
				t.Logf("Expected %d successful writes, got %d", fileSize, result.Success)
			}

			if result.Failure != 0 {
				t.Logf("Expected 0 failures, got %d", result.Failure)
			}

			if result.SuccessRate != 1.0 {
				t.Logf("Expected success rate of 1.0, got %f", result.SuccessRate)
			}

			if len(result.ErrSlice) != 0 {
				t.Errorf("Expected ErrSlice to be empty, got %d items", len(result.ErrSlice))
				t.Errorf("Test for length: %d", fileSize)
				for _, err := range result.ErrSlice {
					t.Errorf("Error: %v", &err)
				}
				for key, value := range result.Info {
					t.Logf("Key: %s, Value: %s", key, value)
				}
			}
			t.Logf("Elapsed time: %.3fs for %d writes", elapsedTime.Seconds(), fileSize)
		})
	}
	t.Log("Test Completed\n")
	t.Logf("Total elapsed time: %.3fs", totalTime.Seconds())
}

func TestBatching1500(t *testing.T) {
	myWriter, myFiles := setupWriter(1500)

	err := myWriter.SetRetries(0)
	if err != nil {
		t.Errorf("Error setting retries: %v", err)
	}
	err = myWriter.SetMaxPool(1500)
	if err != nil {
		t.Errorf("Error setting max pool: %v", err)
	}

	//Start Timer
	start := time.Now()
	results, err := myWriter.Write(1500)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	elapsed := time.Since(start)

	// Close connections first, before any cleanup
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("Error closing connections: %v", err)
	}
	cleanupFiles(myFiles)

	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if results.Success != 1500 {
		t.Errorf("Expected 1500 successes, got %d", results.Success)
	}

	t.Logf("Elapsed time: %.3fs", elapsed.Seconds())
}

func TestBatching3000(t *testing.T) {
	myWriter, myFiles := setupWriter(3000)

	err := myWriter.SetRetries(0)
	if err != nil {
		t.Errorf("Error setting retries: %v", err)
	}
	err = myWriter.SetMaxPool(1000)
	if err != nil {
		t.Errorf("Error setting max pool: %v", err)
	}

	//Start Timer
	start := time.Now()
	results, err := myWriter.Write(4)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	elapsed := time.Since(start)

	// Close connections first, before any cleanup
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("Error closing connections: %v", err)
	}
	cleanupFiles(myFiles)

	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if results.Success != 3000 {
		t.Errorf("Expected 1500 successes, got %d", results.Success)
	}
	t.Logf("Elapsed time: %.3fs", elapsed.Seconds())
}

func TestBatching5000(t *testing.T) {
	myWriter, myFiles := setupWriter(5000)

	err := myWriter.SetRetries(0)
	if err != nil {
		t.Errorf("Error setting retries: %v", err)
	}
	err = myWriter.SetMaxPool(1500)
	if err != nil {
		t.Errorf("Error setting max pool: %v", err)
	}

	//Start Timer
	start := time.Now()
	results, err := myWriter.Write(4)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	elapsed := time.Since(start)

	// Close connections first, before any cleanup
	err = myWriter.CloseAllConns()
	if err != nil {
		t.Errorf("Error closing connections: %v", err)
	}
	cleanupFiles(myFiles)

	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if results.Success != 5000 {
		t.Errorf("Expected 1500 successes, got %d", results.Success)
	}
	t.Logf("Elapsed time: %.3fs", elapsed.Seconds())
}

func BenchmarkWriter(b *testing.B) {
	// Setup debug for verbose logging
	writer.SetDebugMode(false)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {

			// Setup Writer
			myWriter, myFiles := setupWriter(2)
			// Do the work
			result, err := myWriter.Write(2)
			if err != nil {
				b.Errorf("Write returned error: %v", err)
			}
			// Cleanup
			err = myWriter.CloseAllConns()
			if err != nil {
				b.Errorf("Error closing connections: %v", err)
			}
			cleanupFiles(myFiles)

			if err != nil {
				b.Errorf("Write returned error: %v", err)
				continue
			}
			if result == nil {
				b.Error("Write returned nil results")
				continue
			}
			if result.Success != 2 {
				b.Errorf("Expected 10 successful writes, got %d", result.Success)
			}
			if result.Failure != 0 {
				b.Errorf("Expected 0 failures, got %d", result.Failure)
			}
			if result.SuccessRate != 1.0 {
				b.Errorf("Expected success rate of 1.0, got %f", result.SuccessRate)
			}
			if len(result.ErrSlice) != 0 {
				b.Errorf("Expected ErrSlice to be empty, got %d items", len(result.ErrSlice))
				b.Errorf("Error slice: %v", result.ErrSlice)
			}
			b.Log("Closed all connections")
		}
	})
}

func BenchmarkManyFiles(b *testing.B) {
	writer.SetDebugMode(false)

	sizes := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Files%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				w, files := setupWriter(size)
				b.StartTimer()

				result, err := w.Write(runtime.NumCPU())
				if err != nil {
					b.Errorf("Write returned error: %v", err)
				}

				b.StopTimer()
				err = w.CloseAllConns()
				if err != nil {
					b.Errorf("Error closing connections: %v", err)
				}

				if result == nil {
					b.Error("Write returned nil results")
					continue
				}
				if result.Failure != 0 {
					b.Errorf("Expected 0 failures, got %d", result.Failure)
				}
				if result.SuccessRate != 1.0 {
					b.Errorf("Expected success rate of 1.0, got %f", result.SuccessRate)
				}

				cleanupFiles(files)
				b.Log("Closed all connections")
			}
		})
	}
}

func TestDwriterManyFiles(t *testing.T) {
	writer.SetDebugMode(false)
	sizes := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	for _, size := range sizes {
		start := time.Now()
		t.Run(fmt.Sprintf("Files%d", size), func(t *testing.T) {
			// Make files
			files := makeFiles(size)
			// Add files
			errFiles := writer.Dwriter.AddFiles(files)
			if errFiles != nil {
				t.Errorf("Error adding files: %v", errFiles)
			}
			// Write
			result, err := writer.Dwriter.Write(runtime.NumCPU())
			if err != nil {
				t.Errorf("Write returned error: %v", err)
			}

			if result == nil {
				t.Error("Write returned nil results")
			}
			if result.Failure != 0 {
				t.Errorf("Expected 0 failures, got %d", result.Failure)
			}
			if result.SuccessRate != 1.0 {
				t.Errorf("Expected success rate of 1.0, got %f", result.SuccessRate)
			}

			// Close connections
			err = writer.Dwriter.CloseAllConns()
			if err != nil {
				t.Errorf("Error closing connections: %v", err)
			}
			cleanupFiles(files)
			elapsed := time.Since(start)
			t.Logf("Elapsed time: %.3fs for %d writes", elapsed.Seconds(), size)
		})
	}
}
