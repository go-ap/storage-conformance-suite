package conformance

import (
	"fmt"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"github.com/go-ap/storage-conformance-suite/internal"
	"github.com/google/go-cmp/cmp"
)

type ActivityPubStorage interface {
	// Save saves te [vocab.Item] to storage.
	// To conform to what the GoActivityPub library expects out of it, there are a couple of hidden behaviours:
	// * When saving a [vocab.Object] compatible type the backend *MUST* create all the object's collections
	// that have IRIs set. These collections are in [vocab.OfObject].
	// Eg: For
	//
	//	vocab.Object{
	//		Likes: "https://example.com/objects/1/likes",
	//		Replies:"https://example.com/objects/1/replies",
	//		Shares: nil
	//	}
	//
	// We create the collections https://example.com/objects/1/likes, and https://example.com/objects/1/replies.
	// * When saving a [vocab.Actor] compatible type the backend *MUST* create all the actor's collections that
	// have IRIs set. These collections are in [vocab.OfActor].
	// Eg: For
	//
	//	vocab.Actor{
	//		Inbox:"https://example.com/~jdoe/inbox",
	//		Outbox: "https://example.com/~jdoe/outbox",
	//		Followers: "https://example.com/~jdoe/followers",
	//		Following: nil
	//	}
	//
	// We create the collections https://example.com/~jdoe/inbox, https://example.com/~jdoe/outbox and
	// "https://example.com/~jdoe/followers".
	Save(it vocab.Item) (vocab.Item, error)
	// Load loads the item found at the "iri" [vocab.IRI].
	// If iri points to a collection, the filters "f" get applied to the items of the collection.
	// An implicit assumption made by filters is that when the list contains checks for one level deep properties,
	// the storage backend loads those properties and replaces them into the original.
	// For example if filtering in an activities collection with a check that the Actor should have a preferred username
	// of "janeDoe", the actors of the activities need to be loaded and the check applied on them.
	// So when the object loaded is flattened to something like this:
	//
	//	vocab.Activity {
	//		Actor: "https://example.com/~jdoe" ...
	//	}
	//
	// the actor gets dereferenced to:
	//
	//	vocab.Activity{
	//		Actor: vocab.Actor{
	//			ID: "https://example.com/~jdoe",
	//			preferredUsername: "janeDoe" ...
	//		}
	//	}
	//
	// which then can be filtered with the "preferredUsername" check.
	// The filters can also contain pagination checks, and when those get applied (see the github.com/go-ap/filtering package)
	// the result can be different from the actual persisted value.
	// The [filters.Checks.Paginate] method should handle most cases, so it should be enough to call it just before
	// returning, similarly to how the local "memStorage" type does.
	Load(iri vocab.IRI, ff ...filters.Check) (vocab.Item, error)
	Delete(it vocab.Item) error

	// Create
	// NOTE(marius): should we remove this in favour of custom logic for Save()?
	Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error)

	// AddTo adds items to the collection.
	// Similarly to Save(), this method needs to implement some hidden behaviour in order to conform to the expectations
	// of the rest of GoActivityPub library when executing logic for the blocking and ignoring ActivityPub operations.
	// In summary, when adding items to the "blocked" or "ignored" collections of a [vocab.Actor]
	// they need to be created if missing.
	// These two collections are called "hidden" because they do not appear as properties on the Actor,
	// so the only way to build their IDs is to append the paths "/blocked" and "/ignored" the Actor's ID.
	AddTo(colIRI vocab.IRI, items ...vocab.Item) error
	RemoveFrom(colIRI vocab.IRI, items ...vocab.Item) error
}

func initActivityPub(storage ActivityPubStorage) error {
	if storage == nil {
		return errNilStorage
	}
	if _, err := storage.Save(internal.Root); err != nil {
		return err
	}
	return nil
}

func buildPaginationFilters() []filters.Checks {
	return []filters.Checks{
		{filters.WithMaxCount(10)},
	}
}

func buildTypeFilters() []filters.Checks {
	checks := make([]filters.Checks, 0)
	for _, typ := range vocab.Types {
		checks = append(checks, filters.Checks{filters.HasType(typ)})
	}
	return checks
}

func buildActivityAndObjectTypeFilters() []filters.Checks {
	checks := make([]filters.Checks, 0)
	objectTypesChecks := make(filters.Checks, 0)
	for _, typ := range vocab.ObjectTypes {
		objectTypesChecks = append(objectTypesChecks, filters.HasType(typ))
	}
	for _, typ := range vocab.ActivityTypes {
		for _, objectTypeCheck := range objectTypesChecks {
			activityTypeObjectTypeCheck := filters.All(filters.HasType(typ), filters.Object(objectTypeCheck))
			checks = append(checks, filters.Checks{activityTypeObjectTypeCheck})
		}
	}
	return checks
}

var (
	byTypeFilters               = buildTypeFilters()
	byActivityObjectTypeFilters = buildActivityAndObjectTypeFilters()
	withPagination              = buildPaginationFilters()
)

