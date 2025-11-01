package conformance

import (
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
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

var (
	now = time.Now()

	someDate = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Minute(), 0, time.UTC)
)

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
				t.Errorf("the received error does not match NotFound: %s", err)
			}
			if loaded != nil {
				t.Errorf("we shouldn't be able to load deleted client, received %s", cmp.Diff(nil, loaded))
			}
		})
	})
	t.Run("authorize operations", func(t *testing.T) {
		client := osin.DefaultClient{
			Id:          "test",
			Secret:      "asd",
			RedirectUri: "http://127.0.0.1",
			UserData:    "https://example.com/~jdoe",
		}
		if saver, ok := oStorage.(ClientSaver); ok {
			_ = saver.CreateClient(&client)
		}

		auth := osin.AuthorizeData{
			Client:      &client,
			Code:        "xx44aa1!",
			ExpiresIn:   int32(time.Hour.Seconds()),
			Scope:       "none",
			RedirectUri: "http://127.0.0.1",
			State:       "no-state",
			CreatedAt:   someDate,
			UserData:    vocab.IRI("https://example.com/~johndoe"),
		}
		t.Run("save authorize", func(t *testing.T) {
			if err := oStorage.SaveAuthorize(&auth); err != nil {
				t.Errorf("unable to save authorize data: %s", err)
			}
		})
		t.Run("load authorize", func(t *testing.T) {
			loaded, err := oStorage.LoadAuthorize(auth.Code)
			if err != nil {
				t.Errorf("unable to load authorize data: %s", err)
			}
			if !cmp.Equal(&auth, loaded) {
				t.Errorf("invalid authorize data returned from loading %s", cmp.Diff(&auth, loaded))
			}
		})
		t.Run("remove authorize", func(t *testing.T) {
			err := oStorage.RemoveAuthorize(auth.Code)
			if err != nil {
				t.Errorf("unable to load authorize data: %s", err)
			}
			loaded, err := oStorage.LoadAuthorize(auth.Code)
			if !errors.IsNotFound(err) {
				t.Errorf("the received error does not match NotFound: %s", err)
			}
			if loaded != nil {
				t.Errorf("we shouldn't be able to load deleted authorize data, received %s", cmp.Diff(nil, loaded))
			}
		})
	})
	t.Run("access operations", func(t *testing.T) {
		client := osin.DefaultClient{
			Id:          "test",
			Secret:      "asd",
			RedirectUri: "http://127.0.0.1",
			UserData:    "https://example.com/~jdoe",
		}
		if saver, ok := oStorage.(ClientSaver); ok {
			_ = saver.CreateClient(&client)
		}

		auth := osin.AuthorizeData{
			Client:      &client,
			Code:        "xx44aa1!",
			ExpiresIn:   int32(time.Hour.Seconds()),
			Scope:       "none",
			RedirectUri: "http://127.0.0.1",
			State:       "no-state",
			CreatedAt:   someDate,
			UserData:    vocab.IRI("https://example.com/~johndoe"),
		}
		_ = oStorage.SaveAuthorize(&auth)
		access := osin.AccessData{
			Client:        &client,
			AuthorizeData: &auth,
			AccessToken:   "f00b4r",
			RefreshToken:  "s0fresh",
			ExpiresIn:     int32(time.Hour.Seconds()),
			Scope:         "none",
			RedirectUri:   "http://127.0.0.1",
			CreatedAt:     someDate,
			UserData:      "https://example.com/~johndoe",
		}
		t.Run("save access data", func(t *testing.T) {
			if err := oStorage.SaveAccess(&access); err != nil {
				t.Errorf("unable to save access data: %s", err)
			}
		})
		t.Run("load access data", func(t *testing.T) {
			loaded, err := oStorage.LoadAccess(access.AccessToken)
			if err != nil {
				t.Errorf("unable to load access data: %s", err)
			}
			if !cmp.Equal(&access, loaded) {
				t.Errorf("invalid access data returned from loading %s", cmp.Diff(&access, loaded))
			}
		})
		t.Run("remove access data", func(t *testing.T) {
			if err := oStorage.RemoveAccess(access.AccessToken); err != nil {
				t.Errorf("unable to remove access data: %s", err)
			}
			loaded, err := oStorage.LoadAccess(access.AccessToken)
			if !errors.IsNotFound(err) {
				t.Errorf("the received error does not match NotFound: %s", err)
			}
			if loaded != nil {
				t.Errorf("we shouldn't be able to load deleted access data, received %s", cmp.Diff(nil, loaded))
			}
		})
	})
	t.Run("refresh operations", func(t *testing.T) {
		client := osin.DefaultClient{
			Id:          "test",
			Secret:      "asd",
			RedirectUri: "http://127.0.0.1",
			UserData:    "https://example.com/~jdoe",
		}
		if saver, ok := oStorage.(ClientSaver); ok {
			_ = saver.CreateClient(&client)
		}

		auth := osin.AuthorizeData{
			Client:      &client,
			Code:        "xx44aa1!",
			ExpiresIn:   int32(time.Hour.Seconds()),
			Scope:       "none",
			RedirectUri: "http://127.0.0.1",
			State:       "no-state",
			CreatedAt:   someDate,
			UserData:    vocab.IRI("https://example.com/~johndoe"),
		}
		_ = oStorage.SaveAuthorize(&auth)
		access := osin.AccessData{
			Client:        &client,
			AuthorizeData: &auth,
			AccessToken:   "f00b4r",
			RefreshToken:  "s0fresh",
			ExpiresIn:     int32(time.Hour.Seconds()),
			Scope:         "none",
			RedirectUri:   "http://127.0.0.1",
			CreatedAt:     someDate,
			UserData:      "https://example.com/~johndoe",
		}
		_ = oStorage.SaveAccess(&access)
		t.Run("load refresh data", func(t *testing.T) {
			loaded, err := oStorage.LoadRefresh(access.RefreshToken)
			if err != nil {
				t.Errorf("unable to load refresh data: %s", err)
			}
			if !cmp.Equal(&access, loaded) {
				t.Errorf("invalid refresh access data loaded %s", cmp.Diff(&access, loaded))
			}
		})
		t.Run("remove refresh data", func(t *testing.T) {
			if err := oStorage.RemoveRefresh(access.RefreshToken); err != nil {
				t.Errorf("unable to remove refresh data: %s", err)
			}
			loaded, err := oStorage.LoadRefresh(access.RefreshToken)
			if !errors.IsNotFound(err) {
				t.Errorf("the received error does not match NotFound: %s", err)
			}
			if loaded != nil {
				t.Errorf("we shouldn't be able to load deleted refresh data, received %s", cmp.Diff(nil, loaded))
			}
		})
	})
}
