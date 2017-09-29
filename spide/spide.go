//
package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Bulusideng/go-jd/core"
	//	"github.com/axgle/mahonia"
	"flag"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	clog "gopkg.in/clog.v1"
)

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

type CatSpider struct {
	*core.Downloader
	chanCat    chan *Category
	exit       chan int
	ThreadChan chan int
	threadCnt  int
	jd         *core.JingDong
}

type Category struct {
	CatName string
	CatURL  string
	pages   int
}

func NewCatSpider(threadcnt int) *CatSpider {
	return &CatSpider{
		Downloader: &core.Downloader{
			&http.Client{},
		},
		chanCat:    make(chan *Category, 1000),
		ThreadChan: make(chan int, threadcnt),
		exit:       make(chan int, 1),
		threadCnt:  threadcnt,
		jd: core.NewJingDong(core.JDConfig{
			Period:     time.Millisecond * time.Duration(*period),
			ShipArea:   *area,
			AutoRush:   *rush,
			AutoSubmit: *order,
		}, "shouji"),
	}
}

func (this *Category) run() {
	clog.Info("Run cat %s %s", this.CatName, this.CatURL)
	defer clog.Info("Exit cat %s %s", this.CatName, this.CatURL)
	data, err := spider.GetResponse("GET", this.CatURL, nil)
	if err != nil {
		clog.Error(0, "GetItems failed: %+v", err)
		return
	}
	query, _ := goquery.NewDocumentFromReader(bytes.NewReader(data))
	this.pages, _ = strconv.Atoi(query.Find("span.fp-text i").Text())
	clog.Info("Run cat %s %s, pages:%d", this.CatName, this.CatURL, this.pages)

	for page := 1; page <= this.pages; page++ {
		data, err := spider.GetResponse("GET", this.CatURL+"&page="+strconv.Itoa(page), nil)
		if err != nil {
			clog.Error(0, "Page failed: %+v", err)
			return
		}
		cnt := 0
		query, _ := goquery.NewDocumentFromReader(bytes.NewReader(data))
		query.Find("div.gl-i-wrap.j-sku-item").Each(func(i int, s *goquery.Selection) {
			cnt++
			id, _ := s.Attr("data-sku")
			name := strings.Trim(s.Find("div.p-name em").Text(), " \n\r\t")
			spider.jd.SkuIds <- id
			if true {
				clog.Info("Found Item %s:%s", id, name)
			}
			if len(spider.jd.SkuIds) > 10 {
				time.Sleep(time.Second)
			}

		})
		if page > 0 {
			break
		}
	}
}

func (this *CatSpider) GetCatogery() {
	data, err := this.GetResponse("GET", "https://www.jd.com/allSort.aspx", nil)
	clog.Info("GetCatogery start...")
	defer clog.Info("GetCatogery done...")
	cats := 0
	if err != nil {
		clog.Error(0, "失败: %+v", err)
		return
	}
	catreg := regexp.MustCompile(`//list\.jd\.com/list\.html\?cat=[\d,]+$`)

	query, _ := goquery.NewDocumentFromReader(bytes.NewReader(data))
	query.Find("div.clearfix dd a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		ref, _ := url.QueryUnescape(href)
		if catreg.FindAllString(ref, -1) != nil {
			cat := &Category{s.Text(), "http:" + href, 0}
			switch cat.CatName {
			case "手机", "电脑":
			default:
				this.chanCat <- cat
				cats++
				clog.Info("Add cat[%d] %s %s", cats, cat.CatName, cat.CatURL)
				if cats > 6 {
					return false
				}
			}
		}
		return true
	})

}

func (this *CatSpider) run() {
	go func() {
		for {
			cat, ok := <-this.chanCat
			if !ok {
				clog.Info("No item and closed, exit")
				break
			}
			go func(s *CatSpider, cat *Category) {
				s.ThreadChan <- 1
				cat.run()
				<-s.ThreadChan
			}(this, cat)
		}
	}()

	this.GetCatogery()
	clog.Info("Close cat chan...")
	close(this.chanCat)
	clog.Info("Waiting worker exit...")
	for i := 0; i < this.threadCnt; i++ {
		this.ThreadChan <- i
	}
	clog.Info("main exit...")
}

var spider *CatSpider

func init() {
	if err := clog.New(clog.CONSOLE, clog.ConsoleConfig{
		Level:      clog.INFO,
		BufferSize: 100},
	); err != nil {
		fmt.Printf("init console log failed. error %+v.", err)
		os.Exit(1)
	}
	flag.Parse()
}

func main() {
	defer clog.Shutdown()
	if test {
		TestDB()
	} else {
		spider = NewCatSpider(1)
		spider.jd.Start(10)
		spider.run()

	}

}
