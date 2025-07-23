package utils

import "database/sql"

// DerefString converts a string to sql.NullString for database operations.
// Returns a valid NullString if the input is non-empty, otherwise returns an invalid NullString.
func DerefString(s string) sql.NullString {
	if s != "" {
		return sql.NullString{
			String: s,
			Valid:  true,
		}
	} else {
		return sql.NullString{
			Valid: false,
		}
	}

}
