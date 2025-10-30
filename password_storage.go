package conformance

import (
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-conformance-suite/internal"
)

type PasswordStorage interface {
	PasswordSet(it vocab.IRI, pw []byte) error
	PasswordCheck(it vocab.IRI, pw []byte) error
}

var rootPw = []byte("notSoSecretP4ssw0rd")

func initPasswordStorage(storage PasswordStorage) error {
	err := storage.PasswordSet(internal.RootID, rootPw)
	if err != nil {
		return err
	}
	return nil
}

func RunPasswordTests(t *testing.T, storage ActivityPubStorage) {
	pwStorage, ok := storage.(PasswordStorage)
	if !ok {
		t.Skipf("storage %T does not have Password support", storage)
	}

	if err := initPasswordStorage(pwStorage); err != nil {
		t.Errorf("unable to init Password test suite: %s", err)
		return
	}

	t.Run("check Root password", func(t *testing.T) {
		if err := pwStorage.PasswordCheck(internal.RootID, rootPw); err != nil {
			t.Errorf("unable to validate root password: %s", err)
		}
	})
}
