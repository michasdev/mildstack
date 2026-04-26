package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	sqssdk "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/michasdev/mildstack/core/internal/delivery/cli"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	dynamoapp "github.com/michasdev/mildstack/core/internal/resources/dynamodb/application"
)

func TestRegisterServiceRoutesRegistersS3BeforeServing(t *testing.T) {
	t.Helper()

	root := composition.DefaultRoot(fmt.Sprintf("test-instance-smoke-%d", time.Now().UnixNano()))
	manager := runtime.New(root.Services)
	router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)

	if err := registerServiceRoutes(router.Registrar(), root.Services); err != nil {
		t.Fatalf("register service routes: %v", err)
	}

	entry, ok := router.Registrar().Service("s3")
	if !ok {
		t.Fatal("expected s3 service to be registered")
	}
	if got, want := len(entry.Routes), 62; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	found := false
	for _, route := range entry.Routes {
		if route.Method == "GET" && route.Path == "/api/v1/runtime/services/s3/:bucket" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected list objects route to be registered")
	}
	found = false
	for _, route := range entry.Routes {
		if route.Method == "GET" && route.Path == "/api/v1/runtime/services/s3/:bucket?object-lock" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected object lock route to be registered")
	}
}

func TestRegisterNativeS3RoutesExposesAwsCompatibleSmokeSurface(t *testing.T) {
	t.Helper()

	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)

	if err := registerNativeS3Routes(router, root.Services); err != nil {
		t.Fatalf("register native s3 routes: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list buckets status: got %d want %d", got, want)
	}
	if !strings.Contains(recorder.Body.String(), "ListAllMyBucketsResult") {
		t.Fatalf("expected list buckets XML, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "mildstack-assets") {
		t.Fatalf("expected seeded bucket in list buckets response, got %q", recorder.Body.String())
	}

	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPut, "/native-smoke-bucket", nil)
	router.Engine().ServeHTTP(createRecorder, createRequest)
	if got, want := createRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected create bucket status: got %d want %d", got, want)
	}

	putRecorder := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/native-smoke-bucket/native.txt", strings.NewReader("native-mode smoke payload"))
	putRequest.Header.Set("Content-Type", "text/plain")
	router.Engine().ServeHTTP(putRecorder, putRequest)
	if got, want := putRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected put object status: got %d want %d", got, want)
	}

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/native-smoke-bucket/native.txt", nil)
	router.Engine().ServeHTTP(getRecorder, getRequest)
	if got, want := getRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get object status: got %d want %d", got, want)
	}
	if got, want := getRecorder.Body.String(), "native-mode smoke payload"; got != want {
		t.Fatalf("unexpected object body: got %q want %q", got, want)
	}

	deleteObjectRecorder := httptest.NewRecorder()
	deleteObjectRequest := httptest.NewRequest(http.MethodDelete, "/native-smoke-bucket/native.txt", nil)
	router.Engine().ServeHTTP(deleteObjectRecorder, deleteObjectRequest)
	if got, want := deleteObjectRecorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("unexpected delete object status: got %d want %d", got, want)
	}

	deleteBucketRecorder := httptest.NewRecorder()
	deleteBucketRequest := httptest.NewRequest(http.MethodDelete, "/native-smoke-bucket", nil)
	router.Engine().ServeHTTP(deleteBucketRecorder, deleteBucketRequest)
	if got, want := deleteBucketRecorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("unexpected delete bucket status: got %d want %d", got, want)
	}
}

