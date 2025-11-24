package gomvc

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
)

// JoinType, supported MySql join types INNER, LEFT, RIGHT
type JoinType string

// ResultStyle, full dataset result from database OR dataset result with sub-result nested datasets
type ResultStyle int

// QueryType the query type identifier type CRUD (Create, Read, Update, Delete)
type QueryType string

const (
	ModelJoinInner JoinType = "INNER"
	ModelJoinLeft  JoinType = "LEFT"
	ModelJoinRight JoinType = "RIGHT"
)

const (
	ResultStyleFullresult ResultStyle = 0
	ResultStyleSubresult  ResultStyle = 1
)

const (
	QueryTypeInsert QueryType = "c"
	QueryTypeSelect QueryType = "r"
	QueryTypeUpdate QueryType = "u"
	QueryTypeDelete QueryType = "d"
)

// Model is the model struct holding all the data and parameters for each model.
type Model struct {
	DB           *sql.DB
	PKField      string
	TableName    string
	OrderString  string
	Fields       []string
	Labels       map[string]string
	Relations    []Relation
	DefaultQuery string
	lastQuery    string
	lastValues   []interface{}
}

// ResultRow is the result coming from MySql database
type ResultRow struct {
	Values    []interface{}
	Fields    []string
	pointers  []interface{}
	Subresult []ResultRow
}

// Relation is the relation to another table
type Relation struct {
	Join          SQLJoin
	Foreign_model Model
	ResultStyle   ResultStyle
}

// SQLJoin the type of MySql Join used by Relation
type SQLJoin struct {
	Foreign_table string
	Foreign_PK    string
	KeyPair       SQLKeyPair
	Join_type     JoinType
}

// SQLTable the SQL table object
type SQLTable struct {
	TableName string
	PKField   string
}

// SQLField the MySql table field object
type SQLField struct {
	FieldName string
	Value     interface{}
}

// SQLKeyPair local<->foreign key pair
type SQLKeyPair struct {
	LocalKey   string
	ForeignKey string
}

// Filter is user to filter data in WHERE Clause MySql statement
type Filter struct {
	Field    string
	Operator string
	Value    interface{}
	Logic    string
}

// Instance function returns the current model instance
func (m *Model) Instance() Model {
	return *m
}

// GetFieldIndex rerurns the field index using its name
func (r *ResultRow) GetFieldIndex(name string) int {
	for i, v := range r.Fields {
		if name == v {
			return i
		}
	}

	return -1
}

// InitModel pass all initial parammeters to activate the model
func (m *Model) InitModel(db *sql.DB, tableName string, PKField string) error {
	m.DB = db
	m.TableName = tableName
	m.PKField = PKField

	var q = "SHOW COLUMNS FROM " + tableName
	r, err := m.DB.Query(q)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		var rr ResultRow
		rr.Values = make([]interface{}, 6)
		rr.pointers = make([]interface{}, 6)

		for i := 0; i < 6; i++ {
			rr.pointers[i] = &rr.Values[i]
		}

		r.Scan(rr.pointers...)

		b := rr.Values[0].([]byte)
		n := string(b)
		m.Fields = append(m.Fields, n)
	}

	if len(m.Relations) > 0 {
		for _, f := range m.Relations {
			for _, ff := range f.Foreign_model.Fields {
				m.Fields = append(m.Fields, f.Join.Foreign_table+"."+ff)
			}
		}
	}

	return nil
}

// AssignLabels function to assign human friendly Names/Labels to MySql fields
func (m *Model) AssignLabels(labels map[string]string) {
	m.Labels = labels
}

// AddRelation Add Foreign table (model) and assign a relation to another model
func (m *Model) AddRelation(db *sql.DB, tableName string, PKField string, keys SQLKeyPair, join_type JoinType, result_style ResultStyle) {
	fm := new(Model)
	fm.InitModel(db, tableName, PKField)
	if m.Relations == nil {
		m.Relations = make([]Relation, 0)
	}
	m.Relations = append(m.Relations,
		Relation{Join: SQLJoin{Foreign_table: tableName, Foreign_PK: PKField, KeyPair: keys, Join_type: join_type},
			Foreign_model: *fm,
			ResultStyle:   result_style},
	)
}

