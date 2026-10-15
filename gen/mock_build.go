package gen

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/activitypub/build"
	"github.com/go-ap/filters"
	"github.com/go-ap/storage-conformance-suite/gen/names"
)

var (
	BaseTime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

	DefaultHost vocab.IRI = "https://example.com"
	RootID                = DefaultHost.AddPath("~root")

	publicAudience = vocab.ItemCollection{vocab.PublicNS}

	Root = build.Person(
		build.WithID(RootID),
		build.WithPublished(BaseTime),
		build.WithName("Rooty McRootface"),
		build.WithSummary("The base actor for the conformance test suite"),
		build.WithContent("<p>The base actor for the conformance test suite</p>"),
		build.WithURL(RootID),
		build.WithAudience(vocab.PublicNS),
		build.WithTo(vocab.PublicNS),
		build.WithLikes(),
		build.WithShares(),
		build.WithInbox(),
		build.WithOutbox(),
		build.WithFollowing(),
		build.WithFollowers(),
		build.WithLiked(),
		build.WithPreferredUsername("root"),
	)

	typesNeedReasons = vocab.ActivityVocabularyTypes{
		vocab.BlockType,
		vocab.FlagType,
		vocab.IgnoreType,
	}

	typesNeedFullObject = vocab.ActivityVocabularyTypes{
		vocab.CreateType,
		vocab.UpdateType,
	}

	typesWorkNonDestructivelyOnObjects = vocab.ActivityVocabularyTypes{
		vocab.LikeType,
		vocab.DislikeType,
		vocab.FlagType,
		vocab.BlockType,
		vocab.FollowType,
		vocab.IgnoreType,
	}

	typesWorkOnObjects = append(typesWorkNonDestructivelyOnObjects, vocab.DeleteType)

	typesWorkNonDestructivelyOnActors = typesWorkNonDestructivelyOnObjects
	typesWorkOnActors                 = typesWorkOnObjects

	typesWorkNonDestructivelyOnActivities = typesWorkNonDestructivelyOnObjects
	typesWorkOnActivities                 = append(typesWorkNonDestructivelyOnActivities, vocab.UndoType)

	genCache = sync.Map{}

	typeCountMap = make(map[string]int)
)

func addToGenCache(it vocab.Item) {
	k := it.GetLink()
	genCache.Store(k, it)
}

func getFromCache(ff ...filters.Check) vocab.ItemCollection {
	result := make(vocab.ItemCollection, 0)
	genCache.Range(func(_, value any) bool {
		if it, ok := value.(vocab.Item); ok && filters.All(ff...).Match(it) {
			result = append(result, it)
		}
		return true
	})
	return result
}

func typeMatchMimeType(typ vocab.ActivityVocabularyType, mt string) bool {
	switch mt {
	case "image/svg+xml":
		return vocab.DocumentType.Match(typ)
	case "text/html", "text/markdown", "text/plain":
		return vocab.ActivityVocabularyTypes{vocab.NoteType, vocab.ArticleType}.Match(typ)
	case "video/webm", "video/mp4", "video/mpeg":
		return vocab.VideoType.Match(typ)
	case "image/png", "image/jpeg", "image/gif":
		return vocab.ImageType.Match(typ)
	case "audio/mp3", "audio/mpeg", "audio/mpeg3", "application/octet-stream":
		return vocab.AudioType.Match(typ)
	default:
		return false
	}
}

func getContentByType(typ vocab.ActivityVocabularyType) []byte {
	validArray := make([][]byte, 0)
	for mimeType, files := range ContentMap {
		if !typeMatchMimeType(typ, mimeType) {
			continue
		}
		for _, file := range files {
			validArray = append(validArray, file)
		}
	}
	if len(validArray) == 0 {
		return nil
	}
	return validArray[rand.Intn(len(validArray))]
}

func getRandomContent() []byte {
	validArray := make([][]byte, 0)
	for _, files := range ContentMap {
		for _, file := range files {
			validArray = append(validArray, file)
		}
	}
	if len(validArray) == 0 {
		return nil
	}
	return validArray[rand.Int()%len(validArray)]
}

// getRandomTime we use a random time,
// because time.Now() contains monotonic information which we don't care about
func getRandomTime() time.Time {
	hour := time.Duration(rand.Int31n(24)) * time.Hour
	minute := time.Duration(rand.Int31n(59)) * time.Minute
	second := time.Duration(rand.Int31n(59)) * time.Second
	BaseTime = BaseTime.Add(hour + minute + second)
	return BaseTime
}

