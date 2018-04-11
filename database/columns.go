package database

type dataType string

type column struct {
	name     string
	dataType dataType
}

func text() dataType {
	return "text"
}

func boolean() dataType {
	return "boolean"
}

func datetime() dataType {
	return "datetime"
}

var ColumnApikey = column{"api_key", text()}
var ColumnName = column{"name", text()}
var ColumnPasswordSalt = column{"password_salt", text()}
var ColumnPasswordHash = column{"password_hash", text()}
var ColumnAdmin = column{"admin", boolean()}
var ColumnVerified = column{"verified", boolean()}
var ColumnPublic = column{"public", boolean()}
var ColumnId = column{"id", text()}
var ColumnDate = column{"date", datetime()}
