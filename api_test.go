package milvuslite

import (
	"context"
	"math/rand"
	"testing"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	testDim     = 128
	testCollDense  = "test_dense"
	testCollSparse = "test_sparse"
	testCollMixed  = "test_mixed"
)

// testEnv holds a running milvus-lite server and SDK client for tests.
type testEnv struct {
	server *Server
	client client.Client
	t      *testing.T
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dbFile := t.TempDir() + "/api_test.db"
	srv, err := Start(dbFile)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ctx := context.Background()
	c, err := client.NewClient(ctx, client.Config{Address: srv.Addr()})
	if err != nil {
		srv.Stop()
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() {
		c.Close()
		srv.Stop()
	})

	return &testEnv{server: srv, client: c, t: t}
}

func randomFloatVectors(n, dim int) [][]float32 {
	vecs := make([][]float32, n)
	for i := range vecs {
		v := make([]float32, dim)
		for j := range v {
			v[j] = rand.Float32()
		}
		vecs[i] = v
	}
	return vecs
}

// ===========================================
// Collection management tests
// ===========================================

func TestCollectionCreateAndDrop(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	schema := entity.NewSchema().WithName(testCollDense).
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))

	// CreateCollection
	err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	// HasCollection
	has, err := env.client.HasCollection(ctx, testCollDense)
	if err != nil {
		t.Fatalf("HasCollection: %v", err)
	}
	if !has {
		t.Error("HasCollection: expected true")
	}

	// DescribeCollection
	coll, err := env.client.DescribeCollection(ctx, testCollDense)
	if err != nil {
		t.Fatalf("DescribeCollection: %v", err)
	}
	if coll.Name != testCollDense {
		t.Errorf("collection name = %q, want %q", coll.Name, testCollDense)
	}
	if len(coll.Schema.Fields) != 2 {
		t.Errorf("field count = %d, want 2", len(coll.Schema.Fields))
	}

	// ListCollections (use milvuslite.ListCollections to work around milvus-lite bug)
	colls, err := ListCollections(ctx, env.server.Addr())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	found := false
	for _, name := range colls {
		if name == testCollDense {
			found = true
		}
	}
	if !found {
		t.Error("ListCollections: test collection not found")
	}

	// DropCollection
	err = env.client.DropCollection(ctx, testCollDense)
	if err != nil {
		t.Fatalf("DropCollection: %v", err)
	}

	has, _ = env.client.HasCollection(ctx, testCollDense)
	if has {
		t.Error("HasCollection: expected false after drop")
	}
}

// ===========================================
// Index management tests
// ===========================================

func TestIndexCreateDescribeDrop(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	schema := entity.NewSchema().WithName("test_index").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))

	if err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	// CreateIndex (FLAT)
	idx, _ := entity.NewIndexFlat(entity.L2)
	if err := env.client.CreateIndex(ctx, "test_index", "vector", idx, false); err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}

	// DescribeIndex
	indexes, err := env.client.DescribeIndex(ctx, "test_index", "vector")
	if err != nil {
		t.Fatalf("DescribeIndex: %v", err)
	}
	if len(indexes) == 0 {
		t.Fatal("DescribeIndex: no indexes returned")
	}
	t.Logf("index type: %v", indexes[0].IndexType())

	// DropIndex
	if err := env.client.DropIndex(ctx, "test_index", "vector"); err != nil {
		t.Fatalf("DropIndex: %v", err)
	}
}

// ===========================================
// Insert, load, search, query, delete
// ===========================================

