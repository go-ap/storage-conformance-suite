package conformance

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-conformance-suite/gen"
)

func RandomLink(attrTo vocab.Item) vocab.Item {
	return gen.RandomLink(attrTo)
}

func RandomTag(parent vocab.Item) vocab.Item {
	return gen.RandomTag(parent)
}

func RandomObject(parent vocab.Item) vocab.Item {
	return gen.RandomObject(parent)
}

func RandomImage(mimeType vocab.MimeType, parent vocab.Item) vocab.Item {
	return gen.RandomImage(mimeType, parent)
}

func RandomActor(parent vocab.Item) vocab.Item {
	return gen.RandomActor(parent)
}

func RandomActivity(withObject, parent vocab.Item) vocab.Item {
	return gen.RandomActivity(withObject, parent)
}

func RandomCollection(cnt int) vocab.ItemCollection {
	return gen.RandomItemCollection(cnt)
}