func TestRegisterNativeDynamoDBRoutesExposesAwsCompatibleSmokeSurface(t *testing.T) {
	t.Helper()

	dynamoService, err := dynamoapp.NewWithPersistence(dynamoapp.StorageConfig{
		BaseDir:    t.TempDir(),
		InstanceID: fmt.Sprintf("test-instance-smoke-%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("new persistent dynamodb service: %v", err)
	}
	t.Cleanup(func() {
		if err := dynamoService.Stop(context.Background()); err != nil {
			t.Fatalf("stop dynamodb service: %v", err)
		}
	})

	services := []orchestrator.Service{dynamoService}
	manager := runtime.New(services)
	router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)

	if err := registerNativeDynamoDBRoutes(router, services); err != nil {
		t.Fatalf("register native dynamodb routes: %v", err)
	}

	server := httptest.NewServer(router.Engine())
	t.Cleanup(server.Close)

	client := newDynamoDBSmokeClient(t, server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	const tableName = "native-smoke-table"
	createOut, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
	})
	if err != nil {
		t.Fatalf("create table via sdk: %v", err)
	}
	if createOut.TableDescription == nil {
		t.Fatal("expected create table response to include table description")
	}
	if got, want := string(createOut.TableDescription.TableStatus), string(types.TableStatusCreating); got != want {
		t.Fatalf("unexpected create status: got %q want %q", got, want)
	}
	if createOut.TableDescription.CreationDateTime == nil || createOut.TableDescription.CreationDateTime.IsZero() {
		t.Fatal("expected creation date time to be populated")
	}

	waitForTableStatus(t, ctx, client, tableName, types.TableStatusActive)

	itemValue := map[string]types.AttributeValue{
		"id":       &types.AttributeValueMemberS{Value: "item#1"},
		"title":    &types.AttributeValueMemberS{Value: "native-mode smoke payload"},
		"version":  &types.AttributeValueMemberN{Value: "1"},
		"obsolete": &types.AttributeValueMemberS{Value: "remove me"},
	}
	if _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      itemValue,
	}); err != nil {
		t.Fatalf("put item via sdk: %v", err)
	}

	updateOut, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "item#1"},
		},
		UpdateExpression: aws.String("SET title = :title ADD version :inc REMOVE obsolete"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":title": &types.AttributeValueMemberS{Value: "native-mode smoke updated"},
			":inc":   &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueAllNew,
	})
	if err != nil {
		t.Fatalf("update item via sdk: %v", err)
	}
	if updateOut.Attributes == nil {
		t.Fatal("expected update item response to include attributes")
	}
	if got, want := attrValueString(t, updateOut.Attributes["title"]), "native-mode smoke updated"; got != want {
		t.Fatalf("unexpected updated item title: got %q want %q", got, want)
	}
	if got, want := attrValueString(t, updateOut.Attributes["version"]), "2"; got != want {
		t.Fatalf("unexpected updated item version: got %q want %q", got, want)
	}
	if _, ok := updateOut.Attributes["obsolete"]; ok {
		t.Fatal("expected obsolete attribute to be removed from update response")
	}

	getOut, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "item#1"},
		},
	})
	if err != nil {
		t.Fatalf("get item via sdk: %v", err)
	}
	if got, want := attrValueString(t, getOut.Item["id"]), "item#1"; got != want {
		t.Fatalf("unexpected item id: got %q want %q", got, want)
	}
	if got, want := attrValueString(t, getOut.Item["title"]), "native-mode smoke updated"; got != want {
		t.Fatalf("unexpected item title: got %q want %q", got, want)
	}
	if got, want := attrValueString(t, getOut.Item["version"]), "2"; got != want {
		t.Fatalf("unexpected item version: got %q want %q", got, want)
	}
	if _, ok := getOut.Item["obsolete"]; ok {
		t.Fatal("expected obsolete attribute to be absent after update")
	}

	const readTableName = "native-smoke-read-table"
	readCreateOut, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(readTableName),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("sk"),
				KeyType:       types.KeyTypeRange,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("sk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
	})
	if err != nil {
		t.Fatalf("create read table via sdk: %v", err)
	}
	if readCreateOut.TableDescription == nil {
		t.Fatal("expected read table create response to include table description")
	}
	waitForTableStatus(t, ctx, client, readTableName, types.TableStatusActive)

	for _, item := range []map[string]types.AttributeValue{
		{
			"id":    &types.AttributeValueMemberS{Value: "series#1"},
			"sk":    &types.AttributeValueMemberS{Value: "001"},
			"title": &types.AttributeValueMemberS{Value: "skip-one"},
		},
		{
			"id":    &types.AttributeValueMemberS{Value: "series#1"},
			"sk":    &types.AttributeValueMemberS{Value: "002"},
			"title": &types.AttributeValueMemberS{Value: "keep-two"},
		},
		{
			"id":    &types.AttributeValueMemberS{Value: "series#1"},
			"sk":    &types.AttributeValueMemberS{Value: "003"},
			"title": &types.AttributeValueMemberS{Value: "keep-three"},
		},
	} {
		if _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(readTableName),
			Item:      item,
		}); err != nil {
			t.Fatalf("seed read table item via sdk: %v", err)
		}
	}

	queryOut, err := client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(readTableName),
		KeyConditionExpression: aws.String("id = :id AND sk BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id":    &types.AttributeValueMemberS{Value: "series#1"},
			":start": &types.AttributeValueMemberS{Value: "001"},
			":end":   &types.AttributeValueMemberS{Value: "003"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(2),
	})
	if err != nil {
		t.Fatalf("query via sdk: %v", err)
	}
	if got, want := len(queryOut.Items), 2; got != want {
		t.Fatalf("unexpected query item count: got %d want %d", got, want)
	}
	if got, want := attrValueString(t, queryOut.Items[0]["sk"]), "003"; got != want {
		t.Fatalf("unexpected first query sort key: got %q want %q", got, want)
	}
	if got, want := attrValueString(t, queryOut.Items[1]["sk"]), "002"; got != want {
		t.Fatalf("unexpected second query sort key: got %q want %q", got, want)
	}
	if got, want := attrValueString(t, queryOut.LastEvaluatedKey["sk"]), "002"; got != want {
		t.Fatalf("unexpected query cursor: got %q want %q", got, want)
	}

	beginsOut, err := client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(readTableName),
		KeyConditionExpression: aws.String("id = :id AND begins_with(sk, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id":     &types.AttributeValueMemberS{Value: "series#1"},
			":prefix": &types.AttributeValueMemberS{Value: "00"},
		},
	})
	if err != nil {
		t.Fatalf("begins_with query via sdk: %v", err)
	}
	if got, want := len(beginsOut.Items), 3; got != want {
		t.Fatalf("unexpected begins_with query item count: got %d want %d", got, want)
	}
	if got, want := attrValueString(t, beginsOut.Items[0]["sk"]), "001"; got != want {
		t.Fatalf("unexpected begins_with first sort key: got %q want %q", got, want)
	}

	scanOut, err := client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(readTableName),
		FilterExpression: aws.String("begins_with(title, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: "keep"},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("scan via sdk: %v", err)
	}
	if got, want := len(scanOut.Items), 0; got != want {
		t.Fatalf("unexpected first scan page item count: got %d want %d", got, want)
	}
	if got, want := attrValueString(t, scanOut.LastEvaluatedKey["sk"]), "001"; got != want {
		t.Fatalf("unexpected first scan cursor: got %q want %q", got, want)
	}

	scanOut, err = client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(readTableName),
		FilterExpression: aws.String("begins_with(title, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: "keep"},
		},
		Limit:             aws.Int32(1),
		ExclusiveStartKey: scanOut.LastEvaluatedKey,
	})
	if err != nil {
		t.Fatalf("scan second page via sdk: %v", err)
	}
	if got, want := len(scanOut.Items), 1; got != want {
		t.Fatalf("unexpected second scan page item count: got %d want %d", got, want)
	}
	if got, want := attrValueString(t, scanOut.Items[0]["title"]), "keep-two"; got != want {
		t.Fatalf("unexpected second scan title: got %q want %q", got, want)
	}

	const batchTableName = "native-smoke-batch-table"
	batchCreateOut, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(batchTableName),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
	})
	if err != nil {
		t.Fatalf("create batch table via sdk: %v", err)
	}
	if batchCreateOut.TableDescription == nil {
		t.Fatal("expected batch table create response to include table description")
	}
	waitForTableStatus(t, ctx, client, batchTableName, types.TableStatusActive)

	batchWriteItems := make(map[string][]types.WriteRequest, 1)
	putRequests := make([]types.WriteRequest, 0, 26)
	for i := 1; i <= 26; i++ {
		id := fmt.Sprintf("item#%02d", i)
		putRequests = append(putRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: map[string]types.AttributeValue{
					"id":    &types.AttributeValueMemberS{Value: id},
					"title": &types.AttributeValueMemberS{Value: fmt.Sprintf("title-%02d", i)},
				},
			},
		})
	}
	batchWriteItems[batchTableName] = putRequests

	batchWriteOut, err := client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: batchWriteItems,
	})
	if err != nil {
		t.Fatalf("batch write via sdk: %v", err)
	}
	if got, want := len(batchWriteOut.UnprocessedItems[batchTableName]), 1; got != want {
		t.Fatalf("unexpected batch write unprocessed count: got %d want %d", got, want)
	}
	if got, want := attrValueString(t, batchWriteOut.UnprocessedItems[batchTableName][0].PutRequest.Item["id"]), "item#26"; got != want {
		t.Fatalf("unexpected unprocessed batch write id: got %q want %q", got, want)
	}

	batchGetOut, err := client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			batchTableName: {
				Keys: []map[string]types.AttributeValue{
					{"id": &types.AttributeValueMemberS{Value: "item#01"}},
					{"id": &types.AttributeValueMemberS{Value: "item#25"}},
					{"id": &types.AttributeValueMemberS{Value: "item#26"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("batch get via sdk: %v", err)
	}
	if got, want := len(batchGetOut.Responses[batchTableName]), 2; got != want {
		t.Fatalf("unexpected batch get response count: got %d want %d", got, want)
	}
	if got, want := attrValueString(t, batchGetOut.Responses[batchTableName][0]["id"]), "item#01"; got != want {
		t.Fatalf("unexpected first batch get id: got %q want %q", got, want)
	}
	if got, want := attrValueString(t, batchGetOut.Responses[batchTableName][1]["id"]), "item#25"; got != want {
		t.Fatalf("unexpected second batch get id: got %q want %q", got, want)
	}

	transactWriteOut, err := client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: aws.String(batchTableName),
					Item: map[string]types.AttributeValue{
						"id":    &types.AttributeValueMemberS{Value: "item#27"},
						"title": &types.AttributeValueMemberS{Value: "title-27"},
					},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(batchTableName),
					Key: map[string]types.AttributeValue{
						"id": &types.AttributeValueMemberS{Value: "item#01"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("transact write via sdk: %v", err)
	}
	if transactWriteOut == nil {
		t.Fatal("expected transact write response")
	}

	transactGetOut, err := client.TransactGetItems(ctx, &dynamodb.TransactGetItemsInput{
		TransactItems: []types.TransactGetItem{
			{
				Get: &types.Get{
					TableName: aws.String(batchTableName),
					Key: map[string]types.AttributeValue{
						"id": &types.AttributeValueMemberS{Value: "item#27"},
					},
				},
			},
			{
				Get: &types.Get{
					TableName: aws.String(batchTableName),
					Key: map[string]types.AttributeValue{
						"id": &types.AttributeValueMemberS{Value: "item#02"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("transact get via sdk: %v", err)
	}
	if got, want := len(transactGetOut.Responses), 2; got != want {
		t.Fatalf("unexpected transact get response count: got %d want %d", got, want)
	}
	if got, want := attrValueString(t, transactGetOut.Responses[0].Item["id"]), "item#27"; got != want {
		t.Fatalf("unexpected first transact get id: got %q want %q", got, want)
	}
	if got, want := attrValueString(t, transactGetOut.Responses[1].Item["id"]), "item#02"; got != want {
		t.Fatalf("unexpected second transact get id: got %q want %q", got, want)
	}

	_, err = client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: aws.String(batchTableName),
					Item: map[string]types.AttributeValue{
						"id":    &types.AttributeValueMemberS{Value: "item#28"},
						"title": &types.AttributeValueMemberS{Value: "title-28"},
					},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(batchTableName),
					Key: map[string]types.AttributeValue{
						"id": &types.AttributeValueMemberS{Value: "item#28"},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected conflicting transaction to fail")
	}
	var canceled *types.TransactionCanceledException
	if !errors.As(err, &canceled) {
		t.Fatalf("expected transaction canceled error, got %T: %v", err, err)
	}

	paginator := dynamodb.NewListTablesPaginator(client, &dynamodb.ListTablesInput{Limit: aws.Int32(1)})
	var listed []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			t.Fatalf("list tables page via sdk: %v", err)
		}
		listed = append(listed, page.TableNames...)
	}
	if got, want := listed, []string{batchTableName, readTableName, tableName}; !equalStringSlices(got, want) {
		t.Fatalf("unexpected paginated table list: got %v want %v", got, want)
	}

	deleteOut, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		t.Fatalf("delete table via sdk: %v", err)
	}
	if deleteOut.TableDescription == nil {
		t.Fatal("expected delete table response to include table description")
	}
	if got, want := string(deleteOut.TableDescription.TableStatus), string(types.TableStatusDeleting); got != want {
		t.Fatalf("unexpected delete status: got %q want %q", got, want)
	}

	if _, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tableName)}); err == nil {
		t.Fatal("expected describe on deleted table to fail")
	} else {
		var notFound *types.ResourceNotFoundException
		if !errors.As(err, &notFound) {
			t.Fatalf("expected resource not found after delete, got %T: %v", err, err)
		}
	}
}

func TestInstanceRegistrarPersistsAndReleasesActiveInstance(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := cli.NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
	manager := runtime.NewWithPorts(nil, nil)
	registrar := instanceRegistrar{manager: manager, storage: storage}

	if err := registrar.Serve(context.Background(), 9090); err != nil {
		t.Fatalf("serve: %v", err)
	}

	ports, err := storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports: %v", err)
	}
	if len(ports) != 1 || ports[0] != 9090 {
		t.Fatalf("unexpected active ports: %#v", ports)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %#v", instances)
	}

	savedPath := filepath.Join(paths.InstancesDir, "saved", instances[0].InstanceID+".json")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved instance file: %v", err)
	}

	if err := registrar.Release(context.Background(), 9090); err != nil {
		t.Fatalf("release: %v", err)
	}

	if ports := manager.Ports(context.Background()); len(ports) != 0 {
		t.Fatalf("expected manager ports to be cleared after release, got %#v", ports)
	}

	ports, err = storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports after release: %v", err)
	}
	if len(ports) != 0 {
		t.Fatalf("expected no active ports after release, got %#v", ports)
	}

	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("saved instance should remain available: %v", err)
	}
}

