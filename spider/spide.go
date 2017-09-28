package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Bulusideng/go-jd/core"
	clog "gopkg.in/clog.v1"
)

func init() {
	if err := clog.New(clog.CONSOLE, clog.ConsoleConfig{
		Level:      clog.INFO,
		BufferSize: 100},
	); err != nil {
		fmt.Printf("init console log failed. error %+v.", err)
		os.Exit(1)
	}
	createJD()
}

const (
	AreaBeijing = "1_72_2799_0"
)

var (
	area   = flag.String("area", AreaBeijing, "ship location string, default to Beijing")
	period = flag.Int("period", 500, "the refresh period when out of stock, unit: ms.")
	rush   = flag.Bool("rush", false, "continue to refresh when out of stock.")
	order  = flag.Bool("order", false, "submit the order to JingDong when get the Goods.")
	cat    = flag.String("MobilePhone", "9987,653,655", "product category.")
	goods  = flag.String("goods", "1482791", `the goods you want to by, find it from JD website. 
	Single Goods:
	  2567304(:1)
	Multiple Goods:
	  2567304(:1),3133851(:2)`)
)

var jd *core.JingDong

func createJD() {
	flag.Parse()

	gs := parseGoods(*goods)
	clog.Trace("[Area: %+v, Goods: %qv, Period: %+v, Rush: %+v, Order: %+v]",
		*area, gs, *period, *rush, *order)

	jd = core.NewJingDong(core.JDConfig{
		Period:     time.Millisecond * time.Duration(*period),
		ShipArea:   *area,
		AutoRush:   *rush,
		AutoSubmit: *order,
	}, "shouji")

	/*
		defer jd.Release()
			//jd.GetGoodInfo()
			for pg := 1; pg < 10; pg++ {
				jd.GetSkuIds("9987,653,655", pg)
			}

			close(jd.SkuIds)

	*/
	jd.GetDetails(10)

}

// parseGoods parse the input goods list. Support to input multiple goods sperated
// by comma(,). With an (:count) after goods ID to specify the count of each goods.
//
// Example as following:
//
//   2567304				single goods with default count 1
//   2567304:3				single goods with count 3
//   2567304,3133851:4		multiple goods with defferent count 1, 4
//   2567304:2,3133851:5	...
//
func parseGoods(goods string) map[string]int {
	lst := make(map[string]int)
	if goods == "" {
		return lst
	}

	for _, good := range strings.Split(goods, ",") {
		pair := strings.Split(good, ":")
		name := strings.Trim(pair[0], " ")
		if len(pair) == 2 {
			v, _ := strconv.ParseInt(pair[1], 10, 32)
			lst[name] = int(v)
		} else {
			lst[name] = 1
		}
	}

	return lst
}