// Label returns the Label from the field name
func (m *Model) Label(field string) string {
	lb, ok := m.Labels[field]
	if !ok {
		return "Undefined"
	}
	return lb
}

// GetLastId is a function to get the last id from a Table/Model
func (m *Model) GetLastId() (int64, error) {
	if m == nil {
		return 0, errors.New("cannot perform action : GetLastId() on nil model")
	}

	var q string
	q, _ = BuildQuery(QueryTypeSelect,
		[]SQLField{{FieldName: m.PKField}},
		SQLTable{TableName: m.TableName, PKField: m.PKField},
		[]SQLJoin{}, []Filter{}, "", "ORDER BY "+m.PKField+" DESC", 1)

	r, err := m.DB.Query(q)

	if err != nil {
		return 0, err
	}

	var id int64
	r.Next()
	err = r.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, err
}

// GetRecords is function to execute a query against a table/model with filters (WHERE filters)
func (m *Model) GetRecords(filters []Filter, limit int64) ([]ResultRow, error) {
	if m == nil {
		return []ResultRow{}, errors.New("cannot perform action : GetRecords() on nil model")
	}

	var r *sql.Rows
	var err error

	if len(m.DefaultQuery) == 0 {
		j := make([]SQLJoin, 0)
		if len(m.Relations) > 0 {
			for _, i := range m.Relations {
				if i.ResultStyle == ResultStyleFullresult {
					j = append(j, i.Join)
				}
			}
		}

		q, values := BuildQuery(QueryTypeSelect, []SQLField{{FieldName: "*"}},
			SQLTable{TableName: m.TableName, PKField: m.PKField},
			j, filters, "", "", limit)

		//fmt.Println("QUERY:" + q)
		m.lastQuery = q
		m.lastValues = values

		r, err = m.DB.Query(q, values...)
		if err != nil {
			InfoMessage(q)
			return []ResultRow{}, err
		}
	} else {
		m.lastQuery = m.DefaultQuery
		m.lastValues = make([]interface{}, 0)
		r, err = m.DB.Query(m.DefaultQuery)
		if err != nil {
			InfoMessage(m.DefaultQuery)
			return []ResultRow{}, err
		}
	}

	typ, err := r.ColumnTypes()
	if err != nil {
		return []ResultRow{}, err
	}

	fld, err := r.Columns()
	if err != nil {
		return []ResultRow{}, err
	}

	var rrr []ResultRow

	for r.Next() {
		var rr ResultRow
		rr.Values = make([]interface{}, len(typ))
		rr.pointers = make([]interface{}, len(typ))
		rr.Fields = fld

		for i := range typ {
			rr.pointers[i] = &rr.Values[i]
		}

		err = r.Scan(rr.pointers...)
		if err != nil {
			return []ResultRow{}, err
		}

		for i := range rr.Values {
			val, err := constructField(typ[i], rr.Values[i])
			if err != nil {
				return []ResultRow{}, err
			}
			rr.Values[i] = val
		}

		if len(m.Relations) > 0 {
			for _, relation := range m.Relations {
				if relation.ResultStyle == ResultStyleSubresult {
					//PKIndex := rr.GetFieldIndex(m.PKField)
					PKIndex := rr.GetFieldIndex(relation.Join.KeyPair.LocalKey)
					f := make([]Filter, 0)
					//f = append(f, Filter{Field: relation.Join.Foreign_key, Operator: "=", Value: rr.Values[PKIndex]})
					f = append(f, Filter{Field: relation.Join.KeyPair.ForeignKey, Operator: "=", Value: rr.Values[PKIndex]})
					rel_rr, err := relation.Foreign_model.GetRecords(f, 0)
					if err != nil {
						return []ResultRow{}, err
					}
					rr.Subresult = append(rr.Subresult, rel_rr...)
				}
			}
		}

		rrr = append(rrr, rr)
	}

	r.Close()

	return rrr, nil
}

