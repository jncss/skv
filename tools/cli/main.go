package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "put":
		handlePut()
	case "get":
		handleGet()
	case "update":
		handleUpdate()
	case "delete":
		handleDelete()
	case "exists":
		handleExists()
	case "count":
		handleCount()
	case "keys":
		handleKeys()
	case "clear":
		handleClear()
	case "foreach":
		handleForEach()
	case "putfile":
		handlePutFile()
	case "getfile":
		handleGetFile()
	case "updatefile":
		handleUpdateFile()
	case "putstream":
		handlePutStream()
	case "getstream":
		handleGetStream()
	case "updatestream":
		handleUpdateStream()
	case "putbatch":
		handlePutBatch()
	case "getbatch":
		handleGetBatch()
	case "backup":
		handleBackup()
	case "restore":
		handleRestore()
	case "verify":
		handleVerify()
	case "compact":
		handleCompact()
	case "help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("SKV - Simple Key-Value Database CLI")
	fmt.Println()
	fmt.Println("Usage: skv <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  Basic Operations:")
	fmt.Println("    put <db> <key> <value>           Store a key-value pair")
	fmt.Println("    get <db> <key>                   Retrieve a value")
	fmt.Println("    update <db> <key> <value>        Update an existing key")
	fmt.Println("    delete <db> <key>                Delete a key")
	fmt.Println("    exists <db> <key>                Check if key exists")
	fmt.Println("    count <db>                       Count active keys")
	fmt.Println("    keys <db>                        List all keys")
	fmt.Println("    clear <db>                       Remove all keys")
	fmt.Println("    foreach <db>                     Iterate over all keys")
	fmt.Println()
	fmt.Println("  File Operations:")
	fmt.Println("    putfile <db> <key> <file>        Store file contents")
	fmt.Println("    getfile <db> <key> <file>        Retrieve to file")
	fmt.Println("    updatefile <db> <key> <file>     Update with file contents")
	fmt.Println()
	fmt.Println("  Streaming Operations (for large values):")
	fmt.Println("    putstream <db> <key> <file>      Stream file to database")
	fmt.Println("    getstream <db> <key> <file>      Stream value to file")
	fmt.Println("    updatestream <db> <key> <file>   Update via streaming")
	fmt.Println()
	fmt.Println("  Batch Operations:")
	fmt.Println("    putbatch <db> <key1> <val1> ...  Store multiple pairs")
	fmt.Println("    getbatch <db> <key1> <key2> ...  Retrieve multiple keys")
	fmt.Println()
	fmt.Println("  Backup & Maintenance:")
	fmt.Println("    backup <db> <json-file>          Create JSON backup")
	fmt.Println("    restore <db> <json-file>         Restore from backup")
	fmt.Println("    verify <db>                      Check integrity & stats")
	fmt.Println("    compact <db>                     Remove deleted records")
	fmt.Println()
	fmt.Println("  Help:")
	fmt.Println("    help                             Show detailed help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  skv put mydb.skv name 'John Doe'")
	fmt.Println("  skv get mydb.skv name")
	fmt.Println("  skv putfile mydb.skv config config.ini")
	fmt.Println("  skv backup mydb.skv backup.json")
	fmt.Println()
}

func printHelp() {
	printUsage()
	fmt.Println("Detailed Command Information:")
	fmt.Println()
	fmt.Println("PUT - Store a new key-value pair")
	fmt.Println("  Usage: skv put <database> <key> <value>")
	fmt.Println("  Note: Returns error if key already exists. Use 'update' to modify.")
	fmt.Println()
	fmt.Println("GET - Retrieve a value")
	fmt.Println("  Usage: skv get <database> <key>")
	fmt.Println("  Output: Prints the value to stdout")
	fmt.Println()
	fmt.Println("UPDATE - Update an existing key")
	fmt.Println("  Usage: skv update <database> <key> <value>")
	fmt.Println("  Note: Returns error if key doesn't exist. Use 'put' for new keys.")
	fmt.Println()
	fmt.Println("DELETE - Delete a key")
	fmt.Println("  Usage: skv delete <database> <key>")
	fmt.Println()
	fmt.Println("EXISTS - Check if a key exists")
	fmt.Println("  Usage: skv exists <database> <key>")
	fmt.Println("  Output: 'true' or 'false'")
	fmt.Println()
	fmt.Println("COUNT - Count active keys")
	fmt.Println("  Usage: skv count <database>")
	fmt.Println("  Output: Number of active keys")
	fmt.Println()
	fmt.Println("KEYS - List all keys")
	fmt.Println("  Usage: skv keys <database>")
	fmt.Println("  Output: One key per line")
	fmt.Println()
	fmt.Println("CLEAR - Remove all keys")
	fmt.Println("  Usage: skv clear <database>")
	fmt.Println("  Warning: This operation cannot be undone!")
	fmt.Println()
	fmt.Println("FOREACH - Iterate over all key-value pairs")
	fmt.Println("  Usage: skv foreach <database>")
	fmt.Println("  Output: key=value (one per line)")
	fmt.Println()
	fmt.Println("PUTFILE - Store file contents as a value")
	fmt.Println("  Usage: skv putfile <database> <key> <filepath>")
	fmt.Println("  Note: Reads entire file into memory")
	fmt.Println()
	fmt.Println("GETFILE - Retrieve value to a file")
	fmt.Println("  Usage: skv getfile <database> <key> <filepath>")
	fmt.Println("  Note: Creates or overwrites the file")
	fmt.Println()
	fmt.Println("UPDATEFILE - Update key with file contents")
	fmt.Println("  Usage: skv updatefile <database> <key> <filepath>")
	fmt.Println()
	fmt.Println("PUTSTREAM - Stream large file to database")
	fmt.Println("  Usage: skv putstream <database> <key> <filepath>")
	fmt.Println("  Note: Memory-efficient for large files")
	fmt.Println()
	fmt.Println("GETSTREAM - Stream value to file")
	fmt.Println("  Usage: skv getstream <database> <key> <filepath>")
	fmt.Println("  Note: Memory-efficient for large values")
	fmt.Println()
	fmt.Println("UPDATESTREAM - Update via streaming")
	fmt.Println("  Usage: skv updatestream <database> <key> <filepath>")
	fmt.Println()
	fmt.Println("PUTBATCH - Store multiple key-value pairs")
	fmt.Println("  Usage: skv putbatch <database> <key1> <value1> <key2> <value2> ...")
	fmt.Println("  Note: All keys must be new (not exist)")
	fmt.Println()
	fmt.Println("GETBATCH - Retrieve multiple keys")
	fmt.Println("  Usage: skv getbatch <database> <key1> <key2> <key3> ...")
	fmt.Println("  Output: key=value (one per line)")
	fmt.Println()
	fmt.Println("BACKUP - Create JSON backup")
	fmt.Println("  Usage: skv backup <database> <json-file>")
	fmt.Println("  Note: Creates human-readable JSON backup")
	fmt.Println()
	fmt.Println("RESTORE - Restore from JSON backup")
	fmt.Println("  Usage: skv restore <database> <json-file>")
	fmt.Println("  Note: Overwrites existing keys with same name")
	fmt.Println()
	fmt.Println("VERIFY - Check database integrity")
	fmt.Println("  Usage: skv verify <database>")
	fmt.Println("  Output: Database statistics and health info")
	fmt.Println()
	fmt.Println("COMPACT - Remove deleted records")
	fmt.Println("  Usage: skv compact <database>")
	fmt.Println("  Note: Reduces file size by removing wasted space")
	fmt.Println()
}
