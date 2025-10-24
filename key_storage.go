package conformance

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

type KeyStorage interface {
	LoadKey(iri vocab.IRI) (crypto.PrivateKey, error)
	SaveKey(iri vocab.IRI, key crypto.PrivateKey) (*vocab.PublicKey, error)
}

var privateKey, _ = rsa.GenerateKey(rand.Reader, 2048)

func initKeyStorage(storage ActivityPubStorage) error {
	keyStorage, ok := storage.(KeyStorage)
	if ok {
		pk, err := keyStorage.SaveKey(root.ID, privateKey)
		if err != nil {
			return err
		}
		root.PublicKey = *pk
	}
	return nil
}

func RunKeyTests(t *testing.T, storage ActivityPubStorage) {
	if err := initKeyStorage(storage); err != nil {
		t.Fatalf("unable to init Key pair test suite: %s", err)
	}
	t.Skipf("%s", errNotImplemented)
}
