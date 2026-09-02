package main

import (
	"context"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/glue"
	"github.com/apache/iceberg-go/table"
)

const (
	database  = "sdl-lake-demo-data"
	tableName = "products_silver"
)

// buildArrowTable konverterer de rensede Go-structs til en Arrow-tabel,
// som Iceberg-Go bruger som skriveformat.
func buildArrowTable(records []CleanProduct) arrow.Table {
	pool := memory.NewGoAllocator()

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "product_name", Type: arrow.BinaryTypes.String},
		{Name: "color", Type: arrow.BinaryTypes.String},
		{Name: "department", Type: arrow.BinaryTypes.String},
		{Name: "price", Type: arrow.PrimitiveTypes.Int64},
		{Name: "campaign", Type: arrow.BinaryTypes.String},
		{Name: "is_active", Type: arrow.FixedWidthTypes.Boolean},
	}, nil)

	nameB := array.NewStringBuilder(pool)
	colorB := array.NewStringBuilder(pool)
	deptB := array.NewStringBuilder(pool)
	priceB := array.NewInt64Builder(pool)
	campB := array.NewStringBuilder(pool)
	activeB := array.NewBooleanBuilder(pool)

	for _, r := range records {
		nameB.Append(r.ProductName)
		colorB.Append(r.Color)
		deptB.Append(r.Department)
		priceB.Append(r.Price)
		campB.Append(r.Campaign)
		activeB.Append(r.IsActive)
	}

	rec := array.NewRecord(schema, []arrow.Array{
		nameB.NewArray(), colorB.NewArray(), deptB.NewArray(),
		priceB.NewArray(), campB.NewArray(), activeB.NewArray(),
	}, int64(len(records)))

	return array.NewTableFromRecords(schema, []arrow.Record{rec})
}

// ensureTable opretter Iceberg-tabellen, hvis den ikke findes endnu.
func ensureTable(ctx context.Context, cat catalog.Catalog, ident table.Identifier) (*iceberg.Schema, error) {
	schema := iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "product_name", Type: iceberg.PrimitiveTypes.String},
		iceberg.NestedField{ID: 2, Name: "color", Type: iceberg.PrimitiveTypes.String},
		iceberg.NestedField{ID: 3, Name: "department", Type: iceberg.PrimitiveTypes.String},
		iceberg.NestedField{ID: 4, Name: "price", Type: iceberg.PrimitiveTypes.Int64},
		iceberg.NestedField{ID: 5, Name: "campaign", Type: iceberg.PrimitiveTypes.String},
		iceberg.NestedField{ID: 6, Name: "is_active", Type: iceberg.PrimitiveTypes.Bool},
	)

	if exists, _ := cat.CheckTableExists(ctx, ident); exists {
		return schema, nil
	}

	_, err := cat.CreateTable(ctx, ident, schema)
	return schema, err
}

func writeToIceberg(ctx context.Context, glueCat *glue.Catalog, clean []CleanProduct) error {
	ident := glue.TableIdentifier(database, tableName)

	if _, err := ensureTable(ctx, glueCat, ident); err != nil {
		return fmt.Errorf("ensure table: %w", err)
	}

	tbl, err := glueCat.LoadTable(ctx, ident)
	if err != nil {
		return fmt.Errorf("load table: %w", err)
	}

	arrowTbl := buildArrowTable(clean)

	updated, err := tbl.AppendTable(ctx, arrowTbl, 1024, nil)
	if err != nil {
		return fmt.Errorf("append table: %w", err)
	}

	fmt.Printf("Skrev %d rækker. Nyt snapshot: %d\n",
		len(clean), updated.CurrentSnapshot().SnapshotID)

	return nil
}
