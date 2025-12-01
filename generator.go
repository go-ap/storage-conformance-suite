package conformance

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-conformance-suite/internal"
)

func RandomLink(attrTo vocab.Item) vocab.Item {
	return internal.RandomLink(attrTo)
}

func RandomTag(parent vocab.Item) vocab.Item {
	return internal.RandomTag(parent)
}

func RandomObject(parent vocab.Item) vocab.Item {
	return internal.RandomObject(parent)
}

func RandomImage(mimeType vocab.MimeType, parent vocab.Item) vocab.Item {
	return internal.RandomImage(mimeType, parent)
}

func RandomActor(parent vocab.Item) vocab.Item {
	return internal.RandomActor(parent)
}

func RandomActivity(withObject, parent vocab.Item) vocab.Item {
	return internal.RandomActivity(withObject, parent)
}

func RandomCollection(cnt int) vocab.ItemCollection {
	return internal.RandomItemCollection(cnt)
}
