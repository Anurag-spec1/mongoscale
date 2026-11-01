package bsonx

import "go.mongodb.org/mongo-driver/bson"

func E(key string, value any) bson.E {
	return bson.E{Key: key, Value: value}
}

func D(elements ...bson.E) bson.D {
	return bson.D(elements)
}

func A(values ...any) bson.A {
	return bson.A(values)
}
