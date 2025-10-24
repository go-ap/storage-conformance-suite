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

func initPasswordStorage(storage ActivityPubStorage) error {
	pwStorage, ok := storage.(PasswordStorage)
	if ok {
		err := pwStorage.PasswordSet(root.ID, rootPw)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunPasswordTests(t *testing.T, storage ActivityPubStorage) {
	if err := initPasswordStorage(storage); err != nil {
		t.Errorf("unable to init Password test suite: %s", err)
		return
	}
	t.Errorf("%s", errNotImplemented)
}
