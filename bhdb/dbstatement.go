package bhdb

import (
	"database/sql"
	"fmt"
)

type Statement struct {
	cn *Connection
	tx *sql.Tx
}

func NewStatement(cn *Connection) (stmnt *Statement, err error) {
	if err = cn.db.Ping(); err == nil {
		stmnt = &Statement{cn: cn}
	}
	return stmnt, err
}

func (stmnt *Statement) Begin() (err error) {
	if tx, txerr := stmnt.cn.db.Begin(); txerr == nil {
		stmnt.tx = tx
	} else {
		err = txerr
	}
	return err
}

func (stmnt *Statement) Execute(query string, args ...interface{}) (lastInsertId int64, rowsAffected int64, err error) {
	if stmnt.tx == nil {
		err = stmnt.Begin()
	}
	if err == nil {
		if r, rerr := stmnt.tx.Exec(query, args...); rerr == nil {
			lastInsertId, err = r.LastInsertId()
			rowsAffected, err = r.RowsAffected()
			r = nil
			err = stmnt.tx.Commit()
		} else {
			err = rerr
		}
	}
	if err != nil {
		err = stmnt.tx.Rollback()
	}
	return lastInsertId, rowsAffected, err
}

func (stmnt *Statement) Query(query string, args ...interface{}) (rset *ResultSet, err error) {
	if stmnt.tx == nil {
		err = stmnt.Begin()
	}
	if rs, rserr := stmnt.tx.Query(query, args...); rserr == nil {
		if cols, colserr := rs.Columns(); colserr == nil {
			for n, col := range cols {
				if col == "" {
					cols[n] = "COLUMN" + fmt.Sprint(n+1)
				}
			}
			if coltypes, coltypeserr := rs.ColumnTypes(); coltypeserr == nil {
				rset = &ResultSet{rset: rs, stmnt: stmnt, rsmetadata: &ResultSetMetaData{cols: cols[:], colTypes: columnTypes(coltypes[:])}, dosomething: make(chan bool, 1)}
			} else {
				err = coltypeserr
			}
		} else {
			err = colserr
		}
	} else {
		stmnt.tx.Rollback()
		err = rserr
	}
	return rset, err
}

func (stmnt *Statement) Close() {
	if stmnt.tx != nil {
		stmnt.tx.Commit()
		stmnt.tx = nil
	}
	if stmnt.cn != nil {
		stmnt.cn = nil
	}
}

func columnTypes(sqlcoltypes []*sql.ColumnType) (coltypes []*ColumnType) {
	coltypes = make([]*ColumnType, len(sqlcoltypes))
	for n, ctype := range sqlcoltypes {
		coltype := &ColumnType{}
		coltype.databaseType = ctype.DatabaseTypeName()
		coltype.length, coltype.hasLength = ctype.Length()
		coltype.name = ctype.Name()
		coltype.databaseType = ctype.DatabaseTypeName()
		coltype.nullable, coltype.hasNullable = ctype.Nullable()
		coltype.precision, coltype.scale, coltype.hasPrecisionScale = ctype.DecimalSize()
		coltype.scanType = ctype.ScanType()
		coltypes[n] = coltype
	}
	return coltypes
}
