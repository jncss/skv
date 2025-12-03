// Example showing different use cases for SKV
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/skv"
)

// User represents a user object
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	db, err := skv.Open("usecases_example.skv")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Clear()

	fmt.Println("=== SKV Use Cases ===\n")

	// Use Case 1: Simple configuration storage
	fmt.Println("1. Configuration storage:")
	config := map[string]string{
		"app.name":      "MyApp",
		"app.version":   "1.0.0",
		"server.port":   "8080",
		"server.host":   "localhost",
		"db.max_conn":   "100",
		"cache.enabled": "true",
		"log.level":     "info",
	}

	db.PutBatchString(config)

	port := db.GetOrDefaultString("server.port", "3000")
	logLevel := db.GetOrDefaultString("log.level", "debug")
	fmt.Printf("  Server: %s:%s\n", db.GetOrDefaultString("server.host", "0.0.0.0"), port)
	fmt.Printf("  Log level: %s\n", logLevel)

	// Use Case 2: Session storage
	fmt.Println("\n2. Session storage:")
	sessions := map[string]string{
		"session:abc123": `{"user_id":1,"expires":"2024-12-31"}`,
		"session:def456": `{"user_id":2,"expires":"2024-12-31"}`,
		"session:ghi789": `{"user_id":3,"expires":"2024-12-31"}`,
	}

	db.PutBatchString(sessions)

	// Retrieve specific session
	sessionData, _ := db.GetString("session:abc123")
	fmt.Printf("  Session abc123: %s\n", sessionData)

	// Clean expired sessions
	count := 0
	db.ForEachString(func(key string, value string) error {
		if len(key) > 8 && key[:8] == "session:" {
			count++
		}
		return nil
	})
	fmt.Printf("  Active sessions: %d\n", count)

	// Use Case 3: Storing structured data (JSON)
	fmt.Println("\n3. Storing structured data (JSON):")

	user := User{
		ID:        1,
		Name:      "Alice",
		Email:     "alice@example.com",
		CreatedAt: time.Now(),
	}

	// Serialize to JSON
	userData, _ := json.Marshal(user)
	db.Put([]byte("user:1"), userData)

	// Deserialize from JSON
	storedData, _ := db.Get([]byte("user:1"))
	var retrievedUser User
	json.Unmarshal(storedData, &retrievedUser)

	fmt.Printf("  User: %s (%s)\n", retrievedUser.Name, retrievedUser.Email)
	fmt.Printf("  Created: %s\n", retrievedUser.CreatedAt.Format("2006-01-02 15:04:05"))

	// Use Case 4: Cache implementation
	fmt.Println("\n4. Simple cache:")

	// Simulate expensive operation
	getCachedValue := func(key string) string {
		if db.HasString(key) {
			value, _ := db.GetString(key)
			fmt.Printf("  ✓ Cache hit: %s\n", key)
			return value
		}

		fmt.Printf("  ✗ Cache miss: %s (computing...)\n", key)
		time.Sleep(100 * time.Millisecond) // Simulate expensive operation
		result := fmt.Sprintf("computed_value_for_%s", key)
		db.PutString(key, result)
		return result
	}

	getCachedValue("expensive:operation:1") // Cache miss
	getCachedValue("expensive:operation:1") // Cache hit
	getCachedValue("expensive:operation:2") // Cache miss

	// Use Case 5: Feature flags
	fmt.Println("\n5. Feature flags:")

	flags := map[string]string{
		"feature:new_ui":        "true",
		"feature:dark_mode":     "true",
		"feature:beta_features": "false",
		"feature:analytics":     "true",
	}
	db.PutBatchString(flags)

	isEnabled := func(feature string) bool {
		return db.GetOrDefaultString(feature, "false") == "true"
	}

	fmt.Printf("  New UI enabled: %v\n", isEnabled("feature:new_ui"))
	fmt.Printf("  Beta features enabled: %v\n", isEnabled("feature:beta_features"))
	fmt.Printf("  Unknown feature enabled: %v\n", isEnabled("feature:unknown"))

	// Use Case 6: Counters and metrics
	fmt.Println("\n6. Counters:")

	db.PutString("counter:page_views", "0")
	db.PutString("counter:api_calls", "0")

	// Increment counters
	for i := 0; i < 5; i++ {
		views, _ := db.GetString("counter:page_views")
		var count int
		fmt.Sscanf(views, "%d", &count)
		db.UpdateString("counter:page_views", fmt.Sprintf("%d", count+1))
	}

	pageViews, _ := db.GetString("counter:page_views")
	fmt.Printf("  Page views: %s\n", pageViews)

	// Use Case 7: Key-value with namespaces
	fmt.Println("\n7. Namespaced keys:")

	// Different namespaces
	db.PutString("user:settings:theme", "dark")
	db.PutString("user:settings:language", "en")
	db.PutString("app:config:timeout", "30")
	db.PutString("app:config:retries", "3")

	// List all user settings
	fmt.Println("  User settings:")
	db.ForEachString(func(key string, value string) error {
		if len(key) > 13 && key[:13] == "user:settings" {
			fmt.Printf("    %s = %s\n", key[14:], value)
		}
		return nil
	})

	fmt.Printf("\n✓ Total keys in database: %d\n", db.Count())
	fmt.Println("✓ Use cases example completed!")
}
