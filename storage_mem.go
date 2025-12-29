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
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"github.com/openshift/osin"
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

	if len(f) > 0 {
		switch ob.GetType() {
		case vocab.OrderedCollectionType, vocab.OrderedCollectionPageType:
			clone, _ := ob.(*vocab.OrderedCollection)
			obCopy := *clone
			ob = filters.Checks(f).Run(&obCopy)
		case vocab.CollectionType, vocab.CollectionPageType:
			clone, _ := ob.(*vocab.Collection)
			obCopy := *clone
			ob = filters.Checks(f).Run(&obCopy)
		}
	}

	return ob, nil
}

func saveCollectionIfExists(r *memStorage, it, owner vocab.Item) vocab.Item {
	if vocab.IsNil(it) {
		return nil
	}
	r.Map.LoadOrStore(it.GetLink(), createNewCollection(it.GetLink(), owner))
	return it.GetLink()
}

func createNewCollection(colIRI vocab.IRI, owner vocab.Item) vocab.CollectionInterface {
	col := vocab.OrderedCollection{
		ID:        colIRI,
		Type:      vocab.OrderedCollectionType,
		CC:        vocab.ItemCollection{vocab.PublicNS},
		Published: time.Now().Truncate(time.Second).UTC(),
	}
	if !vocab.IsNil(owner) {
		col.AttributedTo = owner.GetLink()
	}
	return &col
}

// createItemCollections
func createItemCollections(ms *memStorage, it vocab.Item) error {
	if vocab.IsNil(it) || !it.IsObject() {
		return nil
	}
	if vocab.ActorTypes.Contains(it.GetType()) {
		_ = vocab.OnActor(it, func(p *vocab.Actor) error {
			p.Inbox = saveCollectionIfExists(ms, p.Inbox, p)
			p.Outbox = saveCollectionIfExists(ms, p.Outbox, p)
			p.Followers = saveCollectionIfExists(ms, p.Followers, p)
			p.Following = saveCollectionIfExists(ms, p.Following, p)
			p.Liked = saveCollectionIfExists(ms, p.Liked, p)
			// NOTE(marius): shadow creating hidden collections for Blocked and Ignored items
			saveCollectionIfExists(ms, filters.BlockedType.Of(p), p)
			saveCollectionIfExists(ms, filters.IgnoredType.Of(p), p)
			return nil
		})
	}
	return vocab.OnObject(it, func(o *vocab.Object) error {
		o.Replies = saveCollectionIfExists(ms, o.Replies, o)
		o.Likes = saveCollectionIfExists(ms, o.Likes, o)
		o.Shares = saveCollectionIfExists(ms, o.Shares, o)
		return nil
	})
}

func (ms *memStorage) Save(it vocab.Item) (vocab.Item, error) {
	if _, ok := ms.Map.Load(it.GetLink()); !ok {
		if err := createItemCollections(ms, it); err != nil {
			return it, errors.Annotatef(err, "could not create object's collections")
		}
	}
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

func (ms *memStorage) AddTo(colIRI vocab.IRI, items ...vocab.Item) error {
	col, err := ms.loadCol(colIRI)
	if err != nil {
		return err
	}

	if err = col.Append(items...); err != nil {
		return err
	}

	_, err = ms.Save(col)
	return err
}

func (ms *memStorage) RemoveFrom(colIRI vocab.IRI, items ...vocab.Item) error {
	col, err := ms.loadCol(colIRI)
	if err != nil {
		return err
	}

	col.Remove(items...)

	_, err = ms.Save(col)
	return err
}

func clientPath(clientID string) string {
	return filepath.Join("oauth", "clients", clientID)
}

func (ms *memStorage) CreateClient(c osin.Client) error {
	ms.Map.Store(clientPath(c.GetId()), c)
	return nil
}

func (ms *memStorage) UpdateClient(c osin.Client) error {
	ms.Map.Store(clientPath(c.GetId()), c)
	return nil
}

func (ms *memStorage) RemoveClient(id string) error {
	ms.Map.Delete(clientPath(id))
	return nil
}

func (ms *memStorage) ListClients() ([]osin.Client, error) {
	clients := make([]osin.Client, 0)
	ms.Map.Range(func(key, value any) bool {
		path, ok := key.(string)
		if ok {
			if strings.HasPrefix(path, "oauth/clients") {
				if cl, ok := value.(osin.Client); ok {
					clients = append(clients, cl)
				}
			}
		}
		return true
	})
	return clients, nil
}

func (ms *memStorage) Clone() osin.Storage {
	return ms
}

func (ms *memStorage) Close() {
}

func (ms *memStorage) GetClient(id string) (osin.Client, error) {
	val, ok := ms.Map.Load(clientPath(id))
	if !ok {
		return nil, errors.NotFoundf("client not found %s", id)
	}
	cl, ok := val.(osin.Client)
	if !ok {
		return nil, errors.Errorf("invalid type for client %T", val)
	}
	return cl, nil
}

func authorizePath(code string) string {
	return filepath.Join("oauth", "authorize", code)
}

func (ms *memStorage) SaveAuthorize(data *osin.AuthorizeData) error {
	ms.Map.Store(authorizePath(data.Code), data)
	return nil
}

func (ms *memStorage) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	val, ok := ms.Map.Load(authorizePath(code))
	if !ok {
		return nil, errors.NotFoundf("authorization data not found %s", code)
	}
	auth, ok := val.(*osin.AuthorizeData)
	if !ok {
		return nil, errors.Errorf("invalid type for authorization data %T", val)
	}
	return auth, nil
}

