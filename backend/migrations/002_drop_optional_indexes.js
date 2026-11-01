const database = db.getSiblingDB(process.env.MONGO_DATABASE || "mongoscale");

["products", "orders", "reviews"].forEach((collection) => {
  database[collection].getIndexes().forEach((index) => {
    if (index.name !== "_id_") {
      database[collection].dropIndex(index.name);
    }
  });
});
