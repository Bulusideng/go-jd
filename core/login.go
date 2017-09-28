package core

import (
	"fmt"
	"io"
	"io/ioutil"

	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"time"

	sjson "github.com/bitly/go-simplejson"
	clog "gopkg.in/clog.v1"
)

//
func (jd *JingDong) validateLogin(URL string) bool {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	if req, err = http.NewRequest("GET", URL, nil); err != nil {
		clog.Info("请求（%+v）失败: %+v", URL, err)
		return false
	}

	jd.downloader.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// disable redirect
		return http.ErrUseLastResponse
	}

	defer func() {
		// restore to default
		jd.downloader.CheckRedirect = nil
	}()

	if resp, err = jd.downloader.Do(req); err != nil {
		clog.Info("需要重新登录: %+v", err)
		return false
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		clog.Info("需要重新登录")
		return false
	}

	clog.Trace("Response Data: %s", string(data))
	return true
}

// load the login page
//
func (jd *JingDong) loginPage(URL string) error {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	if req, err = http.NewRequest("GET", URL, nil); err != nil {
		clog.Info("请求（%+v）失败: %+v", URL, err)
		return err
	}

	applyCustomHeader(req, DefaultHeaders)

	if resp, err = jd.downloader.Do(req); err != nil {
		clog.Info("请求登录页失败: %+v", err)
		return err
	}

	defer resp.Body.Close()
	return nil
}

// download the QR Code
//
func (jd *JingDong) loadQRCode(URL string) (string, error) {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	u, _ := url.Parse(URL)
	q := u.Query()
	q.Set("appid", strconv.Itoa(133))
	q.Set("size", strconv.Itoa(147))
	q.Set("t", strconv.FormatInt(time.Now().Unix()*1000, 10))
	u.RawQuery = q.Encode()

	if req, err = http.NewRequest("GET", u.String(), nil); err != nil {
		clog.Error(0, "请求（%+v）失败: %+v", URL, err)
		return "", err
	}

	applyCustomHeader(req, DefaultHeaders)
	if resp, err = jd.downloader.Do(req); err != nil {
		clog.Error(0, "下载二维码失败: %+v", err)
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		clog.Error(0, "http status : %d/%s", resp.StatusCode, resp.Status)
	}

	// from mime get QRCode image type
	//  content-type:image/png
	//
	filename := qrCodeFile + ".png"
	mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if typ, e := mime.ExtensionsByType(mt); e == nil {
		filename = qrCodeFile + typ[0]
	}

	dir, _ := os.Getwd()
	filename = filepath.Join(dir, filename)
	clog.Trace("QR Image: %s", filename)

	file, _ := os.Create(filename)
	defer file.Close()

	if _, err = io.Copy(file, resp.Body); err != nil {
		clog.Error(0, "下载二维码失败: %+v", err)
		return "", err
	}

	return filename, nil
}

// wait scan result
//
func (jd *JingDong) waitForScan(URL string) error {
	var (
		err    error
		req    *http.Request
		resp   *http.Response
		wlfstk string
	)

	for _, c := range jd.jar.Cookies(nil) {
		if c.Name == "wlfstk_smdl" {
			wlfstk = c.Value
			break
		}
	}

	u, _ := url.Parse(URL)
	q := u.Query()
	q.Set("callback", "jQuery123456")
	q.Set("appid", strconv.Itoa(133))
	q.Set("token", wlfstk)
	q.Set("_", strconv.FormatInt(time.Now().Unix()*1000, 10))
	u.RawQuery = q.Encode()

	if req, err = http.NewRequest("GET", u.String(), nil); err != nil {
		clog.Info("请求（%+v）失败: %+v", URL, err)
		return err
	}

	// mush have
	req.Host = "qr.m.jd.com"
	req.Header.Set("Referer", "https://passport.jd.com/new/login.aspx")
	applyCustomHeader(req, DefaultHeaders)

	for retry := 50; retry != 0; retry-- {
		if resp, err = jd.downloader.Do(req); err != nil {
			clog.Info("二维码失效：%+v", err)
			break
		}

		if resp.StatusCode == http.StatusOK {
			respMsg := string(responseData(resp))
			resp.Body.Close()

			n1 := strings.Index(respMsg, "(")
			n2 := strings.Index(respMsg, ")")

			var js *sjson.Json
			if js, err = sjson.NewJson([]byte(respMsg[n1+1 : n2])); err != nil {
				clog.Error(0, "解析响应数据失败: %+v", err)
				clog.Trace("Response data  : %+v", respMsg)
				clog.Trace("Response Header: %+v", resp.Header)
				break
			}

			code := js.Get("code").MustInt()
			if code == 200 {
				jd.token = js.Get("ticket").MustString()
				clog.Info("token : %+v", jd.token)
				break
			} else {
				clog.Info("%+v : %s", code, js.Get("msg").MustString())
				time.Sleep(time.Second * 3)
			}
		} else {
			resp.Body.Close()
		}
	}

	if jd.token == "" {
		err = fmt.Errorf("未检测到QR扫码结果")
		return err
	}

	return nil
}

// validate QR token
//
func (jd *JingDong) validateQRToken(URL string) error {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)

	u, _ := url.Parse(URL)
	q := u.Query()
	q.Set("t", jd.token)
	u.RawQuery = q.Encode()

	if req, err = http.NewRequest("GET", u.String(), nil); err != nil {
		clog.Info("请求（%+v）失败: %+v", URL, err)
		return err
	}

	if resp, err = jd.downloader.Do(req); err != nil {
		clog.Error(0, "二维码登陆校验失败: %+v", err)
		return nil
	}

	if resp.StatusCode == http.StatusOK {
		clog.Info("登陆成功, P3P: %s", resp.Header.Get("P3P"))
	} else {
		clog.Info("登陆失败")
		err = fmt.Errorf("%+v", resp.Status)
	}

	resp.Body.Close()
	return nil
}

// Login used to login JD by QR code.
// if the cookies file exits, will try cookies first.
//
func (jd *JingDong) Login(args ...interface{}) error {
	clog.Info(strSeperater)

	if jd.validateLogin(URLForQR[4]) {
		clog.Info("无需重新登录")
		return nil
	}

	var (
		err   error
		qrImg string
	)

	clog.Info("请打开京东手机客户端，准备扫码登陆:")
	jd.jar.Clean()

	if err = jd.loginPage(URLForQR[0]); err != nil {
		return err
	}

	if qrImg, err = jd.loadQRCode(URLForQR[1]); err != nil {
		return err
	}

	// for different platform
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", qrImg)
	case "linux":
		cmd = exec.Command("gnome-open", qrImg)
	default:
		cmd = exec.Command("open", qrImg)
	}

	// just start, do not wait it complete
	if err = cmd.Start(); err != nil {
		clog.Info("打开二维码图片失败: %+v.", err)
		return err
	}

	if err = jd.waitForScan(URLForQR[2]); err != nil {
		return err
	}

	if err = jd.validateQRToken(URLForQR[3]); err != nil {
		return err
	}

	//http.Post()
	return nil
}
