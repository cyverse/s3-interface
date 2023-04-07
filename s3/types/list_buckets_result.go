package types

import (
	"encoding/xml"
)

type ListBucketsOutput struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01 ListAllMyBucketsResult"`
	Owner   AwsUser  `xml:"Owner"`
	Buckets []Bucket `xml:"Buckets>Bucket"`
}
