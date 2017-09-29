package main

import (
	//	"bytes"
	"log"
	"net/smtp"

	"github.com/go-gomail/gomail"
)

/*
Name 	Address			SSL port 	Non-SSL port
IMAP 	imap.163.com   	993			143
SMTP	smtp.163.com	465/994		25
POP3	pop.163.com		995			110
*/

const (
	/*
		host     = "smtp.gmail.com"
		addr     = host + ":25"
		mailAddr = "123@gmail.com"
		pwd      = "123"
		port     = 25 //465, 587
	*/
	host     = "smtp.163.com"
	addr     = host + ":25"
	mailAddr = "dbiti@163.com"
	pwd      = "oxiwangyi."
	port     = 25 //465, 587

)

func sendMail() {
	gomailSend("标题", "内容。。。。。", "")
	//sendMail1()
}
func sendMail1() {
	auth := smtp.PlainAuth(
		"",
		mailAddr,
		pwd,
		host,
	)
	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	err := smtp.SendMail(
		addr,
		auth,
		mailAddr,
		[]string{mailAddr},
		[]byte("This is the email bodyasdf."),
	)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Send success")
	}
}

func gomailSend(sub, body, attach string) {
	m := gomail.NewMessage()
	m.SetHeader("From", mailAddr)
	m.SetHeader("To", mailAddr)
	//m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", sub)
	m.SetBody("text/html", body)
	if len(attach) > 0 {
		m.Attach(attach)
	}

	d := gomail.NewDialer(host, port, mailAddr, pwd)

	if err := d.DialAndSend(m); err != nil {
		panic(err)
	} else {
		log.Println("gmail success")
	}
}
