package internal

import (
	"embed"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	ap "github.com/go-ap/activitypub"
	"golang.org/x/text/language"
)

//go:embed data
var contentFS embed.FS

func buildContentMap() map[string]ap.NaturalLanguageValues {
	files := make(map[string]ap.NaturalLanguageValues)
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

			nlv, ok := files[contentType]
			if !ok {
				nlv = make(ap.NaturalLanguageValues)
			}
			lr := ap.NilLangRef
			if strings.HasPrefix(contentType, "text") {
				lr = ap.DefaultLang
				if langIdx := strings.Index(path, "_"); langIdx > 0 {
					lang := path[langIdx+1 : strings.Index(path, ".")]
					if tag, err := language.Parse(lang); err == nil {
						lr = ap.LangRef(tag)
					} else {
						lr = ap.NilLangRef
					}
				}
			}
			_ = nlv.Append(lr, data)
			files[contentType] = nlv
		}
		return nil
	})
	return files
}

var ContentMap = buildContentMap()
