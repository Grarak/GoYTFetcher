package database

import "strings"

type tableBuilder struct {
	name            string
	primaryKeys     []column
	uniqueKeysPairs [][]column
	foreignKeys     []foreignKey
	columns         []column
}

func newTableBuilder(name string) *tableBuilder {
	return &tableBuilder{name: name}
}

func (tableBuilder *tableBuilder) addPrimaryKey(key column) *tableBuilder {
	tableBuilder.primaryKeys = append(tableBuilder.primaryKeys, key)
	return tableBuilder
}

func (tableBuilder *tableBuilder) addUniqueKeyPair(key ...column) *tableBuilder {
	tableBuilder.uniqueKeysPairs = append(tableBuilder.uniqueKeysPairs, key)
	return tableBuilder
}

func (tableBuilder *tableBuilder) addForeignKey(key foreignKey) *tableBuilder {
	tableBuilder.foreignKeys = append(tableBuilder.foreignKeys, key)
	return tableBuilder
}

func (tableBuilder *tableBuilder) addColumn(column column) *tableBuilder {
	tableBuilder.columns = append(tableBuilder.columns, column)
	return tableBuilder
}

func (tableBuilder *tableBuilder) build() string {
	cmd := "CREATE TABLE IF NOT EXISTS " + tableBuilder.name + " ("

	for _, foreignKey := range tableBuilder.foreignKeys {
		cmd += foreignKey.name + " " + string(foreignKey.dataType) + " NOT NULL,"
	}
	for _, primaryKey := range tableBuilder.primaryKeys {
		line := primaryKey.name + " " + string(primaryKey.dataType)
		if !strings.Contains(cmd, line) {
			cmd += line + " NOT NULL,"
		}
	}
	for _, uniqueKeyPair := range tableBuilder.uniqueKeysPairs {
		for _, uniqueKey := range uniqueKeyPair {
			line := uniqueKey.name + " " + string(uniqueKey.dataType)
			if !strings.Contains(cmd, line) {
				cmd += line + ","
			}
		}
	}
	for _, column := range tableBuilder.columns {
		line := column.name + " " + string(column.dataType)
		if !strings.Contains(cmd, line) {
			cmd += line + ","
		}
	}

	referenceTables := make(map[string][]foreignKey)
	for _, foreignKey := range tableBuilder.foreignKeys {
		referenceKeys := referenceTables[foreignKey.referenceTable]
		referenceTables[foreignKey.referenceTable] = append(referenceKeys, foreignKey)
	}
	for table, keys := range referenceTables {
		cmd += "FOREIGN KEY ("
		for _, key := range keys {
			cmd += key.name + ","
		}
		cmd = cmd[:len(cmd)-1] + ") REFERENCES " + table + " ("
		for _, key := range keys {
			cmd += key.referenceKey + ","
		}
		cmd = cmd[:len(cmd)-1] + ") ON UPDATE CASCADE,"
	}

	var primaryKeys []string
	for _, primaryKey := range tableBuilder.primaryKeys {
		primaryKeys = append(primaryKeys, primaryKey.name)
	}
	for _, foreignKey := range tableBuilder.foreignKeys {
		if foreignKey.primaryKey {
			primaryKeys = append(primaryKeys, foreignKey.name)
		}
	}
	if len(primaryKeys) > 0 {
		cmd += "PRIMARY KEY ("
		for _, key := range primaryKeys {
			cmd += key + ","
		}
		cmd = cmd[:len(cmd)-1] + "),"
	}

	for _, uniqueKeyPair := range tableBuilder.uniqueKeysPairs {
		cmd += "UNIQUE ("
		for _, uniqueKey := range uniqueKeyPair {
			cmd += uniqueKey.name + ","
		}
		cmd = cmd[:len(cmd)-1] + "),"
	}

	cmd = cmd[:len(cmd)-1] + ")"
	return cmd
}
