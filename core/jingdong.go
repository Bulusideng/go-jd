package core

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"

	"net/http"
	"net/url"

	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	sjson "github.com/bitly/go-simplejson"
	clog "gopkg.in/clog.v1"
)

const (
	//URLSKUState    = "http://c0.3.cn/stock"
	URLSKUState    = "https://c0.3.cn/stocks"
	URLGoodsDets   = "http://item.jd.com/%s.html"
	URLGoodsPrice  = "http://p.3.cn/prices/mgets"
	URLAdd2Cart    = "https://cart.jd.com/gate.action"
	URLChangeCount = "http://cart.jd.com/changeNum.action"
	URLCartInfo    = "https://cart.jd.com/cart.action"
	URLOrderInfo   = "http://trade.jd.com/shopping/order/getOrderInfo.action"
	URLSubmitOrder = "http://trade.jd.com/shopping/order/submitOrder.action"
	URLCatList     = "https://list.jd.com/list.html"
)

var (
	// URLForQR is the login related URL
	//
	URLForQR = [...]string{
		"https://passport.jd.com/new/login.aspx",
		"https://qr.m.jd.com/show",
		"https://qr.m.jd.com/check",
		"https://passport.jd.com/uc/qrCodeTicketValidation",
		"http://home.jd.com/getUserVerifyRight.action",
	}

	DefaultHeaders = map[string]string{
		"User-Agent":      "Chrome/51.0.2704.103",
		"ContentType":     "application/json", //"text/html; charset=utf-8",
		"Connection":      "keep-alive",
		"Accept-Encoding": "gzip, deflate",
		"Accept-Language": "zh-CN,zh;q=0.8",
	}

	maxNameLen   = 40
	cookieFile   = "jd.cookies"
	qrCodeFile   = "jd.qr"
	strSeperater = strings.Repeat("+", 60)
)

// JDConfig ...
type JDConfig struct {
	Period     time.Duration // refresh period
	ShipArea   string        // shipping area
	AutoRush   bool          // continue rush when out of stock
	AutoSubmit bool          // whether submit the order
}

// JingDong wrap jing dong operation
type JingDong struct {
	JDConfig
	downloader *Downloader
	jar        *SimpleJar
	token      string
	itemType   string
	SkuIds     chan string
	db         *DBSQLite
}

// NewJingDong create an object to wrap JingDong related operation
//
func NewJingDong(option JDConfig, itemType string) *JingDong {
	jd := &JingDong{
		JDConfig: option,
		itemType: itemType,
		SkuIds:   make(chan string, 10000),
		db:       NewDB(true),
	}

	jd.jar = NewSimpleJar(JarOption{
		JarType:  JarJson,
		Filename: cookieFile,
	})

	if err := jd.jar.Load(); err != nil {
		clog.Error(0, "加载Cookies失败: %s", err)
		jd.jar.Clean()
	}

	jd.downloader = &Downloader{
		&http.Client{
			Timeout: time.Second * 5,
			Jar:     jd.jar,
		},
	}

	return jd
}

// Release the resource opened
//
func (jd *JingDong) Release() {
	if jd.jar != nil {
		if err := jd.jar.Persist(); err != nil {
			clog.Error(0, "Failed to persist cookiejar. error %+v.", err)
		}
	}
}

//
//
func truncate(str string) string {
	rs := []rune(str)
	if len(rs) > maxNameLen {
		return string(rs[:maxNameLen-1]) + "..."
	}

	return str
}

// if response data compressed by gzip, unzip first
//
func responseData(resp *http.Response) []byte {
	if resp == nil {
		return nil
	}

	var reader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		//clog.Trace("Encoding: %+v", resp.Header.Get("Content-Encoding"))
		reader, _ = gzip.NewReader(resp.Body)
	default:
		reader = resp.Body
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		clog.Error(0, "读取响应数据失败: %+v", err)
		return nil
	}

	return data
}

//
//
func applyCustomHeader(req *http.Request, header map[string]string) {
	if req == nil || len(header) == 0 {
		return
	}

	for key, val := range header {
		req.Header.Set(key, val)
	}
}

