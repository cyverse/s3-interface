package s3

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

var (
	// if object matches reserved string, no need to encode them
	reservedObjectNames = regexp.MustCompile("^[a-zA-Z0-9-_.~/]+$")
)

const (
	signV4Algorithm = "AWS4-HMAC-SHA256"
	iso8601Format   = "20060102T150405Z"
	yyyymmdd        = "20060102"
)

type AWSCredential struct {
	Username       string // iychoi
	RequestDate    string // YYYYMMDD
	Region         string // us-east-1
	ServiceType    string // s3
	RequestVersion string // aws4_request
}

func (credential AWSCredential) GetScopeString() string {
	return strings.Join([]string{
		credential.RequestDate,
		credential.Region,
		credential.ServiceType,
		credential.RequestVersion,
	}, "/")
}

func getSignedHeaderFields(request *http.Request) map[string]string {
	logger := log.WithFields(log.Fields{
		"package":  "s3",
		"function": "getSignedHeaderFields",
	})

	signedHeadersMap := map[string]string{}

	authFields := getRequestAuthFields(request)
	signedHeadersCSV, hasSignedHeaders := authFields["SignedHeaders"]
	if !hasSignedHeaders {
		return signedHeadersMap
	}

	logger.Debugf("signed headers: %s", signedHeadersCSV)

	signedHeadersFields := strings.Split(signedHeadersCSV, ";")
	for _, signedHeadersField := range signedHeadersFields {
		if signedHeadersField == "host" {
			signedHeadersMap["Host"] = request.Host
		}

		for mk, _ := range request.Header {
			if strings.ToLower(mk) == signedHeadersField {
				signedHeadersMap[mk] = request.Header.Get(mk)
			}
		}
	}

	return signedHeadersMap
}

func getCredential(request *http.Request) *AWSCredential {
	authFields := getRequestAuthFields(request)
	credentialSSV, hasCredential := authFields["Credential"]
	if !hasCredential {
		return nil
	}

	credentialFields := strings.Split(credentialSSV, "/")
	if len(credentialFields) == 5 {
		return &AWSCredential{
			Username:       credentialFields[0],
			RequestDate:    credentialFields[1],
			Region:         credentialFields[2],
			ServiceType:    credentialFields[3],
			RequestVersion: credentialFields[4],
		}
	}

	return nil
}

func getSignature(request *http.Request) string {
	authFields := getRequestAuthFields(request)
	signature, hasSignature := authFields["Signature"]
	if !hasSignature {
		return ""
	}

	return signature
}

func getRequestAuthFields(request *http.Request) map[string]string {
	authorization := request.Header.Get("Authorization")

	fields := map[string]string{}

	authFields := strings.Split(authorization, " ")
	for fieldIdx, authField := range authFields {
		authField = strings.TrimSpace(authField)
		authField = strings.TrimRight(authField, ",")

		if fieldIdx == 0 {
			fields["algorithm"] = authField
			continue
		}

		kv := strings.Split(authField, "=")
		if len(kv) == 2 {
			fields[kv[0]] = kv[1]
		}
	}

	return fields
}

func encodePath(pathName string) string {
	if reservedObjectNames.MatchString(pathName) {
		return pathName
	}
	var encodedPathname strings.Builder
	for _, s := range pathName {
		if 'A' <= s && s <= 'Z' || 'a' <= s && s <= 'z' || '0' <= s && s <= '9' { // §2.3 Unreserved characters (mark)
			encodedPathname.WriteRune(s)
			continue
		}
		switch s {
		case '-', '_', '.', '~', '/': // §2.3 Unreserved characters (mark)
			encodedPathname.WriteRune(s)
			continue
		default:
			len := utf8.RuneLen(s)
			if len < 0 {
				// if utf8 cannot convert return the same string as is
				return pathName
			}
			u := make([]byte, len)
			utf8.EncodeRune(u, s)
			for _, r := range u {
				hex := hex.EncodeToString([]byte{r})
				encodedPathname.WriteString("%" + strings.ToUpper(hex))
			}
		}
	}
	return encodedPathname.String()
}

