const database = db.getSiblingDB(process.env.MONGO_DATABASE || "mongoscale");

const product = database.products.findOne();
const review = database.reviews.findOne();
const order = database.orders.findOne();

const plans = [
  ["products by category sorted by price", database.products.find({ category: product.category }).sort({ price: 1 }).limit(20)],
  ["reviews for a product", database.reviews.find({ productId: review.productId }).sort({ createdAt: -1 }).limit(20)],
  ["orders for a user", database.orders.find({ userId: order.userId }).sort({ createdAt: -1 }).limit(20)],
];

plans.forEach(([label, cursor]) => {
  const stats = cursor.explain("executionStats").executionStats;
  print(label + ": examined=" + stats.totalDocsExamined + " keys=" + stats.totalKeysExamined + " returned=" + stats.nReturned + " ms=" + stats.executionTimeMillis);
});
