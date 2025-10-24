package conformance

import "testing"

func initStorage(_ *testing.T) ActivityPubStorage {
	storage := &memStorage{}
	return storage
}

func Test_Conformance(t *testing.T) {
	var suite TestType = TestActivityPub | TestKey | TestPassword | TestMetadata | TestOAuth
	suite.Run(t, initStorage(t))
}