// scanRows is a helper method to scan database rows into ResultRow slice
func (m *Model) scanRows(r *sql.Rows) ([]ResultRow, error) {
	typ, err := r.ColumnTypes()
	if err != nil {
		return []ResultRow{}, err
	}

	fld, err := r.Columns()
	if err != nil {
		return []ResultRow{}, err
	}

	var rrr []ResultRow

	for r.Next() {
		var rr ResultRow
		rr.Values = make([]interface{}, len(typ))
		rr.pointers = make([]interface{}, len(typ))
		rr.Fields = fld

		for i := range typ {
			rr.pointers[i] = &rr.Values[i]
		}

		err = r.Scan(rr.pointers...)
		if err != nil {
			return []ResultRow{}, err
		}

		for i := range rr.Values {
			val, err := constructField(typ[i], rr.Values[i])
			if err != nil {
				return []ResultRow{}, err
			}
			rr.Values[i] = val
		}

		// Handle relations if configured
		if len(m.Relations) > 0 {
			for _, relation := range m.Relations {
				if relation.ResultStyle == ResultStyleSubresult {
					PKIndex := rr.GetFieldIndex(relation.Join.KeyPair.LocalKey)
					if PKIndex >= 0 && PKIndex < len(rr.Values) {
						f := make([]Filter, 0)
						f = append(f, Filter{
							Field:    relation.Join.KeyPair.ForeignKey,
							Operator: "=",
							Value:    rr.Values[PKIndex],
						})
						rel_rr, err := relation.Foreign_model.GetRecords(f, 0)
						if err != nil {
							return []ResultRow{}, err
						}
						rr.Subresult = append(rr.Subresult, rel_rr...)
					}
				}
			}
		}

		rrr = append(rrr, rr)
	}

	return rrr, nil
}

// Execute is function to execute custon query, same like GetRecords
func (m *Model) Execute(q string, values ...interface{}) ([]ResultRow, error) {
	InfoMessage("WARNING: Model.Execute() is deprecated. Use QueryBuilder instead.")

	if m == nil {
		return []ResultRow{}, errors.New("cannot perform action : Execute() on nil model")
	}

	m.lastQuery = q
	m.lastValues = values
	r, err := m.DB.Query(q, values...)

	if err != nil {
		return nil, err
	}

	typ, err := r.ColumnTypes()
	if err != nil {
		return []ResultRow{}, err
	}

	fld, err := r.Columns()
	if err != nil {
		return []ResultRow{}, err
	}

	var rrr []ResultRow

	for r.Next() {
		var rr ResultRow
		rr.Values = make([]interface{}, len(m.Fields))
		rr.pointers = make([]interface{}, len(m.Fields))
		rr.Fields = fld

		for i := range m.Fields {
			rr.pointers[i] = &rr.Values[i]
		}

		r.Scan(rr.pointers...)

		for i := range rr.Values {
			val, err := constructField(typ[i], rr.Values[i])
			if err != nil {
				return []ResultRow{}, err
			}
			rr.Values[i] = val
		}

		rrr = append(rrr, rr)
	}

	return rrr, nil
}

// Deprecated: Execute save query
func (m *Model) Save(fields []SQLField) (bool, error) {
	return m.Insert(fields)
}

// Execute INSERT query
func (m *Model) Insert(fields []SQLField) (bool, error) {
	if m == nil {
		return false, errors.New("cannot perform action : Insert() on nil model")
	}

	q, values := BuildQuery(QueryTypeInsert, fields,
		SQLTable{TableName: m.TableName, PKField: m.PKField}, []SQLJoin{}, []Filter{}, "", "", 0)

	success, err := executeWithContext(m, q, values)
	if err != nil {
		InfoMessage(q)
		return false, err
	}

	if success {
		return true, nil
	}

	InfoMessage(q)
	return false, errors.New("unknown eror occured, check your sql sytax statement")
}

// Execute UPDATE query
func (m *Model) Update(fields []SQLField, id string) (bool, error) {
	if m == nil {
		return false, errors.New("cannot perform action : Update() on nil model")
	}

	q, values := BuildQuery(QueryTypeUpdate, fields,
		SQLTable{TableName: m.TableName, PKField: m.PKField}, []SQLJoin{}, []Filter{{Field: m.PKField, Operator: "=", Value: id}}, "", "", 0)

	success, err := executeWithContext(m, q, values)
	if err != nil {
		InfoMessage(q)
		return false, err
	}

	if success {
		return true, nil
	}

	InfoMessage(q)
	return false, errors.New("unknown eror occured, check your sql sytax statement")
}

