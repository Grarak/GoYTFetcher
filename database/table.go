package database

import "fmt"

type tableBuilder struct {
	name        string
	primaryKeys []column
	uniqueKeys  []column
	foreignKeys []foreignKey
	columns     []column
}

func newTableBuilder(name string) *tableBuilder {
	return &tableBuilder{name: name}
}

func (tableBuilder *tableBuilder) addPrimaryKey(key column) *tableBuilder {
	tableBuilder.primaryKeys = append(tableBuilder.primaryKeys, key)
	return tableBuilder
}

func (tableBuilder *tableBuilder) addUniqueKey(key column) *tableBuilder {
	tableBuilder.uniqueKeys = append(tableBuilder.uniqueKeys, key)
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
		cmd += primaryKey.name + " " + string(primaryKey.dataType) + " NOT NULL,"
	}
	for _, uniqueKey := range tableBuilder.uniqueKeys {
		cmd += uniqueKey.name + " " + string(uniqueKey.dataType) + " UNIQUE,"
	}
	for _, column := range tableBuilder.columns {
		cmd += column.name + " " + string(column.dataType) + ","
	}

	for _, foreignKey := range tableBuilder.foreignKeys {
		cmd += fmt.Sprintf(
			"foreign key (%s) references %s(%s),",
			foreignKey.name, foreignKey.referenceTable, foreignKey.referenceKey)
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
		cmd += "primary key ("
		for _, key := range primaryKeys {
			cmd += key + ","
		}
		cmd = cmd[:len(cmd)-1] + "),"
	}
	cmd = cmd[:len(cmd)-1] + ")"
	return cmd
}
