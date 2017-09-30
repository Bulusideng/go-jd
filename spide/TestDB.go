package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/Bulusideng/go-jd/core"

	"github.com/Bulusideng/go-jd/core/models"
	//clog "gopkg.in/clog.v1"
)

var test = true

func TestDB() {
	//models.GetItems()
	//models.GetChanged()
	test2()

}

var wg sync.WaitGroup

func test2() {

	ch := make(chan int)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			for {
				foo, ok := <-ch
				if !ok {
					println("done:", i)

					wg.Done()
					return
				}
				time.Sleep(time.Second)
				println(i, ": ", foo)
			}
		}(i)
	}

	ch <- 1
	ch <- 2
	ch <- 3
	ch <- 4
	ch <- 5
	ch <- 6
	close(ch)

	wg.Wait()
}
func TestDBold() {
	c := core.NewDB(false)
	c.FindAll("jditems", false)
	return
	sendMail()
	return
	id := "100"

	item := &models.SKUInfo{
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
