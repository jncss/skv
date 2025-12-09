package main

import (
	"fmt"
	"os"

	"github.com/jncss/skv"
)

func main() {
	fmt.Println("=== SKV Structured Logging Demo ===")
	fmt.Println()

	// Clean up from previous runs
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")

	// Example 1: Default (No Logging)
	fmt.Println("Example 1: Default behavior (NullLogger)")
	fmt.Println("----------------------------------------")
	db1, err := skv.Open("demo.skv")
	if err != nil {
		panic(err)
	}
	db1.Put([]byte("user:1"), []byte("Alice"))
	db1.Close()
	fmt.Println("✓ Operations completed silently (no logs)")
	fmt.Println()

	// Clean up
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")

	// Example 2: Text Logger (Human-Readable)
	fmt.Println("Example 2: Text logger (development mode)")
	fmt.Println("----------------------------------------")
	textLogger := skv.NewTextLogger(os.Stdout, skv.LogLevelDebug)
	db2, err := skv.OpenWithOptions("demo.skv", &skv.Options{
		Logger: textLogger,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("\n--- Performing operations with text logging:")
	db2.Put([]byte("user:1"), []byte("Alice"))
	db2.Put([]byte("user:2"), []byte("Bob"))
	db2.Get([]byte("user:1"))
	db2.Update([]byte("user:1"), []byte("Alice Smith"))
	db2.Close()
	fmt.Println()

	// Clean up
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")

	// Example 3: JSON Logger (Production)
	fmt.Println("Example 3: JSON logger (production mode)")
	fmt.Println("----------------------------------------")
	jsonLogger := skv.NewJSONLogger(os.Stdout, skv.LogLevelInfo)
	db3, err := skv.OpenWithOptions("demo.skv", &skv.Options{
		Logger:      jsonLogger,
		Compression: skv.CompressionLZ4,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("\n--- Performing operations with JSON logging (Info level):")
	// These won't log (Debug level)
	db3.Put([]byte("product:1"), []byte("Laptop"))
	db3.Get([]byte("product:1"))

	// But Verify will log (Info level)
	stats, _ := db3.Verify()
	fmt.Printf("\n--- Stats: %d total records, %.2f%% efficiency\n", stats.TotalRecords, stats.Efficiency)

	db3.Close()
	fmt.Println()

	// Clean up
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")

	// Example 4: Log Levels
	fmt.Println("Example 4: Log levels")
	fmt.Println("----------------------------------------")

	fmt.Println("\n--- LogLevelError (only errors):")
	errorLogger := skv.NewTextLogger(os.Stdout, skv.LogLevelError)
	db4, _ := skv.OpenWithOptions("demo.skv", &skv.Options{
		Logger: errorLogger,
	})
	db4.Put([]byte("key1"), []byte("value1"))
	_, err = db4.Get([]byte("nonexistent"))
	if err != nil {
		fmt.Printf("Expected error: %v (but no log because it's Debug level)\n", err)
	}
	db4.Close()

	// Clean up
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")

	fmt.Println("\n--- LogLevelDebug (all operations):")
	debugLogger := skv.NewTextLogger(os.Stdout, skv.LogLevelDebug)
	db5, _ := skv.OpenWithOptions("demo.skv", &skv.Options{
		Logger: debugLogger,
	})
	db5.Put([]byte("key1"), []byte("value1"))
	_, err = db5.Get([]byte("nonexistent"))
	if err != nil {
		fmt.Printf("\nExpected error: %v\n", err)
	}
	db5.Close()
	fmt.Println()

	// Clean up
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")

	// Example 5: Log to File
	fmt.Println("Example 5: Log to file")
	fmt.Println("----------------------------------------")
	logFile, err := os.OpenFile("demo.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	fileLogger := skv.NewJSONLogger(logFile, skv.LogLevelInfo)
	db6, _ := skv.OpenWithOptions("demo.skv", &skv.Options{
		Logger:      fileLogger,
		Compression: skv.CompressionSnappy,
	})

	// Perform some operations
	for i := 1; i <= 100; i++ {
		key := []byte(fmt.Sprintf("key:%d", i))
		value := []byte(fmt.Sprintf("value:%d", i))
		db6.Put(key, value)
	}

	db6.Verify()
	db6.Compact()
	db6.Close()
	logFile.Close()

	fmt.Println("✓ Logged 100 operations to demo.log")

	// Show log file content
	logContent, _ := os.ReadFile("demo.log")
	fmt.Printf("\n--- First 500 chars of demo.log:\n%s...\n\n", string(logContent[:min(500, len(logContent))]))

	// Example 6: Dynamic Log Level
	fmt.Println("Example 6: Dynamic log level changes")
	fmt.Println("----------------------------------------")
	dynamicLogger := skv.NewTextLogger(os.Stdout, skv.LogLevelError)
	db7, _ := skv.OpenWithOptions("demo.skv", &skv.Options{
		Logger: dynamicLogger,
	})

	fmt.Println("\n--- Level = Error (nothing logged):")
	db7.Put([]byte("test"), []byte("value"))

	fmt.Println("\n--- Changing to Debug level:")
	dynamicLogger.SetLevel(skv.LogLevelDebug)
	db7.Put([]byte("test2"), []byte("value2"))

	db7.Close()

	// Final cleanup
	os.Remove("demo.skv")
	os.Remove("demo.skv.wal")
	os.Remove("demo.log")

	fmt.Println("\n=== Demo completed ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
