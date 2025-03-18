// package writer provides functionality for writing to files with configurable settings
// such as concurrent writing, retry mechanisms, connection pooling, and various file access modes.
//
// The package includes several components:
//
// Writer: The main struct that handles writing operations to multiple files.
// It supports concurrent writing, connection pooling, retry mechanisms, and customizable
// writing modes.
//
// Mode: Represents the file writing mode ('a' for append or 'w' for write/truncate).
//
// Results: Tracks statistics about writing operations, including success/failure counts
// and additional information.
//
// WriterConfig: Configuration struct used for initializing Writer instances through
// structured configuration.
//
// Key features:
//   - Connection pooling to efficiently manage open file handles
//   - Configurable retry mechanisms with exponential backoff
//   - Concurrent writing with worker pools
//   - Context-based operations with timeout and cancellation support
//   - Batch processing for large file sets
//   - Multiple initialization methods (from struct, map, or JSON)
//   - Detailed error tracking and operation statistics
package writer

// Writer Component for the Logger Module
// ----------------------------------------------------

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

// ----------------------------------------------------
// Logger Methods
// ----------------------------------------------------

// Make default logger
func Default() *log.Logger {
	return log.New(os.Stdout, "moduleLogger", log.LstdFlags)
}

// Module Logger
var logger *log.Logger = Default()

// Debug flag to control debug level logging
var debugMode bool = false

// IsDebugMode returns the current debug mode status
func IsDebugMode() bool {
	return debugMode
}

// SetDebugMode enables or disables debug level logging
func SetDebugMode(enabled bool) {
	debugMode = enabled
}

// Debug logs a message only when debug mode is enabled
func Debug(format string, v ...interface{}) {
	if debugMode {
		logger.Printf("[DEBUG] "+format, v...)
	}
}

// GetAvailableModes returns a list of available writing modes
func GetAvailableModes() []string {
	return availableModes
}

// ----------------------------------------------------
// Vars
// ----------------------------------------------------

var availableModes = []string{"a", "w"}

// ----------------------------------------------------
// Structs
// ----------------------------------------------------

// Writer struct
type Writer struct {
	files         *[]*os.File     // Slice of pointers to files
	mode          *Mode           // Mode for writing - a or w
	message       *string         // Message to write
	openFilesPool sync.Map        // Pool of open files
	connPoolLock  sync.RWMutex    // Lock for the connection pool
	connLastUsed  sync.Map        // Map to track when connections were last used
	maxConns      uint64          // Max number of connections
	retries       uint64          // Number of retries
	backoff       uint64          // Backoff between retries
	ctx           context.Context // Context
	mu            sync.RWMutex    // Mutex
}

// WriterConfig struct -> use with NewWriterFromStruct
type WriterConfig struct {
	Files   *[]*os.File
	Mode    *Mode
	Message *string
	MaxPool uint64
	Retries uint64
	Backoff uint64
}

// Mode struct
type Mode struct {
	mode *string // Mode for writing - a or w
}

// Results struct
type Results struct {
	Total       uint64                 `json:"total"`        // Total number of messages
	ErrSlice    []*error               `json:"err_slice"`    // Slice of errors
	Success     uint64                 `json:"success"`      // Number of successful writes
	Failure     uint64                 `json:"failure"`      // Number of failed writes
	SuccessRate float64                `json:"success_rate"` // Percentage of successful writes
	FailureRate float64                `json:"failure_rate"` // Percentage of failed writes
	Info        map[string]interface{} `json:"info"`         // Map of additional information
	mu          sync.RWMutex           // Mutex
}

// struct for JSON unmarshaling
type jsonConfig struct {
	Files   []string `json:"files"`   // Array of file paths
	Mode    string   `json:"mode"`    // Mode as string
	Message string   `json:"message"` // Message as string
	MaxPool uint64   `json:"maxPool"` // Max pool size
	Retries uint64   `json:"retries"` // Number of retries
	Backoff uint64   `json:"backoff"` // Backoff duration
}

// ----------------------------------------------------
// Write Methods
// ----------------------------------------------------

// fullWriteCheck validates the Writer's fields, ensuring they are not nil.
// It checks the files, mode, and message fields, logging and returning an error
// if any of them are nil. It also validates and updates the mode field using
// modeValidation, returning an error if the mode is invalid.
func (w *Writer) fullWriteCheck() error {
	if w == nil {
		logger.Print("Writer is nil")
		return fmt.Errorf("writer is nil")
	}

	if w.files == nil {
		logger.Print("Files is nil")
		return fmt.Errorf("files is nil")
	}

	if w.mode == nil {
		logger.Print("Mode is nil")
		return fmt.Errorf("mode is nil")
	}

	if w.message == nil {
		logger.Print("Message is nil")
		return fmt.Errorf("message is nil")
	}
	return nil
}

