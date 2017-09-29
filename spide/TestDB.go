package main

import (
	"strconv"

	"github.com/Bulusideng/go-jd/core"
)

var test = false

func TestDB() {
	c := core.NewDB(false)
	c.FindAll("jditems", false)
	return
	sendMail()
	return
	id := "100"

	item := &core.SKUInfo{
		ID:    id,
		Price: 999,
	}
	c.Update(item)
	return
	items := c.FindAll("jditems", false)

	for _, item := range items {
		item.Price *= 2
		c.Update(item)
	}
	c.FindAll("jditems", false)

	return

	c.Find("jditems", id)

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
