//: Copyright Verizon Media
//: Licensed under the terms of the Apache 2.0 License. See LICENSE file in the project root for terms.

package vssh

import (
	"hash/fnv"
	"sync"
)

var clientsShardNum = 10

type clients []*clientsShard
type clientsShard struct {
	clients map[string]*clientAttr
	sync.RWMutex
}

func (c *clients) getShard(key string) uint {
	hash := fnv.New32()
	hash.Write([]byte(key))
	hSum32 := hash.Sum32()
	return uint(hSum32) % uint(clientsShardNum)
}

func newClients() clients {
	c := make(clients, clientsShardNum)
	for i := 0; i < clientsShardNum; i++ {
		c[i] = &clientsShard{}
		c[i].clients = make(map[string]*clientAttr)
	}
	return c
}

func (c clients) add(client *clientAttr) {
	shard := c.getShard(client.addr)
	c[shard].Lock()
	defer c[shard].Unlock()
	c[shard].clients[client.addr] = client
}
func (c clients) del(key string) {
	shard := c.getShard(key)
	c[shard].Lock()
	defer c[shard].Unlock()
	delete(c[shard].clients, key)
}
func (c clients) get(key string) (*clientAttr, bool) {
	shard := c.getShard(key)
	c[shard].RLock()
	defer c[shard].RUnlock()
	v, ok := c[shard].clients[key]
	return v, ok
}

func (c clients) enum() chan *clientAttr {
	ch := make(chan *clientAttr, 1)
	go func() {
		for i := 0; i < clientsShardNum; i++ {
			c[i].Lock()
			for _, v := range c[i].clients {
				ch <- v
			}
			c[i].Unlock()
		}
		close(ch)
	}()
	return ch
}
