// Package utils provides utility functions for the HybridPipe system.
package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Storage provides persistent storage for the HybridPipe system.
type Storage struct {
	// directory is the directory where data is stored.
	directory string
	// mutex protects concurrent access to the storage.
	mutex sync.RWMutex
}

// NewStorage creates a new storage instance.
func NewStorage(directory string) (*Storage, error) {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return &Storage{
		directory: directory,
	}, nil
}

// Store stores data with the specified key.
func (s *Storage) Store(key string, data interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Encode the data as JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}
	
	// Create the file path
	filePath := filepath.Join(s.directory, key+".json")
	
	// Write the data to the file
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	return nil
}

// Load loads data with the specified key.
func (s *Storage) Load(key string, target interface{}) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Create the file path
	filePath := filepath.Join(s.directory, key+".json")
	
	// Read the data from the file
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to read data: %w", err)
	}
	
	// Decode the data from JSON
	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}
	
	return nil
}

// Delete deletes data with the specified key.
func (s *Storage) Delete(key string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Create the file path
	filePath := filepath.Join(s.directory, key+".json")
	
	// Delete the file
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to delete data: %w", err)
	}
	
	return nil
}

// List lists all keys in the storage.
func (s *Storage) List() ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Open the directory
	dir, err := os.Open(s.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage directory: %w", err)
	}
	defer dir.Close()
	
	// Read the directory entries
	entries, err := dir.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}
	
	// Extract the keys from the file names
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Get the file name without the extension
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".json" {
			continue
		}
		
		key := name[:len(name)-len(ext)]
		keys = append(keys, key)
	}
	
	return keys, nil
}

// Clear deletes all data in the storage.
func (s *Storage) Clear() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Open the directory
	dir, err := os.Open(s.directory)
	if err != nil {
		return fmt.Errorf("failed to open storage directory: %w", err)
	}
	defer dir.Close()
	
	// Read the directory entries
	entries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}
	
	// Delete all files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Delete the file
		filePath := filepath.Join(s.directory, entry.Name())
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", filePath, err)
		}
	}
	
	return nil
}

// Backup creates a backup of the storage.
func (s *Storage) Backup(writer io.Writer) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Open the directory
	dir, err := os.Open(s.directory)
	if err != nil {
		return fmt.Errorf("failed to open storage directory: %w", err)
	}
	defer dir.Close()
	
	// Read the directory entries
	entries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}
	
	// Create a map to store all data
	data := make(map[string]interface{})
	
	// Load all data
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Get the file name without the extension
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".json" {
			continue
		}
		
		key := name[:len(name)-len(ext)]
		
		// Read the data from the file
		filePath := filepath.Join(s.directory, entry.Name())
		jsonData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		
		// Decode the data from JSON
		var value interface{}
		if err := json.Unmarshal(jsonData, &value); err != nil {
			return fmt.Errorf("failed to decode data from file %s: %w", filePath, err)
		}
		
		// Add to the map
		data[key] = value
	}
	
	// Encode the map as JSON
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode backup data: %w", err)
	}
	
	return nil
}

// Restore restores the storage from a backup.
func (s *Storage) Restore(reader io.Reader) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Clear the storage
	if err := s.Clear(); err != nil {
		return fmt.Errorf("failed to clear storage: %w", err)
	}
	
	// Decode the backup data
	var data map[string]interface{}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode backup data: %w", err)
	}
	
	// Store all data
	for key, value := range data {
		// Encode the data as JSON
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to encode data for key %s: %w", key, err)
		}
		
		// Create the file path
		filePath := filepath.Join(s.directory, key+".json")
		
		// Write the data to the file
		if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write data for key %s: %w", key, err)
		}
	}
	
	return nil
}

// GlobalStorage is a global instance of Storage.
var GlobalStorage *Storage

// InitGlobalStorage initializes the global storage instance.
func InitGlobalStorage(directory string) error {
	var err error
	GlobalStorage, err = NewStorage(directory)
	return err
}
