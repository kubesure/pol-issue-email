package main

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"testing"

	e "github.com/aws/aws-lambda-go/events"
	em "github.com/jordan-wright/email"
)

func TestExtractPoilcyNumber(t *testing.T) {
	key := "unprocessed/1234567890.pdf"
	polNumber := key[strings.Index(key, "/")+1 : strings.Index(key, ".")]
	if polNumber != "1234567890" {
		t.Errorf("policy number not found")
	}
}

func TestProcessEvent(t *testing.T) {
	bucket := e.S3Bucket{Name: "kubesure-cs-1"}
	object := e.S3Object{Key: "unprocessed/1234567890.pdf"}
	r := e.S3EventRecord{}
	r.S3.Bucket = bucket
	r.S3.Object = object
	err := processEvent(r)

	if err != nil {
		t.Errorf("S3 Event Processed %v", err)
	}
}

func TestEmail(t *testing.T) {
	to := "pras.p.in@gmail.com"
	from := "pras.p.in@gmail.com"
	password := "7@NotForget"
	subject := "subject line of email"
	msg := "a one-line email message"

	emailTemplate := `To: %s
  Subject: %s
  
  %s
  `
	body := fmt.Sprintf(emailTemplate, to, subject, msg)
	auth := smtp.PlainAuth("", from, password, "smtp.gmail.com")
	err := smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		from,
		[]string{to},
		[]byte(body),
	)
	if err != nil {
		log.Println(err)
	}
}

func TestEmail2(t *testing.T) {
	e := em.NewEmail()
	e.From = "Kubesure <pras.p.in@gmail.com>"
	e.To = []string{"pras.p.in@gmail.com"}
	e.Subject = "Kubesure : EsyHealth - 1234567890"
	ebody := "hello body"
	e.Text = []byte(ebody)
	err := e.Send("smtp.gmail.com:587", smtp.PlainAuth("", "pras.p.in@gmail.com", "7@NotForget", "smtp.gmail.com"))
	if err != nil {
		t.Errorf("Error sending email %v", err)
	}
}

func TestExtractExt(t *testing.T) {
	key := "unprocessed/1234567890.pdf"
	ext := extractExt(key)
	if ext != ".pdf" {
		t.Error("not a pdf ext")
	}
}
