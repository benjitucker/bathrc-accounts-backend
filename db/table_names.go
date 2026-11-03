package db

// tableNamePrefix is prepended to every DynamoDB table name. It is empty in
// production, so the tables keep their original names (e.g. "TrainingSubmissions").
// A non-production deployment sets it (e.g. "dev-") so that it uses its own tables.
var tableNamePrefix string

// SetTableNamePrefix sets the prefix applied to all table names. It must be called
// before any table is opened. An empty prefix gives the production table names.
func SetTableNamePrefix(prefix string) {
	tableNamePrefix = prefix
}

// tableName returns the full DynamoDB table name for a base table name.
func tableName(base string) string {
	return tableNamePrefix + base
}
