package repository

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"mongoscale/internal/utils"
	"mongoscale/pkg/bsonx"
)

func keysetClause(field string, direction int, cursor utils.Cursor, dateField bool) (bson.E, error) {
	id, err := utils.ParseObjectID(cursor.ID)
	if err != nil {
		return bson.E{}, utils.BadRequest("cursor is not valid")
	}
	operator := "$gt"
	if direction < 0 {
		operator = "$lt"
	}
	var boundary any = cursor.Number
	if dateField {
		boundary = time.UnixMilli(cursor.Millis)
	}
	clause := bsonx.E("$or", bsonx.A(
		bsonx.D(bsonx.E(field, bsonx.D(bsonx.E(operator, boundary)))),
		bsonx.D(bsonx.E(field, boundary), bsonx.E("_id", bsonx.D(bsonx.E(operator, id)))),
	))
	return clause, nil
}

func numberCursor(value float64, id primitive.ObjectID) string {
	return utils.EncodeCursor(utils.Cursor{Number: value, ID: id.Hex()})
}

func dateCursor(value time.Time, id primitive.ObjectID) string {
	return utils.EncodeCursor(utils.Cursor{Millis: value.UnixMilli(), ID: id.Hex()})
}

func offsetFor(page, limit int) int64 {
	return int64((page - 1) * limit)
}
