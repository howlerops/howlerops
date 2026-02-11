package database

func newEditableMetadata(columns []string) *EditableQueryMetadata {
	metadata := &EditableQueryMetadata{
		Enabled:     false,
		Columns:     make([]EditableColumn, 0, len(columns)),
		PrimaryKeys: make([]string, 0),
	}

	for _, col := range columns {
		metadata.Columns = append(metadata.Columns, EditableColumn{
			Name:       col,
			ResultName: col,
			Editable:   false,
			PrimaryKey: false,
		})
	}

	return metadata
}
