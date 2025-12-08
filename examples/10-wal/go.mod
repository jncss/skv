module wal_example

go 1.25.1

replace github.com/jncss/skv => ../..

require github.com/jncss/skv v0.0.0-00010101000000-000000000000

require (
	github.com/golang/snappy v1.0.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
)
