//
package main

import (
	"bytes"
	"net/url"
	"regexp"
	"strconv"
	"sync"

	"strings"

	"github.com/axgle/mahonia"

	"net/http"

	"github.com/Bulusideng/go-jd/core"
	"github.com/PuerkitoBio/goquery"

	clog "gopkg.in/clog.v1"
)

type HomeNav struct {
	Name string
	URL  string
	cat  chan *Category
}

type Category struct {
	CatName string
	CatURL  string
	pages   int
}

func (this *Category) Get() {
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
			//			name := strings.Trim(s.Find("div.p-name em").Text(), " \n\r\t")
			jd.SkuIds <- id
			//clog.Info("%s[%d][%d][%d]: sku id: %s, %s", this.CatName, this.pages, page, cnt, id, name)
			//jd.getPrice(id)
			//skuIds = append(skuIds, id)
			//jd.SkuIds <- id

		})
	}
}

type CatSpider struct {
	*core.Downloader
	wg         sync.WaitGroup
	HomeNavCnt int
}

func (this *HomeNav) run(wg *sync.WaitGroup) {
	catCnt := 0
	wg.Add(1)
	go this.GetCatPages(wg)
	defer wg.Done()

	data, err := spider.GetResponse("GET", this.URL, nil)
	if err != nil {
		clog.Error(0, "HomeNav run failed: %+v", err)
		return
	}

	catreg := regexp.MustCompile(`//list\.jd\.com/list\.html\?cat=[\d,]+$`)
	// return GBK encoding
	dec := mahonia.NewDecoder("gbk")

	decString := dec.ConvertString(string(data))

	query, _ := goquery.NewDocumentFromReader(strings.NewReader(decString))

	/*
			<li class="title-name">
		      	<a href="//list.jd.com/list.html?cat=9987,653,655" target="_blank" clstag="channel|keycount|565|FLA2_3_2_1">全部手机</a>
		      </li>
	*/
	query.Find("li.title-name a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		ref, _ := url.QueryUnescape(href)
		if catreg.FindAllString(ref, -1) != nil {
			name := strings.Trim(s.Text(), " \n\r\t")
			clog.Info("Add  %s[%s]: %s", this.Name, name, ref)
			this.cat <- &Category{CatName: name, CatURL: ref}
			catCnt++
		}
	})
	clog.Info("%s: %s exit, Catgory found:%d", this.Name, this.URL, catCnt)
	close(this.cat)
}

func (this *HomeNav) GetCatPages(wg *sync.WaitGroup) {
	cnt := 0
	wg.Add(1)
	defer wg.Done()
	defer clog.Info("%s GetItems exit", this.Name)

	for cat := range this.cat {
		if !strings.Contains(cat.CatURL, "http") {
			cat.CatURL = "http:" + cat.CatURL
		}
		data, err := spider.GetResponse("GET", cat.CatURL, nil)
		if err != nil {
			clog.Error(0, "GetItems failed: %+v", err)
			return
		}
		query, _ := goquery.NewDocumentFromReader(bytes.NewReader(data))
		cat.pages, _ = strconv.Atoi(query.Find("span.fp-text i").Text())
		cat.Get()
		cnt++
		clog.Info("%s[%d %s]: pages:%d, URL:%s", this.Name, cnt, cat.CatName, cat.pages, cat.CatURL)
	}
}

func NewCatSpider() *CatSpider {
	return &CatSpider{
		Downloader: &core.Downloader{
			&http.Client{},
		},
	}
}

func (this *CatSpider) run() {
	data, err := this.GetResponse("GET", "http://www.jd.com", nil)
	clog.Info("run...")
	defer clog.Info("exit...")

	if err != nil {
		clog.Error(0, "失败: %+v", err)
		return
	}
	query, _ := goquery.NewDocumentFromReader(bytes.NewReader(data))
	query.Find("a.cate_menu_lk").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		nav := &HomeNav{s.Text(), "http:" + href, make(chan *Category, 1000)}
		switch nav.Name {
		case "手机", "电脑":
			go nav.run(&this.wg)
			this.HomeNavCnt++
			clog.Info("Add HomeNav[%d] %s %s", this.HomeNavCnt, s.Text(), href)
		}

	})
}

var spider *CatSpider

func main() {
	defer clog.Shutdown()
	spider = NewCatSpider()
	spider.run()
	spider.wg.Wait()

}
