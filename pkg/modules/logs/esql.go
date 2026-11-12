package logs

import "fmt"

// esqlRows converts the ES|QL `values` payload into one object per row.
//
// Non-columnar responses return one array per row; columnar responses return
// one array per column (each holding that column's values in order), so the
// data has to be transposed before it can be keyed by column name.
func esqlRows(resp ESQLResponse, columnar bool) ([]map[string]interface{}, error) {
	if !columnar {
		rows := make([]map[string]interface{}, 0, len(resp.Values))
		for _, row := range resp.Values {
			obj := make(map[string]interface{}, len(row))
			for i, value := range row {
				if i < len(resp.Columns) {
					obj[resp.Columns[i].Name] = value
				}
			}
			rows = append(rows, obj)
		}
		return rows, nil
	}

	// Columnar: values[i] holds all values of columns[i].
	if len(resp.Values) != 0 && len(resp.Columns) != 0 && len(resp.Values) != len(resp.Columns) {
		return nil, fmt.Errorf("unexpected ES|QL columnar response: %d columns but %d value arrays", len(resp.Columns), len(resp.Values))
	}

	rowCount := 0
	for _, column := range resp.Values {
		if len(column) > rowCount {
			rowCount = len(column)
		}
	}

	rows := make([]map[string]interface{}, 0, rowCount)
	for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
		obj := make(map[string]interface{}, len(resp.Values))
		for columnIndex, column := range resp.Values {
			if rowIndex < len(column) {
				obj[resp.Columns[columnIndex].Name] = column[rowIndex]
			}
		}
		rows = append(rows, obj)
	}
	return rows, nil
}