func TestInstanceRegistrarServeSkipsDuplicateLoadedPort(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := cli.NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
	manager := runtime.NewWithPorts(nil, []int{9090})
	registrar := instanceRegistrar{manager: manager, storage: storage}

	if err := registrar.Serve(context.Background(), 9090); err != nil {
		t.Fatalf("serve: %v", err)
	}

	ports, err := storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports: %v", err)
	}
	if len(ports) != 1 || ports[0] != 9090 {
		t.Fatalf("unexpected active ports: %#v", ports)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %#v", instances)
	}

	savedPath := filepath.Join(paths.InstancesDir, "saved", instances[0].InstanceID+".json")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved instance file: %v", err)
	}
}

func TestRegisterNativeSQSRoutesExposesAwsCompatibleSmokeSurface(t *testing.T) {
	t.Helper()

	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)

	if err := registerNativeSQSRoutes(router, root.Services); err != nil {
		t.Fatalf("register native sqs routes: %v", err)
	}

	healthRecorder := httptest.NewRecorder()
	healthRequest := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/health", nil)
	router.Engine().ServeHTTP(healthRecorder, healthRequest)
	if got, want := healthRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected health status: got %d want %d", got, want)
	}

	rootRecorder := httptest.NewRecorder()
	rootRequest := httptest.NewRequest(http.MethodGet, "/?Action=ListQueues&Version=2012-11-05", nil)
	router.Engine().ServeHTTP(rootRecorder, rootRequest)
	if got, want := rootRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected sqs root status: got %d want %d", got, want)
	}
	if !strings.Contains(rootRecorder.Body.String(), "ListQueuesResponse") {
		t.Fatalf("expected sqs list queues xml response, got %q", rootRecorder.Body.String())
	}

	server := httptest.NewServer(router.Engine())
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "test")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := sqssdk.NewFromConfig(cfg, func(o *sqssdk.Options) {
		o.BaseEndpoint = aws.String(server.URL)
	})

	result, err := client.ListQueues(ctx, &sqssdk.ListQueuesInput{})
	if err != nil {
		t.Fatalf("expected list queues to succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil list queues result")
	}
}


