package library

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MediaRecord struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	Source            string             `bson:"source"`
	MediaType         string             `bson:"mediaType"`
	Title             string             `bson:"title"`
	SortTitle         *string            `bson:"sortTitle,omitempty"`
	ReleaseDate       *string            `bson:"releaseDate,omitempty"`
	Year              *int32             `bson:"year,omitempty"`
	ImageURL          *string            `bson:"imageUrl,omitempty"`
	Summary           *string            `bson:"summary,omitempty"`
	ProviderRefs      bson.M             `bson:"providerRefs,omitempty"`
	ExternalRatings   bson.M             `bson:"externalRatings,omitempty"`
	BarcodeCandidates []string           `bson:"barcodeCandidates,omitempty"`
	Details           bson.M             `bson:"details"`
	LastSyncedAt      *time.Time         `bson:"lastSyncedAt,omitempty"`
	CreatedAt         time.Time          `bson:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt"`
}

type LibraryEntry struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        primitive.ObjectID `bson:"userId"`
	MediaRecordID primitive.ObjectID `bson:"mediaRecordId"`
	Bucket        string             `bson:"bucket"`
	MediaType     string             `bson:"mediaType"`
	Format        *string            `bson:"format,omitempty"`
	Barcode       *string            `bson:"barcode,omitempty"`
	PurchaseDate  *string            `bson:"purchaseDate,omitempty"`
	Notes         *string            `bson:"notes,omitempty"`
	Tags          []string           `bson:"tags,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
}
