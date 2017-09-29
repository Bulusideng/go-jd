package core

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"log"

	_ "github.com/mattn/go-sqlite3" // sqlite3 dirver
)

var (
	jdItems      = "jditems"
	priceChanges = "priceChanges"
)

type DBSQLite struct {
	db *sql.DB
}

func NewDB(truncate bool) *DBSQLite {
	c, err := connectDB("sqlite3", "jd.db")
	if err != "" {
		panic(err)
	}
	c.Create(truncate)
	return c
}

func connectDB(driverName string, dbName string) (*DBSQLite, string) {
	db, err := sql.Open(driverName, dbName)
	if err != nil {
		return nil, err.Error()
	}
	if err = db.Ping(); err != nil {
		return nil, err.Error()
	}
	return &DBSQLite{db}, ""
}

func (c *DBSQLite) Create(truncate bool) {
	if truncate {
		_, err := c.db.Exec("DROP TABLE jditems")
		if err != nil {
			fmt.Printf("Error dorp table:%s\n", err.Error())
		} else {
			fmt.Println("Truncate success...")
		}
	}

	_, err := c.db.Exec(`CREATE TABLE IF NOT EXISTS jditems(
									ID STRING PRIMARY KEY,
									TimeStamp STRING,
									price FLOAT64,
									priceCnt INTEGER,
									State STRING,
									StateName STRING,
									Name STRING,
									Link STRING,
									HistPrices STRING
									)`)

	if err != nil {
		fmt.Println("Create jditems error:", err.Error())
	}
	_, err = c.db.Exec(`CREATE TABLE IF NOT EXISTS priceChanges(
									ID STRING PRIMARY KEY,
									TimeStamp STRING,
									price FLOAT64,
									priceCnt INTEGER,
									State STRING,
									StateName STRING,
									Name STRING,
									Link STRING,
									HistPrices STRING
									)`)

	if err != nil {
		fmt.Println("Create priceChanges error:", err.Error())
	}
}

func (c *DBSQLite) FindAll(tb string) []*SKUInfo {
	items := []*SKUInfo{}
	stm := fmt.Sprintf("SELECT * FROM %s", tb)
	rows, err := c.db.Query(stm)
	if err != nil {
		fmt.Println("Queryall error:", err.Error())
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		item := &SKUInfo{}
		err := rows.Scan(
			&item.ID,
			&item.TimeStamp,
			&item.Price,
			&item.PriceCnt,
			&item.State,
			&item.StateName,
			&item.Name,
			&item.Link,
			&item.HistPrices)
		if err != nil {
			fmt.Println("FindAll Next error:", err.Error())
		} else {
			items = append(items, item)
			log.Printf("FoundAll %s", item)
		}
	}
	return items
}

func (c *DBSQLite) Find(tb, id string) *SKUInfo {

	stm := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tb)
	rows, err := c.db.Query(stm, id)
	if err != nil {
		fmt.Println("querya error:", err.Error())
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		item := &SKUInfo{}
		err := rows.Scan(
			&item.ID,
			&item.TimeStamp,
			&item.Price,
			&item.PriceCnt,
			&item.State,
			&item.StateName,
			&item.Name,
			&item.Link,
			&item.HistPrices)
		if err != nil {
			fmt.Println("Find Next error:", err.Error())
		} else {
			log.Printf("Found    %s", item)
		}
		return item
	}
	return nil
}

func (c *DBSQLite) Update(sku *SKUInfo) {
	old := c.Find(jdItems, sku.ID)
	if old == nil {
		c.insert(jdItems, sku)
	} else {
		if sku.Price != old.Price {
			sku.HistPrices = old.HistPrices + "," + strconv.FormatFloat(old.Price, 'f', 2, 64)
			sku.PriceCnt++
			c.update(priceChanges, sku)
		}
		c.update(jdItems, sku)
	}
}

func (c *DBSQLite) update(tb string, sku *SKUInfo) {
	stm := fmt.Sprintf(`UPDATE %s SET 
			timeStamp = ?,
			price = ?, 
			priceCnt = ?,
			state = ?,
			stateName = ?,
			name = ?,
			link = ?,
			histPrices = ?
			WHERE id = ?`, tb)

	result, err := c.db.Exec(stm,
		sku.TimeStamp,
		sku.Price,
		sku.PriceCnt,
		sku.State,
		sku.StateName,
		sku.Name,
		sku.Link,
		sku.HistPrices,
		sku.ID)

	if err != nil {
		fmt.Printf("Update %s error:%s\n", tb, err.Error())
	}
	affectNum, err := result.RowsAffected()
	if err != nil {
		fmt.Println(err.Error())
		fmt.Printf("UpdateR %s error:%s\n", tb, err.Error())
	}
	fmt.Println("update %s affect rows: %d\n", tb, affectNum)

}

func (c *DBSQLite) Delete(id string) {
	stmt, err := c.db.Prepare("DELETE FROM jditems WHERE id = ?")
	if err != nil {
		fmt.Println(err.Error())
	}
	result, err := stmt.Exec(id)
	if err != nil {
		fmt.Println(err.Error())
	}
	affectNum, err := result.RowsAffected()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("delete affect rows is ", affectNum)
}

func (c *DBSQLite) insert(tb string, item *SKUInfo) {
	stm := fmt.Sprintf(`INSERT INTO %s
		(id, timeStamp, price, priceCnt, state, stateName, name, link, histPrices) 
		values(?,?,?,?,?,?,?,?,?)`, tb)
	item.TimeStamp = time.Now().Format(time.RFC3339)
	result, err := c.db.Exec(stm,
		item.ID,
		item.TimeStamp,
		item.Price,
		item.PriceCnt,
		item.State,
		item.StateName,
		item.Name,
		item.Link,
		item.HistPrices)

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
