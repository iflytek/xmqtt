package mysqlMngr

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"net/url"
	"sync"
	"time"
)

type MysqlMngr struct {
	Addr string
	DB   *gorm.DB
}

func NewMysqlMngr(addr string) (*MysqlMngr, error) {
	client := &MysqlMngr{Addr: addr}
	err := client.InitDB()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (m *MysqlMngr) InitDB() error {

	db, err := gorm.Open("mysql", m.Addr)
	if err != nil {
		return err
	}

	// 开启日志
	db.LogMode(true)
	db.SetLogger(m)

	m.DB = db
	m.DB.DB().SetMaxOpenConns(1000)
	m.DB.DB().SetMaxIdleConns(200)
	m.DB.DB().SetConnMaxLifetime(5 * time.Second)
	return nil
}

func (m *MysqlMngr) Fini() {
	_ = m.DB.Close()
}

func (m *MysqlMngr) Print(values ...interface{}) {
	fmt.Println(values...)
}

type MysqlMngr2 struct {
	mysqlAddr     string
	mysqlUserName string
	mysqlPassword string
	mysqlDBName   string
	maxOpenConns  int
	maxIdleConns  int

	db         *sql.DB
	stop       bool
	lockadd    sync.Mutex
	retrygroup sync.WaitGroup
}

func (mm *MysqlMngr2) Init(mysqlAddr, mysqlUserName, mysqlPassword, mysqlDBName string, MaxOpenConns, MaxIdleConns int) (err error) {
	mm.mysqlAddr = mysqlAddr
	mm.mysqlUserName = mysqlUserName
	mm.mysqlPassword = mysqlPassword
	mm.mysqlDBName = mysqlDBName
	mm.maxOpenConns = MaxOpenConns
	mm.maxIdleConns = MaxIdleConns

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&loc=%s&parseTime=true", mm.mysqlUserName, mm.mysqlPassword, mm.mysqlAddr, mm.mysqlDBName, url.QueryEscape("Asia/Shanghai"))
	mm.db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		return err
	}
	mm.db.SetMaxOpenConns(mm.maxOpenConns)
	mm.db.SetMaxIdleConns(mm.maxIdleConns)
	err = mm.db.Ping()
	if err != nil {
		return err
	}
	return nil
}

// 插入
func (mm *MysqlMngr2) Insert(sqlstr string, args ...interface{}) (int64, error) {
	mm.lockadd.Lock()
	defer mm.lockadd.Unlock()
	stmtIns, err := mm.db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	result, err := stmtIns.Exec(args...)
	if err != nil {
		panic(err.Error())
	}
	return result.LastInsertId()
}

// 修改和删除
func (mm *MysqlMngr2) ExecUpdateDelete(sqlstr string, args ...interface{}) (int64, error) {
	mm.lockadd.Lock()
	defer mm.lockadd.Unlock()
	stmtIns, err := mm.db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	result, err := stmtIns.Exec(args...)
	if err != nil {
		panic(err.Error())
	}
	return result.RowsAffected()
}

// 查询多行
func (mm *MysqlMngr2) QueryRows(sqlstr string, args ...interface{}) (*[]map[string]string, error) {
	mm.lockadd.Lock()
	defer mm.lockadd.Unlock()
	stmtOut, err := mm.db.Prepare(sqlstr)
	if err != nil {
		return nil, err
	}

	// Execute the query
	rows, err := stmtOut.Query(args...)
	if err != nil {
		stmtOut.Close()
		return nil, err
	}
	stmtOut.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))

	results := make([]map[string]string, 0)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		result := make(map[string]string, len(scanArgs))

		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col != nil {
				value = string(col)
				result[columns[i]] = value
			}
		}
		results = append(results, result)
	}

	return &results, nil
}

// 查询一行
func (mm *MysqlMngr2) QueryRow(db *sql.DB, sqlstr string, args ...interface{}) (*map[string]string, error) {
	mm.lockadd.Lock()
	defer mm.lockadd.Unlock()
	stmtOut, err := db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	rows, err := stmtOut.Query(args...)
	if err != nil {
		panic(err.Error())
	}

	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error())
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	ret := make(map[string]string, len(scanArgs))

	for i := range values {
		scanArgs[i] = &values[i]
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error())
		}
		var value string

		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			ret[columns[i]] = value
		}
	}
	return &ret, nil
}

func (mm *MysqlMngr2) Fini() {
	mm.db.Close()
}
