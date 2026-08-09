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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/filters"
	"github.com/go-ap/storage-conformance-suite/gen/names"
)

var (
	BaseTime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

	DefaultHost vocab.IRI = "https://example.com"
	RootID                = DefaultHost.AddPath("~root")

	publicAudience = vocab.ItemCollection{vocab.PublicNS}

	Root = &vocab.Actor{
		ID:                RootID,
		Type:              vocab.PersonType,
		Published:         BaseTime,
		Name:              vocab.DefaultNaturalLanguage("Rooty McRootface"),
		Summary:           vocab.DefaultNaturalLanguage("The base actor for the conformance test suite"),
		Content:           vocab.DefaultNaturalLanguage("<p>The base actor for the conformance test suite</p>"),
		URL:               vocab.Item(RootID),
		Audience:          publicAudience,
		To:                publicAudience,
		Likes:             vocab.Likes.IRI(RootID),
		Shares:            vocab.Shares.IRI(RootID),
		Inbox:             vocab.Inbox.IRI(RootID),
		Outbox:            vocab.Outbox.IRI(RootID),
		Following:         vocab.Following.IRI(RootID),
		Followers:         vocab.Followers.IRI(RootID),
		Liked:             vocab.Liked.IRI(RootID),
		PreferredUsername: vocab.DefaultNaturalLanguage("root"),
	}

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
	return "unknown"
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

func RandomCollection(attrTo vocab.LinkOrIRI) vocab.CollectionInterface {
	col := new(vocab.OrderedCollection)
	col.Type = vocab.OrderedCollectionType
	col.AttributedTo = attrTo.GetLink()
	col.Published = getRandomTime()
	SetItemID(col)

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
	genFns := append(itemGenFns, randomActivity(CreateActivity), randomActivity(RandomNonNonContentActivity))
	fn := genFns[rand.Intn(len(genFns))]
	return fn(attrTo)
}

func RandomObjectByType(attrTo vocab.Item, typ vocab.ActivityVocabularyType) vocab.Item {
	ob := new(vocab.Object)
	ob.Type = typ
	ob.AttributedTo = attrTo.GetLink()
	// NOTE(marius): we use random time, instead of something like time.Now()
	// because the latter contains monotonic information which gets lost at loading form the mock storage we're using
	ob.Published = getRandomTime()

	_ = vocab.OnObject(ob, setContentByType(typ))
	SetItemID(ob)

	ob.Replies = vocab.Replies.IRI(ob)
	ob.Likes = vocab.Likes.IRI(ob)
	ob.Shares = vocab.Shares.IRI(ob)

	return ob
}

func RandomProfile(attrTo vocab.LinkOrIRI) vocab.Item {
	p := new(vocab.Profile)
	p.AttributedTo = attrTo.GetLink()
	p.Published = getRandomTime()
	p.Type = vocab.ProfileType
	p.Audience = publicAudience
	SetItemID(p)
	return p
}

func RandomPlace(attrTo vocab.LinkOrIRI) vocab.Item {
	p := new(vocab.Place)
	p.AttributedTo = attrTo.GetLink()
	p.Published = getRandomTime()
	p.Type = vocab.PlaceType
	p.Audience = publicAudience
	SetItemID(p)
	return p
}

func setContentByType(typ vocab.ActivityVocabularyType) func(ob *vocab.Object) error {
	var mt vocab.MimeType
	var data []byte
	if typ == "-" {
		data = getRandomContent()
	} else {
		data = getContentByType(typ)
	}
	return func(ob *vocab.Object) error {
		typ, mt = getObjectTypes(data)

		ob.Type = typ
		ob.MediaType = mt

		if len(data) == 0 {
			ob.Content = vocab.DefaultNaturalLanguage("no data")
		} else {
			if !strings.Contains(string(mt), "text") {
				buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
				base64.RawStdEncoding.Encode(buf, data)
				data = buf
			} else {
				ob.Summary = vocab.DefaultNaturalLanguage(string(data[:bytes.Index(data, []byte{'.'})]))
			}
			ob.Content = vocab.DefaultNaturalLanguage(string(data))
		}
		return nil
	}
}

func RandomObject(attrTo vocab.LinkOrIRI) vocab.Item {
	ob := new(vocab.Object)
	ob.AttributedTo = attrTo.GetLink()
	ob.Published = getRandomTime()

	_ = vocab.OnObject(ob, setContentByType("-"))
	SetItemID(ob)
	ob.Audience = publicAudience

	ob.Replies = vocab.Replies.IRI(ob)
	ob.Likes = vocab.Likes.IRI(ob)
	ob.Shares = vocab.Shares.IRI(ob)

	return ob
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
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetLink().String() <= items[j].GetLink().String()
	})
}

