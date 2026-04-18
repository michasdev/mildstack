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
		"id":    &types.AttributeValueMemberS{Value: "item#1"},
		"title": &types.AttributeValueMemberS{Value: "native-mode smoke payload"},
	}
	if _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      itemValue,
	}); err != nil {
		t.Fatalf("put item via sdk: %v", err)
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
	if got, want := attrValueString(t, getOut.Item["title"]), "native-mode smoke payload"; got != want {
		t.Fatalf("unexpected item title: got %q want %q", got, want)
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
	if got, want := listed, []string{"mildstack-records", tableName}; !equalStringSlices(got, want) {
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

	savedPath := filepath.Join(paths.InstancesDir, "saved", "9090.json")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved instance file: %v", err)
	}

	if err := registrar.Release(context.Background(), 9090); err != nil {
		t.Fatalf("release: %v", err)
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

	savedPath := filepath.Join(paths.InstancesDir, "saved", "9090.json")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved instance file: %v", err)
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