// Execute DELETE query
func (m *Model) Delete(id string) (bool, error) {
	if m == nil {
		return false, errors.New("cannot perform action : Delete() on nil model")
	}

	q, values := BuildQuery(QueryTypeDelete, []SQLField{},
		SQLTable{TableName: m.TableName, PKField: m.PKField}, []SQLJoin{}, []Filter{{Field: m.PKField, Operator: "=", Value: id}}, "", "", 0)

	success, err := executeWithContext(m, q, values)
	if err != nil {
		InfoMessage(q)
		return false, err
	}

	if success {
		return true, nil
	}

	InfoMessage(q)
	return false, errors.New("unknown eror occured, check your sql sytax statement")
}

func executeWithContext(m *Model, q string, values []interface{}) (bool, error) {
	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Prepare
	stmt, err := m.DB.Prepare(q)
	if err != nil {
		return false, err
	}

	defer stmt.Close()

	// Execute
	_, err = stmt.ExecContext(ctx, values...)

	if err != nil {
		InfoMessage(q)
		return false, err
	}

	return true, nil
}

// Construct Filed function
func constructField(ct *sql.ColumnType, val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}

	var b []byte
	var n string

	switch v := val.(type) {
	case int:
		n = strconv.FormatInt(val.(int64), 10)
	case int64:
		n = strconv.FormatInt(val.(int64), 10)
	case float64:
		n = strconv.FormatFloat(val.(float64), 'f', 64, 64)
		b = []byte(n)
	case float32:
		n = strconv.FormatFloat(float64(val.(float32)), 'f', 32, 32)
		b = []byte(n)
	case []uint8:
		b = val.([]byte)
		n = string(b)
	default:
		_ = v
		//b = val.([]byte)
	}

	switch ct.DatabaseTypeName() {
	case "BIT":
		return b[0], nil
	case "INT", "TINYINT", "SMALLINT", "MEDIUMINT":
		val, err := strconv.ParseInt(n, 10, 32)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return val, nil
	case "BIGINT":
		val, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return val, nil

	case "FLOAT", "DECIMAL":
		n := string(b)
		val, err := strconv.ParseFloat(n, 32)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return val, nil
	case "DOUBLE":
		n := string(b)
		intVar, err := strconv.ParseFloat(n, 64)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return intVar, nil
	case "CHAR", "VARCHAR", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT", "TEXT", "JSON", "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB":
		return string(b), nil
	case "DATE":
		n := string(b)
		t, err := time.Parse("2006-01-02", n)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return t, nil
	case "DATETIME", "TIMESTAMP":
		n := string(b)
		t, err := time.Parse("2006-01-02 15:04:05", n)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return t, nil
	case "TIME":
		n := string(b)
		t, err := time.Parse("15:04:05", n)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return t, nil
	case "YEAR":
		n := string(b)
		t, err := time.Parse("2006", n)
		if err != nil {
			//fmt.Println(err)
			return nil, err
		}
		return t, nil
	}
	return nil, nil
}

// Build query func
func BuildQuery(queryType QueryType, fields []SQLField, table SQLTable, joins []SQLJoin, wheres []Filter, group string, order string, limit int64) (string, []interface{}) {
	q := ""
	s := ""
	j := ""
	w := ""
	g := ""
	o := ""
	l := ""

	//SELECT
	if len(fields) > 0 {
		for _, fld := range fields {
			s = s + fld.FieldName + ", "
		}
		s = s[:len(s)-2]
	} else {
		s = "*"
	}

	//JOIN
	for _, jn := range joins {
		j = j + " " + string(jn.Join_type) + " JOIN " + jn.Foreign_table + " ON "
		j = j + jn.Foreign_table + "." + jn.KeyPair.ForeignKey + "=" + table.TableName + "." + jn.KeyPair.LocalKey
	}

	//WHERE
	var values = make([]interface{}, 0)
	if len(wheres) > 0 {
		w = " WHERE "
		for _, f := range wheres {
			if len(f.Logic) > 0 {
				w = w + " " + f.Logic + " "
			}
			w = w + "(" + f.Field + " " + f.Operator + " ?)"
			values = append(values, f.Value)
		}
	}

	//GROUP BY
	if len(group) > 0 {
		g = " " + group
	}

	//ORDER
	if len(order) > 0 {
		o = " " + order
	}

	//LIMIT
	if limit > 0 {
		l = " LIMIT " + strconv.FormatInt(int64(limit), 10)
	}

	switch queryType {
	case QueryTypeSelect:
		q = "SELECT " + s + " FROM " + table.TableName + j + w + g + o + l
	case QueryTypeInsert:
		q = "INSERT INTO " + table.TableName + " (" + s + ") VALUES ("
		for _, fld := range fields {
			q = q + "?, "
			values = append(values, fld.Value)
		}
		q = q[:len(q)-2] + ")"

	case QueryTypeUpdate:
		q = "UPDATE " + table.TableName + " SET "
		for _, fld := range fields {
			q = q + fld.FieldName + " = ?, "
			values = append(values, fld.Value)
		}
		v0 := values[0]
		values = values[1:]
		values = append(values, v0)
		q = q[:len(q)-2] + w

	case QueryTypeDelete:
		q = "DELETE FROM " + table.TableName + w
	default:
		q = ""
	}

	return q, values
}