// GetFiles returns a pointer to the Writer's files slice of pointers to os.File.
func (w *Writer) GetFiles() *[]*os.File {
	return w.files
}

// GetMode returns a pointer to the Writer's mode struct.
func (w *Writer) GetMode() *Mode {
	return w.mode
}

// GetMessage returns a pointer to the Writer's message string.
func (w *Writer) GetMessage() *string {
	return w.message
}

// GetRetries returns the Writer's number of retries.
func (w *Writer) GetRetries() uint64 {
	return w.retries
}

// GetBackoff returns the Writer's backoff value.
func (w *Writer) GetBackoff() uint64 {
	return w.backoff
}

// GetMaxPool returns the Writer's maximum number of connections.
func (w *Writer) GetMaxPool() uint64 {
	return w.maxConns
}

// SetFiles sets the Writer's files slice of pointers to os.File.
func (w *Writer) SetFiles(files *[]*os.File) error {
	err := w.fullWriteCheck()
	if err != nil {
		return err
	}

	if files == nil {
		logger.Print("Files is nil")
		return fmt.Errorf("files is nil")
	}
	w.files = files
	return nil
}

// AddFiles appends the given files to the Writer's existing files slice,
// and sets the Writer's files field to the new slice.
// It returns an error if the Writer's fullWriteCheck fails.
func (w *Writer) AddFiles(files []*os.File) error {
	err := w.fullWriteCheck()
	if err != nil {
		return err
	}
	copy := *w.files
	copy = append(copy, files...)
	w.files = &copy
	return nil
}

// SetMode sets the Writer's mode struct.
func (w *Writer) SetMode(mode *Mode) error {
	err := w.fullWriteCheck()
	if err != nil {
		return err
	}
	if mode == nil {
		logger.Print("Mode is nil")
		return fmt.Errorf("mode is nil")
	}
	w.mode = mode
	return nil
}

// SetMessage sets the Writer's message string.
func (w *Writer) SetMessage(message *string) error {
	err := w.fullWriteCheck()
	if err != nil {
		return err
	}
	if message == nil {
		logger.Print("Message is nil")
		return fmt.Errorf("message is nil")
	}
	w.message = message
	return nil
}

// SetRetries sets the Writer's number of retries.
func (w *Writer) SetRetries(retries uint64) error {
	err := w.fullWriteCheck()
	if err != nil {
		return err
	}
	w.retries = retries
	return nil
}

// SetBackoff sets the Writer's backoff value.
func (w *Writer) SetBackoff(backoff uint64) error {
	err := w.fullWriteCheck()
	if err != nil {
		return err
	}
	w.backoff = backoff
	return nil
}

// SetMaxPool sets the Writer's maximum number of connections in the openFilesPool.
func (w *Writer) SetMaxPool(maxPool uint64) error {
	err := w.fullWriteCheck()
	if err != nil {
		return err
	}
	w.maxConns = maxPool
	return nil
}

// SetContext sets the Writer's context.
func (w *Writer) SetContext(ctx context.Context) {
	w.ctx = ctx
}

// Constructor
// ----------------------------------------------------

// NewWriter initializes and returns a new Writer instance. It sets up the
// files, mode, and message for the Writer, and initializes the openFilesPool
// with the specified maximum pool size. The Writer is equipped with a mutex
// for handling concurrent access to its fields.
//
// Parameters:
//   - files: A pointer to a slice of pointers to os.File, representing the
//     files to be managed by the Writer.
//   - mode: A Mode struct representing the writing mode (e.g., append or write).
//   - message: A pointer to a string, containing the message to be written to files.
//   - maxPool: A uint64 specifying the maximum number of connections in the openFilesPool.
//   - retries: A uint64 specifying the number of retries.
//   - backoff: A uint64 specifying the backoff between retries.
//
// Returns:
//   - A pointer to the initialized Writer instance.
func NewWriter(
	files *[]*os.File,
	mode *Mode,
	message *string,
	maxPool uint64,
	retries uint64,
	backoff uint64,
) *Writer {
	if files == nil {
		files = &[]*os.File{}
	}
	return &Writer{
		files:         files,
		mode:          mode,
		message:       message,
		openFilesPool: sync.Map{},
		connLastUsed:  sync.Map{},
		maxConns:      maxPool,
		retries:       retries,
		backoff:       backoff,
		ctx:           context.Background(),
		mu:            sync.RWMutex{},
		connPoolLock:  sync.RWMutex{},
	}
}

