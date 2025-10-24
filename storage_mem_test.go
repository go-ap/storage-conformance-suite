package conformance

import "testing"

func initStorage(_ *testing.T) ActivityPubStorage {
	storage := &memStorage{}
	return storage
}

func Test_Conformance(t *testing.T) {
	var enabledTests TestType = TestActivityPub | TestKey | TestPassword | TestMetadata | TestOAuth
	Init(initStorage(t), enabledTests).RunTests(t)
}