// BuildQueryExtended - improved version with OFFSET and IN clause support
func BuildQueryExtended(queryType QueryType, fields []SQLField, table SQLTable,
	joins []SQLJoin, wheres []Filter, group string, order string,
	limit int64, offset int64) (string, []interface{}) {

	q := ""
	s := ""
	j := ""
	w := ""
	g := ""
	o := ""
	l := ""

	// SELECT
	if len(fields) > 0 {
		fieldNames := make([]string, len(fields))
		for i, fld := range fields {
			fieldNames[i] = fld.FieldName
		}
		s = strings.Join(fieldNames, ", ")
	} else {
		s = "*"
	}

	// JOIN
	for _, jn := range joins {
		j = j + " " + string(jn.Join_type) + " JOIN " + jn.Foreign_table + " ON "
		j = j + jn.Foreign_table + "." + jn.KeyPair.ForeignKey + "=" +
			table.TableName + "." + jn.KeyPair.LocalKey
	}

	// WHERE with IN clause support
	var values = make([]interface{}, 0)
	if len(wheres) > 0 {
		w = " WHERE "
		for i, f := range wheres {
			if i > 0 && len(f.Logic) > 0 {
				w = w + " " + f.Logic + " "
			}

			// Handle IN clause
			if f.Operator == "IN" {
				inValues, ok := f.Value.([]interface{})
				if !ok {
					// Try to convert single value to slice
					inValues = []interface{}{f.Value}
				}

				placeholders := make([]string, len(inValues))
				for j := range inValues {
					placeholders[j] = "?"
					values = append(values, inValues[j])
				}
				w = w + "(" + f.Field + " IN (" + strings.Join(placeholders, ", ") + "))"
			} else {
				w = w + "(" + f.Field + " " + f.Operator + " ?)"
				values = append(values, f.Value)
			}
		}
	}

	// GROUP BY
	if len(group) > 0 {
		g = " " + group
	}

	// ORDER
	if len(order) > 0 {
		o = " " + order
	}

	// LIMIT and OFFSET
	if limit > 0 {
		l = " LIMIT " + strconv.FormatInt(limit, 10)
		if offset > 0 {
			l = l + " OFFSET " + strconv.FormatInt(offset, 10)
		}
	}

	switch queryType {
	case QueryTypeSelect:
		q = "SELECT " + s + " FROM " + table.TableName + j + w + g + o + l
	case QueryTypeInsert:
		fieldNames := make([]string, len(fields))
		placeholders := make([]string, len(fields))
		for i, fld := range fields {
			fieldNames[i] = fld.FieldName
			placeholders[i] = "?"
			values = append(values, fld.Value)
		}
		q = "INSERT INTO " + table.TableName +
			" (" + strings.Join(fieldNames, ", ") + ") VALUES (" +
			strings.Join(placeholders, ", ") + ")"
	case QueryTypeUpdate:
		setParts := make([]string, len(fields))
		for i, fld := range fields {
			setParts[i] = fld.FieldName + " = ?"
			values = append(values, fld.Value)
		}
		q = "UPDATE " + table.TableName + " SET " + strings.Join(setParts, ", ") + w
	case QueryTypeDelete:
		q = "DELETE FROM " + table.TableName + w
	default:
		q = ""
	}

	return q, values
}
