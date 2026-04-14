package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/parquet-go/parquet-go"
)

type SessionEvent struct {
	EventID       string  `parquet:"event_id"`
	EventType     string  `parquet:"event_type"`       // impression, click, close, complete
	SessionID     string  `parquet:"session_id"`
	PublisherOrgID string `parquet:"publisher_org_id"`
	AppBundleID   string  `parquet:"app_bundle_id"`
	Platform      string  `parquet:"platform"`          // ios, android
	CampaignID    string  `parquet:"campaign_id"`
	BrandOrgID    string  `parquet:"brand_org_id"`
	CreativeID    string  `parquet:"creative_id"`
	Country       string  `parquet:"country"`
	DurationMs    int64   `parquet:"duration_ms"`
	Revenue       float64 `parquet:"revenue"`
	Timestamp     int64   `parquet:"timestamp"`         // unix millis
}

var (
	publishers = []struct {
		orgID    string
		apps     []struct{ bundleID, platform string }
	}{
		{"pub-001", []struct{ bundleID, platform string }{
			{"com.puzzle.master", "ios"},
			{"com.puzzle.master", "android"},
		}},
		{"pub-002", []struct{ bundleID, platform string }{
			{"com.fitness.daily", "ios"},
		}},
		{"pub-003", []struct{ bundleID, platform string }{
			{"com.news.reader", "android"},
			{"com.news.reader", "ios"},
		}},
	}

	brands = []struct {
		orgID      string
		campaigns  []struct{ id, creativeID string }
	}{
		{"brand-001", []struct{ id, creativeID string }{
			{"camp-001", "cre-001"},
			{"camp-002", "cre-002"},
		}},
		{"brand-002", []struct{ id, creativeID string }{
			{"camp-003", "cre-003"},
		}},
	}

	countries   = []string{"US", "GB", "DE", "FR", "JP", "BR", "IN", "CA"}
	eventTypes  = []string{"impression", "impression", "impression", "click", "close", "complete"} // weighted
)

func main() {
	rng := rand.New(rand.NewSource(42))
	numEvents := 50_000
	now := time.Now()

	events := make([]SessionEvent, 0, numEvents)
	for i := 0; i < numEvents; i++ {
		pub := publishers[rng.Intn(len(publishers))]
		app := pub.apps[rng.Intn(len(pub.apps))]
		brand := brands[rng.Intn(len(brands))]
		camp := brand.campaigns[rng.Intn(len(brand.campaigns))]

		evtType := eventTypes[rng.Intn(len(eventTypes))]
		var durationMs int64
		var revenue float64
		switch evtType {
		case "impression":
			durationMs = int64(rng.Intn(5000) + 500)
			revenue = float64(rng.Intn(50)+5) / 1000.0 // $0.005 - $0.055
		case "click":
			durationMs = int64(rng.Intn(2000) + 200)
			revenue = float64(rng.Intn(200)+50) / 1000.0
		case "complete":
			durationMs = int64(rng.Intn(30000) + 5000)
			revenue = float64(rng.Intn(100)+10) / 1000.0
		case "close":
			durationMs = int64(rng.Intn(3000) + 100)
		}

		ts := now.Add(-time.Duration(rng.Intn(30*24)) * time.Hour) // last 30 days

		events = append(events, SessionEvent{
			EventID:        fmt.Sprintf("evt-%08d", i),
			EventType:      evtType,
			SessionID:      fmt.Sprintf("sess-%06d", rng.Intn(30000)),
			PublisherOrgID: pub.orgID,
			AppBundleID:    app.bundleID,
			Platform:       app.platform,
			CampaignID:     camp.id,
			BrandOrgID:     brand.orgID,
			CreativeID:     camp.creativeID,
			Country:        countries[rng.Intn(len(countries))],
			DurationMs:     durationMs,
			Revenue:        revenue,
			Timestamp:      ts.UnixMilli(),
		})
	}

	// Write parquet file locally
	tmpFile := "/tmp/session_events.parquet"
	f, err := os.Create(tmpFile)
	if err != nil {
		log.Fatalf("create file: %v", err)
	}
	writer := parquet.NewGenericWriter[SessionEvent](f)
	if _, err := writer.Write(events); err != nil {
		log.Fatalf("write parquet: %v", err)
	}
	if err := writer.Close(); err != nil {
		log.Fatalf("close writer: %v", err)
	}
	f.Close()
	log.Printf("Wrote %d events to %s", len(events), tmpFile)

	// Upload to MinIO
	endpoint := envOr("MINIO_ENDPOINT", "localhost:9000")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("minio client: %v", err)
	}

	bucket := "brandmoment-events"
	ctx := context.Background()
	_, err = client.FPutObject(ctx, bucket, "v1/session_events.parquet", tmpFile, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		log.Fatalf("upload to minio: %v", err)
	}
	log.Printf("Uploaded to s3://%s/v1/session_events.parquet", bucket)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