func newDynamoDBSmokeClient(t *testing.T, endpoint string) *dynamodb.Client {
	t.Helper()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "test")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     "us-east-1",
				HostnameImmutable: true,
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	return dynamodb.NewFromConfig(cfg)
}

func waitForTableStatus(t *testing.T, ctx context.Context, client *dynamodb.Client, tableName string, want types.TableStatus) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		out, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			var notFound *types.ResourceNotFoundException
			if errors.As(err, &notFound) {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			t.Fatalf("describe table while waiting for %s: %v", want, err)
		}
		if out.Table != nil && out.Table.TableStatus == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("table %q did not reach status %s", tableName, want)
}

func attrValueString(t *testing.T, value types.AttributeValue) string {
	t.Helper()

	switch v := value.(type) {
	case *types.AttributeValueMemberS:
		return v.Value
	case *types.AttributeValueMemberN:
		return v.Value
	default:
		t.Fatalf("unexpected attribute value type %T", value)
		return ""
	}
}

func equalStringSlices(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestNewInstanceIDUsesRandomNonLegacyValue(t *testing.T) {
	first, err := cli.NewInstanceID()
	if err != nil {
		t.Fatalf("new instance id: %v", err)
	}
	second, err := cli.NewInstanceID()
	if err != nil {
		t.Fatalf("new instance id second call: %v", err)
	}
	if first == "" || second == "" {
		t.Fatal("expected non-empty instance ids")
	}
	if first == second {
		t.Fatalf("expected random ids, got duplicated value %q", first)
	}
	if strings.HasPrefix(first, "mildstack-") || strings.HasPrefix(second, "mildstack-") {
		t.Fatalf("expected non-legacy ids, got %q and %q", first, second)
	}
}
