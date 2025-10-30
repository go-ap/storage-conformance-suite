package conformance

import (
	"testing"

	vocab "github.com/go-ap/activitypub"
)

type MetadataStorage interface {
	LoadMetadata(iri vocab.IRI, m any) error
	SaveMetadata(iri vocab.IRI, m any) error
}

func RunMetadataTests(t *testing.T, storage ActivityPubStorage) {
	_, ok := storage.(MetadataStorage)
	if !ok {
		t.Skipf("storage %T is not compatible with MetaData functionality", storage)
	}
}