func TestInsertSearchQueryDelete(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	schema := entity.NewSchema().WithName("test_crud").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true)).
		WithField(entity.NewField().WithName("category").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64)).
		WithField(entity.NewField().WithName("score").WithDataType(entity.FieldTypeFloat)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))

	if err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	// Insert
	n := 100
	ids := make([]int64, n)
	categories := make([]string, n)
	scores := make([]float32, n)
	vectors := randomFloatVectors(n, testDim)

	for i := 0; i < n; i++ {
		ids[i] = int64(i)
		if i%2 == 0 {
			categories[i] = "even"
		} else {
			categories[i] = "odd"
		}
		scores[i] = float32(i) * 0.1
	}

	_, err := env.client.Insert(ctx, "test_crud", "",
		entity.NewColumnInt64("id", ids),
		entity.NewColumnVarChar("category", categories),
		entity.NewColumnFloat("score", scores),
		entity.NewColumnFloatVector("vector", testDim, vectors),
	)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// CreateIndex + LoadCollection
	idx, _ := entity.NewIndexFlat(entity.L2)
	if err := env.client.CreateIndex(ctx, "test_crud", "vector", idx, false); err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	if err := env.client.LoadCollection(ctx, "test_crud", false); err != nil {
		t.Fatalf("LoadCollection: %v", err)
	}

	// GetCollectionStatistics
	stats, err := env.client.GetCollectionStatistics(ctx, "test_crud")
	if err != nil {
		t.Fatalf("GetCollectionStatistics: %v", err)
	}
	t.Logf("collection stats: %v", stats)

	// Search
	searchVec := []entity.Vector{entity.FloatVector(vectors[0])}
	sp, _ := entity.NewIndexFlatSearchParam()
	results, err := env.client.Search(ctx, "test_crud", nil, "", []string{"category", "score"}, searchVec, "vector", entity.L2, 10, sp)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 || results[0].ResultCount == 0 {
		t.Fatal("Search: no results")
	}
	t.Logf("search returned %d results", results[0].ResultCount)

	// Search with filter
	results, err = env.client.Search(ctx, "test_crud", nil, "category == \"even\"", []string{"category"}, searchVec, "vector", entity.L2, 10, sp)
	if err != nil {
		t.Fatalf("Search with filter: %v", err)
	}
	if results[0].ResultCount == 0 {
		t.Fatal("Search with filter: no results")
	}
	// Verify all results are "even"
	catCol, ok := results[0].Fields.GetColumn("category").(*entity.ColumnVarChar)
	if !ok {
		t.Fatal("category column type mismatch")
	}
	for i := 0; i < results[0].ResultCount; i++ {
		val, _ := catCol.ValueByIdx(i)
		if val != "even" {
			t.Errorf("expected category=even, got %q", val)
		}
	}

	// Query
	rs, err := env.client.Query(ctx, "test_crud", nil, "id < 5", []string{"id", "category", "score"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	idCol, ok := rs.GetColumn("id").(*entity.ColumnInt64)
	if !ok {
		t.Fatal("id column type mismatch")
	}
	if idCol.Len() == 0 {
		t.Fatal("Query: no results")
	}
	t.Logf("query returned %d rows", idCol.Len())

	// Delete
	err = env.client.Delete(ctx, "test_crud", "", "id < 10")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify deletion via query
	rs, err = env.client.Query(ctx, "test_crud", nil, "id < 10", []string{"id"})
	if err != nil {
		t.Fatalf("Query after delete: %v", err)
	}
	idCol2, ok := rs.GetColumn("id").(*entity.ColumnInt64)
	if ok && idCol2.Len() > 0 {
		t.Errorf("expected 0 rows after delete, got %d", idCol2.Len())
	}
}

// ===========================================
// Upsert test
// ===========================================

func TestUpsert(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	schema := entity.NewSchema().WithName("test_upsert").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true)).
		WithField(entity.NewField().WithName("label").WithDataType(entity.FieldTypeVarChar).WithMaxLength(32)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))

	if err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	vecs := randomFloatVectors(10, testDim)

	// Insert initial data
	_, err := env.client.Insert(ctx, "test_upsert", "",
		entity.NewColumnInt64("id", []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		entity.NewColumnVarChar("label", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}),
		entity.NewColumnFloatVector("vector", testDim, vecs),
	)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Upsert: update existing (id=1,2) + insert new (id=11)
	upsertVecs := randomFloatVectors(3, testDim)
	_, err = env.client.Upsert(ctx, "test_upsert", "",
		entity.NewColumnInt64("id", []int64{1, 2, 11}),
		entity.NewColumnVarChar("label", []string{"updated_a", "updated_b", "new_k"}),
		entity.NewColumnFloatVector("vector", testDim, upsertVecs),
	)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Verify
	idx, _ := entity.NewIndexFlat(entity.L2)
	env.client.CreateIndex(ctx, "test_upsert", "vector", idx, false)
	env.client.LoadCollection(ctx, "test_upsert", false)

	rs, err := env.client.Query(ctx, "test_upsert", nil, "id in [1, 2, 11]", []string{"id", "label"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	labelCol, ok := rs.GetColumn("label").(*entity.ColumnVarChar)
	if !ok {
		t.Fatal("label column type mismatch")
	}
	t.Logf("upsert query returned %d rows", labelCol.Len())
	for i := 0; i < labelCol.Len(); i++ {
		val, _ := labelCol.ValueByIdx(i)
		t.Logf("  label[%d] = %q", i, val)
	}
}

// ===========================================
// Load / Release / GetLoadState
// ===========================================

func TestLoadReleaseState(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	schema := entity.NewSchema().WithName("test_load").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))

	if err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	idx, _ := entity.NewIndexFlat(entity.L2)
	env.client.CreateIndex(ctx, "test_load", "vector", idx, false)

	// Load
	if err := env.client.LoadCollection(ctx, "test_load", false); err != nil {
		t.Fatalf("LoadCollection: %v", err)
	}

	// GetLoadState
	state, err := env.client.GetLoadState(ctx, "test_load", nil)
	if err != nil {
		t.Fatalf("GetLoadState: %v", err)
	}
	if state != entity.LoadStateLoaded {
		t.Errorf("load state = %v, want Loaded", state)
	}

	// Release
	if err := env.client.ReleaseCollection(ctx, "test_load"); err != nil {
		t.Fatalf("ReleaseCollection: %v", err)
	}
}

