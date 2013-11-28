package kontrol

import (
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"io/ioutil"
	"koding/db/mongodb/modelhelper"
	"koding/newkite/kite"
	"koding/newkite/protocol"
	"labix.org/v2/mgo/bson"
	"testing"
	"time"
)

func setupTest(t *testing.T) {
	kodingKey := modelhelper.NewKodingKeys()
	kodingKey.Id = bson.ObjectIdHex("528e1b36a819430000000001")
	kodingKey.Key = "FmJpq9nof261Fa50GBb4x1naxQbB9sToQ9cSQ5RUsFgxgH0R9-DHxPwfhpXRe5PM"
	kodingKey.Owner = "5196fcb0bc9bdb0000000011"
	kodingKey.Hostname = "tardis.local-39612b01-b08b-4df7-49f7-641e58541459"

	err := modelhelper.AddKodingKeys(kodingKey)
	if err != nil {
		t.Errorf("Cannot add Koding Key to MongoDB: %s", err.Error())
		return
	}

	err = ioutil.WriteFile("/home/vagrant/.kd/koding.key", []byte(kodingKey.Key), 0644)
	if err != nil {
		t.Errorf("Cannot write Koding Key to disk: %s", err.Error())
		return
	}

	etcdClient := etcd.NewClient(nil)
	_, err = etcdClient.DeleteAll("/kites/devrim")
	if err != nil {
		if err.(etcd.EtcdError).ErrorCode != 100 { // Key Not Found
			t.Errorf("Cannot delete keys from etcd: %s", err)
			return
		}
	}
}

func TestKontrol(t *testing.T) {
	setupTest(t)

	kon := New()
	kon.Start()

	mathKite := mathWorker()
	mathKite.Start()

	exp2Kite := exp2()
	exp2Kite.Start()

	// Wait for kites to register themselves on Kontrol.
	time.Sleep(500 * time.Millisecond)

	query := protocol.KontrolQuery{
		Username:    "devrim",
		Environment: "development",
		Name:        "mathworker",
	}

	kites, err := exp2Kite.Kontrol.GetKites(query, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(kites) == 0 {
		t.Errorf("No mathworker available")
		return
	}

	mathWorker := kites[0]
	err = mathWorker.Dial()
	if err != nil {
		t.Errorf("Cannot connect to remote mathworker")
		return
	}

	response, err := mathWorker.Call("square", 2)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	var result int
	err = response.Unmarshal(&result)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if result != 4 {
		t.Errorf("Invalid result: %d", result)
		return
	}
}

func mathWorker() *kite.Kite {
	options := &protocol.Options{
		Kitename:    "mathworker",
		Version:     "1",
		Port:        "3636",
		Region:      "localhost",
		Environment: "development",
	}

	k := kite.New(options)
	k.HandleFunc("square", Square)
	return k
}

func Square(r *kite.Request) (interface{}, error) {
	a, err := r.Args.Float64()
	if err != nil {
		return nil, err
	}

	result := a * a

	fmt.Printf("Kite call, sending result '%f' back\n", result)

	// Reverse method call
	r.RemoteKite.Go("foo", "bar")

	return result, nil
}

func exp2() *kite.Kite {
	options := &protocol.Options{
		Kitename:    "exp2",
		Version:     "1",
		Port:        "3637",
		Region:      "localhost",
		Environment: "development",
	}

	return kite.New(options)
}