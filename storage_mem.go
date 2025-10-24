package conformance

import (
	"crypto"
	"fmt"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/filters"
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

type memStorage struct{}

func (ms *memStorage) Load(i vocab.IRI, f ...filters.Check) (vocab.Item, error) {
	return nil, errNotImplemented
}

func (ms *memStorage) Save(it vocab.Item) (vocab.Item, error) {
	return nil, errNotImplemented
}

func (ms *memStorage) Delete(it vocab.Item) error {
	return errNotImplemented
}

func (ms *memStorage) Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error) {
	return nil, errNotImplemented
}

func (ms *memStorage) AddTo(colIRI vocab.IRI, it vocab.Item) error {
	return errNotImplemented
}

func (ms *memStorage) RemoveFrom(colIRI vocab.IRI, it vocab.Item) error {
	return errNotImplemented
}

func (ms *memStorage) LoadKey(iri vocab.IRI) (crypto.PrivateKey, error) {
	return nil, errNotImplemented
}

func (ms *memStorage) SaveKey(iri vocab.IRI, key crypto.PrivateKey) (*vocab.PublicKey, error) {
	return nil, errNotImplemented
}

func (ms *memStorage) PasswordSet(it vocab.IRI, pw []byte) error {
	return errNotImplemented
}

func (ms *memStorage) PasswordCheck(it vocab.IRI, pw []byte) error {
	return errNotImplemented
}

func (ms *memStorage) LoadMetadata(iri vocab.IRI, m any) error {
	return errNotImplemented
}

func (ms *memStorage) SaveMetadata(iri vocab.IRI, m any) error {
	return errNotImplemented
}

var _ ActivityPubStorage = &memStorage{}
var _ MetadataStorage = &memStorage{}
var _ PasswordStorage = &memStorage{}
var _ KeyStorage = &memStorage{}
