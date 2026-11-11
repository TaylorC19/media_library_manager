package providerkeys

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoStore encrypts each personal override independently. Associated data
// binds ciphertext to both its owner and provider. The encryption key lives in
// deployment secrets, separately from MongoDB and database backups.
func NewMongoStore(db *mongo.Database, encodedKey string) (*Store, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("invalid provider encryption key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Store{coll: db.Collection("personal_provider_keys"), aead: aead}, nil
}
func (s *Store) encrypt(id primitive.ObjectID, provider, value string) string {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic("secure random source unavailable")
	}
	sealed := s.aead.Seal(nonce, nonce, []byte(value), []byte(id.Hex()+":"+provider))
	return base64.StdEncoding.EncodeToString(sealed)
}
func (s *Store) decrypt(id primitive.ObjectID, provider, value string) (string, error) {
	sealed, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(sealed) < s.aead.NonceSize() {
		return "", errors.New("could not read provider keys")
	}
	n := s.aead.NonceSize()
	plain, err := s.aead.Open(nil, sealed[:n], sealed[n:], []byte(id.Hex()+":"+provider))
	if err != nil {
		return "", errors.New("could not read provider keys")
	}
	return string(plain), nil
}
func (s *Store) mongoGet(id primitive.ObjectID) (Keys, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc struct {
		Values map[string]string `bson:"values"`
	}
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Keys{}, nil
	}
	if err != nil {
		return nil, errors.New("could not read provider keys")
	}
	keys := Keys{}
	for _, p := range Providers {
		if v := doc.Values[p]; v != "" {
			keys[p], err = s.decrypt(id, p, v)
			if err != nil {
				return nil, err
			}
		}
	}
	return keys, nil
}
func (s *Store) mongoUpdate(id primitive.ObjectID, changes Keys) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	set, unset := bson.M{}, bson.M{}
	for _, p := range Providers {
		if v, ok := changes[p]; ok {
			if v == "" {
				unset["values."+p] = ""
			} else {
				set["values."+p] = s.encrypt(id, p, v)
			}
		}
	}
	update := bson.M{}
	if len(set) > 0 {
		update["$set"] = set
	}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if len(update) == 0 {
		return nil
	}
	_, err := s.coll.UpdateByID(ctx, id, update, options.Update().SetUpsert(true))
	// Simultaneous first writes can race the unique _id insert; retry as update.
	if mongo.IsDuplicateKeyError(err) {
		_, err = s.coll.UpdateByID(ctx, id, update)
	}
	if err != nil {
		return errors.New("could not save provider keys")
	}
	return nil
}
