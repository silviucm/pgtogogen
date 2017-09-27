package main

/*
	Nullable types that act as wrappers over pgx types
*/

import (
	"errors"
	"time"

	"github.com/silviucm/pgx/pgtype"
)

/*
 To allow legacy code to still work, we are creating dummy NullTypeX structs
 that embed the official pgtypes types
*/

// NullFloat32 is a compatibility-purpose struct that represents a float4 that may be null.
// It embeds the official pgtype.Float4
// If Valid is false then the value is NULL.
type NullFloat32 struct {
	pgtype.Float4
	Float32 float32
	Valid   bool // Valid is true if Float32 is not NULL
}

func (n *NullFloat32) Scan(src interface{}) error {
	if err := n.Float4.Scan(src); err != nil {
		return err
	}
	n.Float32 = n.Float4.Float
	n.Valid = (n.Float4.Status == pgtype.Present)
	return nil
}

func (n *NullFloat32) GetValue() float32 { return n.Float32 }

func CreateNullFloat32(value float32, valid bool) NullFloat32 {
	n := NullFloat32{Float32: value, Valid: valid}
	n.Set(value)
	return n
}

type Float4 pgtype.Float4

func CreateFloat32(value float32, present bool) Float4 {
	n := Float4{Float: value, Status: pgtype.Present}
	if present == false {
		n.Status = pgtype.Null
	}
	return n
}

// NullFloat64 is a compatibility-purpose struct that represents a float8 that may be null.
// It embeds the official pgtype.Float8
// If Valid is false then the value is NULL.
type NullFloat64 struct {
	pgtype.Float8
	Float64 float64
	Valid   bool // Valid is true if Float64 is not NULL
}

func (n *NullFloat64) Scan(src interface{}) error {
	if err := n.Float8.Scan(src); err != nil {
		return err
	}
	n.Float64 = n.Float8.Float
	n.Valid = (n.Float8.Status == pgtype.Present)
	return nil
}

func (n *NullFloat64) GetValue() float64 { return n.Float64 }

func CreateNullFloat64(value float64, valid bool) NullFloat64 {
	n := NullFloat64{Float64: value, Valid: valid}
	n.Set(value)
	return n
}

// NullString is a compatibility-purpose struct that represents a string that may be null.
// It embeds the official pgtype.Text
// If Valid is false then the value is NULL.
type NullString struct {
	String string
	Valid  bool // Valid is true if String is not NULL
	pgtype.Text
}

func (n *NullString) Scan(src interface{}) error {
	if err := n.Text.Scan(src); err != nil {
		return err
	}
	n.String = n.Text.String
	n.Valid = (n.Text.Status == pgtype.Present)
	return nil
}

func (n *NullString) GetValue() string { return n.String }

func CreateNullString(value string, valid bool) NullString {
	n := NullString{String: value, Valid: valid}
	n.Set(value)
	return n
}

// NullInt16 is a compatibility-purpose struct that represents an int16 that may be null.
// It embeds the official pgtype.Int2
// If Valid is false then the value is NULL.
type NullInt16 struct {
	pgtype.Int2
	Int16 int16
	Valid bool // Valid is true if Int16 is not NULL
}

func (n *NullInt16) Scan(src interface{}) error {
	if err := n.Int2.Scan(src); err != nil {
		return err
	}
	n.Int16 = n.Int2.Int
	n.Valid = (n.Int2.Status == pgtype.Present)
	return nil
}

func (n *NullInt16) GetValue() int16 { return n.Int16 }

func CreateNullInt16(value int16, valid bool) NullInt16 {
	n := NullInt16{Int16: value, Valid: valid}
	n.Set(value)
	return n
}

// NullInt32 is a compatibility-purpose struct that represents an int32 that may be null.
// It embeds the official pgtype.Int4
// If Valid is false then the value is NULL.
type NullInt32 struct {
	pgtype.Int4
	Int32 int32
	Valid bool // Valid is true if Int32 is not NULL
}

