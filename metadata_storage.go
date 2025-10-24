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
	t.Skipf("%s", errNotImplemented)
}
