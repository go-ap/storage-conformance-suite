package conformance

import (
	"fmt"
	"sort"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"github.com/go-ap/storage-conformance-suite/gen"
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
	//			preferredUsername: "janeDoe", // ...
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
	// Deprecated
	// NOTE(marius): should we remove this in favour of custom logic for Save()?
	//Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error)

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
	if _, err := storage.Save(gen.Root); err != nil {
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
		it, err := storage.Load(gen.RootID)
		if err != nil {
			t.Errorf("unable to load root item: %s", err)
		}
		if !cmp.Equal(gen.Root, it) {
			t.Errorf("invalid root actor loaded from storage %s", cmp.Diff(gen.Root, it))
		}
	})

	randomObjects := gen.RandomObjects(64, gen.Root)
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
			typ := ob.GetType()
			if vocab.ActorTypes.Match(typ) {
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
			} else if !vocab.LinkTypes.Match(typ) {
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

	nonExistentCollection := vocab.CollectionPath("non-existent").IRI(randomObjects.First())
	t.Run(fmt.Sprintf("operate on non-existent collection: %s", nonExistentCollection), func(t *testing.T) {
		t.Run("RemoveFrom", func(t *testing.T) {
			err := storage.RemoveFrom(nonExistentCollection, randomObjects...)
			if err == nil {
				t.Errorf("expected error on invalid collection removal")
			}
			if !errors.IsNotFound(err) {
				t.Errorf("error received is not a not-found error: %s", err)
			}
		})
		t.Run("AddTo", func(t *testing.T) {
			err := storage.AddTo(nonExistentCollection, randomObjects...)
			if err == nil {
				t.Errorf("expected error on invalid collection removal")
			}
			if !errors.IsNotFound(err) {
				t.Errorf("error received is not a not-found error: %s", err)
			}
		})
	})

	col := gen.RandomCollection(gen.Root)
	colIRI := col.GetLink()
	colType := colLabel(col.GetType())
	t.Run(fmt.Sprintf("create %s", colType), func(t *testing.T) {
		savedIt, err := storage.Save(col)
		if err != nil {
			t.Errorf("unable to save %s: %v", colType, err)
		}
		if !cmp.Equal(col, savedIt) {
			t.Errorf("invalid %s returned from saving %s", colType, cmp.Diff(col, savedIt))
		}
		loadIt, err := storage.Load(colIRI)
		if err != nil {
			t.Errorf("unable to load %s %s: %v", colType, colIRI, err)
		}
		if !cmp.Equal(col, loadIt, EquateItems) {
			t.Errorf("invalid %s returned from loading %s: %s", colType, colIRI, cmp.Diff(col, loadIt, EquateItems))
		}

		t.Run(fmt.Sprintf("add %d items to %s", randomObjects.Count(), colType), func(t *testing.T) {
			if err = storage.AddTo(colIRI, randomObjects...); err != nil {
				t.Errorf("unable to add objects to %s: %v", err, colType)
			}
			loadedIt, err := storage.Load(colIRI)
			if err != nil {
				t.Errorf("unable to load %s %s: %v", colType, colIRI, err)
			}
			err = vocab.OnCollectionIntf(loadedIt, func(col vocab.CollectionInterface) error {
				if col.Count() != uint(len(randomObjects)) {
					t.Fatalf("invalid %s item counts returned from loading %d, expected %d", colType, col.Count(), len(randomObjects))
				}
				savedItems := col.Collection()
				if len(savedItems) != len(randomObjects) {
					t.Fatalf("invalid %s item counts returned from loading %d, expected %d", colType, len(savedItems), len(randomObjects))
				}
				if vocab.CollectionType.Match(col.GetType()) {
					gen.SortItemCollectionByID(randomObjects)
					gen.SortItemCollectionByID(savedItems)
				}
				for i, it := range randomObjects {
					if !cmp.Equal(it, savedItems[i]) {
						t.Errorf("invalid item at pos %d, unable: %s", i, cmp.Diff(it, savedItems[i]))
					}
				}
				return nil
			})
			if err != nil {
				t.Errorf("loaded object wasn't a(n) %s %s: %v", colType, colIRI, err)
			}
		})
		queryFilters := append(withPagination, append(byTypeFilters, byActivityObjectTypeFilters...)...)
		for _, fil := range queryFilters {
			t.Run(fmt.Sprintf("query %s with filters %#v", colType, fil), func(t *testing.T) {
				loadIt, err = storage.Load(colIRI, fil...)
				if err != nil {
					t.Errorf("unable to load %s %s: %v", colType, colIRI, err)
				}
				var foundItems vocab.ItemCollection
				var totalItems uint
				err = vocab.OnOrderedCollection(loadIt, func(col *vocab.OrderedCollection) error {
					foundItems = col.OrderedItems
					totalItems = col.TotalItems
					return nil
				})
				if err != nil {
					t.Errorf("loaded object wasn't a(n) %s %s: %v", colType, colIRI, err)
				}
				filters.ResetPagination(fil...)
				filteredRandomObjects := fil.Run(randomObjects)
				filteredItems, ok := filteredRandomObjects.(vocab.ItemCollection)
				if !ok {
					t.Fatalf("filtered items are not compatible with an Item Collection %T", filteredRandomObjects)
				}
				if totalItems != uint(len(randomObjects)) {
					t.Fatalf("invalid %s total items count returned from loading %d, expected %d", colType, totalItems, len(randomObjects))
				}
				if len(filteredItems) != len(foundItems) {
					t.Fatalf("invalid %s item counts returned from loading %d, expected %d", colType, len(foundItems), len(filteredItems))
				}
				if len(filteredItems) > 0 {
					if !cmp.Equal(foundItems, filteredItems, EquateItemCollections) {
						t.Errorf("invalid items returned from loading: %s", cmp.Diff(foundItems, filteredItems, EquateItemCollections))
					}
				}
			})
		}
		for _, cnt := range genCountsFor(len(randomObjects)) {
			checks := filters.Checks{filters.WithMaxCount(cnt)}
			t.Run(fmt.Sprintf("traverse %s with pagination %d", colType, cnt), func(t *testing.T) {
				for range len(randomObjects) / cnt {
					t.Run(fmt.Sprintf("query %s with filters %#v", colType, checks), func(t *testing.T) {
						loadIt, err := storage.Load(colIRI, checks...)
						if err != nil {
							t.Errorf("unable to load %s %s: %v", colType, colIRI, err)
						}
						var foundItems vocab.ItemCollection
						var totalItems uint
						err = vocab.OnOrderedCollection(loadIt, func(col *vocab.OrderedCollection) error {
							foundItems = col.OrderedItems
							totalItems = col.TotalItems
							return nil
						})
						if err != nil {
							t.Errorf("loaded object wasn't a %s %s: %v", colType, colIRI, err)
						}
						filters.ResetPagination(checks...)
						filteredRandomObjects := checks.Run(randomObjects)
						filteredItems, ok := filteredRandomObjects.(vocab.ItemCollection)
						if !ok {
							t.Fatalf("filtered items are not compatible with an Item Collection %T", filteredRandomObjects)
						}
						if totalItems != uint(len(randomObjects)) {
							t.Fatalf("invalid %s total items count returned from loading %d, expected %d", colType, totalItems, len(randomObjects))
						}
						if len(filteredItems) != len(foundItems) {
							t.Fatalf("invalid %s item counts returned from loading %d, expected %d", colType, len(foundItems), len(filteredItems))
						}
						if !cmp.Equal(foundItems, filteredItems, EquateItems) {
							t.Errorf("invalid items returned from loading: %s", cmp.Diff(foundItems, filteredItems, EquateItems))
						}
						if len(filteredItems) != cnt {
							t.Fatalf("invalid %s item counts returned from loading %d, expected %d", colType, len(foundItems), cnt)
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

		t.Run(fmt.Sprintf("remove %d items from %s", randomObjects.Count(), colType), func(t *testing.T) {
			if err = storage.RemoveFrom(colIRI, randomObjects...); err != nil {
				t.Errorf("unable to remove objects from %s: %v", colType, err)
			}
			loadedIt, err := storage.Load(colIRI)
			if err != nil {
				t.Errorf("unable to load %s %s: %v", colType, colIRI, err)
			}
			err = vocab.OnCollectionIntf(loadedIt, func(col vocab.CollectionInterface) error {
				if col.Count() != 0 {
					t.Fatalf("invalid %s item counts returned from loading %d, expected %d", colType, col.Count(), 0)
				}
				if remainingItems := col.Collection(); len(remainingItems) != 0 {
					t.Errorf("invalid %s returned from loading it has %d items: expected empty", colType, len(remainingItems))
					t.Logf("%s", cmp.Diff(vocab.ItemCollection{}, remainingItems))
				}
				return nil
			})
			if err != nil {
				t.Errorf("loaded object wasn't a(n) %s %s: %v", colType, colIRI, err)
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

func colLabel(typ vocab.Typer) string {
	switch {
	case vocab.CollectionType.Match(typ):
		return "collection"
	case vocab.CollectionPageType.Match(typ):
		return "collection page"
	case vocab.OrderedCollectionType.Match(typ):
		return "ordered collection"
	case vocab.OrderedCollectionPageType.Match(typ):
		return "ordered collection page"
	default:
		return "not a collection!!!"
	}
	return "unknown"
}
func genCountsFor(cnt int) []int {
	result := make([]int, 0, cnt/2)
	odd := 1
	for {
		half := cnt / 2
		if half <= 1 {
			break
		}
		cnt = half
		result = append(result, half)
	}
	for i := len(result) - 1; i >= 0; i-- {
		if ev := result[i]; ev%2 == 0 {
			result = append(result, ev+odd)
			odd += 2
		}
	}
	sort.Ints(result)
	return result
}

func areItems(a, b any) bool {
	_, ok1 := a.(vocab.Item)
	_, ok2 := b.(vocab.Item)
	return ok1 && ok2
}

func compareItems(x, y any) bool {
	var i1 vocab.Item
	var i2 vocab.Item
	if ic1, ok := x.(vocab.Item); ok {
		i1 = ic1
	}
	if ic2, ok := y.(vocab.Item); ok {
		i2 = ic2
	}
	return vocab.ItemsEqual(i1, i2)
}

var EquateItems = cmp.FilterValues(areItems, cmp.Comparer(compareItems))

func areItemCollections(a, b any) bool {
	_, ok1 := a.(vocab.ItemCollection)
	_, ok3 := a.(*vocab.ItemCollection)
	_, ok2 := b.(vocab.ItemCollection)
	_, ok4 := b.(*vocab.ItemCollection)
	return (ok1 || ok3) && (ok2 || ok4)
}

var EquateItemCollections = cmp.FilterValues(areItemCollections, cmp.Comparer(compareItems))
