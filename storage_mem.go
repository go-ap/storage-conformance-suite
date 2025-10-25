package conformance

import (
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/bcrypt"
)

type memStorageError string

func (e memStorageError) Error() string {
	return string(e)
}

func errf(s string, arg ...any) memStorageError {
	return memStorageError(fmt.Sprintf(s, arg...))
}

var errNotImplemented = errf("not implemented")
var errNilStorage = errf("nil storage")

type memStorage struct {
	*sync.Map
}

func asBytes(s any) []byte {
	if b, ok := s.([]byte); ok {
		return b
	}
	return nil
}

func (ms *memStorage) Load(i vocab.IRI, f ...filters.Check) (vocab.Item, error) {
	raw, ok := ms.Map.Load(i)
	if !ok {
		return nil, errors.NotFoundf("unable to find %s", i)
	}
	return vocab.UnmarshalJSON(asBytes(raw))
}

func (ms *memStorage) Save(it vocab.Item) (vocab.Item, error) {
	raw, err := vocab.MarshalJSON(it)
	if err != nil {
		return nil, err
	}
	ms.Map.Store(it.GetLink(), raw)
	return it, nil
}

func (ms *memStorage) Delete(it vocab.Item) error {
	ms.Map.Delete(it.GetLink())
	return nil
}

func (ms *memStorage) Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error) {
	it, err := ms.Save(col)
	if err != nil {
		return nil, err
	}
	col, ok := it.(vocab.CollectionInterface)
	if !ok {
		return nil, errors.Newf("invalid collection saved")
	}
	itemsKey := col.GetLink().AddPath("items")
	ms.Map.Store(itemsKey, &sync.Map{})
	return col, nil
}

func (ms *memStorage) AddTo(colIRI vocab.IRI, it vocab.Item) error {
	if _, ok := ms.Map.Load(colIRI); ok {
		return errors.NotFoundf("unable to find collection %s", colIRI)
	}
	itemsKey := colIRI.GetLink().AddPath("items")
	var items *sync.Map
	_items, ok := ms.Map.Load(itemsKey)
	if !ok {
		return errors.Newf("unable to find collection items map")
	}
	if items, ok = _items.(*sync.Map); !ok {
		return errors.Newf("invalid items map %T", _items)
	}
	items.Store(itemsKey, it)
	return nil
}

func (ms *memStorage) RemoveFrom(colIRI vocab.IRI, it vocab.Item) error {
	if _, ok := ms.Map.Load(colIRI); ok {
		return errors.NotFoundf("unable to find collection %s", colIRI)
	}
	itemsKey := colIRI.GetLink().AddPath("items")
	var items *sync.Map
	_items, ok := ms.Map.Load(itemsKey)
	if !ok {
		return errors.Newf("unable to find collection items map")
	}
	if items, ok = _items.(*sync.Map); !ok {
		return errors.Newf("invalid items map %T", _items)
	}
	items.Delete(itemsKey)
	return nil
}

func (ms *memStorage) LoadKey(iri vocab.IRI) (crypto.PrivateKey, error) {
	privateKeyKey := iri.GetLink().AddPath("privateKey")
	prvAny, ok := ms.Map.Load(privateKeyKey)
	if !ok {
		return nil, errors.Errorf("unable to find private key for iri %s", iri)
	}
	prvRaw, ok := prvAny.([]byte)
	if !ok {
		return nil, errors.Errorf("unable to load raw private key %T", prvAny)
	}

	b, _ := pem.Decode(prvRaw)
	if b == nil {
		return nil, errors.Errorf("failed decoding pem")
	}
	prvKey, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		return nil, err
	}
	return prvKey, nil
}

func (ms *memStorage) SaveKey(iri vocab.IRI, key crypto.PrivateKey) (*vocab.PublicKey, error) {
	prvEnc, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}

	pemPrvKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: prvEnc,
	})

	privateKeyKey := iri.GetLink().AddPath("privateKey")
	ms.Map.Store(privateKeyKey, pemPrvKey)

	var pub crypto.PublicKey
	switch prv := key.(type) {
	case *ecdsa.PrivateKey:
		pub = prv.Public()
	case *rsa.PrivateKey:
		pub = prv.Public()
	case *dsa.PrivateKey:
		pub = &prv.PublicKey
	case *ed25519.PrivateKey:
		pub = prv.Public()
	default:
		return nil, nil
	}
	pubEnc, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	pubEncoded := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubEnc,
	})

	return &vocab.PublicKey{
		ID:           vocab.IRI(fmt.Sprintf("%s#main", iri)),
		Owner:        iri,
		PublicKeyPem: string(pubEncoded),
	}, nil
}

func (ms *memStorage) PasswordSet(iri vocab.IRI, pw []byte) error {
	privateKeyKey := iri.GetLink().AddPath("password")
	ms.Map.Store(privateKeyKey, pw)
	return nil
}

func (ms *memStorage) PasswordCheck(iri vocab.IRI, pw []byte) error {
	pwKey := iri.GetLink().AddPath("password")
	pwAny, ok := ms.Map.Load(pwKey)
	if !ok {
		return errors.Errorf("unable to find password for iri %s", iri)
	}
	pwRaw, ok := pwAny.([]byte)
	if !ok {
		return errors.Errorf("unable to load raw password %T", pwAny)
	}
	if err := bcrypt.CompareHashAndPassword(pwRaw, pw); err != nil {
		return errors.NewUnauthorized(err, "Invalid pw")
	}
	return nil
}

func (ms *memStorage) LoadMetadata(iri vocab.IRI, m any) error {
	metaKey := iri.GetLink().AddPath("meta")
	metaAny, ok := ms.Map.Load(metaKey)
	if !ok {
		return errors.Errorf("unable to find metadata for iri %s", iri)
	}
	m = &metaAny
	return nil
}

func (ms *memStorage) SaveMetadata(iri vocab.IRI, m any) error {
	metaKey := iri.GetLink().AddPath("meta")
	ms.Map.Store(metaKey, m)
	return nil
}

var _ ActivityPubStorage = &memStorage{}
var _ MetadataStorage = &memStorage{}
var _ PasswordStorage = &memStorage{}
var _ KeyStorage = &memStorage{}
