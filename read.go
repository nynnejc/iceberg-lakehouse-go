package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// RawProduct matcher skemaet fra jeres Glue-tabel `raw`.
type RawProduct struct {
	ProductName   string `json:"productname"`
	Color         string `json:"color"`
	Department    string `json:"department"`
	Product       string `json:"product"`
	ImageURL      string `json:"imageurl"`
	DateSoldSince string `json:"datesoldsince"`
	DateSoldUntil string `json:"datesolduntil"`
	Price         int    `json:"price"`
	Campaign      string `json:"campaign"`
}

const (
	bucket = "sdl-immersion-day-051103554105"
	// Skift til den partition, du faktisk vil læse.
	prefix = "raw/year=2026/month=08/day=31/hour=10/"
)

func readRawRecords(ctx context.Context, s3Client *s3.Client) ([]RawProduct, error) {
	var records []RawProduct

	listOut, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	for _, obj := range listOut.Contents {
		getOut, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    obj.Key,
		})
		if err != nil {
			return nil, fmt.Errorf("get object %s: %w", *obj.Key, err)
		}

		gz, err := gzip.NewReader(getOut.Body)
		if err != nil {
			getOut.Body.Close()
			return nil, fmt.Errorf("gzip reader for %s: %w", *obj.Key, err)
		}

		raw, err := io.ReadAll(gz)
		gz.Close()
		getOut.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", *obj.Key, err)
		}

		// Firehose leverer typisk newline-delimited JSON (én record per linje).
		decoder := json.NewDecoder(bytes.NewReader(raw))
		for decoder.More() {
			var rec RawProduct
			if err := decoder.Decode(&rec); err != nil {
				log.Printf("springer ugyldig record over i %s: %v", *obj.Key, err)
				continue
			}
			records = append(records, rec)
		}
	}

	return records, nil
}
