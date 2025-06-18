package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadCollection(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	// Test collection hash provided by user
	collectionHash := "d48af6bdd4ce12328583611debf2f75e0895a2c5522d23db70c912a6a34538a9"

	collection, err := ReadCollection(ctx, collectionHash)
	if err != nil {
		t.Fatalf("ReadCollection failed: %v", err)
	}

	assert.NotNil(collection, "Collection should not be nil")
	assert.Equal(collectionHash, collection.CollectionHash, "Collection hash should match")
	assert.NotEmpty(collection.Name, "Collection name should not be empty")

	t.Logf("Collection Name: %s", collection.Name)
	t.Logf("Collection Symbol: %s", collection.Symbol)
	t.Logf("Collection Supply: %s", collection.Supply)
}

func TestReadCollectionItems(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	// Test collection hash provided by user
	collectionHash := "d48af6bdd4ce12328583611debf2f75e0895a2c5522d23db70c912a6a34538a9"

	items, err := ReadCollectionItems(ctx, collectionHash)
	if err != nil {
		t.Fatalf("ReadCollectionItems failed: %v", err)
	}

	assert.NotNil(items, "Items should not be nil")
	assert.Greater(len(items), 0, "Should have at least one item")

	t.Logf("Found %d items in collection", len(items))

	// Verify first item structure
	if len(items) > 0 {
		firstItem := items[0]
		assert.NotEmpty(firstItem.InscriptionHash, "Inscription hash should not be empty")
		assert.Equal(collectionHash, firstItem.CollectionHash, "Collection hash should match")

		t.Logf("First item inscription hash: %s", firstItem.InscriptionHash)
		t.Logf("First item content type: %s", firstItem.ContentType)
	}
}

func TestReadInscription(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	// First get items from the collection to get a valid inscription hash
	collectionHash := "d48af6bdd4ce12328583611debf2f75e0895a2c5522d23db70c912a6a34538a9"
	items, err := ReadCollectionItems(ctx, collectionHash)
	if err != nil {
		t.Fatalf("ReadCollectionItems failed: %v", err)
	}

	if len(items) == 0 {
		t.Skip("No items in collection to test with")
	}

	// Test with the first item's inscription hash
	inscriptionHash := items[0].InscriptionHash

	inscription, err := ReadInscription(ctx, inscriptionHash)
	if err != nil {
		t.Fatalf("ReadInscription failed: %v", err)
	}

	assert.NotNil(inscription, "Inscription should not be nil")
	assert.Equal(inscriptionHash, inscription.InscriptionHash, "Inscription hash should match")
	assert.Equal(collectionHash, inscription.CollectionHash, "Collection hash should match")

	t.Logf("Inscription hash: %s", inscription.InscriptionHash)
	t.Logf("Content type: %s", inscription.ContentType)
	t.Logf("Owner: %s", inscription.Owner)
}