func typeAsString(typ vocab.Typer) string {
	if tt, ok := typ.(vocab.ActivityVocabularyType); ok {
		return string(tt)
	}
	if tt, ok := typ.(vocab.ActivityVocabularyTypes); ok && len(tt) > 0 {
		return string(tt[0])
	}
	return "tag"
}

func WithGeneratedID() vocab.WithObjectFn {
	return setObjectID
}

func setObjectID(ob *vocab.Object) error {
	isCollection := vocab.IsCollection(ob)
	pieces := make([]string, 0)
	base := DefaultHost
	if !vocab.IsNil(ob.AttributedTo) {
		base = ob.AttributedTo.GetLink()
	}
	pieces = append(pieces, "/")
	if isCollection {
		typ := strings.ToLower(typeAsString(ob.Type))
		pieces = append(pieces, typ)
	} else {
		typ := strings.ToLower(typeAsString(ob.Type))
		cnt, _ := typeCountMap[typ]
		cnt++
		typeCountMap[typ] = cnt
		pieces = append(pieces, typ, strconv.Itoa(cnt))
	}
	ob.ID = base.AddPath(filepath.Join(pieces...))
	return nil
}

func setLinkID(ob *vocab.Link) error {
	base := DefaultHost
	pieces := make([]string, 0)
	pieces = append(pieces, "/")
	typ := strings.ToLower(typeAsString(ob.Type))
	cnt, _ := typeCountMap[typ]
	cnt++
	typeCountMap[typ] = cnt
	pieces = append(pieces, typ, strconv.Itoa(cnt))
	ob.ID = base.AddPath(filepath.Join(pieces...))
	return nil
}

func DefaultSetter(it vocab.Item) {
	if vocab.IsObject(it) {
		_ = vocab.OnObject(it, setObjectID)
	}
	if vocab.IsLink(it) {
		_ = vocab.OnLink(it, setLinkID)
	}

	addToGenCache(it)
}

var SetItemID = DefaultSetter

var CollectionTypes = vocab.ActivityVocabularyTypes{vocab.OrderedCollectionType, vocab.CollectionType}

func RandomCollection(attrTo vocab.LinkOrIRI) vocab.Item {
	// NOTE(marius): skip creating unordered collections as we don't have a consistent storage for the items
	//  See: https://todo.sr.ht/~mariusor/go-activitypub/508
	typ := vocab.OrderedCollectionType
	//typ := CollectionTypes[rand.Intn(len(CollectionTypes))]

	var col vocab.Item
	switch {
	case vocab.OrderedCollectionType.Match(typ):
		col = new(vocab.OrderedCollection)
	case vocab.CollectionType.Match(typ):
		col = new(vocab.Collection)
	}
	_ = vocab.OnObject(col, func(col *vocab.Object) error {
		col.Type = typ
		col.AttributedTo = attrTo.GetLink()
		col.Published = getRandomTime()
		SetItemID(col)
		return nil
	})

	return col
}

type generatorFn func(iri vocab.LinkOrIRI) vocab.Item

var itemGenFns = append(nonActivityGenFns, RandomQuestion)

func randomActivity(actFn func(vocab.LinkOrIRI, vocab.LinkOrIRI) vocab.Item) func(vocab.LinkOrIRI) vocab.Item {
	return func(attrTo vocab.LinkOrIRI) vocab.Item {
		var obj vocab.Item
		matchFilters := filters.All(filters.HasType(vocab.ObjectTypes...), filters.SameAttributedTo(attrTo.GetLink()))
		if objs := getFromCache(matchFilters); len(objs) == 0 {
			obj = RandomObject(attrTo)
		} else {
			obj = objs[rand.Intn(len(objs))]
		}
		return actFn(obj, attrTo)
	}
}

func RandomItem(attrTo vocab.LinkOrIRI) vocab.Item {
	genFns := append(itemGenFns, randomActivity(CreateActivity), randomActivity(RandomNonCreateActivity))
	fn := genFns[rand.Intn(len(genFns))]
	return fn(attrTo)
}

func RandomObjectByType(attrTo vocab.Item, typ vocab.ActivityVocabularyType) vocab.Item {
	return build.Object(
		WithGeneratedID(),
		build.WithType(typ),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithPublished(getRandomTime()),
		build.WithAudience(vocab.PublicNS),
		WithRandomObjectCollections(),
		setContentByType(typ),
	)
}

