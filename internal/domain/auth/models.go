package auth

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Username     string             `bson:"username"`
	PasswordHash string             `bson:"passwordHash"`
	// Pointer so BSON null decodes; omitempty keeps inserts clean.
	DisplayName *string `bson:"displayName,omitempty"`
	CreatedAt   time.Time `bson:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt"`
}

// ShowName returns a non-empty label for UI (falls back to username).
func (u *User) ShowName() string {
	if u == nil {
		return ""
	}
	if u.DisplayName != nil {
		if s := strings.TrimSpace(*u.DisplayName); s != "" {
			return s
		}
	}
	return u.Username
}

type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"userId"`
	TokenHash string             `bson:"tokenHash"`
	ExpiresAt time.Time          `bson:"expiresAt"`
	CreatedAt time.Time          `bson:"createdAt"`
	LastUsed  time.Time          `bson:"lastUsedAt"`
	UserAgent string             `bson:"userAgent,omitempty"`
	IPAddress string             `bson:"ipAddress,omitempty"`
}
