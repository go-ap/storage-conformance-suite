package conformance

import (
	"testing"

	"github.com/openshift/osin"
)

type OSINStorage interface {
	Clone() osin.Storage
	Close()
	GetClient(id string) (osin.Client, error)
	SaveAuthorize(*osin.AuthorizeData) error
	LoadAuthorize(code string) (*osin.AuthorizeData, error)
	RemoveAuthorize(code string) error
	SaveAccess(*osin.AccessData) error
	LoadAccess(token string) (*osin.AccessData, error)
	RemoveAccess(token string) error
	LoadRefresh(token string) (*osin.AccessData, error)
	RemoveRefresh(token string) error
}

type ClientSaver interface {
	UpdateClient(c osin.Client) error
	CreateClient(c osin.Client) error
	RemoveClient(id string) error
}

type ClientLister interface {
	ListClients() ([]osin.Client, error)
	GetClient(id string) (osin.Client, error)
}

func (s Suite) RunOAuthTests(t *testing.T) {
	t.Errorf("%s", errNotImplemented)
}