// Config
// ----------------------------------------------------

// Helper to validated Map
func validateMap(config map[string]interface{}) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	for key, value := range config {
		switch key {
		case "files":
			_, ok := value.(*[]*os.File)
			if !ok {
				return fmt.Errorf("files is not a []*os.File")
			}
		case "mode":
			_, ok := value.(*Mode)
			if !ok {
				return fmt.Errorf("mode is not a *Mode")
			}
		case "message":
			_, ok := value.(*string)
			if !ok {
				return fmt.Errorf("message is not a *string")
			}
		case "retries":
			_, ok := value.(uint64)
			if !ok {
				return fmt.Errorf("retries is not a uint64")
			}
		case "backoff":
			_, ok := value.(uint64)
			if !ok {
				return fmt.Errorf("backoff is not a uint64")
			}
		case "maxPool":
			_, ok := value.(uint64)
			if !ok {
				return fmt.Errorf("maxPool is not a uint64")
			}
		default:
			return fmt.Errorf("unknown key: %s", key)
		}
	}
	return nil
}

// validateStruct validates a Writer's fields, ensuring they are not nil.
// It checks the files, mode, and message fields, logging and returning an error
func validateStruct(w *WriterConfig) error {
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	if w.Files == nil {
		return fmt.Errorf("files is nil")
	}
	if w.Mode == nil {
		return fmt.Errorf("mode is nil")
	}
	if w.Message == nil {
		return fmt.Errorf("message is nil")
	}
	return nil
}

// NewWriterFromMap creates a new Writer instance from a configuration map.
//
// The map should contain the following keys with the corresponding values:
//   - "files": a []*os.File
//   - "mode": a *Mode
//   - "message": a *string
//   - "retries": a uint64
//   - "backoff": a uint64
//   - "maxPool": a uint64
//
// If the map does not contain all the required keys or if the values are not of the correct type, an error is returned.
// If all the keys and values are valid, a new Writer instance is returned.
func NewWriterFromMap(config map[string]interface{}) (*Writer, error) {
	err := validateMap(config)
	if err != nil {
		return nil, err
	}

	return &Writer{
		files:         config["files"].(*[]*os.File),
		mode:          config["mode"].(*Mode),
		message:       config["message"].(*string),
		retries:       config["retries"].(uint64),
		backoff:       config["backoff"].(uint64),
		openFilesPool: sync.Map{},
		connPoolLock:  sync.RWMutex{},
		connLastUsed:  sync.Map{},
		maxConns:      config["maxPool"].(uint64),
		ctx:           context.Background(),
		mu:            sync.RWMutex{},
	}, nil

}

