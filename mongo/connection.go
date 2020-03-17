package mongo

import (
	"time"

	"github.com/globalsign/mgo"

	"github.com/seniorGolang/gokit/logger"
	"github.com/seniorGolang/gokit/utils"
)

const Default = "default"

var log = logger.Log.WithField("module", "mongo")

var pool = make(map[string]*mongoConnection)

type mongoConnection struct {
	session  *mgo.Session
	dialInfo *mgo.DialInfo
}

func (mc *mongoConnection) Session(collectionName string) (session *mgo.Session, collection *mgo.Collection) {

	session = mc.session.Clone()
	collection = session.DB(mc.dialInfo.Database).C(collectionName)
	return
}

func (mc *mongoConnection) ConnSession() *mgo.Session {
	return mc.session
}

func (mc *mongoConnection) DialInfo() *mgo.DialInfo {
	return mc.dialInfo
}

func (mc *mongoConnection) Addresses() []string {
	return mc.dialInfo.Addrs
}

func Connect(mongoAddr string, nameArg ...string) *mongoConnection {

	name := Default

	if len(nameArg) > 0 {
		name = nameArg[0]
	}

	if mc, found := pool[name]; found {
		return mc
	}

	dialInfo, err := mgo.ParseURL(mongoAddr)
	utils.ExitOnError(log, err, "Could not parse url: "+mongoAddr)
	dialInfo.Timeout = 10 * time.Second

	mgoSession, err := mgo.DialWithInfo(dialInfo)
	utils.ExitOnError(log, err, "Could not create mongo session")

	mgoSession.SetMode(mgo.Monotonic, true)
	pool[name] = &mongoConnection{dialInfo: dialInfo, session: mgoSession}

	return pool[name]
}

func DB(nameArg ...string) *mongoConnection {

	name := Default

	if len(nameArg) > 0 {
		name = nameArg[0]
	}

	if mc, found := pool[name]; found {
		return mc
	}
	return nil
}
