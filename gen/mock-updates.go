package gen

import vocab "github.com/go-ap/activitypub"

func UpdateItem(it vocab.Item) vocab.Item {
	if vocab.IsNil(it) {
		return nil
	}
	nw := vocab.Clone(it)
	typ := nw.GetType()
	switch {
	case vocab.ActivityTypes.Match(typ):
		_ = vocab.OnActivity(nw, updateRandomActivityProperties)
	case vocab.QuestionType.Match(typ):
		_ = vocab.OnQuestion(nw, updateRandomQuestionProperties)
	case vocab.IntransitiveActivityTypes.Match(typ):
		_ = vocab.OnIntransitiveActivity(nw, updateRandomIntransitiveActivityProperties)
	case vocab.ActorTypes.Match(typ):
		_ = vocab.OnActor(nw, updateRandomActorProperties)
	case vocab.PlaceType.Match(typ):
		_ = vocab.OnPlace(nw, updateRandomPlaceProperties)
	case vocab.RelationshipType.Match(typ):
		_ = vocab.OnRelationship(nw, updateRandomRelationshipProperties)
	case vocab.TombstoneType.Match(typ):
		_ = vocab.OnTombstone(nw, updateRandomTombstoneProperties)
	case vocab.ProfileType.Match(typ):
		_ = vocab.OnProfile(nw, updateRandomProfileProperties)
	case vocab.LinkTypes.Match(typ):
		_ = vocab.OnLink(nw, updateRandomLinkProperties)
	case typ == nil:
		fallthrough
	case vocab.ObjectTypes.Match(typ):
		fallthrough
	default:
		_ = vocab.OnObject(nw, updateRandomObjectProperties)
	}
	return nw
}

func updateRandomLinkProperties(l *vocab.Link) error {
	l.Name = getRandomName()
	return nil
}

func updateRandomProfileProperties(pr *vocab.Profile) error {
	return vocab.OnObject(pr, updateRandomObjectProperties)
}

func updateRandomTombstoneProperties(t *vocab.Tombstone) error {
	return vocab.OnObject(t, updateRandomObjectProperties)
}

func updateRandomPlaceProperties(pl *vocab.Place) error {
	return vocab.OnObject(pl, updateRandomObjectProperties)
}

func updateRandomRelationshipProperties(r *vocab.Relationship) error {
	return vocab.OnObject(r, updateRandomObjectProperties)
}

func updateRandomActorProperties(a *vocab.Actor) error {
	return vocab.OnObject(a, updateRandomObjectProperties)
}

func updateRandomActivityProperties(act *vocab.Activity) error {
	return vocab.OnIntransitiveActivity(act, updateRandomIntransitiveActivityProperties)
}

func updateRandomIntransitiveActivityProperties(act *vocab.IntransitiveActivity) error {
	return vocab.OnObject(act, updateRandomObjectProperties)
}

func updateRandomQuestionProperties(q *vocab.Question) error {
	return vocab.OnIntransitiveActivity(q, updateRandomIntransitiveActivityProperties)
}

func updateRandomObjectProperties(ob *vocab.Object) error {
	ob.Name = getRandomName()
	//ob.Summary = getRandomSummary()

	if ob.Type != nil {
		for _, typ := range ob.Type.AsTypes() {
			_ = setContentData(ob, getContentByType(typ))
			break
		}
	}

	ob.Published = getRandomTime()
	ob.Updated = ob.Published

	return nil
}