// NewWriterFromJSON creates a new Writer instance from a JSON configuration byte slice.
// The JSON should contain:
//   - "files": array of file paths
//   - "mode": writing mode ("a" or "w")
//   - "message": string to write
//   - "maxPool": max connections
//   - "retries": number of retries
//   - "backoff": backoff duration in ms
//
// Example JSON:
//
//	{
//	  "files": ["/path/to/file1", "/path/to/file2"],
//	  "mode": "a",
//	  "message": "Hello, World!",
//	  "maxPool": 10,
//	  "retries": 3,
//	  "backoff": 100
//	}
func NewWriterFromJSON(config []byte) (*Writer, error) {
	var jc jsonConfig
	if err := json.Unmarshal(config, &jc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Convert file paths to *os.File slice
	files := make([]*os.File, 0, len(jc.Files))
	for _, path := range jc.Files {
		file, err := os.CreateTemp("", path)
		if err != nil {
			// Close all previously opened files
			for _, f := range files {
				f.Close()
			}
			return nil, fmt.Errorf("failed to open file %s: %v", path, err)
		}
		files = append(files, file)
	}
	// Create mode
	mode, err := NewMode(&jc.Mode)
	if err != nil {
		// Close all opened files
		for _, f := range files {
			f.Close()
		}
		return nil, fmt.Errorf("failed to create mode: %v", err)
	}

	// Convert to map for NewWriterFromMap
	conf := map[string]interface{}{
		"files":   &files,
		"mode":    mode,
		"message": &jc.Message,
		"maxPool": jc.MaxPool,
		"retries": jc.Retries,
		"backoff": jc.Backoff,
	}

	return NewWriterFromMap(conf)
}

// NewWriterFromStruct creates a new Writer instance from a WriterConfig struct.
//
// The function checks if the WriterConfig is not nil and if all the required fields
// are not nil. If any of the checks fail, an error is returned.
//
// Otherwise, a new Writer instance is returned with the fields set to the values
// in the WriterConfig.
//
// The function does not validate the values of the fields, other than checking
// if they are not nil. It is the caller's responsibility to ensure that the values
// are valid.
func NewWriterFromStruct(config *WriterConfig) (*Writer, error) {
	if validateStruct(config) != nil {
		return nil, validateStruct(config)
	}
	// Return Writer
	return &Writer{
		files:         config.Files,
		mode:          config.Mode,
		message:       config.Message,
		retries:       config.Retries,
		backoff:       config.Backoff,
		openFilesPool: sync.Map{},
		connPoolLock:  sync.RWMutex{},
		connLastUsed:  sync.Map{},
		maxConns:      config.MaxPool,
		ctx:           context.Background(),
		mu:            sync.RWMutex{},
	}, nil
}

// Writer Method
// ----------------------------------------------------

// writeToFile writes a message to a file, ensuring the file is open and using the correct mode.
// If the file is not open, it opens the file with the specified mode and updates the openFilesPool.
// If there is an error during the writing process, it logs the error and returns it.
func (w *Writer) writeToFile(file *os.File, message string, results *Results, mu *sync.RWMutex) error {
	// Check if context is done
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	default:
	}

	// Nil file check
	if file == nil {
		mu.Lock()
		defer mu.Unlock()
		results.Info["nil_file"] = "received nil file pointer"
		return fmt.Errorf("nil file pointer received")
	}

	// Get file mode
	fileMode, err := getFileMode(*w.mode.mode)
	if err != nil {
		mu.Lock()
		defer mu.Unlock()
		results.Info[file.Name()] = err.Error()
		return err
	}

	// Check if file is open -> if not open, open it
	if !w.CheckConnStatus(file) {
		newFile, err := os.OpenFile(file.Name(), fileMode, 0666)
		if err != nil {
			mu.Lock()
			defer mu.Unlock()
			results.Info[file.Name()] = err.Error()
			return fmt.Errorf("error opening file %s: %v", file.Name(), err)
		}
		// Ensure the new file is not closed prematurely
		defer func() {
			if err != nil {
				newFile.Close()
				w.openFilesPool.Delete(newFile.Name())
			}
		}()

		// Update the new file to the pool
		fileName := file.Name()
		w.mu.Lock()
		defer w.mu.Unlock()
		w.openFilesPool.Store(fileName, newFile)

		// Use the new file for writing
		file = newFile
	}

	// Write to file
	bufferedWriter := bufio.NewWriter(file)
	_, err = bufferedWriter.WriteString(message)

	// Check for error
	if err != nil {
		mu.Lock()
		defer mu.Unlock()
		results.Info[file.Name()] = err.Error()

		// Check if file is already
		if !strings.Contains(err.Error(), "already closed") {
			file.Close()
		}

		return fmt.Errorf("error writing to file %s: %v", file.Name(), err)
	}

	// Flush the buffer
	err = bufferedWriter.Flush()
	if err != nil {
		mu.Lock()
		defer mu.Unlock()
		results.Info[file.Name()] = err.Error()
		file.Close() // Ensure the file is closed if an error occurs
		return fmt.Errorf("error flushing buffer for file %s: %v", file.Name(), err)
	}

	return nil
}

// ----------------------------------------------------
// Mode Methods
// ----------------------------------------------------

// Mode validation
func modeValidation(mode *string) (*string, error) {
	if mode == nil {
		logger.Print("Mode is nil")
		return nil, fmt.Errorf("Mode is nil")
	}

	var cleanMode string
	cleanMode = strings.TrimSpace(*mode)
	cleanMode = strings.ToLower(cleanMode)

	if !slices.Contains(GetAvailableModes(), cleanMode) {
		logger.Print("Mode is not available, please use 'a' or 'w', got: ", cleanMode)
		return nil, fmt.Errorf("Mode is not available, please use 'a' or 'w', got: %s", cleanMode)
	}

	return &cleanMode, nil

}

// Make new Mode struct
func NewMode(mode *string) (*Mode, error) {
	cleanMode, err := modeValidation(mode)
	if err != nil {
		return nil, err
	}
	return &Mode{
		mode: cleanMode,
	}, nil
}

// Set mode -> a or w
func (m *Mode) SetMode() error {
	cleanMode, err := modeValidation(m.mode)
	if err != nil {
		return err
	}
	m.mode = cleanMode
	return nil
}

// Get mode
func (m *Mode) GetMode() *string {
	return m.mode
}

