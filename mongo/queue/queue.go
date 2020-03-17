package queue

import (
	"encoding/json"
	"io"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/seniorGolang/gokit/mongo"
	"github.com/seniorGolang/gokit/types/uuid"
)

type Queue struct {
	collection string
}

func New(collection string) (queue *Queue, err error) {

	queue = &Queue{collection: collection}

	sess, c := mongo.DB().Session(queue.collection)
	defer sess.Close()

	if err = c.EnsureIndex(mgo.Index{Key: []string{"expire"}, ExpireAfter: time.Second}); err != nil {
		return
	}

	if err = c.EnsureIndex(mgo.Index{Key: []string{"created"}}); err != nil {
		return
	}

	if err = c.EnsureIndex(mgo.Index{Key: []string{"priority"}}); err != nil {
		return
	}

	if err = c.EnsureIndex(mgo.Index{Key: []string{"expire"}}); err != nil {
		return
	}

	if err = c.EnsureIndex(mgo.Index{Key: []string{"relevant"}}); err != nil {
		return
	}

	if err = c.EnsureIndex(mgo.Index{Key: []string{"executed"}}); err != nil {
		return
	}

	if err = c.EnsureIndex(mgo.Index{Key: []string{"created", "-priority"}}); err != nil {
		return
	}

	if err = c.EnsureIndex(mgo.Index{Key: []string{"executed", "expire", "relevant"}}); err != nil {
		return
	}
	return
}

func (queue *Queue) Set(payload []byte, opts ...option) (err error) {

	sess, c := mongo.DB().Session(queue.collection)
	defer sess.Close()

	item := queueItem{Id: uuid.NewV4(), Payload: payload, Created: time.Now(), AutoRemove: true}

	for _, opt := range opts {
		opt(&item)
	}
	return c.Insert(item)
}

func (queue *Queue) Get() (payload []byte, err error) {

	sess, c := mongo.DB().Session(queue.collection)
	defer sess.Close()

	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"executed": time.Now()}},
		ReturnNew: true,
	}

	filter := bson.M{
		"$and": []bson.M{
			{"executed": nil},
			{
				"$or": []bson.M{
					{"expire": nil},
					{"expire": bson.M{"$gte": time.Now()}},
				},
			},
			{
				"$or": []bson.M{
					{"relevant": nil},
					{"relevant": bson.M{"$lte": time.Now()}},
				},
			},
		},
	}

	var item queueItem
	if _, err = c.Find(filter).Sort("created", "-priority").Apply(change, &item); err != nil {

		if err == mgo.ErrNotFound {
			err = io.EOF
		}
		return
	}

	if item.AutoRemove {
		err = queue.Drop(item.Id)
	}
	return item.Payload, err
}

func (queue *Queue) GetJSON(payload interface{}) (err error) {

	var payloadData []byte
	if payloadData, err = queue.Get(); err != nil {
		return
	}
	return json.Unmarshal(payloadData, payload)
}

func (queue *Queue) SetJSON(payload interface{}, opts ...option) (err error) {

	var payloadData json.RawMessage
	if payloadData, err = json.Marshal(payload); err != nil {
		return
	}
	return queue.Set(payloadData, opts...)
}

func (queue *Queue) Drop(id uuid.UUID) (err error) {

	sess, c := mongo.DB().Session(queue.collection)
	defer sess.Close()

	return c.RemoveId(id)
}
