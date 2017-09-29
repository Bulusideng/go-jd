package core

import (
	"bytes"

	"fmt"

	"net/url"

	"strconv"
	"strings"

	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	sjson "github.com/bitly/go-simplejson"
	clog "gopkg.in/clog.v1"
)

// SKUInfo ...
type SKUInfo struct {
	ID         string
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

func (jd *JingDong) GetSkuIds(cat string, page int) error {
	data, err := jd.downloader.GetResponse("GET", URLCatList, func(URL string) string {
		u, _ := url.Parse(URLCatList)
		q := u.Query()
		q.Set("cat", cat)
		q.Set("page", strconv.Itoa(page))
		u.RawQuery = q.Encode()
		return u.String()
	})

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(data))
	if err != nil {
		clog.Error(0, "响应解析失败: %+v", err)
		return err
	}
	doc.Find("div.gl-i-wrap.j-sku-item").Each(func(i int, s *goquery.Selection) {
		id, _ := s.Attr("data-sku")
		clog.Info("item:%s", id)
		//jd.getPrice(id)
		//skuIds = append(skuIds, id)
		jd.SkuIds <- id

	})
	return nil
}

// getPrice return sku price by ID
//
//  [{"id":"J_5105046","p":"1999.00","m":"9999.00","op":"1999.00","tpp":"1949.00"}]
//
func (jd *JingDong) getPrice(ID string) (string, error) {
	data, err := jd.downloader.GetResponse("GET", URLGoodsPrice, func(URL string) string {
		u, _ := url.Parse(URLGoodsPrice)
		q := u.Query()
		q.Set("type", "1")
		q.Set("skuIds", "J_"+ID)
		q.Set("pduid", strconv.FormatInt(time.Now().Unix()*1000, 10))
		u.RawQuery = q.Encode()
		return u.String()
	})

	if err != nil {
		clog.Error(0, "获取商品（%s）价格失败: %+v", ID, err)
		return "", err
	}

	var js *sjson.Json
	if js, err = sjson.NewJson(data); err != nil {
		clog.Info("Response Data: %s", data)
		clog.Error(0, "解析响应数据失败: %+v", err)
		return "", err
	}

	return js.GetIndex(0).Get("p").String()
}

// stockState return stock state
// http://c0.3.cn/stock?skuId=531065&area=1_72_2799_0&cat=1,1,1&buyNum=1
// http://c0.3.cn/stock?skuId=531065&area=1_72_2799_0&cat=1,1,1
// https://c0.3.cn/stocks?type=getstocks&skuIds=4099139&area=1_72_2799_0&_=1499755881870
//
// {"3133811":{"StockState":33,"freshEdi":null,"skuState":1,"PopType":0,"sidDely":"40",
//	"channel":1,"StockStateName":"现货","rid":null,"rfg":0,"ArrivalDate":"",
//  "IsPurchase":true,"rn":-1}}
func (jd *JingDong) stockState(ID string) (string, string, error) {
	data, err := jd.downloader.GetResponse("GET", URLSKUState, func(URL string) string {
		u, _ := url.Parse(URL)
		q := u.Query()
		q.Set("type", "getstocks")
		q.Set("skuIds", ID)
		q.Set("area", jd.ShipArea)
		q.Set("_", strconv.FormatInt(time.Now().Unix()*1000, 10))
		//q.Set("cat", "1,1,1")
		//q.Set("buyNum", strconv.Itoa(1))
		u.RawQuery = q.Encode()
		return u.String()
	})

	if err != nil {
		clog.Error(0, "获取商品（%s）库存失败: %+v", ID, err)
		return "", "", err
	}

	// return GBK encoding
	dec := mahonia.NewDecoder("gbk")
	decString := dec.ConvertString(string(data))
	//clog.Trace(decString)

	var js *sjson.Json
	if js, err = sjson.NewJson([]byte(decString)); err != nil {
		clog.Info("Response Data: %s", data)
		clog.Error(0, "解析库存数据失败: %+v", err)
		return "", "", err
	}

	//if sku, exist := js.CheckGet("stock"); exist {
	if sku, exist := js.CheckGet(ID); exist {
		skuState, _ := sku.Get("StockState").Int()
		skuStateName, _ := sku.Get("StockStateName").String()
		return strconv.Itoa(skuState), skuStateName, nil
	}

	return "", "", fmt.Errorf("无效响应数据")
}

// skuDetail get sku detail information
//
func (jd *JingDong) skuDetail(ID string) (*SKUInfo, error) {
	g := &SKUInfo{ID: ID}

	// response context encoding by GBK
	//
	itemURL := fmt.Sprintf("http://item.jd.com/%s.html", ID)
	data, err := jd.downloader.GetResponse("GET", itemURL, nil)
	if err != nil {
		clog.Error(0, "获取商品页面失败: %+v", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(data))
	if err != nil {
		clog.Error(0, "解析商品页面失败: %+v", err)
		return nil, err
	}

	if link, exist := doc.Find("a#InitCartUrl").Attr("href"); exist {
		g.Link = link
		if !strings.HasPrefix(link, "https:") {
			g.Link = "https:" + link
		}
	}

	dec := mahonia.NewDecoder("gbk")
	//rd := dec.NewReader()
	name := dec.ConvertString(doc.Find("div.sku-name").Text())
	g.Name = strings.Trim(name, " \t\n")
	//g.Name = truncate(g.Name)

	price, _ := jd.getPrice(ID)
	g.Price, _ = strconv.ParseFloat(price, 64)
	g.State, g.StateName, _ = jd.stockState(ID)

	info := fmt.Sprintf("SKU Info 编号: %s, 库存: %s, 价格: %s, %f, 名称: %s:%s, 链接: %s",
		g.ID, g.StateName, price, g.Price, name, g.Name, g.Link)
	clog.Info(info)

	jd.db.Update(g)

	return g, nil
}

func (jd *JingDong) GetPrices() {
	for skuId := range jd.SkuIds {
		if p, err := jd.getPrice(skuId); err != nil {
			clog.Info("error ...")
		} else {
			clog.Info("%s, price:%s", skuId, p)
		}
	}
}
func (jd *JingDong) getDetail(id int) {
	clog.Info("Worker %d start", id)
	for skuId := range jd.SkuIds {
		clog.Info("get info for %s", skuId)
		if _, err := jd.skuDetail(skuId); err != nil {
			clog.Info("error ...")
		} else {

		}
	}
	clog.Info("Worker %d exit", id)
}

func (jd *JingDong) Start(threads int) {
	for i := 0; i < threads; i++ {
		go jd.getDetail(i)
	}

}
