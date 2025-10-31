package conformance

import (
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-conformance-suite/internal"
	"github.com/google/go-cmp/cmp"
)

type MetadataStorage interface {
	LoadMetadata(iri vocab.IRI, m any) error
	SaveMetadata(iri vocab.IRI, m any) error
}

type PassAndKeyMetadata struct {
	Pw  string            `json:"pw"`
	Key crypto.PrivateKey `json:"key"`
}

func decPrv(raw []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errf("unable to PEM decode data")
	}

	if prv, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return prv, nil
	}
	if prv, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return prv, nil
	}
	if prv, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return prv, nil
	}

	return nil, errf("unable to parse private key")
}

func encPrv(key crypto.PrivateKey) ([]byte, error) {
	var raw []byte
	var err error
	valid := true
	switch prv := key.(type) {
	case *ecdsa.PrivateKey:
		raw, err = x509.MarshalECPrivateKey(prv)
	case *rsa.PrivateKey:
		raw = x509.MarshalPKCS1PrivateKey(prv)
	case *dsa.PrivateKey:
		raw, err = x509.MarshalPKCS8PrivateKey(prv)
	case ed25519.PrivateKey:
		raw, err = x509.MarshalPKCS8PrivateKey(prv)
	default:
		valid = false
	}
	if !valid {
		return nil, errf("unknown %T type for private key", key)
	}
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: raw,
	}), nil
}

func (p *PassAndKeyMetadata) UnmarshalJSON(raw []byte) error {
	mm := new(map[string]string)
	if err := json.Unmarshal(raw, &mm); err != nil {
		return err
	}
	if pw, ok := (*mm)["pw"]; ok {
		p.Pw = pw
	}
	if rk, ok := (*mm)["key"]; ok {
		pk, err := decPrv([]byte(rk))
		if err != nil {
			return err
		}
		p.Key = pk
	}
	return nil
}
func (p PassAndKeyMetadata) MarshalJSON() ([]byte, error) {
	ss := strings.Builder{}
	ss.WriteRune('{')
	if p.Pw != "" {
		pw, err := json.Marshal(p.Pw)
		if err != nil {
			return nil, err
		}
		ss.WriteString(`"pw":`)
		ss.Write(pw)
		ss.WriteRune(',')
	}
	if p.Key != nil {
		if raw, err := encPrv(p.Key); err == nil {
			ss.WriteString(`"key":`)
			if kk, err := json.Marshal(string(raw)); err == nil {
				ss.Write(kk)
			}
		}
	}
	ss.WriteRune('}')
	return []byte(ss.String()), nil
}

var validPwChars = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+[]\{}|;':",./<>?`

func getRandomPw() string {
	ss := strings.Builder{}
	for range 8 + rand.IntN(32) {
		ss.WriteRune(rune(validPwChars[rand.IntN(len(validPwChars))]))
	}
	return ss.String()
}

func buildMetaData() []json.Marshaler {
	result := make([]json.Marshaler, 0)
	result = append(result, PassAndKeyMetadata{
		Pw:  getRandomPw(),
		Key: getPrivateKey(),
	})
	//result = append(result, )
	//result = append(result, 123.6666)
	//result = append(result, "Lorem ipsum dolor sic amet")
	return result
}

