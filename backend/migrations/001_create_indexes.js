const database = db.getSiblingDB(process.env.MONGO_DATABASE || "mongoscale");

database.users.createIndex({ email: 1 }, { unique: true, name: "email_1" });

database.products.createIndex({ category: 1, price: 1 }, { name: "category_1_price_1" });
database.products.createIndex({ name: "text" }, { name: "name_text" });
database.products.createIndex({ rating: -1 }, { name: "rating_-1" });

database.orders.createIndex({ userId: 1, createdAt: -1 }, { name: "userId_1_createdAt_-1" });
database.orders.createIndex({ status: 1, createdAt: -1 }, { name: "status_1_createdAt_-1" });
database.orders.createIndex({ userId: "hashed" }, { name: "userId_hashed" });

database.reviews.createIndex({ productId: 1, createdAt: -1 }, { name: "productId_1_createdAt_-1" });
database.reviews.createIndex({ productId: 1, rating: -1 }, { name: "productId_1_rating_-1" });
