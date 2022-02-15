package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

type Faker struct {
	transport *Transport
	Client    *http.Client
}

func NewFaker(t *testing.T) *Faker {
	t.Helper()

	transport := &Transport{
		t: t,
		fakeResponses: &fakeResponses{
			responseMap:     make(map[string]*http.Response),
			requestCountMap: make(map[string]int),
		},
	}
	return &Faker{
		transport: transport,
		Client: &http.Client{
			Transport: transport,
		},
	}
}

// AddResponse is RequestされたURLに対するResponseを登録する
// 同じURLを複数回呼ぶ時は複数回Addする
func (faker *Faker) AddResponse(url string, method string, response *http.Response) error {
	faker.transport.fakeResponses.Add(url, method, response)
	return nil
}

// AddGetObjectResponse is 指定したobjectの読み込みに対してのResponseを登録する
// 同じObjectを複数回読み込む時は複数回Addする
func (faker *Faker) AddGetObjectResponse(bucket string, object string, response *http.Response) error {
	faker.transport.fakeResponses.Add(fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object), http.MethodGet, response)
	return nil
}

// WriteObjectResponse is Cloud StorageにPostを行った時のResponse Body
type WriteObjectResponse struct {
	Kind                    string    `json:"kind"`
	ID                      string    `json:"id"`
	SelfLink                string    `json:"selfLink"`
	Name                    string    `json:"name"`
	Bucket                  string    `json:"bucket"`
	Generation              string    `json:"generation"`
	Metageneration          string    `json:"metageneration"`
	ContentType             string    `json:"contentType"`
	TimeCreated             time.Time `json:"timeCreated"`
	Updated                 time.Time `json:"updated"`
	StorageClass            string    `json:"storageClass"`
	TimeStorageClassUpdated time.Time `json:"timeStorageClassUpdated"`
	Size                    string    `json:"size"`
	Md5Hash                 string    `json:"md5Hash"`
	MediaLink               string    `json:"mediaLink"`
	ACL                     []*ACL    `json:"acl"`
	Owner                   *Owner    `json:"owner"`
	Crc32C                  string    `json:"crc32c"`
	Etag                    string    `json:"etag"`
}

type ACL struct {
	Kind        string       `json:"kind"`
	ID          string       `json:"id"`
	SelfLink    string       `json:"selfLink"`
	Bucket      string       `json:"bucket"`
	Object      string       `json:"object"`
	Generation  string       `json:"generation"`
	Entity      string       `json:"entity"`
	Role        string       `json:"role"`
	ProjectTeam *ProjectTeam `json:"projectTeam,omitempty"`
	Etag        string       `json:"etag"`
	Email       string       `json:"email,omitempty"`
}

type ProjectTeam struct {
	ProjectNumber string `json:"projectNumber"`
	Team          string `json:"team"`
}

type Owner struct {
	Entity string `json:"entity"`
}

// GenerateSimplePostObjectOKResponse is 最低限指定したそうな場所だけ指定すれば残りは適当に埋めたOKResponseを返す
func GenerateSimplePostObjectOKResponse(bucket string, object string, contentType string, size int64) *WriteObjectResponse {
	return &WriteObjectResponse{
		Kind:                    "storage#object",
		ID:                      fmt.Sprintf("%s/%s/1570087904014021", bucket, object),
		SelfLink:                fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s/o/%s", bucket, object),
		Name:                    object,
		Bucket:                  bucket,
		Generation:              "1570087904014021",
		Metageneration:          "1",
		ContentType:             contentType,
		TimeCreated:             time.Now(),
		Updated:                 time.Now(),
		StorageClass:            "REGIONAL",
		TimeStorageClassUpdated: time.Now(),
		Size:                    fmt.Sprintf("%d", size),
		Md5Hash:                 "3fv0VXHjk3nCc3znVNrcRw==",
		MediaLink:               fmt.Sprintf("https://www.googleapis.com/download/storage/v1/b/%s/o/%s?generation=1570087904014021&alt=media", bucket, object),
		ACL: []*ACL{
			{
				Kind:       "storage#objectAccessControl",
				ID:         fmt.Sprintf("%s/%s/1570087904014021/project-owners-168610916801", bucket, object),
				SelfLink:   fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s/o/%s/acl/project-owners-168610916801", bucket, object),
				Bucket:     bucket,
				Object:     object,
				Generation: "1570087904014021",
				Entity:     "project-owners-168610916801",
				Role:       "OWNER",
				ProjectTeam: &ProjectTeam{
					ProjectNumber: "168610916801",
					Team:          "owners",
				},
				Etag:  "CMXdo57J/+QCEAE=",
				Email: "",
			},
			{
				Kind:       "storage#objectAccessControl",
				ID:         fmt.Sprintf("%s/%s/1570087904014021/project-owners-168610916801", bucket, object),
				SelfLink:   fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s/o/%s/acl/project-owners-168610916801", bucket, object),
				Bucket:     bucket,
				Object:     object,
				Generation: "1570087904014021",
				Entity:     "project-owners-168610916801",
				Role:       "OWNER",
				Etag:       "CMXdo57J/+QCEAE=",
				Email:      "faker@example.com",
			},
		},
		Owner: &Owner{
			Entity: "user-faker@example.com",
		},
		Crc32C: "vOMu5Q==",
		Etag:   "CMXdo57J/+QCEAE=",
	}
}