func getCanonicalRequest(signedHeaderFields map[string]string, contentCheckSum string, queryString string, urlPath string, method string) string {
	rawQuery := strings.ReplaceAll(queryString, "+", "%20")

	encodedPath := encodePath(urlPath)
	canonicalRequest := strings.Join([]string{
		method,
		encodedPath,
		rawQuery,
		getCanonicalHeaders(signedHeaderFields),
		getSignedHeaders(signedHeaderFields),
		contentCheckSum,
	}, "\n")
	return canonicalRequest
}

func getCanonicalHeaders(signedHeaderFields map[string]string) string {
	var headers []string
	vals := map[string]string{}
	for k, vv := range signedHeaderFields {
		headers = append(headers, strings.ToLower(k))
		vals[strings.ToLower(k)] = vv
	}
	sort.Strings(headers)

	// order by key
	var sb strings.Builder
	for _, k := range headers {
		sb.WriteString(k)
		sb.WriteByte(':')

		valFields := strings.Split(vals[k], ",")
		for idx, v := range valFields {
			if idx > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(v)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func getSignedHeaders(signedHeaderFields map[string]string) string {
	var headers []string
	for k := range signedHeaderFields {
		headers = append(headers, strings.ToLower(k))
	}
	sort.Strings(headers)
	return strings.Join(headers, ";")
}

func getStringToSign(canonicalRequest string, requestTime time.Time, scopeString string) string {
	stringToSign := signV4Algorithm + "\n" + requestTime.Format(iso8601Format) + "\n"
	stringToSign += scopeString + "\n"
	canonicalRequestBytes := sha256.Sum256([]byte(canonicalRequest))
	stringToSign += hex.EncodeToString(canonicalRequestBytes[:])
	return stringToSign
}

func sumHMAC(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func getSigningKey(secretKey string, requestTime time.Time, region string, serviceType string) []byte {
	date := sumHMAC([]byte("AWS4"+secretKey), []byte(requestTime.Format(yyyymmdd)))
	regionBytes := sumHMAC(date, []byte(region))
	service := sumHMAC(regionBytes, []byte(serviceType))
	signingKey := sumHMAC(service, []byte("aws4_request"))
	return signingKey
}

func generateSignature(signingKey []byte, stringToSign string) string {
	return hex.EncodeToString(sumHMAC(signingKey, []byte(stringToSign)))
}

func checkSignature(request *http.Request, secretKey string) (bool, error) {
	logger := log.WithFields(log.Fields{
		"package":  "s3",
		"function": "checkSignature",
	})

	queryString := request.Form.Encode()

	signedHeaderFields := getSignedHeaderFields(request)
	contentCheckSum := request.Header.Get("X-Amz-Content-SHA256")

	canonicalRequest := getCanonicalRequest(signedHeaderFields, contentCheckSum, queryString, request.URL.Path, request.Method)
	logger.Debugf("canonical request: %s", canonicalRequest)

	requestDate := request.Header.Get("X-Amz-Date")
	requestTime, err := time.Parse(iso8601Format, requestDate)
	if err != nil {
		return false, err
	}

	credential := getCredential(request)
	if credential == nil {
		return false, xerrors.Errorf("failed to get credential from request")
	}

	stringToSign := getStringToSign(canonicalRequest, requestTime, credential.GetScopeString())
	logger.Debugf("string to sign: %s", stringToSign)

	signingKey := getSigningKey(secretKey, requestTime, credential.Region, credential.ServiceType)
	logger.Debugf("signing key: %s", signingKey)

	newSignature := generateSignature(signingKey, stringToSign)
	logger.Debugf("new signature: %s", newSignature)

	oldSignature := getSignature(request)
	logger.Debugf("old signature: %s", oldSignature)

	return newSignature == oldSignature, nil
}
