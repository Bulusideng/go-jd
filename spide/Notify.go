package main

import (
	"bytes"
	"log"
	"net/smtp"

	"github.com/go-gomail/gomail"
)

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
	mailAddr = "123@163.com"
	pwd      = "123"
	port     = 25 //465, 587

)

func dial() {
	// Connect to the remote SMTP server.
	c, err := smtp.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	// Set the sender and recipient.
	c.Mail(mailAddr)
	c.Rcpt(mailAddr)
	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		log.Fatal(err)
	}
	defer wc.Close()
	buf := bytes.NewBufferString("This is the email body.")
	if _, err = buf.WriteTo(wc); err != nil {
		log.Fatal(err)
	}
}

func notify() {
	//testgomail()
	//return
	// Set up authentication information.
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
		[]byte("This is the email body."),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func testgomail() {

	m := gomail.NewMessage()
	m.SetHeader("From", mailAddr)
	m.SetHeader("To", mailAddr)
	//m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Hello!")
	m.SetBody("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")
	//m.Attach("/home/Alex/lolcat.jpg")

	d := gomail.NewDialer(host, port, mailAddr, pwd)

	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}
