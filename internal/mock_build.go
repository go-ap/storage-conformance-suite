package internal

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-conformance-suite/internal/names"
)

var (
	defaultTime = time.Date(1999, time.April, 1, 6, 6, 6, 0, time.UTC)

	RootID = vocab.IRI("https://example.com/~root")

	publicAudience = vocab.ItemCollection{vocab.PublicNS}

	Root = &vocab.Actor{
		ID:                RootID,
		Type:              vocab.PersonType,
		Published:         defaultTime,
		Name:              vocab.DefaultNaturalLanguage("Rooty McRootface"),
		Summary:           vocab.DefaultNaturalLanguage("The base actor for the conformance test suite"),
		Content:           vocab.DefaultNaturalLanguage("<p>The base actor for the conformance test suite</p>"),
		URL:               vocab.Item(RootID),
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
)

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

func getRandomTime() time.Time {
	year := int(1900 + rand.Int31n(199))
	month := time.Month(rand.Int31n(12) + 1)
	day := int(rand.Int31n(30))
	hour := int(rand.Int31n(24))
	minute := int(rand.Int31n(59))
	second := int(rand.Int31n(59))
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
}

var typeCountMap = make(map[string]int)

func SetID(it vocab.Item) {
	_ = vocab.OnObject(it, func(ob *vocab.Object) error {
		isCollection := it.IsCollection()
		pieces := make([]string, 0)
		base := "https://example.com"
		if !vocab.IsNil(ob.AttributedTo) {
			base = ob.AttributedTo.GetLink().String()
		}
		pieces = append(pieces, base)
		if isCollection {
			typ := strings.ToLower(string(ob.Type))
			pieces = append(pieces, typ)
		} else {
			typ := strings.ToLower(string(ob.Type))
			cnt, _ := typeCountMap[typ]
			cnt++
			typeCountMap[typ] = cnt
			pieces = append(pieces, typ, strconv.Itoa(cnt))
		}
		ob.ID = vocab.IRI(filepath.Join(pieces...))
		return nil
	})
}

func RandomCollection(attrTo vocab.Item) vocab.CollectionInterface {
	col := new(vocab.OrderedCollection)
	col.Type = vocab.OrderedCollectionType
	col.AttributedTo = attrTo
	col.Published = getRandomTime()
	SetID(col)

	return col
}

type generatorFn func(vocab.Item) vocab.Item

func RandomItem(attrTo vocab.Item) vocab.Item {
	genFns := []generatorFn{
		RandomObject,
		RandomActor,
		func(attrTo vocab.Item) vocab.Item {
			return RandomActivity(RandomObject(attrTo), attrTo)
		},
		RandomLink,
	}

	fn := genFns[rand.Intn(len(genFns))]
	return fn(attrTo)
}

func RandomObject(attrTo vocab.Item) vocab.Item {
	ob := new(vocab.Object)
	ob.Type = vocab.NoteType
	ob.AttributedTo = attrTo
	// NOTE(marius): we use random time, instead of something like time.Now()
	// because the later contains monotonic information which gets lost at loading form the mock storage we're using
	ob.Published = getRandomTime()

	ob.Content = vocab.DefaultNaturalLanguage("no data")
	if data := getRandomContent(); len(data) > 0 {
		typ, mt := getObjectTypes(data)
		ob.Type = typ
		ob.MediaType = mt

		if !strings.Contains(string(mt), "text") {
			buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
			base64.RawStdEncoding.Encode(buf, data)
			data = buf
		} else {
			ob.Summary = vocab.DefaultNaturalLanguage(string(data[:bytes.Index(data, []byte{'.'})]))
		}
		ob.Content = vocab.DefaultNaturalLanguage(string(data))
	}
	SetID(ob)

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
	case "video/webm", "video/mp4":
		objectType = vocab.VideoType
	case "audio/mp3":
		objectType = vocab.AudioType
	case "image/png", "image/jpg":
		objectType = vocab.ImageType
	}
	return objectType, vocab.MimeType(contentType)
}

func SortItemCollectionByID(items vocab.ItemCollection) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetLink().String() <= items[j].GetLink().String()
	})
}

func GetRandomItemCollection(count int) vocab.ItemCollection {
	items := make(vocab.ItemCollection, 0, count)
	for range count {
		items = append(items, RandomItem(Root))
	}
	SortItemCollectionByID(items)
	return items
}

