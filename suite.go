package conformance

import "testing"

type Suite struct {
	types   TestType
	storage ActivityPubStorage
}

type TestType uint16

const (
	TestActivityPub = 1
	TestKey         = 1 << iota
	TestPassword
	TestMetadata
	TestOAuth

	TestNone = 0
)

func Init(storage ActivityPubStorage, tests TestType) Suite {
	return Suite{storage: storage, types: tests}
}

func (s Suite) RunTests(t *testing.T) {
	if s.types == TestNone {
		t.Logf("No tests to run")
		return
	}
	if s.types&TestActivityPub == TestActivityPub {
		s.RunActivityPubTests(t)
	}
	if s.types&TestKey == TestKey {
		s.RunKeyTests(t)
	}
	if s.types&TestMetadata == TestMetadata {
		s.RunMetadataTests(t)
	}
	if s.types&TestPassword == TestPassword {
		s.RunPasswordTests(t)
	}
	if s.types&TestOAuth == TestOAuth {
		s.RunOAuthTests(t)
	}
}