// ===========================================
// Multiple field types
// ===========================================

func TestMultipleFieldTypes(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	schema := entity.NewSchema().WithName("test_types").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true)).
		WithField(entity.NewField().WithName("flag").WithDataType(entity.FieldTypeBool)).
		WithField(entity.NewField().WithName("age").WithDataType(entity.FieldTypeInt32)).
		WithField(entity.NewField().WithName("score").WithDataType(entity.FieldTypeDouble)).
		WithField(entity.NewField().WithName("name").WithDataType(entity.FieldTypeVarChar).WithMaxLength(128)).
		WithField(entity.NewField().WithName("meta").WithDataType(entity.FieldTypeJSON)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))

	if err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	n := 20
	ids := make([]int64, n)
	flags := make([]bool, n)
	ages := make([]int32, n)
	scoreVals := make([]float64, n)
	names := make([]string, n)
	metas := make([][]byte, n)
	vecs := randomFloatVectors(n, testDim)

	for i := 0; i < n; i++ {
		ids[i] = int64(i)
		flags[i] = i%2 == 0
		ages[i] = int32(20 + i)
		scoreVals[i] = float64(i) * 1.1
		names[i] = "user_" + string(rune('A'+i%26))
		metas[i] = []byte(`{"key":"value"}`)
	}

	_, err := env.client.Insert(ctx, "test_types", "",
		entity.NewColumnInt64("id", ids),
		entity.NewColumnBool("flag", flags),
		entity.NewColumnInt32("age", ages),
		entity.NewColumnDouble("score", scoreVals),
		entity.NewColumnVarChar("name", names),
		entity.NewColumnJSONBytes("meta", metas),
		entity.NewColumnFloatVector("vector", testDim, vecs),
	)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	idx, _ := entity.NewIndexFlat(entity.L2)
	env.client.CreateIndex(ctx, "test_types", "vector", idx, false)
	env.client.LoadCollection(ctx, "test_types", false)

	// Query with various field types
	rs, err := env.client.Query(ctx, "test_types", nil, "age > 30", []string{"id", "flag", "age", "score", "name", "meta"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	idCol := rs.GetColumn("id")
	if idCol == nil || idCol.Len() == 0 {
		t.Fatal("Query: no results")
	}
	t.Logf("query with age>30 returned %d rows", idCol.Len())
}

// ===========================================
// IVF_FLAT index test
// ===========================================

func TestIVFFlatIndex(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	schema := entity.NewSchema().WithName("test_ivf").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))

	if err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	// IVF_FLAT index
	idx, _ := entity.NewIndexIvfFlat(entity.L2, 16)
	if err := env.client.CreateIndex(ctx, "test_ivf", "vector", idx, false); err != nil {
		t.Fatalf("CreateIndex IVF_FLAT: %v", err)
	}

	indexes, err := env.client.DescribeIndex(ctx, "test_ivf", "vector")
	if err != nil {
		t.Fatalf("DescribeIndex: %v", err)
	}
	t.Logf("index type: %v", indexes[0].IndexType())
}

// ===========================================
// Multiple collections
// ===========================================

func TestMultipleCollections(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	names := []string{"coll_a", "coll_b", "coll_c"}
	for _, name := range names {
		schema := entity.NewSchema().WithName(name).
			WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
			WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(testDim))
		if err := env.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
			t.Fatalf("CreateCollection %s: %v", name, err)
		}
	}

	collNames, err := ListCollections(ctx, env.server.Addr())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}

	found := map[string]bool{}
	for _, n := range collNames {
		found[n] = true
	}
	for _, name := range names {
		if !found[name] {
			t.Errorf("collection %s not found in list", name)
		}
	}

	// Drop all
	for _, name := range names {
		if err := env.client.DropCollection(ctx, name); err != nil {
			t.Fatalf("DropCollection %s: %v", name, err)
		}
	}

	collNames, err = ListCollections(ctx, env.server.Addr())
	if err != nil {
		t.Fatalf("ListCollections after drop: %v", err)
	}
	for _, n := range collNames {
		for _, name := range names {
			if n == name {
				t.Errorf("collection %s still exists after drop", name)
			}
		}
	}
}