func RunMetadataTests(t *testing.T, storage ActivityPubStorage) {
	mStorage, ok := storage.(MetadataStorage)
	if !ok {
		t.Skipf("storage %T is not compatible with MetaData functionality", storage)
	}

	{
		toSave := PassAndKeyMetadata{
			Pw:  getRandomPw(),
			Key: getPrivateKey(),
		}
		t.Run(fmt.Sprintf("store %T", toSave), func(t *testing.T) {
			if err := mStorage.SaveMetadata(internal.RootID, toSave); err != nil {
				t.Errorf("unable to save Metadata %T: %s", toSave, err)
			}

			loadInto := PassAndKeyMetadata{}
			if err := mStorage.LoadMetadata(internal.RootID, &loadInto); err != nil {
				t.Errorf("unable to load metadata %T for iri %s: %s", toSave, internal.RootID, err)
			}
			if !cmp.Equal(toSave, loadInto) {
				t.Errorf("loaded metadata is not equal: %s", cmp.Diff(toSave, loadInto))
			}
		})
	}
	{
		toStore := struct{ A string }{A: "lorem ipsum, dolor sic amet"}
		t.Run(fmt.Sprintf("store %T", toStore), func(t *testing.T) {
			if err := mStorage.SaveMetadata(internal.RootID, toStore); err != nil {
				t.Errorf("unable to save Metadata %T: %s", toStore, err)
			}

			loadInto := struct{ A string }{}
			if err := mStorage.LoadMetadata(internal.RootID, &loadInto); err != nil {
				t.Errorf("unable to load metadata %T for iri %s: %s", toStore, internal.RootID, err)
			}
			if !cmp.Equal(toStore, loadInto) {
				t.Errorf("loaded metadata is not equal: %s", cmp.Diff(toStore, loadInto))
			}
		})
	}
	{
		toStore := 6666
		t.Run(fmt.Sprintf("store %T", toStore), func(t *testing.T) {
			if err := mStorage.SaveMetadata(internal.RootID, toStore); err != nil {
				t.Errorf("unable to save Metadata %T: %s", toStore, err)
			}

			var loadInto int
			if err := mStorage.LoadMetadata(internal.RootID, &loadInto); err != nil {
				t.Errorf("unable to load metadata %T for iri %s: %s", toStore, internal.RootID, err)
			}
			if !cmp.Equal(toStore, loadInto) {
				t.Errorf("loaded metadata is not equal: %s", cmp.Diff(toStore, loadInto))
			}
		})
	}
	{
		toStore := 0.1111111
		t.Run(fmt.Sprintf("store %T", toStore), func(t *testing.T) {
			if err := mStorage.SaveMetadata(internal.RootID, toStore); err != nil {
				t.Errorf("unable to save Metadata %T: %s", toStore, err)
			}

			var loadInto float64
			if err := mStorage.LoadMetadata(internal.RootID, &loadInto); err != nil {
				t.Errorf("unable to load metadata %T for iri %s: %s", toStore, internal.RootID, err)
			}
			if !cmp.Equal(toStore, loadInto) {
				t.Errorf("loaded metadata is not equal: %s", cmp.Diff(toStore, loadInto))
			}
		})
	}
	{
		toStore := []byte("Lorem ipsum dolor sic amet")
		t.Run(fmt.Sprintf("store %T", toStore), func(t *testing.T) {
			if err := mStorage.SaveMetadata(internal.RootID, toStore); err != nil {
				t.Errorf("unable to save Metadata %T: %s", toStore, err)
			}

			var loadInto []byte
			if err := mStorage.LoadMetadata(internal.RootID, &loadInto); err != nil {
				t.Errorf("unable to load metadata %T for iri %s: %s", toStore, internal.RootID, err)
			}
			if !cmp.Equal(toStore, loadInto) {
				t.Errorf("loaded metadata is not equal: %s", cmp.Diff(toStore, loadInto))
			}
		})
	}
	{
		toStore := "Lorem ipsum dolor sic amet"
		t.Run(fmt.Sprintf("store %T", toStore), func(t *testing.T) {
			if err := mStorage.SaveMetadata(internal.RootID, toStore); err != nil {
				t.Errorf("unable to save Metadata %T: %s", toStore, err)
			}

			var loadInto string
			if err := mStorage.LoadMetadata(internal.RootID, &loadInto); err != nil {
				t.Errorf("unable to load metadata %T for iri %s: %s", toStore, internal.RootID, err)
			}
			if !cmp.Equal(toStore, loadInto) {
				t.Errorf("loaded metadata is not equal: %s", cmp.Diff(toStore, loadInto))
			}
		})
	}
}
