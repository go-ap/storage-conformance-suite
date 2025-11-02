package conformance

import "testing"

type TestType uint16

const (
	TestActivityPub = 1
	TestKey         = 1 << iota
	TestPassword
	TestMetadata
	TestOAuth

	TestNone = 0

	TestsFull = TestActivityPub | TestKey | TestPassword | TestMetadata | TestOAuth
)

func Suite(tt ...TestType) TestType {
	var result TestType = TestNone
	for _, t := range tt {
		result = result | t
	}
	return result
}

type Opener interface {
	Open() error
}

type NilCloser interface {
	Close()
}

func maybeOpen(t *testing.T, storage ActivityPubStorage) func() {
	if opener, ok := storage.(Opener); ok {
		err := opener.Open()
		if err != nil {
			t.Fatalf("Unable to open storage: %s", err)
		}
	}
	if closer, ok := storage.(NilCloser); ok {
		return closer.Close
	}
	return func() {}
}

func (tt TestType) Run(t *testing.T, storage ActivityPubStorage) {
	maybeClose := maybeOpen(t, storage)
	defer maybeClose()

	t.Helper()

	if tt == TestNone {
		t.Logf("No tests to run")
		return
	}
	if tt&TestActivityPub == TestActivityPub {
		t.Run("ActivityPub tests", func(t *testing.T) {
			RunActivityPubTests(t, storage)
		})
	}
	if tt&TestOAuth == TestOAuth {
		t.Run("OAuth2 tests", func(t *testing.T) {
			RunOAuthTests(t, storage)
		})
	}
	if tt&TestKey == TestKey {
		t.Run("Key tests", func(t *testing.T) {
			RunKeyTests(t, storage)
		})
	}
	if tt&TestPassword == TestPassword {
		t.Run("Password tests", func(t *testing.T) {
			RunPasswordTests(t, storage)
		})
	}
	if tt&TestMetadata == TestMetadata {
		t.Run("MetaData tests", func(t *testing.T) {
			RunMetadataTests(t, storage)
		})
	}
}
