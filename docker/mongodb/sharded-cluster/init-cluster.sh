#!/bin/bash
set -euo pipefail

initiate_replica_set() {
  local name="$1"
  local members="$2"
  local config_server="$3"
  local first_host="${members%%,*}"
  RS_NAME="$name" RS_MEMBERS="$members" RS_CONFIGSVR="$config_server" \
    mongosh --host "$first_host" --quiet --file /scripts/init-replica-set.js
}

initiate_replica_set cfgrs cfg1,cfg2,cfg3 true
initiate_replica_set shard1rs shard1a,shard1b,shard1c false
initiate_replica_set shard2rs shard2a,shard2b,shard2c false
