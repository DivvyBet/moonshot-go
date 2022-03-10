package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func (c *Context) PingStore(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	// defer cancel()

	// conn, err := c.rs.Pool.GetContext(ctx)
	// defer conn.Close()

	//i, err := redis.String(conn.Do("PING"))
	pong, err := c.rs.Client.Ping().Result()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		c._log.Error("pinging memory store", err)
		return
	}

	w.Write([]byte(pong))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

}

func (c *Context) GetKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := ps.ByName("id")
	c._log.Info("getting key", key)
	result, err := c.rs.Client.Get(key).Result()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		c._log.Error("getting key", key, err)
		return
	}

	var output map[string]interface{}

	json.Unmarshal([]byte(result), &output)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	byt, _ := json.Marshal(output)
	w.Write(byt)

}

func (c *Context) CreateKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := ps.ByName("id")
	var val map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&val)
	if err != nil {
		c._log.Error("creating key", key, err)
	}

	byt, _ := json.Marshal(val)

	result, err := c.rs.Client.Set(key, string(byt), 36000*time.Hour).Result()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		c._log.Error("creating key", key, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	bytes, _ := json.Marshal(result)
	w.Write(bytes)
}

func (c *Context) DeleteKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := ps.ByName("id")
	result, err := c.rs.Client.Del(key).Result()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		c._log.Error("delete key", key, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	bytes, _ := json.Marshal(result)
	w.Write(bytes)
}
