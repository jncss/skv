package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jncss/skv"
)

// Example 1: User Session Storage
func sessionStorageExample() {
	fmt.Println("=== Use Case 1: User Session Storage ===\n")

	os.MkdirAll("data", 0755)

	db, err := skv.Open("data/sessions")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any previous data
	db.Clear()

	// Create a session
	sessionID := "sess_abc123xyz"
	sessionData := `{"user_id": "user_456", "email": "alice@example.com", "login_time": "2024-12-06T10:30:00Z"}`

	err = db.PutString(sessionID, sessionData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Session created: %s\n", sessionID)

	// Retrieve session
	data, err := db.GetString(sessionID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Session data: %s\n", data)

	// Update session (e.g., update last activity)
	updatedData := `{"user_id": "user_456", "email": "alice@example.com", "last_activity": "2024-12-06T11:45:00Z"}`
	err = db.UpdateString(sessionID, updatedData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Session updated\n")

	// Check if session exists (e.g., for authentication)
	if db.HasString(sessionID) {
		fmt.Printf("✓ Session is valid\n")
	}

	// Delete session (logout)
	err = db.DeleteString(sessionID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Session deleted (user logged out)\n\n")
}

// Example 2: Application Configuration
func configurationExample() {
	fmt.Println("=== Use Case 2: Application Configuration ===\n")

	db, err := skv.Open("data/config")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any previous data
	db.Clear()

	// Store configuration settings
	config := map[string]string{
		"app:name":          "MyApp",
		"app:version":       "1.2.0",
		"db:host":           "localhost",
		"db:port":           "5432",
		"cache:enabled":     "true",
		"cache:ttl":         "3600",
		"feature:dark_mode": "true",
	}

	err = db.PutBatchString(config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Stored %d configuration settings\n", len(config))

	// Read specific settings
	appName := db.GetOrDefaultString("app:name", "Unknown App")
	cacheEnabled := db.GetOrDefaultString("cache:enabled", "false")

	fmt.Printf("✓ App Name: %s\n", appName)
	fmt.Printf("✓ Cache Enabled: %s\n", cacheEnabled)

	// List all database-related settings
	fmt.Println("\nDatabase settings:")
	db.ForEachString(func(key string, value string) error {
		if len(key) >= 3 && key[:3] == "db:" {
			fmt.Printf("  %s = %s\n", key, value)
		}
		return nil
	})

	fmt.Println()
}

// Example 3: Simple Cache
func cacheExample() {
	fmt.Println("=== Use Case 3: Simple Cache ===\n")

	db, err := skv.Open("data/cache")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any previous data
	db.Clear()

	// Simulate caching API responses
	type CacheEntry struct {
		key   string
		value string
	}

	entries := []CacheEntry{
		{"api:users:123", `{"id": 123, "name": "Alice"}`},
		{"api:posts:456", `{"id": 456, "title": "Hello World"}`},
		{"api:comments:789", `{"id": 789, "text": "Great post!"}`},
	}

	// Cache miss - fetch and store
	for _, entry := range entries {
		if !db.HasString(entry.key) {
			fmt.Printf("Cache MISS: %s\n", entry.key)
			db.PutString(entry.key, entry.value)
			fmt.Printf("  ✓ Cached: %s\n", entry.key)
		}
	}

	// Cache hit - retrieve from cache
	fmt.Println("\nRetrieving from cache:")
	for _, entry := range entries {
		if db.HasString(entry.key) {
			data, _ := db.GetString(entry.key)
			fmt.Printf("Cache HIT: %s -> %s\n", entry.key, data)
		}
	}

	// Cache invalidation
	fmt.Println("\nCache invalidation:")
	db.DeleteString("api:users:123")
	fmt.Println("✓ Invalidated cache for api:users:123")

	fmt.Println()
}

// Example 4: Job Queue / Task Storage
func jobQueueExample() {
	fmt.Println("=== Use Case 4: Job Queue / Task Storage ===\n")

	db, err := skv.Open("data/jobs")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any previous data
	db.Clear()

	// Add jobs to queue
	jobs := map[string]string{
		"job:001": `{"type": "email", "to": "user@example.com", "status": "pending"}`,
		"job:002": `{"type": "report", "format": "pdf", "status": "pending"}`,
		"job:003": `{"type": "backup", "target": "s3", "status": "pending"}`,
	}

	err = db.PutBatchString(jobs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Queued %d jobs\n", len(jobs))

	// Process jobs
	fmt.Println("\nProcessing jobs:")
	processedCount := 0

	// First, collect job keys to process
	var jobKeys []string
	db.ForEachString(func(key string, value string) error {
		if len(key) >= 4 && key[:4] == "job:" {
			jobKeys = append(jobKeys, key)
		}
		return nil
	})

	// Then process them (outside of ForEach to avoid deadlock)
	for _, key := range jobKeys {
		fmt.Printf("  Processing %s...\n", key)

		// Simulate processing
		time.Sleep(10 * time.Millisecond)

		// Update status
		updatedValue := `{"status": "completed", "completed_at": "2024-12-06T12:00:00Z"}`
		db.UpdateString(key, updatedValue)

		processedCount++
		fmt.Printf("  ✓ Completed %s\n", key)
	}

	fmt.Printf("\n✓ Processed %d jobs\n", processedCount)
	fmt.Println()
}

// Example 5: Feature Flags
func featureFlagsExample() {
	fmt.Println("=== Use Case 5: Feature Flags ===\n")

	db, err := skv.Open("data/features")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any previous data
	db.Clear()

	// Define feature flags
	features := map[string]string{
		"feature:new_ui":          "true",
		"feature:beta_api":        "false",
		"feature:dark_mode":       "true",
		"feature:experimental":    "false",
		"feature:payment_gateway": "true",
	}

	err = db.PutBatchString(features)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Initialized %d feature flags\n", len(features))

	// Check feature flags
	fmt.Println("\nFeature status:")

	checkFeature := func(name string) {
		enabled := db.GetOrDefaultString(name, "false")
		status := "❌ Disabled"
		if enabled == "true" {
			status = "✅ Enabled"
		}
		fmt.Printf("  %s: %s\n", name, status)
	}

	checkFeature("feature:new_ui")
	checkFeature("feature:beta_api")
	checkFeature("feature:dark_mode")

	// Toggle a feature
	fmt.Println("\nToggling feature:beta_api to enabled...")
	db.UpdateString("feature:beta_api", "true")

	checkFeature("feature:beta_api")

	// List all enabled features
	fmt.Println("\nAll enabled features:")
	db.ForEachString(func(key string, value string) error {
		if value == "true" {
			featureName := key
			if len(key) >= 8 && key[:8] == "feature:" {
				featureName = key[8:]
			}
			fmt.Printf("  ✅ %s\n", featureName)
		}
		return nil
	})

	fmt.Println()
}

// Example 6: User Preferences
func userPreferencesExample() {
	fmt.Println("=== Use Case 6: User Preferences ===\n")

	db, err := skv.Open("data/preferences")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Clear any previous data
	db.Clear()

	// Store user preferences
	userID := "user_789"

	prefs := map[string]string{
		fmt.Sprintf("%s:theme", userID):         "dark",
		fmt.Sprintf("%s:language", userID):      "en",
		fmt.Sprintf("%s:timezone", userID):      "UTC",
		fmt.Sprintf("%s:notifications", userID): "enabled",
		fmt.Sprintf("%s:email_digest", userID):  "weekly",
	}

	err = db.PutBatchString(prefs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Stored preferences for %s\n", userID)

	// Retrieve user preferences
	fmt.Println("\nUser preferences:")
	db.ForEachString(func(key string, value string) error {
		prefix := userID + ":"
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			setting := key[len(prefix):]
			fmt.Printf("  %s: %s\n", setting, value)
		}
		return nil
	})

	// Update a preference
	fmt.Println("\nUpdating theme preference...")
	themeKey := fmt.Sprintf("%s:theme", userID)
	db.UpdateString(themeKey, "light")

	newTheme, _ := db.GetString(themeKey)
	fmt.Printf("✓ Theme updated to: %s\n", newTheme)

	fmt.Println()
}

func main() {
	fmt.Println("╔════════════════════════════════════════════╗")
	fmt.Println("║   SKV - Real-World Use Cases Examples     ║")
	fmt.Println("╚════════════════════════════════════════════╝")
	fmt.Println()

	sessionStorageExample()
	configurationExample()
	cacheExample()
	jobQueueExample()
	featureFlagsExample()
	userPreferencesExample()

	fmt.Println("✅ All use case examples completed successfully!")
}