func (n *NullInt32) Scan(src interface{}) error {
	if err := n.Int4.Scan(src); err != nil {
		return err
	}
	n.Int32 = n.Int4.Int
	n.Valid = (n.Int4.Status == pgtype.Present)
	return nil
}

func (n *NullInt32) GetValue() int32 { return n.Int32 }

func CreateNullInt32(value int32, valid bool) NullInt32 {
	n := NullInt32{Int32: value, Valid: valid}
	n.Set(value)
	return n
}

// NullInt64 is a compatibility-purpose struct that represents an int64 that may be null.
// It embeds the official pgtype.Text
// If Valid is false then the value is NULL.
type NullInt64 struct {
	pgtype.Int8
	Int64 int64
	Valid bool // Valid is true if Int64 is not NULL
}

func (n *NullInt64) Scan(src interface{}) error {
	if err := n.Int8.Scan(src); err != nil {
		return err
	}
	n.Int64 = n.Int8.Int
	n.Valid = (n.Int8.Status == pgtype.Present)
	return nil
}

func (n *NullInt64) GetValue() int64 { return n.Int64 }

func CreateNullInt64(value int64, valid bool) NullInt64 {
	n := NullInt64{Int64: value, Valid: valid}
	n.Set(value)
	return n
}

// NullBool is a compatibility-purpose struct that represents a bool that may be null.
// It embeds the official pgtype.Bool
// If Valid is false then the value is NULL.
type NullBool struct {
	pgtype.Bool
	Valid bool // Valid is true if String is not NULL
}

func (n *NullBool) Scan(src interface{}) error {
	if err := n.Bool.Scan(src); err != nil {
		return err
	}
	n.Valid = (n.Bool.Status == pgtype.Present)
	return nil
}

func (n *NullBool) GetValue() bool { return n.Bool.Bool }

func CreateNullBool(value bool, valid bool) NullBool {
	n := NullBool{Valid: valid}
	n.Set(value)
	return n
}

// NullTime is a compatibility-purpose struct that represents a timestamp with
// timezone that may be null.
// It embeds the official pgtype.Text
// If Valid is false then the value is NULL.
type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if String is not NULL
	pgtype.Timestamptz
}

func (n *NullTime) Scan(src interface{}) error {
	if err := n.Timestamptz.Scan(src); err != nil {
		return err
	}
	n.Time = n.Timestamptz.Time
	n.Valid = (n.Timestamptz.Status == pgtype.Present)
	return nil
}

func (n *NullTime) GetValue() time.Time { return n.Timestamptz.Time }

func CreateNullTime(value time.Time, valid bool) NullTime {
	n := NullTime{Time: value, Valid: valid}
	n.Set(value)
	return n
}

// NullHstore represents an hstore column that can be null or have null values
// associated with its keys.
// It embeds the official pgtype.Hstore
// If Valid is false then the value is NULL.
type NullHstore struct {
	pgtype.Hstore
	Valid bool // Valid is true if String is not NULL
}

func (n *NullHstore) Scan(src interface{}) error {
	if err := n.Hstore.Scan(src); err != nil {
		return err
	}
	n.Valid = (n.Hstore.Status == pgtype.Present)
	return nil
}

func (n *NullHstore) GetValue() map[string]NullString {
	if n.Hstore.Map == nil {
		return nil
	}
	m := make(map[string]NullString, len(n.Hstore.Map))
	for k, v := range n.Hstore.Map {
		m[k] = NullString{String: v.String, Valid: v.Status == pgtype.Present}
	}
	return m
}

func CreateNullHstore(value map[string]NullString, valid bool) NullHstore {
	n := NullHstore{Valid: valid}
	if value == nil {
		return n
	}
	m := make(map[string]string, len(value))
	for k, v := range value {
		m[k] = v.String
	}
	n.Set(m)
	return n
}