// Helper function to get OS file mode from mode string
func getFileMode(modeStr string) (int, error) {
	switch modeStr {
	case "a":
		return os.O_RDWR | os.O_CREATE | os.O_APPEND, nil
	case "w":
		return os.O_RDWR | os.O_CREATE | os.O_TRUNC, nil
	default: // Should not reach here after validation, but for safety
		return 0, fmt.Errorf("invalid mode: %s", modeStr)
	}
}

// ----------------------------------------------------
// Results Methods
// ----------------------------------------------------

// NewResults initializes and returns a new Results instance with all fields set to their default values (0 for integers, empty slice for ErrSlice, and empty map for Info).
func NewResults() *Results {
	return &Results{
		Total:       0,
		ErrSlice:    make([]*error, 0),
		Success:     0,
		Failure:     0,
		SuccessRate: 0,
		FailureRate: 0,
		Info:        make(map[string]interface{}),
		mu:          sync.RWMutex{},
	}
}

// Print prints out the Results struct fields in a human-readable format.
// It is thread-safe.
func (r *Results) Print() {
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Printf("Total: %d\n", r.Total)
	fmt.Printf("Success: %d\n", r.Success)
	fmt.Printf("Failure: %d\n", r.Failure)
	fmt.Printf("Success Rate: %f\n", r.SuccessRate)
	fmt.Printf("Failure Rate: %f\n", r.FailureRate)
	fmt.Print("Info:\n")
	for key, value := range r.Info {
		fmt.Printf("%s: %v\n", key, value)
	}
}

// GetStringRepresentation returns a string that contains a formatted representation
// of the Results struct fields, including total counts, success and failure metrics,
// and any additional information stored in the Info map. The method is thread-safe
// and locks the mutex to ensure the integrity of the data when accessed concurrently.
func (r *Results) GetStringRepresentation() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var infoString string
	for key, value := range r.Info {
		infoString += fmt.Sprintf("%s: %v\n", key, value)
	}

	return fmt.Sprintf("Total: %d\nSuccess: %d\nFailure: %d\nSuccess Rate: %f\nFailure Rate: %f\nInfo: %v", r.Total, r.Success, r.Failure, r.SuccessRate, r.FailureRate, infoString)
}

// ----------------------------------------------------
// Pool Methods
// ----------------------------------------------------

// Get openFilesPool
func (w *Writer) GetOpenFilesPool() *sync.Map {
	return &w.openFilesPool
}

// Helper function to add file to openFilesPool
func (w *Writer) AddConn(file *os.File) error {
	// Check nil file
	if file == nil {
		return fmt.Errorf("nil file pointer received")
	}

	// Get file name
	fileName := file.Name()

	w.connPoolLock.Lock()
	defer w.connPoolLock.Unlock()

	// Check if file already exists
	if _, exists := w.openFilesPool.Load(fileName); exists {
		Debug("File already exists in pool")
		return nil
	}

	// Store file in pool
	w.openFilesPool.Store(fileName, file)
	w.connLastUsed.Store(fileName, time.Now())

	Debug("File %s added to pool", fileName)
	return nil
}

// Function to remove file from openFilesPool
func (w *Writer) RemoveConn(file *os.File) error {
	// Get file name
	fileName := file.Name()
	// Check if file exists
	if _, ok := w.openFilesPool.Load(fileName); ok {
		// Remove file from openFilesPool
		Debug("File %s removed from pool", fileName)
		w.openFilesPool.Delete(fileName)
		return nil
	}
	Debug("File %s not found", fileName)
	return fmt.Errorf("file %s not found", fileName)
}

// GetConn returns the file from openFilesPool if it exists, or creates a new connection
// if the pool is not full. If the pool is full, it closes the last element and then
// creates a new connection. The function returns the file and an error if the file
// does not exist or if creating a new connection fails.
func (w *Writer) GetConn(file *os.File) (*os.File, error) {
	// Nil check
	if file == nil {
		return nil, fmt.Errorf("nil file pointer received")
	}

	// Get file name
	fileName := file.Name()

	// Check if file exists in pool
	if existingFile, ok := w.openFilesPool.Load(fileName); ok {
		Debug("File %s found in pool", fileName)
		fileObj := existingFile.(*os.File)

		// Verify file is still usable
		if _, err := fileObj.Stat(); err == nil {
			w.connLastUsed.Store(fileName, time.Now())
			return fileObj, nil
		} else {
			// File is not usable, remove it from pool
			w.openFilesPool.Delete(fileName)
			w.connLastUsed.Delete(fileName)
			Debug("File %s in pool is no longer usable: %v", fileName, err)
		}
	}

	// Find least recently used connection if pool is full
	var count int
	w.openFilesPool.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	// Check if pool is full
	if uint64(count) >= w.maxConns {
		var oldestFile string
		var oldestTime time.Time

		w.connLastUsed.Range(func(key, value interface{}) bool {
			lastUsed := value.(time.Time)
			if oldestFile == "" || lastUsed.Before(oldestTime) {
				oldestFile = key.(string)
				oldestTime = lastUsed
			}
			return true
		})

		// Close and remove oldest connection
		if oldestFile != "" {
			if oldFile, ok := w.openFilesPool.Load(oldestFile); ok {
				oldFile.(*os.File).Close()
				w.openFilesPool.Delete(oldestFile)
				w.connLastUsed.Delete(oldestFile)
			}
		}
	}

	// Store new connection
	w.openFilesPool.Store(fileName, file)
	w.connLastUsed.Store(fileName, time.Now())

	return file, nil

}

