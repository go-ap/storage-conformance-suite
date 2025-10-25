package conformance

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"mime"
	"net/http"
	"strings"
	"time"

	ap "github.com/go-ap/activitypub"
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

func RandomObject(attrTo ap.Item) ap.Item {
	ob := new(ap.Object)
	ob.Type = ap.NoteType
	ob.AttributedTo = attrTo
	// NOTE(marius): we use random time, instead of something like time.Now()
	// because the later contains monotonic information which gets lost at loading form the mock storage we're using
	ob.Published = getRandomTime()

	ob.Content = ap.DefaultNaturalLanguage("no data")
	if data := getRandomContent(); len(data) > 0 {
		typ, mt := getObjectTypes(data)
		ob.Type = typ
		ob.MediaType = mt

		if !strings.Contains(string(mt), "text") {
			buf := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
			base64.RawStdEncoding.Encode(buf, data)
			data = buf
		} else {
			ob.Summary = ap.DefaultNaturalLanguage(string(data[:bytes.Index(data, []byte{'.'})]))
		}
		ob.Content = ap.DefaultNaturalLanguage(string(data))
	}

	return ob
}

var svgDocumentStart = []byte{'<', 's', 'v', 'g'}

func getObjectTypes(data []byte) (ap.ActivityVocabularyType, ap.MimeType) {
	contentType := http.DetectContentType(data)
	var objectType ap.ActivityVocabularyType

	contentType, _, _ = mime.ParseMediaType(contentType)
	switch contentType {
	case "text/html", "text/markdown", "text/plain":
		objectType = ap.NoteType
		if len(data) > 600 {
			objectType = ap.ArticleType
		}
		if bytes.Contains(data, svgDocumentStart) {
			objectType = ap.DocumentType
			contentType = "image/svg+xml"
		}
	case "image/svg+xml":
		objectType = ap.DocumentType
	case "video/webm":
		fallthrough
	case "video/mp4":
		objectType = ap.VideoType
	case "audio/mp3":
		objectType = ap.AudioType
	case "image/png":
		fallthrough
	case "image/jpg":
		objectType = ap.ImageType
	}
	return objectType, ap.MimeType(contentType)
}
