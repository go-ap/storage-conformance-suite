package internal

import (
	"bytes"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	ap "github.com/go-ap/activitypub"
	"golang.org/x/text/language"
)

func buildContentMapTest() (map[string]string, map[string]ap.LangRef, map[string][]byte) {
	contentTypes := make(map[string]string)
	langRefs := make(map[string]ap.LangRef)
	content := make(map[string][]byte)
	_ = fs.WalkDir(contentFS, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d != nil && !d.IsDir() {
			ff, err := contentFS.Open(path)
			if err != nil {
				return err
			}
			defer ff.Close()

			data, err := io.ReadAll(ff)
			if err != nil {
				return err
			}
			contentType, _, _ := mime.ParseMediaType(http.DetectContentType(data))
			if filepath.Ext(path) == ".md" {
				contentType = "text/markdown"
			}
			contentTypes[path] = contentType
			content[path] = data

			if strings.HasPrefix(contentType, "text") {
				lr := ap.DefaultLang
				if langIdx := strings.Index(path, "_"); langIdx > 0 {
					lang := path[langIdx+1 : strings.Index(path, ".")]
					if tag, err := language.Parse(lang); err == nil {
						lr = ap.LangRef(tag)
					} else {
						lr = ap.NilLangRef
					}
				}
				langRefs[path] = lr
			}
		}
		return nil
	})
	return contentTypes, langRefs, content
}

func Test_buildContentMap(t *testing.T) {
	rawContentTypes, rawLangRefs, rawContent := buildContentMapTest()
	got := buildContentMap()

	if len(got) != len(rawLangRefs) {
		t.Errorf("invalid number of lang refs elements %d, expected %d", len(got), len(rawLangRefs))
	}

	for mt, nlv := range got {
		foundMt := false
		for _, m := range rawContentTypes {
			if m == mt {
				foundMt = true
				break
			}
		}
		if !foundMt {
			t.Errorf("unable to find mime-type %s", mt)
		}
		for lang, val := range nlv {
			foundLang := !strings.Contains(mt, "text")
			foundContent := false
			for _, l := range rawLangRefs {
				if l == lang {
					foundLang = true
					break
				}
			}
			for _, v := range rawContent {
				if bytes.Equal(v, val) {
					foundContent = true
					break
				}
			}
			if !foundLang {
				t.Errorf("unable to find language ref elements for %s", mt)
			}
			if !foundContent {
				t.Errorf("unable to find raw content for %s", mt)
			}
		}
	}
}