// CheckConnStatus verifies whether a given file is open and usable within the Writer's openFilesPool.
// It first checks if the file exists in the openFilesPool by its name. If the file is found,
// it further checks if the file is still usable by calling Stat on it. If the file is usable,
// it returns true. If the file is not found in the pool or is not usable, it logs the status
// and returns false.
func (w *Writer) CheckConnStatus(file *os.File) bool {
	w.connPoolLock.RLock()
	defer w.connPoolLock.RUnlock()

	// Check if file is nil
	if file == nil {
		Debug("File is nil")
		return false
	}

	// Get file name
	fileName := file.Name()

	// Check if file exists in pool
	if poolFile, ok := w.openFilesPool.Load(fileName); ok {
		Debug("File %s found in pool", fileName)

		// Type assert from interface{} to *os.File
		fileObj, ok := poolFile.(*os.File)
		if !ok {
			Debug("File %s in pool is not a valid os.File", fileName)
			return false
		}

		// Try to check if file is usable
		_, err := fileObj.Stat()
		if err != nil {
			Debug("File %s is closed or has error: %v", fileName, err)
			return false
		}

		return true
	}

	Debug("File %s not found in pool", fileName)
	return false
}

// CloseConn closes a file if it is present in the openFilesPool and removes it
// from the pool. It logs and returns an error if the file cannot be closed or
// if the file is not found in the pool. Upon successful closure, the file is
// removed from openFilesPool and a log entry for the closure is made.
func (w *Writer) CloseConn(file *os.File) error {
	if file == nil {
		return fmt.Errorf("file is nil")
	}

	// Get file name
	fileName := file.Name()
	// Check if file exists and get it if it does
	w.connPoolLock.Lock()
	poolFile, ok := w.openFilesPool.Load(fileName)
	if !ok {
		w.connPoolLock.Unlock()
		Debug("File %s not found in pool", fileName)
		return fmt.Errorf("file %s not found in pool", fileName)
	}

	// Remove file from openFilesPool first
	w.openFilesPool.Delete(fileName)
	// Remove from last used tracking
	w.connLastUsed.Delete(fileName)
	w.connPoolLock.Unlock()

	// Now close the file (outside of the lock to avoid blocking)
	fileObj, ok := poolFile.(*os.File)
	if !ok {
		Debug("File %s in pool is not a valid os.File", fileName)
		return fmt.Errorf("file %s in pool is not a valid os.File", fileName)
	}

	// Close file
	err := fileObj.Close()
	if err != nil {
		Debug("Error closing file %s: %v", fileName, err)
		return fmt.Errorf("error closing file %s: %v", fileName, err)
	}

	Debug("File %s closed", fileName)
	return nil
}

// CloseAllConns closes all files in the openFilesPool and removes them from the
// pool. If any of the files cannot be closed, it logs and returns an error. If
// all files are closed successfully, it returns nil.
func (w *Writer) CloseAllConns() error {
	var errSlice []error

	// Create a copy of the pool to avoid modification during iteration
	filesToClose := make(map[string]*os.File)
	w.mu.Lock()
	w.openFilesPool.Range(func(key, value interface{}) bool {
		filesToClose[key.(string)] = value.(*os.File)
		return true
	})
	w.mu.Unlock()

	for name, file := range filesToClose {
		// First check if file is already closed
		if _, err := file.Stat(); err != nil {
			Debug("File %s appears already closed: %v", name, err)
			w.mu.Lock()
			w.openFilesPool.Delete(name)
			w.connLastUsed.Delete(name)
			w.mu.Unlock()
			continue
		}

		// Try to close the file
		err := file.Close()
		if err != nil {
			// Only consider it an error if it's not already closed
			if !strings.Contains(err.Error(), "file already closed") {
				Debug("Error closing file %s: %v", name, err)
				errSlice = append(errSlice, fmt.Errorf("error closing file %s: %v", name, err))
			} else {
				Debug("File %s was already closed", name)
			}
		}
		// Remove from pool regardless of close error
		w.mu.Lock()
		w.openFilesPool.Delete(name)
		w.connLastUsed.Delete(name)
		w.mu.Unlock()
		Debug("File %s closed or removed from pool", name)
	}

	if len(errSlice) > 0 {
		return fmt.Errorf("multiple errors closing files: %v", errSlice)
	}
	return nil
}

