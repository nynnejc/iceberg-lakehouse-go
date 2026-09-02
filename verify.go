package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/athena/types"
)

func runAthenaQuery(ctx context.Context, athenaClient *athena.Client, query, outputLocation string) error {
	start, err := athenaClient.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString: aws.String(query),
		QueryExecutionContext: &types.QueryExecutionContext{
			Database: aws.String(database),
		},
		ResultConfiguration: &types.ResultConfiguration{
			OutputLocation: aws.String(outputLocation), // fx s3://din-bucket/athena-results/
		},
	})
	if err != nil {
		return fmt.Errorf("start query: %w", err)
	}

	queryID := *start.QueryExecutionId

	for {
		status, err := athenaClient.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{
			QueryExecutionId: aws.String(queryID),
		})
		if err != nil {
			return fmt.Errorf("get query execution: %w", err)
		}

		state := status.QueryExecution.Status.State
		if state == types.QueryExecutionStateSucceeded {
			break
		}
		if state == types.QueryExecutionStateFailed || state == types.QueryExecutionStateCancelled {
			return fmt.Errorf("query %s: %s", state, aws.ToString(status.QueryExecution.Status.StateChangeReason))
		}
		time.Sleep(1 * time.Second)
	}

	results, err := athenaClient.GetQueryResults(ctx, &athena.GetQueryResultsInput{
		QueryExecutionId: aws.String(queryID),
	})
	if err != nil {
		return fmt.Errorf("get query results: %w", err)
	}

	for _, row := range results.ResultSet.Rows {
		var cols []string
		for _, d := range row.Data {
			cols = append(cols, aws.ToString(d.VarCharValue))
		}
		fmt.Println(cols)
	}

	return nil
}
