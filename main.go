package main

import (
	"context"
	"encoding/json"
	"fmt"
	io "io/ioutil"
	"log"
	"net/smtp"
	"strconv"
	"strings"

	e "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	em "github.com/jordan-wright/email"
)

type policydata struct {
	Name         string `json:"name"`
	AddressLine1 string `json:"addressLine1"`
	AddressLine2 string `json:"addressLine2"`
	AddressLine3 string `json:"addressLine3"`
	City         string `json:"city"`
	PinCode      int    `json:"pinCode"`
	MobileNumber int    `json:"mobileNumber"`
	PolicyNumber int    `json:"policyNumber"`
}

type email struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type polmetadata struct {
	Email email      `json:"email"`
	Data  policydata `json:"data"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event e.S3Event) (string, error) {

	for _, record := range event.Records {
		log.Println("bucket " + record.S3.Bucket.Name)
		log.Println("object " + record.S3.Object.Key)
		err := processEvent(record)
		if err != nil {
			log.Println(err)
			return "processing error ", err
		}
	}
	return fmt.Sprintf("object processed "), nil
}

func processEvent(record e.S3EventRecord) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	pdf, pdferr := policyPDF(sess, record)
	if pdferr != nil {
		return nil
	}

	pmd, err := policyMetaData(sess, record)
	if err != nil {
		return nil
	}

	serr := sendEmail(pmd, pdf)
	if serr != nil {
		log.Println("error sending email " + serr.Error())
		return nil
	}

	mverr := moveFiles(sess)
	if mverr != nil {
		return mverr
	}
	return nil
}

func moveFiles(sess *session.Session) error {
	return nil
}

func sendEmail(pmd *polmetadata, pdf []byte) error {
	e := em.NewEmail()
	e.From = "Kubesure <" + pmd.Email.From + ">"
	e.To = []string{pmd.Email.To}
	e.Subject = "Kubesure : EsyHealth - " + strconv.Itoa(pmd.Data.PolicyNumber)
	ebody := []byte("Hello " + pmd.Data.Name + "," + "\n")
	e.Text = ebody
	return e.Send("smtp.gmail.com:587", smtp.PlainAuth("", "edakghar@gmail.com", "", "smtp.gmail.com"))
}

func policyPDF(sess *session.Session, record e.S3EventRecord) ([]byte, error) {
	svc := s3.New(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String(record.S3.Bucket.Name),
		Key:    aws.String(record.S3.Object.Key),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		log.Println("error getting pdf object " + err.Error())
		return nil, err
	}
	defer result.Body.Close()
	bodyBytes, err := io.ReadAll(result.Body)
	return bodyBytes, nil
}

func policyMetaData(sess *session.Session, record e.S3EventRecord) (*polmetadata, error) {
	svc := s3.New(sess)

	key := record.S3.Object.Key
	polNumber := key[strings.Index(key, "/")+1 : strings.Index(key, ".")]

	input := &s3.GetObjectInput{
		Bucket: aws.String(record.S3.Bucket.Name),
		Key:    aws.String("unprocessed/" + polNumber + ".json"),
	}

	result, jerr := svc.GetObject(input)
	if jerr != nil {
		log.Println("error getting json object " + jerr.Error())
		return nil, jerr
	}
	defer result.Body.Close()
	bodyBytes, merr := io.ReadAll(result.Body)
	pmd, merr := marshallReq(string(bodyBytes))
	if merr != nil {
		log.Println("json marshall error " + merr.Error())
		return nil, merr
	}
	return pmd, nil
}

func marshallReq(data string) (*polmetadata, error) {
	var pd polmetadata
	err := json.Unmarshal([]byte(data), &pd)
	if err != nil {

		return nil, err
	}
	return &pd, nil
}
