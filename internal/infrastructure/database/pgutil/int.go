package pgutil

import "github.com/jackc/pgx/v5/pgtype"

func ToPgInt8Ptr(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{
		Int64: *v,
		Valid: true,
	}
}

func FromPgInt8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}

	n := v.Int64
	return &n
}

func ToPgInt4Ptr(v *int32) *int32 {
	return v
}

func ToPgInt4PtrFromInt(v *int) *int32 {
	if v == nil {
		return nil
	}

	n := int32(*v)
	return &n
}

func FromPgInt4PtrToInt(v *int32) *int {
	if v == nil {
		return nil
	}

	n := int(*v)
	return &n
}
