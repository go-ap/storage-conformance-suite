package conformance

import (
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
	defaultTime = time.Date(1999, time.April, 1, 6, 6, 6, 0, time.UTC)

	rootID = vocab.IRI("https://example.com/~root")

	publicAudience = vocab.ItemCollection{vocab.PublicNS}

	root = &vocab.Actor{
		ID:                rootID,
		Type:              vocab.PersonType,
		Published:         defaultTime,
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
	}
)

func initActivityPub(storage ActivityPubStorage) error {
	if storage == nil {
		return errNilStorage
	}
	if _, err := storage.Save(root); err != nil {
		return err
	}
	return nil
}

func RunActivityPubTests(t *testing.T, storage ActivityPubStorage) {
	if err := initActivityPub(storage); err != nil {
		t.Fatalf("unable to init ActivityPub test suite: %s", err)
	}

	// Load root item
	t.Run("Load Root item", func(t *testing.T) {
		it, err := storage.Load(rootID)
		if err != nil {
			t.Errorf("unable to load root item: %s", err)
		}
		if !cmp.Equal(root, it) {
			t.Errorf("invalid root actor loaded from storage %s", cmp.Diff(root, it))
		}
	})

	// Save items
	t.Run("Save items", func(t *testing.T) {
		t.Run("save random object", func(t *testing.T) {
			ob := RandomObject(root)
			savedIt, err := storage.Save(ob)
			if err != nil {
				t.Errorf("unable to save object: %s", err)
			}
			if !cmp.Equal(ob, savedIt) {
				t.Errorf("invalid object returned from saving %s", cmp.Diff(ob, savedIt))
			}
			loadIt, err := storage.Load(savedIt.GetLink())
			if err != nil {
				t.Errorf("unable to load object %s: %s", ob.GetLink(), err)
			}
			if !cmp.Equal(ob, loadIt) {
				t.Errorf("invalid object returned from loading %s: %s", ob.GetLink(), cmp.Diff(ob, loadIt))
			}
		})
		t.Run("create collection", func(t *testing.T) {
			col := RandomCollection(root)
			savedIt, err := storage.Create(col)
			if err != nil {
				t.Errorf("unable to save collection: %s", err)
			}
			if !cmp.Equal(col, savedIt) {
				t.Errorf("invalid collection returned from saving %s", cmp.Diff(col, savedIt))
			}
			loadIt, err := storage.Load(savedIt.GetLink())
			if err != nil {
				t.Errorf("unable to load collection %s: %s", col.GetLink(), err)
			}
			if !cmp.Equal(col, loadIt) {
				t.Errorf("invalid collection returned from loading %s: %s", col.GetLink(), cmp.Diff(col, loadIt))
			}
			t.Run("add items collection", func(t *testing.T) {
				ob := RandomObject(root)
				err := storage.AddTo(col.GetLink(), ob)
				if err != nil {
					t.Errorf("unable to add object to collection: %s", err)
				}
				loadedIt, err := storage.Load(col.GetLink())
				if err != nil {
					t.Errorf("unable to load collection %s: %s", col.GetLink(), err)
				}
				err = vocab.OnCollectionIntf(loadedIt, func(col vocab.CollectionInterface) error {
					for pos, it := range col.Collection() {
						if !cmp.Equal(ob, it) {
							t.Errorf("invalid collection item returned from loading at pos %d %s: %s", pos, col.GetLink(), cmp.Diff(ob, it))
						}
					}
					return nil
				})
				if err != nil {
					t.Errorf("loaded object wasn't a collection %s: %s", col.GetLink(), err)
				}
			})
		})
	})
}
