package gomvc

import (
	"errors"
	"strings"
)

// QueryBuilder provides a safe way to build complex queries
type QueryBuilder struct {
	model      *Model
	selectCols []string
	joins      []SQLJoin
	wheres     []Filter
	groupBy    string
	orderBy    string
	limit      int64
	offset     int64
}

// NewQueryBuilder creates a new query builder for a model
func (m *Model) NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		model:      m,
		selectCols: []string{"*"},
		joins:      make([]SQLJoin, 0),
		wheres:     make([]Filter, 0),
	}
}

// Select specifies columns to select
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.selectCols = columns
	return qb
}

// Join adds an INNER JOIN
func (qb *QueryBuilder) Join(foreignTable, foreignPK, localKey, foreignKey string) *QueryBuilder {
	qb.joins = append(qb.joins, SQLJoin{
		Foreign_table: foreignTable,
		Foreign_PK:    foreignPK,
		KeyPair:       SQLKeyPair{LocalKey: localKey, ForeignKey: foreignKey},
		Join_type:     ModelJoinInner,
	})
	return qb
}

// LeftJoin adds a LEFT JOIN
func (qb *QueryBuilder) LeftJoin(foreignTable, foreignPK, localKey, foreignKey string) *QueryBuilder {
	qb.joins = append(qb.joins, SQLJoin{
		Foreign_table: foreignTable,
		Foreign_PK:    foreignPK,
		KeyPair:       SQLKeyPair{LocalKey: localKey, ForeignKey: foreignKey},
		Join_type:     ModelJoinLeft,
	})
	return qb
}

// Where adds a WHERE condition with AND logic
func (qb *QueryBuilder) Where(field, operator string, value interface{}) *QueryBuilder {
	logic := ""
	if len(qb.wheres) > 0 {
		logic = "AND"
	}
	qb.wheres = append(qb.wheres, Filter{
		Field:    field,
		Operator: operator,
		Value:    value,
		Logic:    logic,
	})
	return qb
}

// OrWhere adds a WHERE condition with OR logic
func (qb *QueryBuilder) OrWhere(field, operator string, value interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, Filter{
		Field:    field,
		Operator: operator,
		Value:    value,
		Logic:    "OR",
	})
	return qb
}

// WhereIn adds a WHERE IN condition
func (qb *QueryBuilder) WhereIn(field string, values []interface{}) *QueryBuilder {
	logic := ""
	if len(qb.wheres) > 0 {
		logic = "AND"
	}
	// Create placeholder for IN clause
	qb.wheres = append(qb.wheres, Filter{
		Field:    field,
		Operator: "IN",
		Value:    values,
		Logic:    logic,
	})
	return qb
}

// GroupBy adds GROUP BY clause
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	qb.groupBy = "GROUP BY " + strings.Join(columns, ", ")
	return qb
}

// OrderBy adds ORDER BY clause
func (qb *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
	if qb.orderBy == "" {
		qb.orderBy = "ORDER BY "
	} else {
		qb.orderBy += ", "
	}
	qb.orderBy += column + " " + strings.ToUpper(direction)
	return qb
}

// Limit sets the LIMIT
func (qb *QueryBuilder) Limit(limit int64) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the OFFSET
func (qb *QueryBuilder) Offset(offset int64) *QueryBuilder {
	qb.offset = offset
	return qb
}

// buildQuery constructs the SQL query with proper parameterization
func (qb *QueryBuilder) buildQuery() (string, []interface{}) {
	fields := make([]SQLField, 0)
	for _, col := range qb.selectCols {
		fields = append(fields, SQLField{FieldName: col})
	}

	// Modify BuildQuery to handle IN clauses and OFFSET
	q, values := BuildQueryExtended(
		QueryTypeSelect,
		fields,
		SQLTable{TableName: qb.model.TableName, PKField: qb.model.PKField},
		qb.joins,
		qb.wheres,
		qb.groupBy,
		qb.orderBy,
		qb.limit,
		qb.offset,
	)

	return q, values
}

// Execute executes the query and returns results
func (qb *QueryBuilder) Execute() ([]ResultRow, error) {
	q, values := qb.buildQuery()

	qb.model.lastQuery = q
	qb.model.lastValues = values

	r, err := qb.model.DB.Query(q, values...)
	if err != nil {
		InfoMessage("Query failed: " + q)
		return []ResultRow{}, err
	}
	defer r.Close()

	return qb.model.scanRows(r)
}

// First executes the query and returns the first result
func (qb *QueryBuilder) First() (ResultRow, error) {
	qb.Limit(1)
	results, err := qb.Execute()
	if err != nil {
		return ResultRow{}, err
	}
	if len(results) == 0 {
		return ResultRow{}, errors.New("no records found")
	}
	return results[0], nil
}

// Count returns the count of matching records
func (qb *QueryBuilder) Count() (int64, error) {
	qb.selectCols = []string{"COUNT(*) as count"}
	result, err := qb.First()
	if err != nil {
		return 0, err
	}

	countIdx := result.GetFieldIndex("count")
	if countIdx == -1 {
		return 0, errors.New("count field not found")
	}

	count, ok := result.Values[countIdx].(int64)
	if !ok {
		return 0, errors.New("invalid count value")
	}

	return count, nil
}
