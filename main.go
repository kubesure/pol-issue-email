package main

import (
	"bytes"
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

	mverr := moveFiles(sess, record)
	if mverr != nil {
		return mverr
	}
	return nil
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
	//polNumber := key[strings.Index(key, "/")+1 : strings.Index(key, ".")]
	polNumber := extactPolNo(key)

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

func sendEmail(pmd *polmetadata, pdf []byte) error {
	e := em.NewEmail()
	e.From = "Kubesure <" + pmd.Email.From + ">"
	e.To = []string{pmd.Email.To}
	e.Subject = "Kubesure : EsyHealth - " + strconv.Itoa(pmd.Data.PolicyNumber)
	msg := `Hello %s,
				Your policy has been issued and the policy document has been attached.
				Please use policy number to make any enquiries.

		Best Wishes,
		Kubesure Customer Service	 
	    `
	body := fmt.Sprintf(msg, pmd.Data.Name)
	ebody := []byte(body)
	e.Text = ebody
	e.Attach(bytes.NewReader(pdf), strconv.Itoa(pmd.Data.PolicyNumber), "application/pdf")
	var err error
	err = e.Send("smtp.gmail.com:587", smtp.PlainAuth("", "pras.p.in@gmail.com", "", "smtp.gmail.com"))
	if err != nil {
		return err
	}
	return nil
}

func moveFiles(sess *session.Session, record e.S3EventRecord) error {
	svc := s3.New(sess)
	polNumber := extactPolNo(record.S3.Object.Key)
	listInput := &s3.ListObjectsInput{
		Bucket:    aws.String(record.S3.Bucket.Name),
		Prefix:    aws.String("unprocessed/" + polNumber),
		Delimiter: aws.String("/"),
	}

	outputList, err := svc.ListObjects(listInput)
	if err != nil {
		log.Println("error while listing objects " + err.Error())
		return nil
	}

	//movedkeys := make([]string, 0)
	//failed := false

	for _, o := range outputList.Contents {
		fileName := extractFileName(*o.Key)
		dkey := "processed/" + fileName

		copyInput := &s3.CopyObjectInput{
			Bucket:     aws.String(record.S3.Bucket.Name),
			CopySource: aws.String(record.S3.Bucket.Name + "/" + *o.Key),
			Key:        aws.String("/" + dkey),
		}
		_, cerr := svc.CopyObject(copyInput)
		if cerr != nil {
			//movedkeys = append(movedkeys, *o.Key)
			log.Println("copy failed due to " + cerr.Error())
			//continue
		}
		deleteInput := &s3.DeleteObjectInput{
			Bucket: aws.String(record.S3.Bucket.Name),
			Key:    o.Key,
		}

		_, derr := svc.DeleteObject(deleteInput)
		if derr != nil {
			log.Println("delete failed due to " + cerr.Error())
			//movedkeys = append(movedkeys, *o.Key)
			//continue
		}
	}

	/*for _, fo : range movedkeys {

		deleteInput := &s3.DeleteObjectInput{
			Bucket: aws.String(record.S3.Bucket.Name),
			Key:    aws.String(dkey),
		}

		_, derr := svc.DeleteObject(deleteInput)
		if derr != nil {
			movedkeys = append(movedkeys, *o.Key)
			continue
		}
	}*/

	return nil
}

func marshallReq(data string) (*polmetadata, error) {
	var pd polmetadata
	err := json.Unmarshal([]byte(data), &pd)
	if err != nil {

		return nil, err
	}
	return &pd, nil
}

func extactPolNo(key string) string {
	return key[strings.Index(key, "/")+1 : strings.Index(key, ".")]
}

func extractExt(key string) string {
	return key[strings.Index(key, "."):len(key)]
}

func extractFileName(key string) string {
	return key[strings.Index(key, "/")+1 : len(key)]
}