func RandomProfile(attrTo vocab.LinkOrIRI) vocab.Item {
	return build.Profile(
		WithGeneratedID(),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithPublished(getRandomTime()),
		build.WithAudience(vocab.PublicNS),
		WithRandomObjectCollections(),
	)
}

func RandomPlace(attrTo vocab.LinkOrIRI) vocab.Item {
	return build.Place(
		WithGeneratedID(),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithPublished(getRandomTime()),
		build.WithAudience(vocab.PublicNS),
		WithRandomObjectCollections(),
	)
}

func RandomTombstone(attrTo vocab.LinkOrIRI) vocab.Item {
	return build.Tombstone(
		WithGeneratedID(),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithPublished(getRandomTime()),
		build.WithAudience(vocab.PublicNS),
		build.WithFormerType(getRandomActorType()),
		WithRandomObjectCollections(),
	)
}

func setContentByType(typ vocab.ActivityVocabularyType) vocab.WithObjectFn {
	var data []byte
	if emptyType.Match(typ) {
		data = getRandomContent()
	} else {
		data = getContentByType(typ)
	}
	return func(ob *vocab.Object) error {
		return setContentData(ob, data)
	}
}

func setContentData(ob *vocab.Object, data []byte) error {
	typ, mt := getObjectTypes(data)
	ob.MediaType = mt
	ob.Type = typ

	if len(data) == 0 {
		ob.Content = vocab.DefaultLangValue("no data")
	} else {
		if !strings.Contains(string(mt), "text") {
			buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
			base64.RawStdEncoding.Encode(buf, data)
			data = buf
		} else {
			ob.Summary = vocab.DefaultLangValue(string(data[:bytes.Index(data, []byte{'.'})]))
		}
		ob.Content = vocab.DefaultLangValue(string(data))
	}
	return nil
}

func RandomTags(parent vocab.LinkOrIRI, cnt int) vocab.ItemCollection {
	tags := make(vocab.ItemCollection, 0, cnt)
	for range cnt {
		_ = tags.Append(RandomTag(parent))
	}
	return tags
}

var emptyType = vocab.ActivityVocabularyType("-")

func RandomObject(attrTo vocab.LinkOrIRI) vocab.Item {
	return build.Object(
		WithGeneratedID(),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithAudience(vocab.PublicNS),
		build.WithPublished(getRandomTime()),
		WithRandomObjectCollections(),
		setContentByType(emptyType),
	)
}

func randPercent(p int) bool {
	return rand.Int31n(101) > int32(100-p)
}

func WithRandomActorCollections() vocab.WithActorFn {
	return SetActorCollections
}

func SetActorCollections(act *vocab.Actor) error {
	if randPercent(60) {
		act.Following = vocab.Following.IRI(act)
	}
	if randPercent(60) {
		act.Followers = vocab.Followers.IRI(act)
	}
	if randPercent(20) {
		act.Liked = vocab.Liked.IRI(act)
	}
	return vocab.OnObject(act, SetObjectCollections)
}

func WithRandomObjectCollections() vocab.WithObjectFn {
	return SetObjectCollections
}

func SetObjectCollections(ob *vocab.Object) error {
	if randPercent(50) {
		ob.Replies = vocab.Replies.IRI(ob)
	}
	if randPercent(50) {
		ob.Likes = vocab.Likes.IRI(ob)
	}
	if randPercent(50) {
		ob.Shares = vocab.Shares.IRI(ob)
	}
	return nil
}

var svgDocumentStart = []byte{'<', 's', 'v', 'g'}

func getObjectTypes(data []byte) (vocab.ActivityVocabularyType, vocab.MimeType) {
	contentType := http.DetectContentType(data)
	var objectType vocab.ActivityVocabularyType

	contentType, _, _ = mime.ParseMediaType(contentType)
	switch contentType {
	case "text/html", "text/markdown", "text/plain":
		objectType = vocab.NoteType
		if len(data) > 600 {
			objectType = vocab.ArticleType
		}
		if bytes.Contains(data, svgDocumentStart) {
			objectType = vocab.DocumentType
			contentType = "image/svg+xml"
		}
	case "image/svg+xml":
		objectType = vocab.DocumentType
	case "video/webm", "video/mp4", "video/mpeg":
		objectType = vocab.VideoType
	case "audio/mp3", "audio/mpeg", "audio/mpeg3", "application/octet-stream":
		objectType = vocab.AudioType
	case "image/png", "image/jpeg", "image/gif":
		objectType = vocab.ImageType
	default:
		objectType = "Unknown"
	}
	return objectType, vocab.MimeType(contentType)
}

