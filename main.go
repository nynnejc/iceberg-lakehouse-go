package main

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/iceberg-go/catalog/glue"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	ctx := context.Background()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("eu-north-1"))
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	raw, err := readRawRecords(ctx, s3Client)
	if err != nil {
		log.Fatalf("read raw records: %v", err)
	}
	log.Printf("Læste %d rå records", len(raw))

	clean := transform(raw)
	log.Printf("Efter transformation: %d records tilbage", len(clean))

	glueCat := glue.NewCatalog(glue.WithAwsConfig(cfg))

	if err := writeToIceberg(ctx, glueCat, clean); err != nil {
		log.Fatalf("write to iceberg: %v", err)
	}

	athenaClient := athena.NewFromConfig(cfg)
	query := fmt.Sprintf(`SELECT department, COUNT(*) FROM "%s"."%s" GROUP BY department`, database, tableName)
	if err := runAthenaQuery(ctx, athenaClient, query, "s3://sdl-immersion-day-051103554105/athena-results/"); err != nil {
		log.Fatalf("verify: %v", err)
	}
}
