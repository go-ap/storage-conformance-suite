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

var (
	orderedCollectionTypes = vocab.ActivityVocabularyTypes{
		vocab.OrderedCollectionType,
		vocab.OrderedCollectionPageType,
	}
	collectionTypes = vocab.ActivityVocabularyTypes{
		vocab.CollectionType,
		vocab.CollectionPageType,
	}

	allCollectionTypes = append(collectionTypes, orderedCollectionTypes...)
)

func (ms *memStorage) Load(i vocab.IRI, f ...filters.Check) (vocab.Item, error) {
	raw, ok := ms.Map.Load(i)
	if !ok {
		return nil, errors.NotFoundf("unable to find %s", i)
	}
	ob, ok := raw.(vocab.Item)
	if !ok {
		return nil, errors.Newf("invalid item type in storage %T", raw)
	}

	if allCollectionTypes.Contains(ob.GetType()) {
		itemsKey := i.AddPath("items")
		var itemMap *sync.Map
		_items, ok := ms.Map.Load(itemsKey)
		if !ok {
			return ob, errors.Newf("unable to find collection items map")
		}
		if itemMap, ok = _items.(*sync.Map); !ok {
			return ob, errors.Newf("invalid items map %T", _items)
		}

		items := make(vocab.ItemCollection, 0)
		itemMap.Range(func(_, raw any) bool {
			if it, ok := raw.(vocab.Item); ok {
				_ = items.Append(it)
			}
			return true
		})
		if len(items) > 0 {
			var err error
			if orderedCollectionTypes.Contains(ob.GetType()) {
				err = vocab.OnOrderedCollection(ob, func(col *vocab.OrderedCollection) error {
					col.OrderedItems = items
					return nil
				})
			}
			if collectionTypes.Contains(ob.GetType()) {
				err = vocab.OnCollection(ob, func(col *vocab.Collection) error {
					col.Items = items
					return nil
				})
			}
			if err != nil {
				return ob, err
			}
		}
	}
	return ob, nil
}

func (ms *memStorage) Save(it vocab.Item) (vocab.Item, error) {
	ms.Map.Store(it.GetLink(), it)
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

func (ms *memStorage) loadCol(colIRI vocab.IRI) (vocab.CollectionInterface, error) {
	it, ok := ms.Map.Load(colIRI)
	if !ok {
		return nil, errors.Newf("unable to load collection %s", colIRI)
	}
	col, ok := it.(vocab.CollectionInterface)
	if !ok {
		return nil, errors.Newf("invalid collection type %T %s", it, colIRI)
	}
	return col, nil
}

func (ms *memStorage) AddTo(colIRI vocab.IRI, it vocab.Item) error {
	col, err := ms.loadCol(colIRI)
	if err != nil {
		return err
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
	items.Store(it.GetLink(), it)

	// NOTE(marius): increase total items count
	err = vocab.OnCollection(col, func(col *vocab.Collection) error {
		col.TotalItems += 1
		return nil
	})
	if err != nil {
		return err
	}
	_, err = ms.Save(col)
	return err
}

func (ms *memStorage) RemoveFrom(colIRI vocab.IRI, it vocab.Item) error {
	col, err := ms.loadCol(colIRI)
	if err != nil {
		return err
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
	items.Delete(it.GetLink())

	// NOTE(marius): decrease total items count
	err = vocab.OnCollection(col, func(col *vocab.Collection) error {
		col.Items.Remove(it)
		if col.TotalItems > 0 {
			col.TotalItems -= 1
		}
		return nil
	})
	if err != nil {
		return err
	}
	_, err = ms.Save(col)
	return err
}

func (ms *memStorage) LoadKey(iri vocab.IRI) (crypto.PrivateKey, error) {
	privateKeyKey := iri.GetLink().AddPath("privateKey")
	prvKey, ok := ms.Map.Load(privateKeyKey)
	if !ok {
		return nil, errors.Errorf("unable to find private key for iri %s", iri)
	}
	return prvKey, nil
}

func (ms *memStorage) SaveKey(iri vocab.IRI, key crypto.PrivateKey) (*vocab.PublicKey, error) {
	privateKeyKey := iri.GetLink().AddPath("privateKey")
	ms.Map.Store(privateKeyKey, key)

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
	if err := bcrypt.CompareHashAndPassword(asBytes(pwAny), pw); err != nil {
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