func (ms *memStorage) RemoveAuthorize(code string) error {
	ms.Map.Delete(authorizePath(code))
	return nil
}

func accessPath(token string) string {
	return filepath.Join("oauth", "access", token)
}

func (ms *memStorage) SaveAccess(data *osin.AccessData) error {
	ms.Map.Store(accessPath(data.AccessToken), data)
	if data.RefreshToken != "" {
		ms.Map.Store(refreshPath(data.RefreshToken), data.AccessToken)
	}
	return nil
}

func (ms *memStorage) LoadAccess(token string) (*osin.AccessData, error) {
	val, ok := ms.Map.Load(accessPath(token))
	if !ok {
		return nil, errors.NotFoundf("access data not found %s", token)
	}
	access, ok := val.(*osin.AccessData)
	if !ok {
		return nil, errors.Errorf("invalid type for access data %T", val)
	}
	return access, nil
}

func (ms *memStorage) RemoveAccess(token string) error {
	exists, _ := ms.LoadAccess(token)
	ms.Map.Delete(accessPath(token))
	if exists != nil && exists.RefreshToken != "" {
		ms.Map.Delete(refreshPath(exists.RefreshToken))
	}
	return nil
}

func refreshPath(token string) string {
	return filepath.Join("oauth", "refresh", token)
}

func (ms *memStorage) LoadRefresh(token string) (*osin.AccessData, error) {
	val, ok := ms.Map.Load(refreshPath(token))
	if !ok {
		return nil, errors.NotFoundf("refresh data not found %s", token)
	}
	token, ok = val.(string)
	if !ok {
		return nil, errors.Errorf("invalid type for refresh data %T", val)
	}
	return ms.LoadAccess(token)
}

func (ms *memStorage) RemoveRefresh(token string) error {
	ms.Map.Delete(refreshPath(token))
	return nil
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
	case ed25519.PrivateKey:
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
	privateKeyKey := iri.GetLink().AddPath("__password")
	hashed, err := bcrypt.GenerateFromPassword(pw, bcrypt.MinCost)
	if err != nil {
		return err
	}
	ms.Map.Store(privateKeyKey, hashed)
	return nil
}

func (ms *memStorage) PasswordCheck(iri vocab.IRI, pw []byte) error {
	pwKey := iri.GetLink().AddPath("__password")
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
	metaKey := iri.GetLink().AddPath("__meta")
	metaAny, ok := ms.Map.Load(metaKey)
	if !ok {
		return errors.Errorf("unable to find metadata for iri %s", iri)
	}
	copy(metaAny, m)
	return nil
}

func (ms *memStorage) SaveMetadata(iri vocab.IRI, m any) error {
	metaKey := iri.GetLink().AddPath("__meta")
	ms.Map.Store(metaKey, m)
	return nil
}

var _ ActivityPubStorage = &memStorage{}
var _ MetadataStorage = &memStorage{}
var _ PasswordStorage = &memStorage{}
var _ KeyStorage = &memStorage{}
var _ OSINStorage = &memStorage{}
var _ ClientLister = &memStorage{}
var _ ClientSaver = &memStorage{}

// copy copies from one instance of type T to another
func copy[T any](from, to T) {
	r := reflect.ValueOf(to).Elem()
	r.Set(reflect.ValueOf(from))
}
