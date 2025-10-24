package conformance

import (
	"crypto"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/filters"
	"github.com/google/go-cmp/cmp"
)

type ActivityPubStorage interface {
	Load(i vocab.IRI, f ...filters.Check) (vocab.Item, error)
	Save(it vocab.Item) (vocab.Item, error)
	Delete(it vocab.Item) error

	Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error)
	AddTo(colIRI vocab.IRI, it vocab.Item) error
	RemoveFrom(colIRI vocab.IRI, it vocab.Item) error
}

var (
	rootID = vocab.IRI("https://example.com/~root")

	rootPw = []byte("notSoSecretP4ssw0rd")

	publicAudience = vocab.ItemCollection{vocab.PublicNS}

	root = &vocab.Actor{
		ID:                rootID,
		Type:              vocab.PersonType,
		Published:         time.Now(),
		Name:              vocab.DefaultNaturalLanguage("Rooty McRootface"),
		Summary:           vocab.DefaultNaturalLanguage("The base actor for the conformance test suite"),
		Content:           vocab.DefaultNaturalLanguage("<p>The base actor for the conformance test suite</p>"),
		URL:               vocab.Item(rootID),
		To:                publicAudience,
		Likes:             vocab.Likes.IRI(rootID),
		Shares:            vocab.Shares.IRI(rootID),
		Inbox:             vocab.Inbox.IRI(rootID),
		Outbox:            vocab.Outbox.IRI(rootID),
		Following:         vocab.Following.IRI(rootID),
		Followers:         vocab.Followers.IRI(rootID),
		Liked:             vocab.Liked.IRI(rootID),
		PreferredUsername: vocab.DefaultNaturalLanguage("root"),
		PublicKey:         vocab.PublicKey{},
	}

	privateKey crypto.PrivateKey = nil
)

func initActivityPub(s Suite) error {
	if s.storage == nil {
		return errNilStorage
	}
	if _, err := s.storage.Save(root); err != nil {
		return err
	}
	return nil
}

func (s Suite) RunActivityPubTests(t *testing.T) {
	if err := initActivityPub(s); err != nil {
		t.Fatalf("unable to init ActivityPub test suite: %s", err)
	}

	// Load root item
	t.Run("Load Root item", func(t *testing.T) {
		it, err := s.storage.Load(rootID)
		if err != nil {
			t.Errorf("unable to load root item: %s", err)
		}
		if !cmp.Equal(root, it) {
			t.Errorf("invalid root actor loaded from storage %s", cmp.Diff(root, it))
		}
	})

	// Save items
	t.Run("Save items", func(t *testing.T) {
		t.Fatalf("%s", errNotImplemented)
	})

	t.Fatalf("%s", errNotImplemented)
}