/*
 * TODO
 *  - Make sure the tests are independent, currently collection save/query/delete are dependent on having objects saved
 *    This would make the tests have a two phase structure:
 *        1. mock the expected storage layout,
 *        2. test expectations
 *    Separate tests into different test functions.
 *  - Add collection creation tests for multiple versions:
 *      1. Expected collection names: inbox, outbox, followers, following, shares, liked, likes, (blocked, ignored).
 *      2. Random IRI paths without any structure to them.
 *  - Build a proper collection filter querying matrix
 *   * Paginate a collection: maxItems, after(, before?).
 *   * Combine Any/All filters.
 *   * Add content filters.
 */

func RunActivityPubTests(t *testing.T, storage ActivityPubStorage) {
	if err := initActivityPub(storage); err != nil {
		t.Fatalf("unable to init ActivityPub test suite: %s", err)
	}

	// Load root item
	t.Run("Load Root item", func(t *testing.T) {
		it, err := storage.Load(internal.RootID)
		if err != nil {
			t.Errorf("unable to load root item: %s", err)
		}
		if !cmp.Equal(internal.Root, it) {
			t.Errorf("invalid root actor loaded from storage %s", cmp.Diff(internal.Root, it))
		}
	})

	randomObjects := internal.RandomItemCollection(48)
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

			// NOTE(marius): check Object and Actor collections being created:
			// @see https://todo.sr.ht/~mariusor/go-activitypub/402
			collectionIRISToCheck := make(vocab.IRIs, 0)
			if vocab.ActorTypes.Contains(ob.GetType()) {
				for _, colPath := range vocab.OfActor {
					if maybeCollection := colPath.Of(ob); !vocab.IsNil(maybeCollection) {
						_ = collectionIRISToCheck.Append(maybeCollection.GetLink())
					}
				}
				// //TODO(marius): this should be checked for AddTo() collections
				//hiddenPaths := vocab.CollectionPaths{"blocked", "ignored"}
				//for _, hiddenPath := range hiddenPaths {
				//	_ = collectionIRISToCheck.Append(hiddenPath.IRI(ob))
				//}
			} else if !vocab.LinkTypes.Contains(ob.GetType()) {
				for _, colPath := range vocab.OfObject {
					if maybeCollection := colPath.Of(ob); !vocab.IsNil(maybeCollection) {
						_ = collectionIRISToCheck.Append(maybeCollection.GetLink())
					}
				}
			}

			for _, itemCollection := range collectionIRISToCheck {
				t.Run(itemCollection.String(), func(t *testing.T) {
					_, which := vocab.Split(itemCollection)
					t.Skipf("Checking %s skipped: we stopped creating them automatically in the storage backend", itemCollection)
					if needsCheck := which.Of(ob); vocab.IsNil(needsCheck) {
						return
					}
					loadedCol, err := storage.Load(itemCollection)
					if err != nil {
						t.Errorf("unable to load %s collection %s: %s", ob.GetType(), itemCollection, err)
					}
					err = vocab.OnCollectionIntf(loadedCol, func(col vocab.CollectionInterface) error {
						if !col.GetLink().Equal(itemCollection) {
							t.Errorf("invalid %s collection returned from loading %s: %s", ob.GetType(), itemCollection, loadedCol)
						}
						if len(col.Collection()) != 0 {
							t.Errorf("freshly created collection should have zero items, found %d", len(col.Collection()))
						}
						if col.Count() != 0 {
							t.Errorf("freshly created collection should have zero total items, found %d", col.Count())
						}
						return nil
					})
					if err != nil {
						t.Errorf("invalid %T collection type, expected %v", loadedCol, allCollectionTypes)
					}
				})
			}
		}
	})

	col := internal.RandomCollection(internal.Root)
	_ = vocab.OnObject(col, func(ob *vocab.Object) error {
		// NOTE(marius): this is a corner case for the storage-fs backend which doesn't work well with collections
		// that don't have IRIs ending in the traditional collection names (inbox, outbox, followers, etc)
		ob.ID = vocab.Inbox.IRI(ob.AttributedTo.GetLink())
		ob.AttributedTo = ob.AttributedTo.GetLink()
		return nil
	})
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

		t.Run(fmt.Sprintf("add %d items to collection", randomObjects.Count()), func(t *testing.T) {
			if err = storage.AddTo(colIRI, randomObjects...); err != nil {
				t.Errorf("unable to add objects to collection: %s", err)
			}
			loadedIt, err := storage.Load(colIRI)
			if err != nil {
				t.Errorf("unable to load collection %s: %s", colIRI, err)
			}
			err = vocab.OnCollectionIntf(loadedIt, func(col vocab.CollectionInterface) error {
				if col.Count() != uint(len(randomObjects)) {
					t.Fatalf("invalid collection item counts returned from loading %d, expected %d", col.Count(), len(randomObjects))
				}
				savedItems := col.Collection()
				if len(savedItems) != len(randomObjects) {
					t.Fatalf("invalid collection item counts returned from loading %d, expected %d", len(savedItems), len(randomObjects))
				}
				internal.SortItemCollectionByID(randomObjects)
				internal.SortItemCollectionByID(savedItems)
				for i, it := range randomObjects {
					if !cmp.Equal(it, savedItems[i]) {
						t.Errorf("invalid item at pos %d, unable: %s", i, cmp.Diff(it, savedItems[i]))
					}
				}
				return nil
			})
			if err != nil {
				t.Errorf("loaded object wasn't a collection %s: %s", colIRI, err)
			}
		})
		queryFilters := append(withPagination, append(byTypeFilters, byActivityObjectTypeFilters...)...)
		for _, fil := range queryFilters {
			t.Run(fmt.Sprintf("query collection with filters %s", fil), func(t *testing.T) {
				loadIt, err := storage.Load(colIRI, fil...)
				if err != nil {
					t.Errorf("unable to load collection %s: %s", colIRI, err)
				}
				var foundItems vocab.ItemCollection
				var totalItems uint
				err = vocab.OnOrderedCollection(loadIt, func(col *vocab.OrderedCollection) error {
					foundItems = col.OrderedItems
					totalItems = col.TotalItems
					return nil
				})
				if err != nil {
					t.Errorf("loaded object wasn't a collection %s: %s", colIRI, err)
				}
				filteredRandomObjects := fil.Run(randomObjects)
				filteredItems, ok := filteredRandomObjects.(vocab.ItemCollection)
				if !ok {
					t.Fatalf("filtered items are not compatible with an Item Collection %T", filteredRandomObjects)
				}
				if totalItems != uint(len(randomObjects)) {
					t.Fatalf("invalid collection total items count returned from loading %d, expected %d", totalItems, len(randomObjects))
				}
				if len(filteredItems) != len(foundItems) {
					t.Fatalf("invalid collection item counts returned from loading %d, expected %d", len(foundItems), len(filteredItems))
				}
				if !cmp.Equal(foundItems, filteredItems) {
					t.Errorf("invalid items returned from loading: %s", cmp.Diff(foundItems, filteredItems))
				}
			})
		}
		for idx := range len(randomObjects) + 1 {
			cnt := idx + 1
			checks := filters.Checks{filters.WithMaxCount(cnt)}
			t.Run(fmt.Sprintf("traverse collection with pagination %d", cnt), func(t *testing.T) {
				for range len(randomObjects) / cnt {
					t.Run(fmt.Sprintf("query collection with filters %s", checks), func(t *testing.T) {
						loadIt, err := storage.Load(colIRI, checks...)
						if err != nil {
							t.Errorf("unable to load collection %s: %s", colIRI, err)
						}
						var foundItems vocab.ItemCollection
						var totalItems uint
						err = vocab.OnOrderedCollection(loadIt, func(col *vocab.OrderedCollection) error {
							foundItems = col.OrderedItems
							totalItems = col.TotalItems
							return nil
						})
						if err != nil {
							t.Errorf("loaded object wasn't a collection %s: %s", colIRI, err)
						}
						filteredRandomObjects := checks.Run(randomObjects)
						filteredItems, ok := filteredRandomObjects.(vocab.ItemCollection)
						if !ok {
							t.Fatalf("filtered items are not compatible with an Item Collection %T", filteredRandomObjects)
						}
						if totalItems != uint(len(randomObjects)) {
							t.Fatalf("invalid collection total items count returned from loading %d, expected %d", totalItems, len(randomObjects))
						}
						if len(filteredItems) != len(foundItems) {
							t.Fatalf("invalid collection item counts returned from loading %d, expected %d", len(foundItems), len(filteredItems))
						}
						if !cmp.Equal(foundItems, filteredItems) {
							t.Errorf("invalid items returned from loading: %s", cmp.Diff(foundItems, filteredItems))
						}
						if len(filteredItems) != cnt {
							t.Fatalf("invalid collection item counts returned from loading %d, expected %d", len(foundItems), cnt)
						}
						_ = vocab.OnCollectionIntf(loadIt, func(col vocab.CollectionInterface) error {
							nextIRI := filters.NextPageFromCollection(col).GetLink()
							if !colIRI.Equal(nextIRI) {
								checks, _ = filters.FromIRI(nextIRI)
							}
							return nil
						})
					})
				}
			})
		}

		t.Run(fmt.Sprintf("remove %d items from collection", randomObjects.Count()), func(t *testing.T) {
			if err = storage.RemoveFrom(colIRI, randomObjects...); err != nil {
				t.Errorf("unable to remove objects from collection: %s", err)
			}
			loadedIt, err := storage.Load(colIRI)
			if err != nil {
				t.Errorf("unable to load collection %s: %s", colIRI, err)
			}
			err = vocab.OnCollectionIntf(loadedIt, func(col vocab.CollectionInterface) error {
				if col.Count() != 0 {
					t.Fatalf("invalid collection item counts returned from loading %d, expected %d", col.Count(), 0)
				}
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

	t.Run(fmt.Sprintf("delete %d random objects", len(randomObjects)), func(t *testing.T) {
		for _, ob := range randomObjects {
			if err := storage.Delete(ob); err != nil {
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
}