// ClearAll clears all the file connections in the openFilesPool and the last used
// file connections in connLastUsed. It is used to clear the file connections after
// writing to all files.
func (w *Writer) ClearAll() {
	w.mu.Lock()
	w.openFilesPool = sync.Map{}
	w.connLastUsed = sync.Map{}
	w.mu.Unlock()
}

// ClearFiles clears the Writer's files slice of pointers to os.File by
// setting it to a new empty slice. It is used to clear the files slice after
// writing to all files.
func (w *Writer) ClearFiles() {
	w.mu.Lock()
	emptySlice := make([]*os.File, 0)
	w.files = &emptySlice
	w.mu.Unlock()
}

// FactoryReset closes all open file connections and clears the pool, the last used
// file connections, and the files slice. It is used to reset the Writer to its
// initial state after writing to all files. It returns an error if closing the
// open file connections fails.
func (w *Writer) FactoryReset() error {
	err := w.CloseAllConns()
	if err != nil {
		return err
	}
	w.ClearAll()
	w.ClearFiles()
	return nil
}

// retry attempts to execute a given function multiple times, with a specified number of retries
// and a backoff period between attempts. If the number of retries is set to zero, the function
// is executed once without retrying. The function accepts a function as an argument that it
// will attempt to execute. If the function execution fails, it logs the error and waits for
// a backoff period before attempting again. It continues to retry until it succeeds or the
// retry count is exhausted.
//
// Parameters:
//   - function: The function to be executed, which takes an os.File, a string message,
//     a Results struct, and a RWMutex as arguments and returns an error.
//   - file: The os.File instance to be passed to the function.
//   - message: The message string to be passed to the function.
//   - results: A pointer to a Results struct to track outcomes of the function execution.
//   - mu: A pointer to a RWMutex for managing concurrent access during function execution.
//
// Returns:
//   - An error if the function execution fails after exhausting all retries, or nil if
//     the function succeeds.
func (w *Writer) retry(
	function func(*os.File, string, *Results, *sync.RWMutex) error,
	file *os.File,
	message string,
	results *Results,
	mu *sync.RWMutex,
) error {

	tries := w.retries
	backoff := w.backoff

	if tries == 0 {
		// Do func without retry
		err := function(file, message, results, mu)
		if err != nil {
			return err
		}
		return nil
	}

	for i := tries; i > 0; i-- {
		err := function(file, message, results, mu)
		if err == nil {
			return nil
		}
		Debug("Error: %v", err)
		if i == 1 { // Last retry
			return fmt.Errorf("exhausted retries: last error: %v", err)
		}
		Debug("Retrying... %d tries left", i-1)
		time.Sleep(time.Duration(backoff) * time.Millisecond)

		if backoff < 1000 {
			backoff *= 2
		}
	}

	return nil
}

// ----------------------------------------------------
// Writers Methods
// ----------------------------------------------------