func RandomItemCollection(count int, attrTo vocab.LinkOrIRI) vocab.ItemCollection {
	items := make(vocab.ItemCollection, 0, count)
	for range count {
		items = append(items, RandomObject(attrTo))
	}
	SortItemCollectionByID(items)
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

func getNonDestructiveActivityTypeByObject(ob vocab.Item) vocab.ActivityVocabularyType {
	if vocab.IsNil(ob) {
		return typesWorkOnObjects[rand.Int()%len(typesWorkOnObjects)]
	}

	typ := ob.GetType()
	switch {
	case vocab.ActivityTypes.Match(typ):
		return typesWorkNonDestructivelyOnActivities[rand.Intn(len(typesWorkNonDestructivelyOnActivities))]
	case vocab.ActorTypes.Match(typ):
		return typesWorkNonDestructivelyOnActors[rand.Intn(len(typesWorkNonDestructivelyOnActors))]
	}

	return typesWorkNonDestructivelyOnActors[rand.Intn(len(typesWorkNonDestructivelyOnObjects))]
}

func RandomNonNonContentActivity(ob vocab.LinkOrIRI, attrTo vocab.LinkOrIRI) vocab.Item {
	act := new(vocab.Activity)
	if oob, ok := ob.(vocab.Item); ok {
		act.Type = getNonDestructiveActivityTypeByObject(oob)
		if ob != nil {
			act.Object = oob
		}
	} else {
		act.Object = ob.GetLink()
	}

	act.AttributedTo = attrTo.GetLink()
	act.Actor = attrTo.GetLink()
	act.To = vocab.ItemCollection{RootID, vocab.PublicNS}
	act.Published = getRandomTime()

	if typesNeedReasons.Match(act.Type) {
		act.Content = vocab.DefaultNaturalLanguage(getRandomReason())
		act.Summary = vocab.DefaultNaturalLanguage(getRandomReason())
	}
	SetItemID(act)

	return act
}

func RandomQuestion(attrTo vocab.LinkOrIRI) vocab.Item {
	act := new(vocab.Question)
	act.Type = vocab.QuestionType

	act.AttributedTo = attrTo.GetLink()
	act.Actor = attrTo.GetLink()
	act.To = vocab.ItemCollection{RootID, vocab.PublicNS}
	act.Published = getRandomTime()

	oneOf := rand.Intn(2) == 1
	count := rand.Intn(10)
	opts := RandomItemCollection(count, attrTo)
	if oneOf {
		act.OneOf = opts
	} else {
		act.AnyOf = opts
	}
	SetItemID(act)

	return act
}

func getRandomActorType() vocab.ActivityVocabularyType {
	return vocab.ActorTypes[rand.Intn(len(vocab.ActorTypes))]
}

func RandomActor(attrTo vocab.LinkOrIRI) vocab.Item {
	act := new(vocab.Actor)
	act.Name = vocab.DefaultNaturalLanguage(names.GetRandom())
	act.PreferredUsername = act.Name
	act.Type = getRandomActorType()
	act.AttributedTo = attrTo.GetLink()
	act.Icon = RandomImage("image/png", attrTo.GetLink())
	act.Published = getRandomTime()
	SetItemID(act)
	act.Audience = publicAudience

	act.Inbox = vocab.Inbox.IRI(act)
	act.Outbox = vocab.Outbox.IRI(act)
	act.Following = vocab.Following.IRI(act)
	act.Followers = vocab.Followers.IRI(act)
	act.Liked = vocab.Liked.IRI(act)

	act.Shares = vocab.Shares.IRI(act)
	act.Replies = vocab.Replies.IRI(act)
	act.Likes = vocab.Likes.IRI(act)

	return act
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
	tag := new(vocab.Object)
	tag.AttributedTo = attrTo.GetLink()
	tag.Name = vocab.DefaultNaturalLanguage("#" + getRandomWord())
	tag.Published = getRandomTime()
	tag.Audience = publicAudience
	SetItemID(tag)
	return tag
}

func RandomImage(mime vocab.MimeType, parent vocab.Item) vocab.Item {
	img := new(vocab.Image)
	img.Type = vocab.ImageType
	img.MediaType = mime
	img.AttributedTo = parent.GetLink()
	img.Published = getRandomTime()

	data := getRandomContentByMimeType(mime)
	buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
	base64.RawStdEncoding.Encode(buf, data)
	img.Content = vocab.DefaultNaturalLanguage(string(buf))
	SetItemID(img)
	return img
}

func getRandomLinkType() vocab.ActivityVocabularyType {
	return vocab.LinkTypes[rand.Intn(len(vocab.LinkTypes))]
}

func getRandomName() vocab.NaturalLanguageValues {
	return vocab.DefaultNaturalLanguage(names.GetRandom())
}

func getRandomHref() vocab.IRI {
	return DefaultHost.AddPath(filepath.Join(strings.Split(names.GetRandom(), "_")...))
}

func RandomLink(_ vocab.LinkOrIRI) vocab.Item {
	l := new(vocab.Link)
	l.Type = getRandomLinkType()
	l.Name = getRandomName()
	l.Href = getRandomHref()
	l.HrefLang = vocab.DefaultLang

	SetItemID(l)
	return l
}

func CreateActivity(ob vocab.LinkOrIRI, attrTo vocab.LinkOrIRI) vocab.Item {
	act := new(vocab.Activity)
	act.Type = vocab.CreateType
	act.AttributedTo = attrTo.GetLink()
	act.Actor = attrTo.GetLink()
	act.To = vocab.ItemCollection{RootID, vocab.PublicNS}
	act.Published = getRandomTime()

	if it, ok := ob.(vocab.Item); ok {
		act.Object = it
	}

	SetItemID(act)
	return act
}

var nonActivityGenFns = []generatorFn{
	RandomObject,
	RandomPlace,
	RandomProfile,
	RandomActor,
	RandomLink,
	RandomTag,
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
		activities = append(activities, RandomNonNonContentActivity(ob, act))
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
