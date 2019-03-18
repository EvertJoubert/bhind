package bhdb

import (
	"strings"
	"sync"

	"../bhio"

	// "github.com/HouzuoGuo/tiedot/db"
	_ "github.com/HouzuoGuo/tiedot/dberr"
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx"
	_ "github.com/syndtr/goleveldb/leveldb"
)

type DbManager struct {
	dbaliases map[string]*Connection
}

func (dbmngr *DbManager) Open(alias string, driver string, datasourcename string) (cn *Connection, err error) {
	if dbmngr.dbaliases == nil {
		dbmngr.dbaliases = make(map[string]*Connection)
	}
	if cn, _ = dbmngr.dbaliases[alias]; cn == nil {
		if cn, err = openConnection(dbmngr, driver, datasourcename); err == nil {
			dbmngr.dbaliases[alias] = cn
		}
	}
	return cn, err
}

func (dbmngr *DbManager) RegisterConnection(alias string, driver string, datasourcename string) (err error) {
	_, err = dbmngr.Open(alias, driver, datasourcename)
	return err
}

var dbmngr *DbManager
var dbmngrlock *sync.Mutex = &sync.Mutex{}

func DatabaseManager() *DbManager {
	if dbmngr == nil {
		dbmngrlock.Lock()
		if dbmngr == nil {
			dbmngr = &DbManager{}
		}
		dbmngrlock.Unlock()
	}
	return dbmngr
}

func (dbmngr *DbManager) Connnection(alias string) (cn *Connection) {
	if len(dbmngr.dbaliases) > 0 {
		cn, _ = dbmngr.dbaliases[alias]
	}
	return cn
}

func OutputResultSet(out *bhio.BhIORW, name string, contentext string, res *ResultSet, err error, setting ...string) {
	if err == nil {
		if contentext == ".js" || contentext == ".json" {
			if contentext == ".js" {
				out.Print("var dataset_" + name + "=")
			}
			out.Print("[")
		} else if contentext == ".csv" {
		}
		for ci, col := range res.MetaData().Columns() {
			if contentext == ".js" || contentext == ".json" {
				if ci == 0 {
					out.Print("[")
				}
				if strings.Index(col, "\"") > -1 || strings.Index(col, "'") > -1 {
					if strings.Index(col, "\"") > -1 {
						col = strings.Replace(col, "\"", "\\\"", -1)
					}
					if strings.Index(col, "'") > -1 {
						col = strings.Replace(col, "'", "\\'", -1)
					}
				}
				out.Print("\"" + col + "\"")
				if ci == len(res.MetaData().Columns())-1 {
					out.Print("]")
				} else {
					out.Print(",")
				}
			} else if contentext == ".csv" {
				if strings.Index(col, "\"") > -1 || strings.Index(col, "'") > -1 {
					if strings.Index(col, "\"") > -1 {
						col = strings.Replace(col, "\"", "\"\"", -1)
					}
				}
				out.Print("\"" + col + "\"")
				if ci == len(res.MetaData().Columns())-1 {
					out.Print("\r\n")
				} else {
					out.Print(",")
				}
			}
		}
		if contentext == ".js" || contentext == ".json" {
			out.Print(",[")
		}
		foundResc := false
		for {
			if next, nexterr := res.Next(); next {
				for n, a := range res.Data() {
					if contentext == ".js" || contentext == ".json" {
						if n == 0 {
							if foundResc {
								out.Print(",[")
							} else {
								out.Print("[")
							}
						}
						if a == nil {
							if contentext == ".json" {
								out.Print("null")
							} else {
								out.Print("\"\"")
							}
						} else if a == "" {
							out.Print("\"\"")
						} else {
							if sa, oks := a.(string); oks {
								if res.MetaData().ColumnTypes()[n].Numeric() {
									out.Print(sa)
								} else {
									out.Print("\"", strings.Replace(strings.Replace(sa, "\"", "\\\"", -1), "'", "\\'", -1), "\"")
								}
							} else {
								out.Print(a)
							}
						}
						if n == len(res.MetaData().Columns())-1 {
							out.Print("]")
						} else {
							out.Print(",")
						}
					} else if contentext == ".csv" {
						if a != nil {
							if sa, oks := a.(string); oks {
								if sa != "" {
									if res.MetaData().ColumnTypes()[n].Numeric() {
										out.Print(sa)
									} else {
										out.Print("\"", strings.TrimSpace(strings.Replace(sa, "\"", "\"\"", -1)), "\"")
									}
								}
							}
						} else {
							out.Print(a)
						}

						if n == len(res.MetaData().Columns())-1 {
							out.Print("\r\n")
						} else {
							out.Print(",")
						}
					}
				}
				if !foundResc {
					foundResc = true
				}
			} else {
				if nexterr != nil {
					out.Print(nexterr.Error())
				}
				break
			}
		}
		if contentext == ".js" || contentext == ".json" {
			out.Print("]]")
			if contentext == ".js" {
				out.Println(";")
			}
		} else if contentext == "csv" {
		}

	} else {
		out.Print(err.Error())
	}
}

type DBExecuted struct {
	LastInsertId int64
	RowsAffected int64
	Err          error
}

func (dbmngr *DbManager) Execute(alias string, query string, args ...interface{}) (dbexecuted *DBExecuted) {
	if cn := dbmngr.Connnection(alias); cn != nil {
		dbexecuted = &DBExecuted{}
		dbexecuted.LastInsertId, dbexecuted.RowsAffected, dbexecuted.Err = cn.Execute(query, args...)
	}
	return dbexecuted
}

type DBQuery struct {
	RSet *ResultSet
	Err  error
}

func (dbqry *DBQuery) PrintResult(out *bhio.BhIORW, name string, contentext string, setting ...string) {
	OutputResultSet(out, name, contentext, dbqry.RSet, dbqry.Err, setting...)
}

func (dbqry *DBQuery) MetaData() *ResultSetMetaData {
	if dbqry.RSet == nil {
		return nil
	}
	return dbqry.RSet.MetaData()
}

func (dbqry *DBQuery) Data() []interface{} {
	if dbqry.RSet == nil {
		return nil
	}
	return dbqry.RSet.Data()
}

func (dbqry *DBQuery) Next() bool {
	if dbqry.RSet == nil {
		return false
	}
	next, err := dbqry.RSet.Next()
	if err != nil {
		dbqry.Err = err
		dbqry.RSet = nil
	}
	return next
}

func (dbmngr *DbManager) Query(alias string, query string, args ...interface{}) (dbquery *DBQuery) {
	if cn := dbmngr.Connnection(alias); cn != nil {
		dbquery = &DBQuery{}
		dbquery.RSet, dbquery.Err = cn.Query(query, args...)
	}
	return dbquery
}
