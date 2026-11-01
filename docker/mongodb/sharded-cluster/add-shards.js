const shards = [
  "shard1rs/shard1a:27017,shard1b:27017,shard1c:27017",
  "shard2rs/shard2a:27017,shard2b:27017,shard2c:27017",
];

shards.forEach((shard) => {
  printjson(sh.addShard(shard));
});

printjson(db.adminCommand({ listShards: 1 }));
