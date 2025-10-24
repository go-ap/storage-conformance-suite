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

func initKeyStorage(s Suite) error {
	keyStorage, ok := s.storage.(KeyStorage)
	if ok {
		pk, err := keyStorage.SaveKey(root.ID, privateKey)
		if err != nil {
			return err
		}
		root.PublicKey = *pk
	}
	return nil
}

func (s Suite) RunKeyTests(t *testing.T) {
	if err := initKeyStorage(s); err != nil {
		t.Fatalf("unable to init Key pair test suite: %s", err)
	}
	t.Skipf("%s", errNotImplemented)
}