func (faker *Faker) AddPostObjectOKResponse(bucket string, object string, header map[string][]string, resp *WriteObjectResponse) error {
	body, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	r := ioutil.NopCloser(bytes.NewReader(body))
	res := &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Header:        header,
		Body:          r,
		ContentLength: int64(len(body)),
	}
	return faker.AddResponse(fmt.Sprintf("https://storage.googleapis.com/upload/storage/v1/b/%s/o?alt=json&name=%s&prettyPrint=false&projection=full&uploadType=multipart", bucket, object), http.MethodPost, res)
}

var _ http.RoundTripper = &Transport{}

type Transport struct {
	t             *testing.T
	Transport     http.RoundTripper
	fakeResponses *fakeResponses
}

func (tran *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	fake, err := tran.fakeResponses.Get(req.URL.String(), req.Method)
	if err != nil {
		tran.t.Fatal("unexpected: ", err)
	}
	return fake, nil
}

type fakeResponses struct {
	// responseMap is RequestされたObjectのURLに対して返すResponseを保持する
	// 同じObjectURLへのRequestの場合、順番に返していく
	responseMap map[string]*http.Response

	// requestCountMap is RequestされたObjectのURLをカウントする
	requestCountMap map[string]int
}

func (f *fakeResponses) keyForResponseMap(url string, method string, count int) string {
	return fmt.Sprintf("%s-%s-%d", url, strings.ToUpper(method), count)
}

func (f *fakeResponses) keyForRequestCountMap(url string, method string) string {
	return fmt.Sprintf("%s-%s", url, strings.ToUpper(method))
}

func (f *fakeResponses) Add(url string, method string, response *http.Response) {
	for i := 0; ; i++ {
		key := f.keyForResponseMap(url, method, i)
		_, ok := f.responseMap[key]
		if ok {
			continue
		}
		f.responseMap[key] = response
		break
	}
}

func (f *fakeResponses) Get(url string, method string) (*http.Response, error) {
	count, ok := f.requestCountMap[f.keyForRequestCountMap(url, method)]
	if !ok {
		count = 0
	}

	v, ok := f.responseMap[f.keyForResponseMap(url, method, count)]
	if !ok {
		return nil, fmt.Errorf("response is not registered. %s:%s RequestCount is %d", method, url, count+1)
	}
	return v, nil
}

func GetObjectOKResponseSample() *http.Response {
	header := make(map[string][]string)
	header["Accept-Ranges"] = []string{"bytes"}
	header["Age"] = []string{"268"}
	header["Alt-Svc"] = []string{`quic=":443"; ma=2592000; v="46,43",h3-Q046=":443"; ma=2592000,h3-Q043=":443"; ma=2592000`}
	header["Cache-Control"] = []string{"public", "max-age=3600"}
	header["Content-Length"] = []string{"25"}
	header["Content-Type"] = []string{"text/plain"}
	header["Date"] = []string{"Mon, 30 Sep 2019 10:23:16 GMT"}
	header["Etag"] = []string{"c4d22707e0d79bd01e33fe19a5e21487"}
	header["Expires"] = []string{"Mon, 30 Sep 2019 11:23:16 GMT"}
	header["Last-Modified"] = []string{"Mon, 30 Sep 2019 10:01:47 GMT"}
	header["X-Goog-Generation"] = []string{"1569837707444808"}
	header["X-Goog-Hash"] = []string{"crc32c=CrEDEg== md5=xNInB+DXm9AeM/4ZpeIUhw=="}
	header["X-Goog-Metageneration"] = []string{"2"}
	header["X-Goog-Storage-Class"] = []string{"REGIONAL"}
	header["X-Goog-Stored-Content-Encoding"] = []string{"identity"}
	header["X-Goog-Stored-Content-Length"] = []string{"25"}
	header["X-Guploader-Uploadid"] = []string{"AEnB2UoygSa1dB8aXstLosALQoifLpXnQ5kIx_lyzTyIvk5bFuIcG7nqk-sR5GdihmWdTtHDuiKCtSgxyRJ9iLJmHnQ7RHmvoQ"}

	r := ioutil.NopCloser(strings.NewReader(`{"message":"Hello Hoge"}`))

	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Header:        header,
		Body:          r,
		ContentLength: 25,
	}
}

