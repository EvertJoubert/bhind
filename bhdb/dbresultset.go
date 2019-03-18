package bhdb

import (
	"database/sql"
	"reflect"
	"strings"
	"time"
)

type ResultSetMetaData struct {
	cols     []string
	colTypes []*ColumnType
}

func (rsetmeta *ResultSetMetaData) Columns() []string {
	return rsetmeta.cols
}

func (rsetmeta *ResultSetMetaData) ColumnTypes() []*ColumnType {
	return rsetmeta.colTypes
}

type ColumnType struct {
	name string

	hasNullable       bool
	hasLength         bool
	hasPrecisionScale bool

	nullable     bool
	length       int64
	databaseType string
	precision    int64
	scale        int64
	scanType     reflect.Type
}

func (colType *ColumnType) Name() string {
	return colType.name
}

func (colType *ColumnType) Numeric() bool {
	if colType.hasPrecisionScale {
		return true
	} else {
		return strings.Index(colType.databaseType, "CHAR") == -1 && strings.Index(colType.databaseType, "DATE") == -1 && strings.Index(colType.databaseType, "TIME") == -1
	}
}

func (coltype *ColumnType) HasNullable() bool {
	return coltype.hasNullable
}

func (colType *ColumnType) HasLength() bool {
	return colType.hasLength
}

func (colType *ColumnType) HasPrecisionScale() bool {
	return colType.hasPrecisionScale
}

func (colType *ColumnType) Nullable() bool {
	return colType.nullable
}

func (colType *ColumnType) Length() int64 {
	return colType.length
}

func (colType *ColumnType) DatabaseType() string {
	return colType.databaseType
}

func (colType *ColumnType) Precision() int64 {
	return colType.precision
}

func (colType *ColumnType) Scale() int64 {
	return colType.scale
}

func (colType *ColumnType) Type() reflect.Type {
	return colType.scanType
}

type ResultSet struct {
	rsmetadata  *ResultSetMetaData
	stmnt       *Statement
	rset        *sql.Rows
	data        []interface{}
	dispdata    []interface{}
	dataref     []interface{}
	err         error
	dosomething chan bool
}

func (rset *ResultSet) MetaData() *ResultSetMetaData {
	return rset.rsmetadata
}

func (rset *ResultSet) Data() []interface{} {
	go func() {
		for n, _ := range rset.data {
			coltype := rset.rsmetadata.colTypes[n]
			rset.dispdata[n] = castSqlTypeValue(rset.data[n], coltype)
		}
		rset.dosomething <- true
	}()
	<-rset.dosomething
	return rset.dispdata
}

func castSqlTypeValue(valToCast interface{}, colType *ColumnType) (castedVal interface{}) {
	if valToCast != nil {

		if d, dok := valToCast.([]uint8); dok {
			castedVal = string(d)
		} else if sd, dok := valToCast.(string); dok {
			castedVal = sd
		} else if dtime, dok := valToCast.(time.Time); dok {
			castedVal = dtime.Format("2006-01-02T15:04:05")
		} else {
			castedVal = valToCast
		}

	} else {
		castedVal = valToCast
	}
	return castedVal
}

func (rset *ResultSet) Next() (next bool, err error) {
	if next = rset.rset.Next(); next {
		if rset.data == nil {
			rset.data = make([]interface{}, len(rset.rsmetadata.cols))
			rset.dataref = make([]interface{}, len(rset.rsmetadata.cols))
			rset.dispdata = make([]interface{}, len(rset.rsmetadata.cols))
		}

		for n, _ := range rset.data {
			rset.dataref[n] = &rset.data[n]
		}

		if scerr := rset.rset.Scan(rset.dataref...); scerr != nil {
			rset.Close()
			err = scerr
			next = false
		}
	} else {
		if rseterr := rset.rset.Err(); rseterr != nil {
			err = rseterr
		}
		rset.Close()
	}
	return next, err
}

func (rset *ResultSet) Close() {
	if rset.data != nil {
		rset.data = nil
	}
	if rset.dataref != nil {
		rset.dataref = nil
	}
	if rset.dispdata != nil {
		rset.dispdata = nil
	}
	if rset.rsmetadata != nil {
		rset.rsmetadata.colTypes = nil
		rset.rsmetadata.cols = nil
	}
	if rset.stmnt != nil {
		rset.stmnt.Close()
		rset.stmnt = nil
	}
	if rset.dosomething != nil {
		close(rset.dosomething)
		rset.dosomething = nil
	}
}
