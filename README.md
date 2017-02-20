# semaphore

Distributed semaphores


## usage
```go
  // create semaphore client
  ec := NewEtcdKeysAPI(etcdKeysApiClient, "/path/to/semaphore")
  // create a semaphore
  sem := NewSemaphore(ec, "holder_id")
  // set its limit to 10
  sem.SetLimit(10)
  // aquire it
  if err := sem.Acquire(); err != nil {
    panic(err)
  }
  // do stuff, max limit concurrent here.

  // release it
  if err := sem.Release(); err != nil {
    panic(err)
  }
```
