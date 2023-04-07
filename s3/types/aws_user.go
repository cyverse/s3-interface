package types

type AwsUser struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

func NewAwsUser(name string) AwsUser {
	return AwsUser{
		ID:          name,
		DisplayName: name,
	}
}
