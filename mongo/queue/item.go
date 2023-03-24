package queue

import (
	"encoding/json"
	"time"

	"github.com/seniorGolang/gokit/types/uuid"
)

type queueItem struct {
	Id         uuid.UUID       `bson:"_id"`
	Priority   int             `bson:"priority"`
	AutoRemove bool            `bson:"autoRemove"`
	Created    time.Time       `bson:"created"`
	Expire     *time.Time      `json:"expire"`
	Executed   *time.Time      `bson:"executed"`
	Relevant   *time.Time      `bson:"relevant"`
	Payload    json.RawMessage `bson:"payload"`
}
