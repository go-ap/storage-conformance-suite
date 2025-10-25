package conformance

import (
	"fmt"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
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

	randomObjects := getRandomItemCollection(20)
	t.Run(fmt.Sprintf("save %d random objects", len(randomObjects)), func(t *testing.T) {
		for _, ob := range randomObjects {
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
		}
	})
	t.Run(fmt.Sprintf("delete %d random objects", len(randomObjects)), func(t *testing.T) {
		for _, ob := range randomObjects {
			err := storage.Delete(ob)
			if err != nil {
				t.Errorf("unable to save object: %s", err)
			}
			loadIt, err := storage.Load(ob.GetLink())
			if err != nil && !errors.IsNotFound(err) {
				t.Errorf("unable to load object %s: %s", ob.GetLink(), err)
			}
			if loadIt != nil {
				t.Errorf("invalid object returned from loading %s: it should have been empty", ob.GetLink())
			}
		}
	})

	col := RandomCollection(root)
	colIRI := col.GetLink()
	t.Run("create collection", func(t *testing.T) {
		savedIt, err := storage.Create(col)
		if err != nil {
			t.Errorf("unable to save collection: %s", err)
		}
		if !cmp.Equal(col, savedIt) {
			t.Errorf("invalid collection returned from saving %s", cmp.Diff(col, savedIt))
		}
		loadIt, err := storage.Load(colIRI)
		if err != nil {
			t.Errorf("unable to load collection %s: %s", colIRI, err)
		}
		if !cmp.Equal(col, loadIt) {
			t.Errorf("invalid collection returned from loading %s: %s", colIRI, cmp.Diff(col, loadIt))
		}

		// NOTE(marius): generate random Item Collection
		randomObjects := getRandomItemCollection(32)
		t.Run(fmt.Sprintf("add %d items to collection", randomObjects.Count()), func(t *testing.T) {
			for i, ob := range randomObjects {
				if err = storage.AddTo(colIRI, ob); err != nil {
					t.Errorf("unable to add object to collection at pos %d: %s", i, err)
				}
			}
			loadedIt, err := storage.Load(colIRI)
			if err != nil {
				t.Errorf("unable to load collection %s: %s", colIRI, err)
			}
			err = vocab.OnCollectionIntf(loadedIt, func(col vocab.CollectionInterface) error {
				savedItems := col.Collection()
				if len(savedItems) != len(randomObjects) {
					t.Fatalf("invalid collection item counts returned from loading %d, expected %d", len(savedItems), len(randomObjects))
				}
				sortItemCollectionByID(savedItems)
				for i, it := range randomObjects {
					if !cmp.Equal(it, savedItems[i]) {
						t.Errorf("invalid item at pos %d, unable: %s", i, cmp.Diff(it, savedItems))
					}
				}
				return nil
			})
			if err != nil {
				t.Errorf("loaded object wasn't a collection %s: %s", colIRI, err)
			}
		})

		t.Run(fmt.Sprintf("remove %d items from collection", randomObjects.Count()), func(t *testing.T) {
			for i, ob := range randomObjects {
				if err = storage.RemoveFrom(colIRI, ob); err != nil {
					t.Errorf("unable to remove object from collection at pos %d: %s", i, err)
				}
			}
			loadedIt, err := storage.Load(colIRI)
			if err != nil {
				t.Errorf("unable to load collection %s: %s", colIRI, err)
			}
			err = vocab.OnCollectionIntf(loadedIt, func(col vocab.CollectionInterface) error {
				if remainingItems := col.Collection(); len(remainingItems) != 0 {
					t.Errorf("invalid collection returned from loading it has %d items: expected empty", len(remainingItems))
					t.Logf("%s", cmp.Diff(vocab.ItemCollection{}, remainingItems))
				}
				return nil
			})
			if err != nil {
				t.Errorf("loaded object wasn't a collection %s: %s", colIRI, err)
			}
		})
	})
}
