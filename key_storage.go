package conformance

import (
	"bytes"
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-conformance-suite/internal"
	"github.com/google/go-cmp/cmp"
)

type KeyStorage interface {
	LoadKey(iri vocab.IRI) (crypto.PrivateKey, error)
	SaveKey(iri vocab.IRI, key crypto.PrivateKey) (*vocab.PublicKey, error)
}

var privateKey, _ = rsa.GenerateKey(rand.Reader, 2048)

func initKeyStorage(storage KeyStorage) error {
	pk, err := storage.SaveKey(internal.RootID, privateKey)
	if err != nil {
		return err
	}
	internal.Root.PublicKey = *pk
	apStorage, ok := storage.(ActivityPubStorage)
	if ok {
		_, _ = apStorage.Save(internal.Root)
	}
	return nil
}

func RunKeyTests(t *testing.T, storage ActivityPubStorage) {
	keyStorage, ok := storage.(KeyStorage)
	if !ok {
		t.Fatalf("storage is not compatible with KeyStorage %T", storage)
	}
	if err := initKeyStorage(keyStorage); err != nil {
		t.Fatalf("unable to init Key pair test suite: %s", err)
	}

	t.Run("load Root key", func(t *testing.T) {
		prv, err := keyStorage.LoadKey(internal.RootID)
		if err != nil {
			t.Fatalf("unable to load private key %s", err)
		}
		if !cmp.Equal(privateKey, prv) {
			t.Errorf("Loaded private key is different %s", cmp.Diff(privateKey, prv))
		}
		actor, err := storage.Load(internal.RootID)
		if err != nil {
			t.Fatalf("unable to load actor item %s", err)
		}
		err = vocab.OnActor(actor, func(actor *vocab.Actor) error {
			if !cmp.Equal(actor.PublicKey, internal.Root.PublicKey) {
				t.Errorf("invalid root actor public key loaded from storage %s", cmp.Diff(internal.Root.PublicKey, actor.PublicKey))
			}

			pub, ok := publicKey(prv)
			if !ok {
				t.Errorf("unable to extract public key %T from the private one %T", pub, prv)
			}

			pubEnc, err := x509.MarshalPKIXPublicKey(pub)
			if err != nil {
				t.Errorf("unable to marshal public key from the private one: %s", err)
			}
			pubPem := pem.EncodeToMemory(&pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: pubEnc,
			})
			if !bytes.Equal(pubPem, []byte(actor.PublicKey.PublicKeyPem)) {
				t.Errorf("invalid actor public PEM key: %s", cmp.Diff(string(pubPem), actor.PublicKey.PublicKeyPem))
			}

			return nil
		})
		if err != nil {
			t.Fatalf("the item loaded for %s couldn't be converted to Actor: %s", internal.RootID, err)
		}
	})

	keys := genPrivateKeys()
	for _, key := range keys {
		t.Run(fmt.Sprintf("save %T key", key), func(t *testing.T) {
			it := internal.RandomActor(internal.RootID)
			actor, err := vocab.ToActor(it)
			if err != nil {
				t.Fatalf("unable to generate random actor: %s", err)
			}
			pub, err := keyStorage.SaveKey(it.GetLink(), key)
			if err != nil {
				t.Fatalf("unable to save random key %T: %s", key, err)
			}
			actor.PublicKey = *pub
			_, _ = storage.Save(actor)
			t.Run(fmt.Sprintf("load %T key", key), func(t *testing.T) {
				actorIRI := actor.GetLink()
				prv, err := keyStorage.LoadKey(actorIRI)
				if err != nil {
					t.Fatalf("unable to load private key %s: %s", actorIRI, err)
				}
				if !cmp.Equal(key, prv) {
					t.Errorf("Loaded private key is different %s", cmp.Diff(privateKey, prv))
				}
				actor, err := storage.Load(actorIRI)
				if err != nil {
					t.Fatalf("unable to load actor item %s", err)
				}
				err = vocab.OnActor(actor, func(actor *vocab.Actor) error {
					pub, ok := publicKey(prv)
					if !ok {
						t.Errorf("unable to extract public key %T from the private one %T", pub, prv)
					}

					pubEnc, err := x509.MarshalPKIXPublicKey(pub)
					if err != nil {
						t.Errorf("unable to marshal public key from the private one: %s", err)
					}
					pubPem := pem.EncodeToMemory(&pem.Block{
						Type:  "PUBLIC KEY",
						Bytes: pubEnc,
					})
					if !bytes.Equal(pubPem, []byte(actor.PublicKey.PublicKeyPem)) {
						t.Errorf("invalid actor public PEM key: %s", cmp.Diff(string(pubPem), actor.PublicKey.PublicKeyPem))
					}

					return nil
				})
				if err != nil {
					t.Fatalf("the item loaded for %s couldn't be converted to Actor: %s", internal.RootID, err)
				}
			})
		})
	}
}

func genPrivateKeys() []crypto.PrivateKey {
	keys := make([]crypto.PrivateKey, 0, 7)

	e224, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err == nil {
		keys = append(keys, e224)
	}
	e256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err == nil {
		keys = append(keys, e256)
	}
	e384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err == nil {
		keys = append(keys, e384)
	}
	e521, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err == nil {
		keys = append(keys, e521)
	}
	_, ed, err := ed25519.GenerateKey(rand.Reader)
	if err == nil {
		keys = append(keys, ed)
	}
	r2048, err := rsa.GenerateKey(rand.Reader, 2048)
	if err == nil {
		keys = append(keys, r2048)
	}
	r4096, err := rsa.GenerateKey(rand.Reader, 4096)
	if err == nil {
		keys = append(keys, r4096)
	}
	return keys
}

func publicKey(key crypto.PrivateKey) (crypto.PublicKey, bool) {
	var pub crypto.PublicKey
	valid := true
	switch prv := key.(type) {
	case *ecdsa.PrivateKey:
		pub = prv.Public()
	case *rsa.PrivateKey:
		pub = prv.Public()
	case *dsa.PrivateKey:
		pub = &prv.PublicKey
	case ed25519.PrivateKey:
		pub = prv.Public()
	default:
		valid = false
	}
	return pub, valid
}
