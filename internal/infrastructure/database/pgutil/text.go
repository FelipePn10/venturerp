package pgutil

import "github.com/jackc/pgx/v5/pgtype"

func ToPgTextFromString(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  true,
	}
}

func ToPgTextFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: *s,
		Valid:  true,
	}
}

func FromPgText(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func FromPgTextPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}

	s := v.String
	return &s
}

func ToPgText(v string) pgtype.Text {
	if v == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: v,
		Valid:  true,
	}
}
