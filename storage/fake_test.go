package storage_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/sinmetalcraft/gcpfaker/hook/hars"
	"github.com/vvakame/go-harlog"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	storagefaker "github.com/sinmetalcraft/gcpfaker/storage"
)

func TestNewFakerWithoutTesting(t *testing.T) {
	ctx := context.Background()

	faker := storagefaker.NewFakerWithoutTesting()

	stg, err := storage.NewClient(ctx, option.WithHTTPClient(faker.Client))
	if err != nil {
		t.Fatal(err)
	}

	const bucket = "sinmetal-ci-fake"
	const object = "hoge.txt"
	const body = `{"message":"Hello Hoge"}`
	header := make(map[string][]string)
	header["content-type"] = []string{"application/json;utf-8"}
	header["content-length"] = []string{fmt.Sprintf("%d", len([]byte(body)))}
	r := io.NopCloser(strings.NewReader(body))
	res := &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Header:        header,
		Body:          r,
		ContentLength: int64(len([]byte(body))),
	}
	if err := faker.AddGetObjectResponse(bucket, object, res); err != nil {
		t.Fatal(err)
	}

	reader, err := stg.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := reader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(string(got), body) {
		t.Errorf("unexpected response body got %s", string(got))
	}
}

func TestGetObject(t *testing.T) {
	ctx := context.Background()

	faker := storagefaker.NewFaker(t)

	stg, err := storage.NewClient(ctx, option.WithHTTPClient(faker.Client))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		want  string
		count int
	}{
		{"hello", `{"message":"Hello Hoge"}`, 2},
		{"world", `{"message":"Hello World"}`, 2},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			const bucket = "sinmetal-ci-fake"
			const object = "hoge.txt"
			for i := 0; i < tt.count; i++ {
				header := make(map[string][]string)
				header["content-type"] = []string{"application/json;utf-8"}
				header["content-length"] = []string{fmt.Sprintf("%d", len([]byte(tt.want)))}
				r := io.NopCloser(strings.NewReader(tt.want))
				res := &http.Response{
					Status:        "200 OK",
					StatusCode:    http.StatusOK,
					Header:        header,
					Body:          r,
					ContentLength: int64(len([]byte(tt.want))),
				}

				if err := faker.AddGetObjectResponse(bucket, object, res); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < tt.count; i++ {
				reader, err := stg.Bucket(bucket).Object(object).NewReader(ctx)
				if err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				if err := reader.Close(); err != nil {
					t.Fatal(err)
				}
				if !cmp.Equal(string(got), tt.want) {
					t.Errorf("unexpected response body got %s", string(got))
				}
			}
		})
	}
}

func TestRealGetObjectHar(t *testing.T) {
	ctx := context.Background()

	hc, err := google.DefaultClient(ctx, storage.ScopeReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	// inject HAR logger!
	har := &harlog.Transport{
		Transport: hc.Transport,
	}
	hc.Transport = har
	stg, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}

	_, err = stg.Bucket("sinmetal-ci-fake").Object("hoge.txt").NewReader(ctx)
	if err != nil {
		t.Fatal(err)
	}

	hars.Compare(t, "object.get.har.golden", har.HAR())
}

