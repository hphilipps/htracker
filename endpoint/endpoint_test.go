package endpoint

import (
	"context"
	"os"
	"testing"
	"time"

	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"golang.org/x/exp/slog"
)

func TestMiddleware(t *testing.T) {

	sub1 := &htracker.Subscription{URL: "http://site1.example/blah", Filter: "foo", ContentType: "text", Interval: time.Hour}
	sub2 := &htracker.Subscription{URL: "http://site1.example/blub", Filter: "bar", ContentType: "byte", Interval: time.Minute}
	sub3 := &htracker.Subscription{URL: "http://unknown.example", Filter: "foo", ContentType: "text", Interval: time.Hour}

	content1 := []byte("This is Site1")
	content2 := []byte("This is Site2")

	req1 := UpdateReq{
		site: &htracker.Site{sub1, time.Now(), time.Now(), content1, service.Checksum(content1), ""},
	}
	req2 := UpdateReq{
		site: &htracker.Site{sub2, time.Now(), time.Now(), content2, service.Checksum(content2), ""},
	}
	req3 := UpdateReq{
		site: &htracker.Site{sub1, time.Now(), time.Now(), content2, service.Checksum(content2), ""},
	}
	req4 := GetReq{subscription: sub1}
	req5 := GetReq{subscription: sub2}
	req6 := GetReq{subscription: sub3}

	ctx := context.Background()
	logger := slog.New(slog.HandlerOptions{Level: slog.LevelDebug}.NewTextHandler(os.Stdout))
	storage := memory.NewSiteStorage(logger)
	svc := service.NewSiteArchive(storage)
	updateEp := MakeUpdateEndpoint(svc)
	updateEp = LoggingMiddleware[UpdateReq, UpdateResp](logger)(updateEp)
	getEp := MakeGetEndpoint(svc)
	getEp = LoggingMiddleware[GetReq, GetResp](logger)(getEp)

	updResp, err := updateEp(ctx, req1)
	if err != nil {
		t.Fatal(err)
	}
	if err := updResp.Failed(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Resp: %v", updResp)

	updResp, err = updateEp(ctx, req2)
	if err != nil {
		t.Fatal(err)
	}
	if err := updResp.Failed(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Resp: %v", updResp)

	updResp, err = updateEp(ctx, req3)
	if err != nil {
		t.Fatal(err)
	}
	if err := updResp.Failed(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Resp: %v", updResp)

	getResp, err := getEp(ctx, req4)
	if err != nil {
		t.Fatal(err)
	}
	if err := getResp.Failed(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Resp: %v", getResp.site.Checksum)

	getResp, err = getEp(ctx, req5)
	if err != nil {
		t.Fatal(err)
	}
	if err := getResp.Failed(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Resp: %v", getResp.site.Checksum)

	getResp, err = getEp(ctx, req6)
	if err != nil {
		t.Fatal(err)
	}
	if err := getResp.Failed(); err == nil {
		t.Error("Expected get to fail, but got nil error")
	}
	t.Logf("Resp: %v", getResp.site.Checksum)
}
