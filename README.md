# semaphore

Distributed semaphores


## usage
```go
  ec := NewEtcdKeysAPI(etcdKeysApiClient, "/path/to/semaphore")
	sem := NewSemaphore(ec, "holder_id")
  sem.SetLimit(10)
  if err := sem.Acquire(); err != nil {
    panic(err)
  }
  // do stuff
  if err := sem.Release(); err != nil {
    panic(err)
  }
```
