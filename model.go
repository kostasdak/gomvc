package gomvc

import (
	"database/sql"
	"errors"
	"strconv"
	"time"
)

type Model struct {
	DB        *sql.DB
	IdField   string
	TableName string
	Fields    []string
}

type ResultRow struct {
	Values   []interface{}
	pointers []interface{}
}

func (m *Model) Instance() Model {
	return *m
}

//Pass initial Parammeters
func (m *Model) InitModel(db *sql.DB, tableName string, idField string) error {
	m.DB = db
	m.TableName = tableName
	m.IdField = idField

	var q = "SHOW COLUMNS FROM " + tableName
	r, err := m.DB.Query(q)
	if err != nil {
		return err
	}

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

	return nil
}

//Get last id from Table
func (m *Model) GetLastId() (int64, error) {
	if m == nil {
		return 0, errors.New("cannot perform action : GetLastId() on nil model")
	}
	var q = "SELECT " + m.IdField + " FROM " + m.TableName + " ORDER BY " + m.IdField + " DESC LIMIT 1"
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

//Query table by Primary Key
func (m *Model) GetRecordByPK(id int64) (ResultRow, error) {
	if m == nil {
		return ResultRow{}, errors.New("cannot perform action : GetRecordByPK() on nil model")
	}
	var q = "SELECT * FROM " + m.TableName + " WHERE " + m.IdField + "=" + strconv.FormatInt(int64(id), 10) + " LIMIT 1"
	r, err := m.DB.Query(q)
	if err != nil {
		return ResultRow{}, err
	}

	r.Next()

	var rr ResultRow
	rr.Values = make([]interface{}, len(m.Fields))
	rr.pointers = make([]interface{}, len(m.Fields))
	for i := range m.Fields {
		rr.pointers[i] = &rr.Values[i]
	}

	typ, _ := r.ColumnTypes()

	err = r.Scan(rr.pointers...)
	if err != nil {
		return ResultRow{}, err
	}

	for i := range rr.Values {
		val, err := constructField(typ[i], rr.Values[i])
		if err != nil {
			return ResultRow{}, err
		}
		rr.Values[i] = val
	}

	return rr, nil
}

//Query table and get all rows
func (m *Model) GetAllRecords(limit int64) ([]ResultRow, error) {
	if m == nil {
		return []ResultRow{}, errors.New("cannot perform action : GetAllRecords() on nil model")
	}

	var q string

	if limit > 0 {
		q = "SELECT * FROM " + m.TableName + " WHERE 1=1 LIMIT " + strconv.FormatInt(int64(limit), 10)
	} else {
		q = "SELECT * FROM " + m.TableName + " WHERE 1=1"
	}

	r, err := m.DB.Query(q)
	if err != nil {
		return []ResultRow{}, err
	}

	typ, _ := r.ColumnTypes()
	var rrr []ResultRow

	for r.Next() {
		var rr ResultRow
		rr.Values = make([]interface{}, len(m.Fields))
		rr.pointers = make([]interface{}, len(m.Fields))

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

func (m *Model) GetRecords2(filters []string, limit int64) ([]ResultRow, error) {
	if m == nil {
		return []ResultRow{}, errors.New("cannot perform action : GetRecords() on nil model")
	}
	//field=value
	//field=*value
	//field=*value*

	return []ResultRow{}, nil
}

//Execute custon query
func (m *Model) Execute(q string) ([]ResultRow, error) {
	if m == nil {
		return []ResultRow{}, errors.New("cannot perform action : Execute() on nil model")
	}

	r, err := m.DB.Query(q)
	if err != nil {
		return nil, err
	}

	typ, _ := r.ColumnTypes()

	var rrr []ResultRow

	for r.Next() {
		var rr ResultRow
		rr.Values = make([]interface{}, len(m.Fields))
		rr.pointers = make([]interface{}, len(m.Fields))

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

//Execute save query
func (m *Model) Save(vals map[string]string) (bool, error) {
	if m == nil {
		return false, errors.New("cannot perform action : Save() on nil model")
	}

	var q = "INSERT INTO " + m.TableName + " ("

	if len(vals) > 0 {
		var values = make([]interface{}, 0)
		for k, v := range vals {
			q = q + k + ","
			values = append(values, v)
		}
		q = string(q[:len(q)-1]) + ") VALUES ("
		for i := 0; i < len(vals); i++ {
			q = q + "?,"
		}

		q = string(q[:len(q)-1]) + ")"

		stmt, err := m.DB.Prepare(q)
		if err != nil {
			return false, err
		}

		defer stmt.Close()

		if err != nil {
			return false, err
		}

		stmt.Exec(values...)

		return true, nil
	}

	return false, nil
}

//Execute update query
func (m *Model) Update(vals map[string]string, id string) (bool, error) {
	if m == nil {
		return false, errors.New("cannot perform action : Update() on nil model")
	}

	var q = "UPDATE " + m.TableName + " SET "

	if len(vals) > 0 && len(id) > 0 {
		var values = make([]interface{}, 0)
		for k, v := range vals {
			q = q + k + " = ?, "
			values = append(values, v)
		}

		q = string(q[:len(q)-2]) + " WHERE " + m.IdField + "=" + id

		stmt, err := m.DB.Prepare(q)
		if err != nil {
			return false, err
		}

		defer stmt.Close()

		if err != nil {
			return false, err
		}

		stmt.Exec(values...)

		return true, nil
	}

	return false, nil
}

//Execute delete query
func (m *Model) Delete(id string) (bool, error) {
	if m == nil {
		return false, errors.New("cannot perform action : Delete() on nil model")
	}

	var q = "DELETE FROM " + m.TableName + " WHERE " + m.IdField + "=" + id

	stmt, err := m.DB.Prepare(q)
	if err != nil {
		return false, err
	}

	defer stmt.Close()

	if err != nil {
		return false, err
	}

	stmt.Exec()

	return true, nil
}

//Construct Filed function
func constructField(ct *sql.ColumnType, val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	b := val.([]byte)
	n := string(b)

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
