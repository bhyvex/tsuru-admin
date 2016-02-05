// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provision

import (
	"reflect"

	"github.com/tsuru/tsuru/db"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ScopedConfig struct {
	Scope        string `bson:"_id"`
	Envs         []Entry
	Pools        []PoolEntry
	Extra        map[string]interface{} `bson:",inline"`
	entryMap     EntryMap
	poolEntryMap PoolEntryMap
}

type Entry struct {
	Name    string
	Value   interface{}
	Private bool
}

type PoolEntry struct {
	Name string
	Envs []Entry
}

type EntryMap map[string]Entry
type PoolEntryMap map[string]EntryMap

type ConfigEntryList []Entry

func (l ConfigEntryList) Len() int           { return len(l) }
func (l ConfigEntryList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ConfigEntryList) Less(i, j int) bool { return l[i].Name < l[j].Name }

type ConfigPoolEntryList []PoolEntry

func (l ConfigPoolEntryList) Len() int           { return len(l) }
func (l ConfigPoolEntryList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ConfigPoolEntryList) Less(i, j int) bool { return l[i].Name < l[j].Name }

func FindScopedConfig(scope string) (*ScopedConfig, error) {
	result := ScopedConfig{
		Scope: scope,
	}
	err := result.reload()
	return &result, err
}

func (c *ScopedConfig) Add(name string, value interface{}) {
	c.add("", name, value, false)
}

func (c *ScopedConfig) AddPool(pool, name string, value interface{}) {
	c.add(pool, name, value, false)
}

func (c *ScopedConfig) UpdateWith(other *ScopedConfig) error {
	for _, env := range other.Envs {
		c.add("", env.Name, env.Value, env.Private)
	}
	for _, pool := range other.Pools {
		for _, env := range pool.Envs {
			c.add(pool.Name, env.Name, env.Value, env.Private)
		}
	}
	return c.SaveEnvs()
}

func (c *ScopedConfig) SetExtra(name string, value interface{}) error {
	conn, err := db.Conn()
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.ScopedConfig().UpsertId(c.Scope, bson.M{"$set": bson.M{name: value}})
	if err != nil {
		return err
	}
	return c.reload()
}

func (c *ScopedConfig) SetExtraAtomic(name string, value interface{}) (bool, error) {
	conn, err := db.Conn()
	if err != nil {
		return false, err
	}
	defer conn.Close()
	_, err = conn.ScopedConfig().Upsert(bson.M{
		"_id": c.Scope,
		"$or": []bson.M{{name: ""}, {name: bson.M{"$exists": false}}},
	}, bson.M{"$set": bson.M{name: value}})
	reloadErr := c.reload()
	if err == nil {
		return true, reloadErr
	}
	if mgo.IsDup(err) {
		return false, reloadErr
	}
	return false, err
}

func (c *ScopedConfig) GetExtraString(name string) string {
	if extra, ok := c.Extra[name].(string); ok {
		return extra
	}
	return ""
}

func (c *ScopedConfig) PoolEntries(pool string) EntryMap {
	m := make(EntryMap)
	for _, e := range c.entries("") {
		m[e.Name] = e
	}
	for _, e := range c.entries(pool) {
		m[e.Name] = e
	}
	return m
}

func (c *ScopedConfig) PoolEntry(pool, name string) string {
	entry, ok := c.entries(pool)[name]
	var value interface{}
	if ok && entry.Value != nil {
		value = entry.Value
	} else {
		entry, ok = c.entries("")[name]
		if ok && entry.Value != nil {
			value = entry.Value
		}
	}
	if ret, ok := value.(string); ok {
		return ret
	}
	return ""
}

func (c *ScopedConfig) ResetEnvs() {
	c.entryMap = make(EntryMap)
	c.poolEntryMap = make(PoolEntryMap)
	c.Envs = nil
	c.Pools = nil
}

func (c *ScopedConfig) ResetBaseEnvs() {
	c.entryMap = make(EntryMap)
	c.Envs = nil
}

func (c *ScopedConfig) ResetPoolEnvs(pool string) {
	if c.poolEntryMap != nil {
		delete(c.poolEntryMap, pool)
	}
	c.updateFromMap()
}

func (c *ScopedConfig) SaveEnvs() error {
	c.updateFromMap()
	conn, err := db.Conn()
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.ScopedConfig().UpsertId(c.Scope, bson.M{
		"$set": bson.M{
			"envs":  c.Envs,
			"pools": c.Pools,
		},
	})
	return err
}

func (c *ScopedConfig) FilterPools(pools []string) {
	if pools == nil {
		return
	}
	poolEntries := make([]PoolEntry, 0, len(pools))
	for _, pool := range pools {
		for _, poolEntry := range c.Pools {
			if poolEntry.Name == pool {
				poolEntries = append(poolEntries, poolEntry)
				break
			}
		}
	}
	c.Pools = poolEntries
}

func (c *ScopedConfig) loadMap() {
	c.entryMap = make(EntryMap)
	c.poolEntryMap = make(PoolEntryMap)
	for _, entry := range c.Envs {
		c.entryMap[entry.Name] = entry
	}
	for _, pe := range c.Pools {
		if c.poolEntryMap[pe.Name] == nil {
			c.poolEntryMap[pe.Name] = make(EntryMap)
		}
		for _, entry := range pe.Envs {
			c.poolEntryMap[pe.Name][entry.Name] = entry
		}
	}
}

func (c *ScopedConfig) updateFromMap() {
	c.Envs = nil
	c.Pools = nil
	for _, value := range c.entryMap {
		c.Envs = append(c.Envs, value)
	}
	for poolName, entryMap := range c.poolEntryMap {
		poolEntry := PoolEntry{Name: poolName}
		for _, value := range entryMap {
			poolEntry.Envs = append(poolEntry.Envs, value)
		}
		c.Pools = append(c.Pools, poolEntry)
	}
}

func (c *ScopedConfig) reload() error {
	conn, err := db.Conn()
	if err != nil {
		return err
	}
	defer conn.Close()
	err = conn.ScopedConfig().FindId(c.Scope).One(c)
	c.loadMap()
	if err == mgo.ErrNotFound {
		return nil
	}
	return err
}

func (c *ScopedConfig) add(pool, name string, value interface{}, private bool) {
	var m EntryMap
	if pool == "" {
		m = c.entryMap
	} else {
		if c.poolEntryMap[pool] == nil {
			c.poolEntryMap[pool] = make(EntryMap)
		}
		m = c.poolEntryMap[pool]
	}
	if value == nil || value == reflect.Zero(reflect.ValueOf(value).Type()).Interface() {
		delete(m, name)
		return
	}
	m[name] = Entry{
		Name:    name,
		Value:   value,
		Private: private,
	}
	c.updateFromMap()
}

func (c *ScopedConfig) entries(pool string) EntryMap {
	if pool == "" {
		return c.entryMap
	}
	if c.poolEntryMap[pool] == nil {
		c.poolEntryMap[pool] = make(EntryMap)
	}
	return c.poolEntryMap[pool]
}
