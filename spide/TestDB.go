package main

import (
	"strconv"

	"github.com/Bulusideng/go-jd/core"
)

var test = true

func TestDB() {

	id := "100"
	c := core.NewDB(true)

	item := &core.SKUInfo{
		ID:    id,
		Price: 999,
	}
	c.Update(item)
	return
	items := c.FindAll()

	for _, item := range items {
		item.Price *= 2
		c.Update(item)
	}
	c.FindAll()

	return

	c.Find(id)

	for i := 0; i < 10; i++ {
		item.Price += float64(i)
		item.ID = strconv.Itoa(i)
		c.Update(item)
		for j := 0; j < 5; j++ {
			item.Price = float64(j)
			c.Update(item)
		}
	}
}