func PostObjectOKResponseSample() *http.Response {
	header := make(map[string][]string)
	header["Server"] = []string{"UploadServer"}
	header["Alt-Svc"] = []string{`quic=":443"; ma=2592000; v="46,43",h3-Q048=":443"; ma=2592000,h3-Q046=":443"; ma=2592000,h3-Q043=":443"; ma=2592000`}
	header["Vary"] = []string{"Origin"}
	header["Vary"] = []string{"X-Origin"}
	header["Content-Type"] = []string{"application/json; charset=UTF-8"}
	header["Cache-Control"] = []string{"no-cache, no-store, max-age=0, must-revalidate"}
	header["Date"] = []string{"Thu, 03 Oct 2019 07:31:44 GMT"}
	header["Content-Length"] = []string{"2324"}
	header["X-Guploader-Uploadid"] = []string{"AEnB2UpF0rRDJSlY8seVYqjxCchiX2GwvYwiGqkFfaduXRlzuNpGEDdlCsKtpvVe5gn0WMsW3HSeqFw4nqyNZ0v3apu9Il_VMw"}
	header["Etag"] = []string{"CMXdo57J/+QCEAE="}
	header["Pragma"] = []string{"no-cache"}
	header["Expires"] = []string{"Mon, 01 Jan 1990 00:00:00 GMT"}
	r := ioutil.NopCloser(strings.NewReader(`eyJraW5kIjoic3RvcmFnZSNvYmplY3QiLCJpZCI6ImhvZ2UvcG9zdC50eHQvMTU3MDA4NzkwNDAxNDAyMSIsInNlbGZMaW5rIjoiaHR0cHM6Ly93d3cuZ29vZ2xlYXBpcy5jb20vc3RvcmFnZS92MS9iL2hvZ2Uvby9wb3N0LnR4dCIsIm5hbWUiOiJwb3N0LnR4dCIsImJ1Y2tldCI6ImhvZ2UiLCJnZW5lcmF0aW9uIjoiMTU3MDA4NzkwNDAxNDAyMSIsIm1ldGFnZW5lcmF0aW9uIjoiMSIsImNvbnRlbnRUeXBlIjoidGV4dC9wbGFpbjsgY2hhcnNldD11dGYtOCIsInRpbWVDcmVhdGVkIjoiMjAxOS0xMC0wM1QwNzozMTo0NC4wMTNaIiwidXBkYXRlZCI6IjIwMTktMTAtMDNUMDc6MzE6NDQuMDEzWiIsInN0b3JhZ2VDbGFzcyI6IlJFR0lPTkFMIiwidGltZVN0b3JhZ2VDbGFzc1VwZGF0ZWQiOiIyMDE5LTEwLTAzVDA3OjMxOjQ0LjAxM1oiLCJzaXplIjoiMjQiLCJtZDVIYXNoIjoiM2Z2MFZYSGprM25DYzN6blZOcmNSdz09IiwibWVkaWFMaW5rIjoiaHR0cHM6Ly93d3cuZ29vZ2xlYXBpcy5jb20vZG93bmxvYWQvc3RvcmFnZS92MS9iL2hvZ2Uvby9wb3N0LnR4dD9nZW5lcmF0aW9uPTE1NzAwODc5MDQwMTQwMjEmYWx0PW1lZGlhIiwiYWNsIjpbeyJraW5kIjoic3RvcmFnZSNvYmplY3RBY2Nlc3NDb250cm9sIiwiaWQiOiJob2dlL3Bvc3QudHh0LzE1NzAwODc5MDQwMTQwMjEvcHJvamVjdC1vd25lcnMtMTY4NjEwOTE2ODAxIiwic2VsZkxpbmsiOiJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9zdG9yYWdlL3YxL2IvaG9nZS9vL3Bvc3QudHh0L2FjbC9wcm9qZWN0LW93bmVycy0xNjg2MTA5MTY4MDEiLCJidWNrZXQiOiJob2dlIiwib2JqZWN0IjoicG9zdC50eHQiLCJnZW5lcmF0aW9uIjoiMTU3MDA4NzkwNDAxNDAyMSIsImVudGl0eSI6InByb2plY3Qtb3duZXJzLTE2ODYxMDkxNjgwMSIsInJvbGUiOiJPV05FUiIsInByb2plY3RUZWFtIjp7InByb2plY3ROdW1iZXIiOiIxNjg2MTA5MTY4MDEiLCJ0ZWFtIjoib3duZXJzIn0sImV0YWciOiJDTVhkbzU3Si8rUUNFQUU9In0seyJraW5kIjoic3RvcmFnZSNvYmplY3RBY2Nlc3NDb250cm9sIiwiaWQiOiJob2dlL3Bvc3QudHh0LzE1NzAwODc5MDQwMTQwMjEvcHJvamVjdC1lZGl0b3JzLTE2ODYxMDkxNjgwMSIsInNlbGZMaW5rIjoiaHR0cHM6Ly93d3cuZ29vZ2xlYXBpcy5jb20vc3RvcmFnZS92MS9iL2hvZ2Uvby9wb3N0LnR4dC9hY2wvcHJvamVjdC1lZGl0b3JzLTE2ODYxMDkxNjgwMSIsImJ1Y2tldCI6ImhvZ2UiLCJvYmplY3QiOiJwb3N0LnR4dCIsImdlbmVyYXRpb24iOiIxNTcwMDg3OTA0MDE0MDIxIiwiZW50aXR5IjoicHJvamVjdC1lZGl0b3JzLTE2ODYxMDkxNjgwMSIsInJvbGUiOiJPV05FUiIsInByb2plY3RUZWFtIjp7InByb2plY3ROdW1iZXIiOiIxNjg2MTA5MTY4MDEiLCJ0ZWFtIjoiZWRpdG9ycyJ9LCJldGFnIjoiQ01YZG81N0ovK1FDRUFFPSJ9LHsia2luZCI6InN0b3JhZ2Ujb2JqZWN0QWNjZXNzQ29udHJvbCIsImlkIjoiaG9nZS9wb3N0LnR4dC8xNTcwMDg3OTA0MDE0MDIxL3Byb2plY3Qtdmlld2Vycy0xNjg2MTA5MTY4MDEiLCJzZWxmTGluayI6Imh0dHBzOi8vd3d3Lmdvb2dsZWFwaXMuY29tL3N0b3JhZ2UvdjEvYi9ob2dlL28vcG9zdC50eHQvYWNsL3Byb2plY3Qtdmlld2Vycy0xNjg2MTA5MTY4MDEiLCJidWNrZXQiOiJob2dlIiwib2JqZWN0IjoicG9zdC50eHQiLCJnZW5lcmF0aW9uIjoiMTU3MDA4NzkwNDAxNDAyMSIsImVudGl0eSI6InByb2plY3Qtdmlld2Vycy0xNjg2MTA5MTY4MDEiLCJyb2xlIjoiUkVBREVSIiwicHJvamVjdFRlYW0iOnsicHJvamVjdE51bWJlciI6IjE2ODYxMDkxNjgwMSIsInRlYW0iOiJ2aWV3ZXJzIn0sImV0YWciOiJDTVhkbzU3Si8rUUNFQUU9In0seyJraW5kIjoic3RvcmFnZSNvYmplY3RBY2Nlc3NDb250cm9sIiwiaWQiOiJob2dlL3Bvc3QudHh0LzE1NzAwODc5MDQwMTQwMjEvdXNlci1zaW5tZXRhbEBtZXJjYXJpLmNvbSIsInNlbGZMaW5rIjoiaHR0cHM6Ly93d3cuZ29vZ2xlYXBpcy5jb20vc3RvcmFnZS92MS9iL2hvZ2Uvby9wb3N0LnR4dC9hY2wvdXNlci1zaW5tZXRhbEBtZXJjYXJpLmNvbSIsImJ1Y2tldCI6ImhvZ2UiLCJvYmplY3QiOiJwb3N0LnR4dCIsImdlbmVyYXRpb24iOiIxNTcwMDg3OTA0MDE0MDIxIiwiZW50aXR5IjoidXNlci1zaW5tZXRhbEBtZXJjYXJpLmNvbSIsInJvbGUiOiJPV05FUiIsImVtYWlsIjoic2lubWV0YWxAbWVyY2FyaS5jb20iLCJldGFnIjoiQ01YZG81N0ovK1FDRUFFPSJ9XSwib3duZXIiOnsiZW50aXR5IjoidXNlci1zaW5tZXRhbEBtZXJjYXJpLmNvbSJ9LCJjcmMzMmMiOiJ2T011NVE9PSIsImV0YWciOiJDTVhkbzU3Si8rUUNFQUU9In0=`))

	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Header:        header,
		Body:          r,
		ContentLength: 2324,
	}
}
