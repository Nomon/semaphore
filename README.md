# semaphore

Distributed semaphores


## aquiring lock
```go
  // create semaphore client
  ec := NewEtcdKeysAPI(etcdKeysApiClient, "/path/to/semaphore")
  // create a semaphore
  sem := NewSemaphore(ec, "holder_id")
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

## creating a new semaphore

```go
  // create semaphore client
  ec := NewEtcdKeysAPI(etcdKeysApiClient, "/path/to/semaphore")
  // create a semaphore
  sem := NewSemaphore(ec, "holder_id")
  // not strictly necessary but useful if reusing a semaphore path.
  // removes all holders, sets limit to 1.
  sem.Reset()
  // set the proper limit of 10.
  if err := sem.SetLimit(10); err != nil {
    panic(err)
  }
```
