package conformance

import (
	"testing"

	vocab "github.com/go-ap/activitypub"
)

type MetadataStorage interface {
	LoadMetadata(iri vocab.IRI, m any) error
	SaveMetadata(iri vocab.IRI, m any) error
}

func (s Suite) RunMetadataTests(t *testing.T) {
	t.Errorf("%s", errNotImplemented)
}
