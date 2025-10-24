package conformance

import (
	"testing"

	vocab "github.com/go-ap/activitypub"
)

type PasswordStorage interface {
	PasswordSet(it vocab.IRI, pw []byte) error
	PasswordCheck(it vocab.IRI, pw []byte) error
}

var rootPw = []byte("notSoSecretP4ssw0rd")

func initPasswordStorage(s Suite) error {
	pwStorage, ok := s.storage.(PasswordStorage)
	if ok {
		err := pwStorage.PasswordSet(root.ID, rootPw)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Suite) RunPasswordTests(t *testing.T) {
	if err := initPasswordStorage(s); err != nil {
		t.Errorf("unable to init Password test suite: %s", err)
		return
	}
	t.Errorf("%s", errNotImplemented)
}