func SortItemCollectionByID(items vocab.ItemCollection) {
	slices.SortStableFunc(items, func(a, b vocab.Item) int {
		return -strings.Compare(a.GetLink().String(), b.GetLink().String())
	})
}

func RandomItemCollection(count int, attrTo vocab.LinkOrIRI) vocab.ItemCollection {
	items := make(vocab.ItemCollection, 0, count)
	for range count {
		items = append(items, RandomItem(attrTo))
	}
	return items
}

func ItemCollectionFromLinkSlice(col []vocab.LinkOrIRI) vocab.ItemCollection {
	res := make(vocab.ItemCollection, 0, len(col))
	for _, li := range col {
		if it, ok := li.(vocab.Item); ok {
			_ = res.Append(it)
		}
	}
	return res
}

func RandomObjects(count int, attrTo vocab.LinkOrIRI) vocab.ItemCollection {
	items := make(vocab.ItemCollection, 0, count)
	for range count {
		_ = items.Append(RandomNonActivity(attrTo))
	}

	tags := filters.Checks{filters.NoType}.Run(items).(vocab.ItemCollection)
	for _, it := range items {
		if vocab.NilType.Match(it.GetType()) {
			// NOTE(marius): skip on tag items to avoid having a tag referencing itself and triggering a stack overflow
			continue
		}
		if rand.Intn(2) == 0 {
			// NOTE(marius): also skip on two thirds of remaining items
			continue
		}
		_ = vocab.OnObject(it, func(ob *vocab.Object) error {
			// NOTE(marius): sample some random tags and set them as the Tag property on the object
			randomTags := ItemCollectionFromLinkSlice(RandomFromCollection(tags, rand.Intn(len(tags)/2)))
			ob.Tag = randomTags.Normalize()
			return nil
		})

	}
	return items
}

var reasons = []string{
	"Lorem ipsum, dolor sic amet",
	"A random reason for a stupid activity",
}

func getRandomReason() string {
	return reasons[rand.Intn(len(reasons))]
}

func getActivityTypeByObject(ob vocab.Item) vocab.ActivityVocabularyType {
	if vocab.IsNil(ob) {
		return typesWorkOnObjects[rand.Int()%len(typesWorkOnObjects)]
	}

	typ := ob.GetType()
	switch {
	case vocab.ActivityTypes.Match(typ):
		return typesWorkOnActivities[rand.Intn(len(typesWorkOnActivities))]
	case vocab.ActorTypes.Match(typ):
		return typesWorkOnActors[rand.Intn(len(typesWorkOnActors))]
	}

	return typesWorkOnActors[rand.Intn(len(typesWorkOnObjects))]
}

func getNonDestructiveActivityTypeByObject(ob vocab.LinkOrIRI) vocab.ActivityVocabularyType {
	if vocab.IsNil(ob) {
		return vocab.IntransitiveActivityTypes[rand.Intn(len(vocab.IntransitiveActivityTypes))]
	}

	if it, ok := ob.(vocab.Item); ok {
		typ := it.GetType()
		switch {
		case vocab.ActivityTypes.Match(typ):
			return typesWorkNonDestructivelyOnActivities[rand.Intn(len(typesWorkNonDestructivelyOnActivities))]
		case vocab.ActorTypes.Match(typ):
			return typesWorkNonDestructivelyOnActors[rand.Intn(len(typesWorkNonDestructivelyOnActors))]
		}
	}

	return typesWorkNonDestructivelyOnActors[rand.Intn(len(typesWorkNonDestructivelyOnObjects))]
}

