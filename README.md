## Conformance tests for GoActivityPub storage backends

### Usage

If you have implemented a backend for the [GoActivityPub library](https://github.com/go-ap), this here you have the test suite to verify that it will behave predictably for the other packages.

```go
import (
    "testing"

    conformance "github.com/go-ap/storage-conformance-suite"
)

// TODO
// write your own initializing function that returns a ready to use instance
// of calls t.Fatal if errors are encountered.
var storageInit func(*testing.T) conformance.ActivityPubStorage

func Test_Conformance(t *testing.T) {
    suite := conformance.Suite(conformance.TestActivityPub, conformance.TestKey)
    suite.Run(t, storageInit(t))
}
```
