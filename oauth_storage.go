package conformance

import (
	"testing"

	"github.com/go-ap/errors"
	"github.com/google/go-cmp/cmp"
	"github.com/openshift/osin"
)

// OSINStorage is a verbatim copy of the [osin.Storage] interface
// We use this method instead of aliasing it, so it's more obvious
// what needs to be implemented.
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

func RunOAuthTests(t *testing.T, storage ActivityPubStorage) {
	oStorage, ok := storage.(OSINStorage)
	if !ok {
		t.Skipf("storage %T is not compatible with OAuth2 operations", storage)
	}

	t.Run("client operations", func(t *testing.T) {
		saver, ok := oStorage.(ClientSaver)
		if !ok {
			t.Skipf("storage %T is not compatible with OAuth2 client saver operations", storage)
		}
		client := osin.DefaultClient{
			Id:          "test",
			Secret:      "asd",
			RedirectUri: "http://127.0.0.1",
			UserData:    "https://example.com/~jdoe",
		}
		t.Run("create client", func(t *testing.T) {
			if err := saver.CreateClient(&client); err != nil {
				t.Errorf("unable to create client: %s", err)
			}
		})
		t.Run("get client", func(t *testing.T) {
			loaded, err := oStorage.GetClient(client.Id)
			if err != nil {
				t.Errorf("unable to load client: %s", err)
			}
			if !cmp.Equal(&client, loaded) {
				t.Errorf("invalid client returned from loading %s", cmp.Diff(&client, loaded))
			}
		})
		t.Run("list clients", func(t *testing.T) {
			loader := oStorage.(ClientLister)
			clients, err := loader.ListClients()
			if err != nil {
				t.Errorf("unable to list clients: %s", err)
			}
			if len(clients) != 1 {
				t.Fatalf("unexpected number of clients received %d, expected 1", len(clients))
			}
			if !cmp.Equal(&client, clients[0]) {
				t.Errorf("invalid client returned from loading %s", cmp.Diff(&client, clients[0]))
			}
		})
		t.Run("update client", func(t *testing.T) {
			toUpdate := *(&client)
			toUpdate.RedirectUri = "https://127.0.0.1"
			toUpdate.Secret = "dsa"
			toUpdate.UserData = "lorem ipsum dolor sic amet"
			if err := saver.UpdateClient(&toUpdate); err != nil {
				t.Errorf("unable to update client: %s", err)
			}
			loaded, err := oStorage.GetClient(client.Id)
			if err != nil {
				t.Errorf("unable to load client: %s", err)
			}
			if !cmp.Equal(&toUpdate, loaded) {
				t.Errorf("invalid client returned from loading %s", cmp.Diff(&toUpdate, loaded))
			}
		})
		t.Run("delete client", func(t *testing.T) {
			if err := saver.RemoveClient(client.Id); err != nil {
				t.Errorf("unable to remove client: %s", err)
			}
			loaded, err := oStorage.GetClient(client.Id)
			if !errors.IsNotFound(err) {
				t.Errorf("received error does not match NotFound: %s", err)
			}
			if loaded != nil {
				t.Errorf("we shouldn't be able to load deleted client, received %s", cmp.Diff(nil, loaded))
			}
		})
	})
	t.Run("authorize operations", func(t *testing.T) {
		t.Run("save authorize", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
		t.Run("load authorize", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
		t.Run("remove authorize", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
	})
	t.Run("access operations", func(t *testing.T) {
		t.Run("save access", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
		t.Run("load access", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
		t.Run("remove access", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
	})
	t.Run("refresh operations", func(t *testing.T) {
		t.Run("load refresh", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
		t.Run("remove refresh", func(t *testing.T) {
			t.Skipf("%s", errNotImplemented)
		})
	})
}