func getActivityInit(ob vocab.LinkOrIRI, attrTo vocab.LinkOrIRI) ([]build.InitFn, vocab.Typer) {
	initFns := []build.InitFn{
		WithGeneratedID(),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithActor(attrTo.GetLink()),
		build.WithTo(RootID, vocab.PublicNS),
		build.WithPublished(getRandomTime()),
		WithRandomObjectCollections(),
	}
	typ := getNonDestructiveActivityTypeByObject(ob)
	initFns = append(initFns, build.WithType(typ))
	if typesNeedReasons.Match(typ) {
		initFns = append(initFns, build.WithContent(vocab.DefaultLangValue(getRandomReason())))
		initFns = append(initFns, build.WithSummary(vocab.DefaultLangValue(getRandomReason())))
	}

	if vocab.QuestionType.Match(typ) {
		initFns = append(initFns, WithRandomQuestionOptions(attrTo, rand.Intn(10)))
	} else if !vocab.IsNil(ob) {
		if oob, ok := ob.(vocab.Item); ok {
			initFns = append(initFns, build.WithObject(oob))
		} else {
			initFns = append(initFns, build.WithObject(ob.GetLink()))
		}
	}

	return initFns, typ
}

func RandomNonCreateActivity(ob vocab.LinkOrIRI, attrTo vocab.LinkOrIRI) vocab.Item {
	initFns, typ := getActivityInit(ob, attrTo)
	if intransitiveActivity := vocab.IsNil(ob); intransitiveActivity {
		if vocab.QuestionType.Match(typ) {
			return build.Question(initFns...)
		}
		return build.IntransitiveActivity(initFns...)
	}
	return build.Activity(initFns...)
}

func WithRandomQuestionOptions(attrTo vocab.LinkOrIRI, count int) vocab.WithQuestionFn {
	opts := make(vocab.ItemCollection, 0, count)
	for range count {
		opts = append(opts, RandomObject(attrTo))
	}

	fn := build.WithAnyOf
	if oneOf := rand.Intn(2) == 1; oneOf {
		fn = build.WithOneOf
	}
	return fn(opts...)
}

func RandomQuestion(attrTo vocab.LinkOrIRI) vocab.Item {
	initFns, _ := getActivityInit(nil, attrTo)
	return build.Question(initFns...)
}

func getRandomActorType() vocab.ActivityVocabularyType {
	return vocab.ActorTypes[rand.Intn(len(vocab.ActorTypes))]
}

func RandomActor(attrTo vocab.LinkOrIRI) vocab.Item {
	name := names.GetRandom()
	return build.Actor(
		WithGeneratedID(),
		build.WithType(getRandomActorType()),
		build.WithIcon(RandomImage("image/png", attrTo.GetLink())),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithAudience(vocab.PublicNS),
		build.WithName(name),
		build.WithPreferredUsername(name),
		build.WithPublished(getRandomTime()),
		build.WithInbox(),
		build.WithOutbox(),
		WithRandomActorCollections(),
	)
}

func getRandomContentByMimeType(mimeType vocab.MimeType) []byte {
	if validArray, ok := ContentMap[string(mimeType)]; ok {
		return validArray.First()
	}
	return nil
}

var (
	text = bytes.Join(
		[][]byte{
			getRandomContentByMimeType("text/plain"),
			getRandomContentByMimeType("text/markdown"),
		},
		[]byte{'\n'},
	)

	words = func() []string {
		words := make([]string, 0, 400)
		for _, w := range regexp.MustCompile(`\s+|\p{P}+`).Split(string(text), -1) {
			if len(w) == 0 {
				continue
			}
			words = append(words, w)
		}
		return words
	}()
)

func getRandomWord() string {
	return words[rand.Intn(len(words))]
}

func RandomTag(attrTo vocab.LinkOrIRI) vocab.Item {
	return build.Object(
		WithGeneratedID(),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithAudience(vocab.PublicNS),
		build.WithName(vocab.DefaultLangValue("#"+getRandomWord())),
		build.WithPublished(getRandomTime()),
		WithRandomObjectCollections(),
	)
}

func RandomImage(mime vocab.MimeType, parent vocab.Item) vocab.Item {
	data := getRandomContentByMimeType(mime)
	buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
	base64.RawStdEncoding.Encode(buf, data)
	return build.Image(
		WithGeneratedID(),
		build.WithAttributedTo(parent.GetLink()),
		build.WithMediaType(mime),
		build.WithAudience(vocab.PublicNS),
		build.WithName(vocab.DefaultLangValue("#"+getRandomWord())),
		build.WithPublished(getRandomTime()),
		build.WithContent(buf),
		WithRandomObjectCollections(),
	)
}

func getRandomLinkType() vocab.ActivityVocabularyType {
	return vocab.LinkTypes[rand.Intn(len(vocab.LinkTypes))]
}