func getRandomReason() string {
	return "A random reason for a stupid activity"
}

var typesNeedReasons = vocab.ActivityVocabularyTypes{vocab.BlockType, vocab.FlagType, vocab.IgnoreType}

var validForActorActivityTypes = vocab.ActivityVocabularyTypes{
	vocab.UpdateType,
	vocab.LikeType,
	vocab.DislikeType,
	vocab.FlagType,
	vocab.BlockType,
	vocab.FollowType,
	vocab.IgnoreType,
}

var validForObjectActivityTypes = vocab.ActivityVocabularyTypes{
	vocab.UpdateType,
	vocab.LikeType,
	vocab.DislikeType,
	vocab.DeleteType,
	vocab.FlagType,
	vocab.BlockType,
	vocab.FollowType,
	vocab.IgnoreType,
}

var validForActivityActivityTypes = vocab.ActivityVocabularyTypes{
	vocab.UndoType,
}

var validActivityTypes = append(validForObjectActivityTypes[:], validForActivityActivityTypes[:]...)

func getActivityTypeByObject(ob vocab.Item) vocab.ActivityVocabularyType {
	if vocab.IsNil(ob) {
		return validForObjectActivityTypes[rand.Int()%len(validForObjectActivityTypes)]
	}
	if vocab.ActivityTypes.Contains(ob.GetType()) {
		return validForActivityActivityTypes[rand.Int()%len(validForActivityActivityTypes)]
	}
	if vocab.ActorTypes.Contains(ob.GetType()) {
		return validForActorActivityTypes[rand.Int()%len(validForActorActivityTypes)]
	}
	return validForObjectActivityTypes[rand.Int()%len(validForObjectActivityTypes)]
}

func RandomActivity(ob vocab.Item, attrTo vocab.Item) *vocab.Activity {
	act := new(vocab.Activity)
	act.Type = getActivityTypeByObject(ob)
	if ob != nil {
		act.Object = ob
	}
	act.AttributedTo = attrTo
	act.Actor = attrTo
	act.To = vocab.ItemCollection{RootID, vocab.PublicNS}

	if typesNeedReasons.Contains(act.Type) {
		act.Content = vocab.DefaultNaturalLanguage(getRandomReason())
		act.Summary = vocab.DefaultNaturalLanguage(getRandomReason())
	}
	SetID(act)

	return act
}

func getRandomActorType() vocab.ActivityVocabularyType {
	return vocab.ActorTypes[rand.Intn(len(vocab.ActorTypes))]
}

func RandomActor(attrTo vocab.Item) vocab.Item {
	act := new(vocab.Actor)
	act.Name = vocab.DefaultNaturalLanguage(names.GetRandom())
	act.PreferredUsername = act.Name
	act.Type = getRandomActorType()
	act.AttributedTo = attrTo
	act.Icon = RandomImage("image/png", attrTo.GetLink())
	SetID(act)
	return act
}

func getRandomContentByMimeType(mimeType vocab.MimeType) []byte {
	if validArray, ok := ContentMap[string(mimeType)]; ok {
		return validArray.First()
	}
	return nil
}

func RandomImage(mime vocab.MimeType, parent vocab.Item) vocab.Item {
	img := new(vocab.Image)
	img.Type = vocab.ImageType
	img.MediaType = mime
	img.AttributedTo = parent

	data := getRandomContentByMimeType(mime)
	buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
	base64.RawStdEncoding.Encode(buf, data)
	img.Content = vocab.DefaultNaturalLanguage(string(buf))
	SetID(img)
	return img
}

func getRandomLinkType() vocab.ActivityVocabularyType {
	return vocab.LinkTypes[rand.Intn(len(vocab.LinkTypes))]
}

func getRandomName() vocab.NaturalLanguageValues {
	return vocab.DefaultNaturalLanguage(names.GetRandom())
}

func getRandomHref() vocab.IRI {
	return vocab.IRI("https://example.com").AddPath(filepath.Join(strings.Split(names.GetRandom(), "_")...))
}

func RandomLink(attrTo vocab.Item) vocab.Item {
	ob := new(vocab.Link)
	ob.Type = getRandomLinkType()
	ob.Name = getRandomName()
	ob.Href = getRandomHref()
	ob.HrefLang = vocab.DefaultLang
	ob.ID = ob.Href

	return ob
}
