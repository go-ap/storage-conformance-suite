## Conformance tests for GoActivityPub storage backends

### Usage

If you have implemented a backend for the GoActivityPub library, this here you have the test suite to verify that it will behave predictably for the other packages.

```go
import (
    "testing"

    conformance "github.com/go-ap/storage-conformance-suite"
)

var initStorage func() conformance.ActivityPubStorage


func Test_Conformance(t *testing.T) {
    suite := conformance.Init(initStorage(), conformance.TestActivityPub)
    suite.Run(t)
}
```
