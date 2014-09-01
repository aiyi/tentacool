package gateway

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/boltdb/bolt"
	"github.com/docker/libcontainer/netlink"
)

const (
	routesBucket = "routes"
	defaultKey   = "default"
)

var db *bolt.DB

type Gateway struct {
	IP   string `json:"ip"`
	Link string `json:"link"`
}

func PostGateway(w rest.ResponseWriter, req *rest.Request) {
	gateway := Gateway{}
	if err := req.DecodeJsonPayload(&gateway); err != nil {
		log.Printf(err.Error())
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db.Update(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(routesBucket))
		data, err := json.Marshal(gateway)
		if err != nil {
			return
		}
		err = b.Put([]byte(defaultKey), []byte(data))
		return
	})

	if err := netlink.AddDefaultGw(gateway.IP, gateway.Link); err != nil {
		log.Printf(err.Error())
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(gateway)
}

func DBinit(d *bolt.DB) (err error) {
	db = d
	err = db.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists([]byte(routesBucket))
		return
	})
	if err != nil {
		return err
	}

	log.Printf("Reinstall previous gateway from DB")
	err = db.View(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte(routesBucket))
		gateway := Gateway{}
		v := b.Get([]byte(defaultKey))
		if v != nil {
			if err := json.Unmarshal(v, &gateway); err != nil {
				log.Printf(err.Error())
			}
			if err := netlink.AddDefaultGw(gateway.IP, gateway.Link); err != nil {
				log.Printf(err.Error())
			}
		}
		return
	})
	return
}