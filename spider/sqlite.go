package main

import (
	"database/sql"
	"fmt"

	"log"

	"github.com/Bulusideng/go-jd/core"

	_ "github.com/mattn/go-sqlite3" // sqlite3 dirver
)

// People - database fields
type People struct {
	id   int
	name string
	age  int
}

type appContext struct {
	db *sql.DB
}

func connectDB(driverName string, dbName string) (*appContext, string) {
	db, err := sql.Open(driverName, dbName)
	if err != nil {
		return nil, err.Error()
	}
	if err = db.Ping(); err != nil {
		return nil, err.Error()
	}
	return &appContext{db}, ""
}

/*
ID         string
	Price      string
	Count      int    // buying count
	State      string // stock state 33 : on sale, 34 : out of stock
	StateName  string // "现货" / "无货"
	Name       string
	Link       string
	HistPrices string //p1,p2,p3
*/
func (c *appContext) Create(item *core.SKUInfo) {
	stmt, err := c.db.Prepare("INSERT INTO jditems(id, price, count, state, stateName, name, link, histPrice) values(?,?,?,?,?,?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	result, err := stmt.Exec(item.ID, item.Price, item.Count, item.State, item.StateName, item.Name, item.Link, item.HistPrices)
	if err != nil {
		fmt.Printf("add error: %v", err)
		return
	}
	lastID, err := result.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("inserted id is ", lastID)
}

// Read
func (c *appContext) Read() *core.SKUInfo {
	rows, err := c.db.Query("SELECT * FROM users")
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		p := new(core.SKUInfo)
		err := rows.Scan(&p.Count)
		if err != nil {
			fmt.Println(err)
		}
		return p

	}
	return nil

}

// UPDATE
func (c *appContext) Update() {
	stmt, err := c.db.Prepare("UPDATE users SET age = ? WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	result, err := stmt.Exec(10, 1)
	if err != nil {
		log.Fatal(err)
	}
	affectNum, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("update affect rows is ", affectNum)
}

// DELETE
func (c *appContext) Delete() {
	stmt, err := c.db.Prepare("DELETE FROM users WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	result, err := stmt.Exec(1)
	if err != nil {
		log.Fatal(err)
	}
	affectNum, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("delete affect rows is ", affectNum)
}

// Mysqlite3 - sqlite3 CRUD
func init1() {
	c, err := connectDB("sqlite3", "jd.db")
	if err != "" {
		print(err)
	}

	//c.Create()
	fmt.Println("add action done!")

	c.Read()
	fmt.Println("get action done!")

	c.Update()
	fmt.Println("update action done!")

	c.Delete()
	fmt.Println("delete action done!")
}
