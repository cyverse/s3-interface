package types

import (
	"encoding/xml"
	"time"
)

type ListBucketsOutput struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Owner   AwsUser  `xml:"Owner"`
	Buckets []Bucket `xml:"Buckets"`
}

type Bucket struct {
	Name         string    `xml:"Name"`
	CreationDate time.Time `xml:"CreationDate"`
}
