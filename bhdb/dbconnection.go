package bhdb

import (
	"database/sql"
	"strings"
)

type Connection struct {
	driver         string
	datasourcename string
	db             *sql.DB
	dbmngr         *DbManager
}

func (cn *Connection) Execute(query string, args ...interface{}) (lastInsertId int64, rowsAffected int64, err error) {
	stmnt := &Statement{cn: cn}
	lastInsertId, rowsAffected, err = stmnt.Execute(query, args...)
	stmnt = nil
	return lastInsertId, rowsAffected, err
}

func (cn *Connection) ParseQuery(query string) (parsedquery string, params []string) {
	startParam := false
	startText := false
	pname := ""
	for n, _ := range query {
		c := string(query[n])
		if startParam {
			if strings.TrimSpace(c) == "" || strings.Index("[](),@$%&|!<>$*+-'", c) > -1 {
				if pname != "" {
					if params == nil {
						params = []string{}
					}
					params = append(params, pname)
					pname = ""
					parsedquery = parsedquery + "?" + c
				} else {
					parsedquery = parsedquery + ":" + c
				}
				startParam = false

			} else {
				pname = pname + c
			}
		} else {
			if c == "'" {
				if startText {
					startText = false
				} else {
					startText = true
				}
			}
			if !startParam {
				if !startText {
					if !startParam && c == ":" && n < len(query)-1 {
						if strings.TrimSpace(c) != "" && strings.Index("[](),@$%&|!<>$*+-'", c) == -1 {
							startParam = true
						} else {
							parsedquery = parsedquery + c
						}
					} else {
						parsedquery = parsedquery + c
					}
				} else {
					parsedquery = parsedquery + c
				}
			}
		}
	}

	if startParam {
		if pname != "" {
			if params == nil {
				params = []string{}
			}
			params = append(params, pname)
			pname = ""
			parsedquery = parsedquery + "?"
		} else {
			parsedquery = parsedquery + ":"
		}
		startParam = false
	}
	return parsedquery, params
}

func (cn *Connection) Query(query string, args ...interface{}) (rset *ResultSet, err error) {
	stmnt := &Statement{cn: cn}
	rset, err = stmnt.Query(query, args...)
	return rset, err
}

func openDB(drvr string, datasourcename string) (driver string, db *sql.DB, err error) {
	driver = drvr
	db, err = sql.Open(driver, datasourcename)
	if err = db.Ping(); err != nil {
		db = nil
	}
	return driver, db, err
}

func openConnection(dbMngr *DbManager, driver string, datasourcename string) (cn *Connection, err error) {
	if driver, db, dberr := openDB(driver, datasourcename); dberr == nil {
		cn = &Connection{dbmngr: dbMngr, driver: driver, datasourcename: datasourcename, db: db}
	} else {
		err = dberr
	}
	return cn, err
}
