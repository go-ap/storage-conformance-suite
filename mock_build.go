package conformance

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
	"github.com/go-ap/storage-conformance-suite/internal"
)

func getRandomContent() []byte {
	validArray := make([][]byte, 0)
	for _, files := range internal.ContentMap {
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

func sortItemCollectionByID(items vocab.ItemCollection) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetLink().String() <= items[j].GetLink().String()
	})
}

func getRandomItemCollection(count int) vocab.ItemCollection {
	items := make(vocab.ItemCollection, 0, count)
	for range count {
		items = append(items, RandomObject(root))
	}
	sortItemCollectionByID(items)
	return items
}
