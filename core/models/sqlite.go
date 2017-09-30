package models

import (
	"os"
	"path"

	"github.com/Unknwon/com"
	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	var _DB_NAME, _DB_DRIVER string
	var dtype orm.DriverType
	orm.RegisterModel(&SKUInfo{})

	dtype = orm.DRSqlite
	_DB_DRIVER = "sqlite3"
	_DB_NAME = "DB/JDItems.db"
	if !com.IsExist(_DB_NAME) {
		os.MkdirAll(path.Dir(_DB_NAME), os.ModePerm)
		os.Create(_DB_NAME)
	}

	orm.RegisterDriver(_DB_DRIVER, dtype)
	orm.RegisterDataBase("default", _DB_DRIVER, _DB_NAME, 10)

	orm.Debug = false
	orm.DefaultRowsLimit = -1
	orm.RunSyncdb("default", false, true)

}
