package models

import "time"

type Topology struct {
	Kind    string `json:"kind"`
	SetName string `json:"setName,omitempty"`
}

func (t Topology) Sharded() bool {
	return t.Kind == "sharded"
}

func (t Topology) ReplicaSet() bool {
	return t.Kind == "replica-set"
}

func (t Topology) SupportsTransactions() bool {
	return t.Kind != "standalone"
}

type PoolUsage struct {
	Open    int64 `json:"open"`
	InUse   int64 `json:"inUse"`
	Waiting int64 `json:"waiting"`
	Max     int64 `json:"max"`
	Failed  int64 `json:"failed"`
}

type ConnectionStatus struct {
	Current   int64 `json:"current"`
	Available int64 `json:"available"`
	Active    int64 `json:"active"`
}

type MemberStatus struct {
	Name          string  `json:"name"`
	State         string  `json:"state"`
	Healthy       bool    `json:"healthy"`
	UptimeSeconds int64   `json:"uptimeSeconds"`
	LagSeconds    float64 `json:"lagSeconds"`
}

type ReplicaSetStatus struct {
	Name        string         `json:"name"`
	PrimaryName string         `json:"primaryName"`
	Members     []MemberStatus `json:"members"`
}

type ShardStatus struct {
	Name      string         `json:"name"`
	Host      string         `json:"host"`
	Documents int64          `json:"documents"`
	SizeBytes int64          `json:"sizeBytes"`
	Members   []MemberStatus `json:"members"`
}

type ShardDistribution struct {
	Collection string        `json:"collection"`
	Sharded    bool          `json:"sharded"`
	ShardKey   string        `json:"shardKey,omitempty"`
	Shards     []ShardStatus `json:"shards"`
}

type IndexUsage struct {
	Collection string `json:"collection"`
	Name       string `json:"name"`
	Accesses   int64  `json:"accesses"`
}

type QueryPlan struct {
	Name            string `json:"name"`
	Collection      string `json:"collection"`
	Filter          string `json:"filter"`
	Stage           string `json:"stage"`
	IndexName       string `json:"indexName,omitempty"`
	DocsExamined    int64  `json:"docsExamined"`
	KeysExamined    int64  `json:"keysExamined"`
	Returned        int64  `json:"returned"`
	ExecutionMillis int64  `json:"executionMillis"`
}

type CollectionCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type MongoStatus struct {
	Topology      Topology          `json:"topology"`
	Version       string            `json:"version"`
	UptimeSeconds float64           `json:"uptimeSeconds"`
	Connections   ConnectionStatus  `json:"connections"`
	OpCounters    map[string]int64  `json:"opCounters"`
	ResidentMB    int64             `json:"residentMB"`
	VirtualMB     int64             `json:"virtualMB"`
	Pool          PoolUsage         `json:"pool"`
	SlowQueries   int64             `json:"slowQueries"`
	ReplicaSet    *ReplicaSetStatus `json:"replicaSet,omitempty"`
	Shards        ShardDistribution `json:"shards"`
	Indexes       []IndexUsage      `json:"indexes"`
	QueryPlans    []QueryPlan       `json:"queryPlans"`
	Collections   []CollectionCount `json:"collections"`
	CollectedAt   time.Time         `json:"collectedAt"`
}

type HealthReport struct {
	Status        string  `json:"status"`
	Mongo         string  `json:"mongo"`
	Redis         string  `json:"redis"`
	Topology      string  `json:"topology"`
	Mode          Mode    `json:"mode"`
	UptimeSeconds float64 `json:"uptimeSeconds"`
}
