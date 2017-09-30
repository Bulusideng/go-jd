package models

import (
	"strconv"

	"fmt"

	"github.com/astaxie/beego/orm"
)

var (
	skuTbl = "s_k_u_info"
)

// SKUInfo ...
type SKUInfo struct {
	ID         string `orm:"pk"`
	Price      float64
	PriceCnt   int
	Count      int    // buying count
	State      string // stock state 33 : on sale, 34 : out of stock
	StateName  string // "现货" / "无货"
	Name       string
	Link       string
	HistPrices string //p1,p2,p3
	TimeStamp  string
}

func (s *SKUInfo) String() string {
	return fmt.Sprintf("ID:%-12s, Price:%-6.2f, State:%-6s, Name:%-10s, HistPrices:%-50s",
		s.ID, s.Price, s.StateName, s.Name, s.HistPrices)
}

var cnt = 0

func UpdateItem(sku *SKUInfo) (err error) {
	o := orm.NewOrm()
	old := &SKUInfo{
		ID: sku.ID,
	}
	if o.Read(old) == nil { //Exist, update it
		if sku.Price != old.Price {
			sku.HistPrices = old.HistPrices + "," + strconv.FormatFloat(old.Price, 'f', 2, 64)
			sku.PriceCnt++
			fmt.Printf("Price changed %f -> %f\n %s\n", old.Price, sku.Price, sku)
		}
		_, err = o.Update(sku)
	} else { //Not exist, insert it
		_, err = o.Insert(sku)
		if err != nil {
			fmt.Printf("Error insert: %s, found %s", sku, old)
		}
	}
	cnt++
	//if cnt%100 == 0 {
	fmt.Printf("Items updated: %d, %s\n", cnt, sku)

	//}
	return err
}

func GetItem(id string) (*SKUInfo, error) {
	o := orm.NewOrm()

	pat := &SKUInfo{}
	qs := o.QueryTable(skuTbl)
	err := qs.Filter("i_d", id).One(pat)
	return pat, err
}

func GetItems() (skus []*SKUInfo, err error) {
	skus = []*SKUInfo{}
	o := orm.NewOrm()

	qs := o.QueryTable(skuTbl)
	_, err = qs.All(&skus)

	fmt.Printf("Items: %d\n", len(skus))
	for k, item := range skus {
		fmt.Printf("%d: %s\n", k, item)
	}
	return skus, err
}

func GetChanged() (skus []*SKUInfo, err error) {
	skus = []*SKUInfo{}
	o := orm.NewOrm()
	qs := o.QueryTable(skuTbl)
	_, err = qs.Filter("price_cnt__gte", 1).All(&skus)
	fmt.Printf("Changed: %d\n", len(skus))
	for k, item := range skus {
		fmt.Printf("%d: %s\n", k, item)
	}
	return skus, err
}

func DeleteItem(id string) error {
	o := orm.NewOrm()
	pat := &SKUInfo{ID: id}
	if o.Read(pat) == nil {
		if _, err := o.Delete(pat); err != nil {
			return err
		}
	}
	return nil
}