func TestPostObject(t *testing.T) {
	ctx := context.Background()

	faker := storagefaker.NewFaker(t)

	stg, err := storage.NewClient(ctx, option.WithHTTPClient(faker.Client))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		want  string
		count int
	}{
		{"hello", `{"message":"Hello Hoge"}`, 2},
		{"world", `{"message":"Hello World"}`, 2},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			const bucket = "sinmetal-ci-fake"
			const object = "hoge.txt"

			for i := 0; i < tt.count; i++ {
				resp := storagefaker.GenerateSimplePostObjectOKResponse(bucket, object, "application/json; charset=UTF-8", uint64(len([]byte(tt.want))))
				if err := faker.AddPostObjectOKResponse(bucket, object, make(map[string][]string), resp); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < tt.count; i++ {
				w := stg.Bucket(bucket).Object(object).NewWriter(ctx)
				n, err := w.Write([]byte(tt.want))
				if err != nil {
					t.Fatal(err)
				}
				if n < 1 {
					t.Error("write result bytes is Zero")
				}
				if err := w.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPostObjectHar(t *testing.T) {
	ctx := context.Background()

	hc, err := google.DefaultClient(ctx, storage.ScopeReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	// inject HAR logger!
	har := &harlog.Transport{
		Transport: hc.Transport,
	}
	hc.Transport = har
	stg, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}

	w := stg.Bucket("sinmetal-ci-fake").Object("post.txt").NewWriter(ctx)
	_, err = w.Write([]byte(`{"message":"hello fake"}`))
	if err != nil {
		t.Fatal("unexpected: ", err)
	}
	w.ContentType = "application/json"
	if err := w.Close(); err != nil {
		t.Fatal("unexpected: ", err)
	}

	hars.Compare(t, "object.post.har.golden", har.HAR())
}

func TestPostObjectHarToCode(t *testing.T) {
	hars.LogFakeResponseCode(t, "object.post.har.golden")
}

// TestObjectListACL is 指定したObjectのACLListを取得する処理にMockResponseを返す
func TestObjectListACL(t *testing.T) {
	ctx := context.Background()

	faker := storagefaker.NewFaker(t)

	stg, err := storage.NewClient(ctx, option.WithHTTPClient(faker.Client))
	if err != nil {
		t.Fatal(err)
	}

	const bucket = "sinmetal-ci-fake"
	const object = "hoge.txt"
	const email = "example@example.com"
	rules := []storage.ACLRule{
		{
			Email:  email,
			Role:   storage.RoleOwner,
			Entity: storage.ACLEntity(fmt.Sprintf("user-%s", email)),
		},
	}
	if err := faker.AddListObjectACLOKResponse(bucket, object, rules); err != nil {
		t.Fatal(err)
	}

	got, err := stg.Bucket(bucket).Object(object).ACL().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if e, g := 1, len(got); e != g {
		t.Errorf("want roles.len %d but got %d", e, g)
	}
	if !cmp.Equal(got, rules) {
		t.Errorf("unexpected response got %+v", got)
	}
}

// TestRealObjectListACL is 指定したObjectのACLListを取得するAPIを実行してharとして保存する
func TestRealObjectListACL(t *testing.T) {
	ctx := context.Background()

	hc, err := google.DefaultClient(ctx, storage.ScopeReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	// inject HAR logger!
	har := &harlog.Transport{
		Transport: hc.Transport,
	}
	hc.Transport = har
	stg, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}

	_, err = stg.Bucket("sinmetal-ci-fake").Object("hoge.txt").ACL().List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	hars.Compare(t, "object.acl.list.har.golden", har.HAR())
}

// TestAddUpdateObjectAttrsSimpleOKResponse
// Object.Attrsを更新するがどのように変化したかのResultはチェックしないSimpleなResponseを返すものを試す
func TestAddUpdateObjectAttrsSimpleOKResponse(t *testing.T) {
	ctx := context.Background()

	faker := storagefaker.NewFaker(t)

	stg, err := storage.NewClient(ctx, option.WithHTTPClient(faker.Client))
	if err != nil {
		t.Fatal(err)
	}

	const bucket = "sinmetal-ci-fake"
	const object = "hoge.txt"
	if err := faker.AddUpdateObjectAttrsSimpleOKResponse(bucket, object); err != nil {
		t.Fatal(err)
	}

	// 返ってくるObjectAttrsが特に意味が無いものなので、チェックしない
	_, err = stg.Bucket(bucket).Object(object).Update(ctx, storage.ObjectAttrsToUpdate{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRealObjectUpdateACL(t *testing.T) {
	ctx := context.Background()

	const bucket = "sinmetal-ci-fake"
	const object = "hoge.txt"

	defaultClient, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roles, err := defaultClient.Bucket(bucket).Object(object).ACL().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roles = append(roles, storage.ACLRule{Entity: storage.ACLEntity(fmt.Sprintf("user-%s", "example@example.com")), Role: storage.RoleReader})

	hc, err := google.DefaultClient(ctx, storage.ScopeReadWrite)
	if err != nil {
		t.Fatal(err)
	}

	// inject HAR logger!
	har := &harlog.Transport{
		Transport: hc.Transport,
	}
	hc.Transport = har
	stg, err := storage.NewClient(ctx, option.WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}

	_, err = stg.Bucket("sinmetal-ci-fake").Object("hoge.txt").Update(ctx, storage.ObjectAttrsToUpdate{ACL: roles})
	if err != nil {
		t.Fatal(err)
	}

	hars.Compare(t, "object.update.attrs.har.golden", har.HAR())
}
