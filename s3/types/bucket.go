package types

import (
	"time"
)

type Bucket struct {
	Name         string    `xml:"Name"`
	CreationDate time.Time `xml:"CreationDate"`
}

func NewBucket(name string, creationDate time.Time) Bucket {
	return Bucket{
		Name:         name,
		CreationDate: creationDate,
	}
}
