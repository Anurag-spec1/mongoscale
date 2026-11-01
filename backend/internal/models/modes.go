package models

import "fmt"

type Mode string

const (
	ModeUnoptimized    Mode = "unoptimized"
	ModeIndexed        Mode = "indexed"
	ModeReplicaSet     Mode = "replica-set"
	ModeSharded        Mode = "sharded"
	ModeFullyOptimized Mode = "fully-optimized"
)

type Tuning struct {
	Mode             Mode   `json:"mode"`
	Label            string `json:"label"`
	Description      string `json:"description"`
	Indexes          bool   `json:"indexes"`
	SecondaryReads   bool   `json:"secondaryReads"`
	MajorityWrites   bool   `json:"majorityWrites"`
	Sharding         bool   `json:"sharding"`
	Projections      bool   `json:"projections"`
	KeysetPagination bool   `json:"keysetPagination"`
	Cache            bool   `json:"cache"`
	RateLimit        bool   `json:"rateLimit"`
	MaxPoolSize      uint64 `json:"maxPoolSize"`
	MinPoolSize      uint64 `json:"minPoolSize"`
}

var tunings = map[Mode]Tuning{
	ModeUnoptimized: {
		Mode:        ModeUnoptimized,
		Label:       "Unoptimized",
		Description: "No secondary indexes, regex search, skip pagination, full documents, tiny connection pool, primary reads, w:1 writes.",
		MaxPoolSize: 10,
		MinPoolSize: 1,
	},
	ModeIndexed: {
		Mode:        ModeIndexed,
		Label:       "Indexed",
		Description: "Single-field, compound, text and unique indexes. Everything else unchanged.",
		Indexes:     true,
		MaxPoolSize: 10,
		MinPoolSize: 1,
	},
	ModeReplicaSet: {
		Mode:           ModeReplicaSet,
		Label:          "Replica set",
		Description:    "Indexes plus secondaryPreferred reads, majority write concern for orders and a larger connection pool.",
		Indexes:        true,
		SecondaryReads: true,
		MajorityWrites: true,
		MaxPoolSize:    60,
		MinPoolSize:    5,
	},
	ModeSharded: {
		Mode:             ModeSharded,
		Label:            "Sharded",
		Description:      "Replica-set behaviour plus orders sharded on a hashed userId key, projections and keyset pagination.",
		Indexes:          true,
		SecondaryReads:   true,
		MajorityWrites:   true,
		Sharding:         true,
		Projections:      true,
		KeysetPagination: true,
		MaxPoolSize:      120,
		MinPoolSize:      10,
	},
	ModeFullyOptimized: {
		Mode:             ModeFullyOptimized,
		Label:            "Fully optimized",
		Description:      "Everything above plus Redis caching, per-client rate limiting on expensive endpoints and the largest pool.",
		Indexes:          true,
		SecondaryReads:   true,
		MajorityWrites:   true,
		Sharding:         true,
		Projections:      true,
		KeysetPagination: true,
		Cache:            true,
		RateLimit:        true,
		MaxPoolSize:      200,
		MinPoolSize:      20,
	},
}

func TuningFor(mode Mode) (Tuning, bool) {
	tuning, ok := tunings[mode]
	return tuning, ok
}

func ParseMode(value string) (Mode, error) {
	switch value {
	case "optimized":
		return ModeFullyOptimized, nil
	case string(ModeUnoptimized), string(ModeIndexed), string(ModeReplicaSet), string(ModeSharded), string(ModeFullyOptimized):
		return Mode(value), nil
	}
	return "", fmt.Errorf("unknown optimization mode %q", value)
}

func AllTunings() []Tuning {
	order := []Mode{ModeUnoptimized, ModeIndexed, ModeReplicaSet, ModeSharded, ModeFullyOptimized}
	result := make([]Tuning, 0, len(order))
	for _, mode := range order {
		result = append(result, tunings[mode])
	}
	return result
}