// Write writes the message to each file in the files slice, using a worker pool with
// the specified maximum number of workers. If maxWorkers is 0 or negative, the
// number of workers is set to the number of CPUs available. If maxWorkers is greater
// than the number of files, the number of workers is capped at the number of files.
//
// The function returns a Results struct containing the total number of files, the
// number of successful writes, the number of failed writes, the success rate, and
// the failure rate. If any of the writes fail, the error is stored in the ErrSlice
// field of the Results struct.
//
// The function also returns an error if the Writer's fullWriteCheck fails, if the
// files slice is empty, or if there is an error closing all file connections.
//
// The function uses a retry mechanism to handle transient errors when writing to
// files. The number of retries and the backoff between retries are specified by
// the Writer's retries and backoff fields, respectively. If all retries are exhausted
// without success, the error is returned.
//
// If the length of the files slices is greater than 1000, the function splits the
// files into chunks, calculated as the length of the file slice divided by the number
// of cpus, and writes them in parallel.
func (w *Writer) Write(maxWorkers int) (*Results, error) {
	// Check Context
	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	default:
	}

	if err := w.fullWriteCheck(); err != nil {
		return nil, err
	}
	if len(*w.files) == 0 {
		return nil, fmt.Errorf("files is empty")
	}

	// Initialize results
	results := NewResults()

	// Initialize Worker Count
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	// Cap Worker Count
	if maxWorkers > len(*w.files) {
		maxWorkers = len(*w.files)
	}

	// Initialize wait group
	wg := sync.WaitGroup{}

	// Create jobs channel
	jobs := make(chan *os.File, len(*w.files))

	// Start worker pool
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				// Get Connection
				file, errConn := w.GetConn(file)
				if errConn != nil {
					errCopy := errConn
					results.mu.Lock()
					results.ErrSlice = append(results.ErrSlice, &errCopy)
					results.Failure++
					results.mu.Unlock()
					continue
				}
				// Retry Wrapper
				err := w.retry(w.writeToFile, file, *w.message, results, &results.mu)
				if err != nil {
					errCopy := err
					results.mu.Lock()
					results.ErrSlice = append(results.ErrSlice, &errCopy)
					results.Failure++
					results.mu.Unlock()
				} else {
					results.mu.Lock()
					results.Success++
					results.mu.Unlock()
				}
			}
		}()
	}

	// Determine if batching is needed
	if len(*w.files) > 1000 {
		// Process in batches
		batches := w.batcher(0)
		for _, batch := range batches {
			for _, file := range batch {
				jobs <- file
			}
		}
	} else {
		// Process all files at once
		for _, file := range *w.files {
			jobs <- file
		}
	}

	// Close jobs channel to signal workers
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()

	// Calculate final results
	results.mu.Lock()
	defer results.mu.Unlock()

	// Set total
	results.Total = uint64(len(*w.files))

	// Calculate rates
	if results.Total > 0 {
		results.SuccessRate = float64(results.Success) / float64(results.Total)
		results.FailureRate = float64(results.Failure) / float64(results.Total)
	} else {
		results.SuccessRate = 0.0
		results.FailureRate = 0.0
	}

	return results, nil
}

// WriteWithTimeout writes the message to each file in the files slice with a specified timeout.
//
// This function creates a context with a timeout and uses it to control the execution
// of the Write method. If the timeout is reached before the write operation completes,
// the context is canceled, terminating the operation.
//
// Parameters:
//   - maxWorkers: The maximum number of concurrent workers to use for writing.
//   - timeout: The maximum duration allowed for the write operation before it is canceled.
//
// Returns:
//   - A Results struct containing statistics about the write operation, including the total
//     number of files processed, the number of successful writes, and the success rate.
//   - An error if the write operation fails, or if the context is canceled due to the timeout.
func (w *Writer) WriteWithTimeout(maxWorkers int, timeout time.Duration) (*Results, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Update the writer's context
	w.SetContext(ctx)

	// Call the regular Write method
	return w.Write(maxWorkers)
}

// StartWriteWithCancel starts a goroutine to write to all files in the writer with
// maxWorkers workers and returns a cancel function, a results channel, and an error
// channel. The cancel function can be used to cancel the writing operation. The
// results channel will receive a Results struct containing statistics about the
// write operation when it completes. The error channel will receive an error if
// the write operation fails. If the context is canceled, the write operation will
// be terminated and an error will be sent on the error channel.
func StartWriteWithCancel(w *Writer, maxWorkers int) (cancel func(), resultCh chan *Results, errCh chan error) {
	ctx, cancel := context.WithCancel(context.Background())
	resultCh = make(chan *Results, 1)
	errCh = make(chan error, 1)

	w.SetContext(ctx)

	go func() {
		results, err := w.Write(maxWorkers)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- results
	}()

	return cancel, resultCh, errCh
}

// ----------------------------------------------------
// Batcher
// ----------------------------------------------------

// batcher splits the files into batches of size batchSize
func (w *Writer) batcher(batchSize int) [][]*os.File {
	if batchSize <= 0 {
		batchSize = len(*w.files) / runtime.NumCPU()
		if batchSize == 0 {
			batchSize = 1 // Ensure at least batch size 1
		}
	}

	// Calculate number of batches
	numFiles := len(*w.files)
	numBatches := (numFiles + batchSize - 1) / batchSize

	batches := make([][]*os.File, numBatches)

	for i := 0; i < numBatches; i++ {
		// Calculate start and end index for this batch
		start := i * batchSize
		end := start + batchSize

		// Handle remainder
		if end > numFiles {
			end = numFiles
		}

		// Create the batch
		batches[i] = (*w.files)[start:end]
	}

	return batches
}

// ----------------------------------------------------
// End of File
// ----------------------------------------------------