func getRandomName() vocab.NaturalLanguageValues {
	return vocab.DefaultLangValue(names.GetRandom())
}

func getRandomHref() vocab.IRI {
	return DefaultHost.AddPath(filepath.Join(strings.Split(names.GetRandom(), "_")...))
}

func RandomLink(_ vocab.LinkOrIRI) vocab.Item {
	return build.Link(
		WithGeneratedID(),
		build.WithType(getRandomLinkType()),
		build.WithName(getRandomName()),
		build.WithHref(getRandomHref()),
		build.WithHrefLang(vocab.DefaultLang),
	)
}

func CreateActivity(ob vocab.LinkOrIRI, attrTo vocab.LinkOrIRI) vocab.Item {
	initFns := []build.InitFn{
		WithGeneratedID(),
		build.WithAttributedTo(attrTo.GetLink()),
		build.WithTo(RootID, vocab.PublicNS),
		build.WithPublished(getRandomTime()),
		build.WithActor(attrTo.GetLink()),
		WithRandomObjectCollections(),
	}
	if it, ok := ob.(vocab.Item); ok {
		initFns = append(initFns, build.WithObject(it))
	}
	return build.Create(initFns...)
}

var nonActivityGenFns = []generatorFn{
	RandomObject,
	RandomPlace,
	RandomProfile,
	RandomTombstone,
	RandomActor,
	RandomLink,
	RandomTag,
	RandomCollection,
}

func RandomNonActivity(attrTo vocab.LinkOrIRI) vocab.Item {
	fn := nonActivityGenFns[rand.Intn(len(nonActivityGenFns))]
	return fn(attrTo)
}

func RandomPlausible(cnt int) vocab.ItemCollection {
	result := make(vocab.ItemCollection, 0, cnt)
	for _, ob := range PlausibleStorage(Root, cnt) {
		if it, ok := ob.(vocab.Item); ok {
			result = append(result, it)
		}
	}
	return result
}

func RandomFromCollection(col vocab.ItemCollection, cnt int) []vocab.LinkOrIRI {
	if len(col) == 0 {
		return nil
	}
	ri := make([]int, 0, cnt)
	for range cnt {
		for {
			r := rand.Intn(len(col))
			if !slices.Contains(ri, r) {
				ri = append(ri, r)
				break
			}
		}
	}
	res := make([]vocab.LinkOrIRI, 0, cnt)
	for i := range ri {
		for j, it := range col {
			if i == j {
				res = append(res, it)
			}
		}
	}
	return res
}

func PlausibleStorage(attrTo vocab.Item, cnt int) []vocab.LinkOrIRI {
	obCnt := cnt / 4

	actors := make([]vocab.LinkOrIRI, 0, obCnt/2)
	objects := make([]vocab.LinkOrIRI, 0, obCnt)
	activities := make([]vocab.LinkOrIRI, 0, cnt)

	for range cap(actors) {
		act := RandomActor(attrTo)
		actors = append(actors, act)
		activities = append(activities, CreateActivity(act, attrTo))
	}

	for range obCnt {
		act := actors[rand.Intn(len(actors))]
		ob := RandomNonActivity(act)
		objects = append(objects, ob)
		activities = append(activities, CreateActivity(ob, act))
	}

	objects = append(objects, actors...)
	remActCnt := cnt - len(objects)
	for range remActCnt {
		act := actors[rand.Intn(len(actors))]
		ob := objects[rand.Intn(len(objects))]
		activities = append(activities, RandomNonCreateActivity(ob, act))
	}

	slices.SortStableFunc(activities, ItemCollectionByTimestamp)
	return activities
}

func ItemCollectionByTimestamp(i1, i2 vocab.LinkOrIRI) int {
	if vocab.IsNil(i1) || vocab.IsNil(i2) {
		return 0
	}

	var t1 time.Time
	var t2 time.Time
	_ = vocab.OnObject(i1, func(o1 *vocab.Object) error {
		t1 = o1.Published
		if o1.Updated.After(t1) {
			t1 = o1.Updated
		}
		return nil
	})
	_ = vocab.OnObject(i2, func(o2 *vocab.Object) error {
		t2 = o2.Published
		if o2.Updated.After(t2) {
			t2 = o2.Updated
		}
		return nil
	})
	return int(t1.Sub(t2))
}