// CartDetails get the shopping cart details
//
func (jd *JingDong) CartDetails() error {
	clog.Info(strSeperater)
	clog.Info("购物车详情>")

	var (
		err  error
		req  *http.Request
		resp *http.Response
		doc  *goquery.Document
	)

	if req, err = http.NewRequest("GET", URLCartInfo, nil); err != nil {
		clog.Error(0, "请求（%+v）失败: %+v", URLCartInfo, err)
		return err
	}

	if resp, err = jd.downloader.Do(req); err != nil {
		clog.Error(0, "获取购物车详情错误: %+v", err)
		return err
	}

	defer resp.Body.Close()
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		clog.Error(0, "分析购物车页面错误: %+v.", err)
		return err
	}

	clog.Info("购买  数量  价格      总价      编号      商品")
	cartFormat := "%-6s%-6s%-10s%-10s%-10s%s"

	doc.Find("div.item-form").Each(func(i int, p *goquery.Selection) {
		check := " -"
		checkTag := p.Find("div.cart-checkbox input").Eq(0)
		if _, exist := checkTag.Attr("checked"); exist {
			check = " +"
		}

		count := "0"
		countTag := p.Find("div.quantity-form input").Eq(0)
		if val, exist := countTag.Attr("value"); exist {
			count = val
		}

		pid := ""
		hrefTag := p.Find("div.p-img a").Eq(0)
		if href, exist := hrefTag.Attr("href"); exist {
			// http://item.jd.com/2967929.html
			pos1 := strings.LastIndex(href, "/")
			pos2 := strings.LastIndex(href, ".")
			pid = href[pos1+1 : pos2]
		}

		price := strings.Trim(p.Find("div.p-price strong").Eq(0).Text(), " ")
		total := strings.Trim(p.Find("div.p-sum strong").Eq(0).Text(), " ")
		gname := strings.Trim(p.Find("div.p-name a").Eq(0).Text(), " \n\t")
		gname = truncate(gname)
		clog.Info(cartFormat, check, count, price, total, pid, gname)
	})

	totalCount := strings.Trim(doc.Find("div.amount-sum em").Eq(0).Text(), " ")
	totalValue := strings.Trim(doc.Find("span.sumPrice em").Eq(0).Text(), " ")
	clog.Info("总数: %s", totalCount)
	clog.Info("总额: %s", totalValue)

	return nil
}

// OrderInfo shows the order detail information
//
func (jd *JingDong) OrderInfo() error {
	var (
		err  error
		req  *http.Request
		resp *http.Response
		doc  *goquery.Document
	)

	clog.Info(strSeperater)
	clog.Info("订单详情>")

	u, _ := url.Parse(URLOrderInfo)
	q := u.Query()
	q.Set("rid", strconv.FormatInt(time.Now().Unix()*1000, 10))
	u.RawQuery = q.Encode()

	if req, err = http.NewRequest("GET", u.String(), nil); err != nil {
		clog.Error(0, "请求（%+v）失败: %+v", URLCartInfo, err)
		return err
	}

	if resp, err = jd.downloader.Do(req); err != nil {
		clog.Error(0, "获取订单页错误: %+v", err)
		return err
	}

	defer resp.Body.Close()
	if doc, err = goquery.NewDocumentFromReader(resp.Body); err != nil {
		clog.Error(0, "分析订单页错误: %+v.", err)
		return err
	}

	//h, _ := doc.Find("div.order-summary").Html()
	//clog.Trace("订单页：%s", h)

	if order := doc.Find("div.order-summary").Eq(0); order != nil {
		warePrice := strings.Trim(order.Find("#warePriceId").Text(), " \t\n")
		shipPrice := strings.Trim(order.Find("#freightPriceId").Text(), " \t\n")
		clog.Info("总金额: %s", warePrice)
		clog.Info("　运费: %s", shipPrice)

	}

	if sum := doc.Find("div.trade-foot").Eq(0); sum != nil {
		payment := strings.Trim(sum.Find("#sumPayPriceId").Text(), " \t\n")
		phone := strings.Trim(sum.Find("#sendMobile").Text(), " \t\n")
		addr := strings.Trim(sum.Find("#sendAddr").Text(), " \t\n")
		clog.Info("应付款: %s", payment)
		clog.Info("%s", phone)
		clog.Info("%s", addr)
	}

	return nil
}

