package pgutil

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// scanTextPtr is a scan destination that writes a nullable text column into *string.
type scanTextPtr struct{ p **string }

func (s scanTextPtr) Scan(src any) error {
	if src == nil {
		*s.p = nil
		return nil
	}
	var t pgtype.Text
	if err := t.Scan(src); err != nil {
		return err
	}
	if !t.Valid {
		*s.p = nil
		return nil
	}
	v := t.String
	*s.p = &v
	return nil
}

// ScanPgTextPtr returns a scan destination that writes nullable text into *string.
func ScanPgTextPtr(p **string) interface{ Scan(any) error } { return scanTextPtr{p} }

// scanInt8Ptr writes a nullable int8 (bigint) into *int64.
type scanInt8Ptr struct{ p **int64 }

func (s scanInt8Ptr) Scan(src any) error {
	if src == nil {
		*s.p = nil
		return nil
	}
	var i pgtype.Int8
	if err := i.Scan(src); err != nil {
		return err
	}
	if !i.Valid {
		*s.p = nil
		return nil
	}
	v := i.Int64
	*s.p = &v
	return nil
}

// ScanPgInt8Ptr returns a scan destination that writes nullable bigint into *int64.
func ScanPgInt8Ptr(p **int64) interface{ Scan(any) error } { return scanInt8Ptr{p} }

// scanNumericPtr writes a nullable numeric into *float64.
type scanNumericPtr struct{ p **float64 }

func (s scanNumericPtr) Scan(src any) error {
	if src == nil {
		*s.p = nil
		return nil
	}
	var n pgtype.Numeric
	if err := n.Scan(src); err != nil {
		return err
	}
	if !n.Valid {
		*s.p = nil
		return nil
	}
	val, err := n.Value()
	if err != nil {
		*s.p = nil
		return nil
	}
	f, err := strconv.ParseFloat(val.(string), 64)
	if err != nil {
		*s.p = nil
		return nil
	}
	*s.p = &f
	return nil
}

// ScanPgNumericPtr returns a scan destination that writes nullable numeric into *float64.
func ScanPgNumericPtr(p **float64) interface{ Scan(any) error } { return scanNumericPtr{p} }
