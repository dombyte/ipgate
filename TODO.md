# TODO

### [] Fix Cache 
Potentially use a different caching mechanism or library to improve performance and reliability of the cache system. Also to fix the Gosec G115 issue in cache.go with the current implementation of the cache.
```go
func (c *Cache) getShard(key string) *shard {
    shardIndex := xxhash.Sum64([]byte(key)) % uint64(c.numShards) // #nosec G115 needs to be addressed todo
    return &c.shards[shardIndex]
}
```

### [] Refactor codebase with multiple files / modules
Analyze and then refactor codebase to put components in different files or modules where it make sense to maintain better readability and an ease of understandment of codebase

### [] Refactor codebase to use api for component interface
So that different components of the codebase like ipmatcher etc. are all interfaced with an clear and structured api for better understanndability and reusability.