// SubmitOrder ... submit order to JingDong, return orderID or error
//
func (jd *JingDong) SubmitOrder() (string, error) {
	clog.Info(strSeperater)
	clog.Info("提交订单>")

	data, err := jd.downloader.GetResponse("POST", URLSubmitOrder, func(URL string) string {
		queryString := map[string]string{
			"overseaPurchaseCookies":             "",
			"submitOrderParam.fp":                "",
			"submitOrderParam.eid":               "",
			"submitOrderParam.btSupport":         "1",
			"submitOrderParam.sopNotPutInvoice":  "false",
			"submitOrderParam.ignorePriceChange": "0",
			"submitOrderParam.trackID":           jd.jar.Get("TrackID"),
		}
		u, _ := url.Parse(URLSubmitOrder)
		q := u.Query()
		for k, v := range queryString {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		return u.String()
	})

	if err != nil {
		clog.Error(0, "提交订单失败: %+v", err)
		return "", err
	}

	var js *sjson.Json
	if js, err = sjson.NewJson(data); err != nil {
		clog.Info("Reponse Data: %s", data)
		clog.Error(0, "无法解析订单响应数据: %+v", err)
		return "", err
	}

	clog.Trace("订单: %s", data)

	if succ, _ := js.Get("success").Bool(); succ {
		orderID, _ := js.Get("orderId").Int64()
		clog.Info("下单成功，订单号：%d", orderID)
		return fmt.Sprintf("%d", orderID), nil
	}

	res, _ := js.Get("resultCode").String()
	msg, _ := js.Get("message").String()
	clog.Error(0, "下单失败, %s : %s", res, msg)
	return "", fmt.Errorf("failed to submit order (%s : %s)", res, msg)
}

func (jd *JingDong) changeCount(ID string, count int) (int, error) {
	data, err := jd.downloader.GetResponse("POST", URLChangeCount, func(URL string) string {
		u, _ := url.Parse(URL)
		q := u.Query()
		q.Set("venderId", "8888")
		q.Set("targetId", "0")
		q.Set("promoID", "0")
		q.Set("outSkus", "")
		q.Set("ptype", "1")
		q.Set("pid", ID)
		q.Set("pcount", strconv.Itoa(count))

		q.Set("random", strconv.FormatFloat(rand.Float64(), 'f', 16, 64))
		q.Set("locationId", jd.ShipArea)
		u.RawQuery = q.Encode()
		return u.String()
	})

	if err != nil {
		clog.Error(0, "淇®鏀瑰晢鍝佹暟閲忓け璐¥: %+v", err)
		return 0, err
	}
	js, _ := sjson.NewJson(data)
	return js.Get("pcount").Int()
}

func (jd *JingDong) buyGood(sku *SKUInfo) error {
	var (
		err  error
		data []byte
		doc  *goquery.Document
	)
	clog.Info(strSeperater)
	clog.Info("购买商品: %s", sku.ID)

	// 33 : on sale
	// 34 : out of stock
	for sku.State != "33" && jd.AutoRush {
		clog.Warn("%s : %s", sku.StateName, sku.Name)
		time.Sleep(jd.Period)
		sku.State, sku.StateName, err = jd.stockState(sku.ID)
		if err != nil {
			clog.Error(0, "获取(%s)库存失败: %+v", sku.ID, err)
			return err
		}
	}

	if sku.Link == "" || sku.Count != 1 {
		u, _ := url.Parse(URLAdd2Cart)
		q := u.Query()
		q.Set("pid", sku.ID)
		q.Set("pcount", strconv.Itoa(sku.Count))
		q.Set("ptype", "1")
		u.RawQuery = q.Encode()
		sku.Link = u.String()
	}

	if _, err := url.Parse(sku.Link); err != nil {
		clog.Error(0, "商品购买链接无效: <%s>", sku.Link)
		return fmt.Errorf("无效商品购买链接<%s>", sku.Link)
	}

	if data, err = jd.downloader.GetResponse("GET", sku.Link, nil); err != nil {
		clog.Error(0, "商品(%s)购买失败: %+v", sku.ID, err)
		return err
	}

	if doc, err = goquery.NewDocumentFromReader(bytes.NewBuffer(data)); err != nil {
		clog.Error(0, "响应解析失败: %+v", err)
		return err
	}

	succFlag := doc.Find("h3.ftx-02").Text()
	//fmt.Println(succFlag)

	if succFlag == "" {
		succFlag = doc.Find("div.p-name a").Text()
	}

	if succFlag != "" {
		count := 0
		if sku.Count > 1 {
			count, err = jd.changeCount(sku.ID, sku.Count)
		}

		if count > 0 {
			clog.Info("成功加入进购物车 %d 个 %s", count, sku.Name)
			return nil
		}
	}

	return err
}

func (jd *JingDong) RushBuy(skuLst map[string]int) {
	var wg sync.WaitGroup
	for id, cnt := range skuLst {
		wg.Add(1)
		go func(id string, count int) {
			defer wg.Done()
			if sku, err := jd.skuDetail(id); err == nil {
				sku.Count = count
				jd.buyGood(sku)
			}
		}(id, cnt)
	}

	wg.Wait()
	jd.OrderInfo()

	if jd.AutoSubmit {
		jd.SubmitOrder()
	}
}
