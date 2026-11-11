package dbsetup

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"media_library_manager/internal/repository"
	authsvc "media_library_manager/internal/service/auth"
	"time"
)

// RuntimeUser provisions a reserved CRUD-only role for the bundled database.
// Existing user passwords are never changed during restart or upgrade.
func RuntimeUser(ctx context.Context, database *mongo.Database, username, password string) error {
	if username == "" || password == "" {
		return errors.New("runtime database credentials are required")
	}
	privileges := bson.A{}
	for _, name := range Collections {
		privileges = append(privileges, bson.M{"resource": bson.M{"db": database.Name(), "collection": name}, "actions": bson.A{"find", "insert", "update", "remove"}})
	}
	role := bson.D{{Key: "createRole", Value: "mlmRuntime"}, {Key: "privileges", Value: privileges}, {Key: "roles", Value: bson.A{}}}
	if err := database.RunCommand(ctx, role).Err(); err != nil {
		var ce mongo.CommandError
		if !errors.As(err, &ce) || ce.Code != 51002 {
			return errors.New("could not create runtime database role")
		}
		role[0] = bson.E{Key: "updateRole", Value: "mlmRuntime"}
		if err := database.RunCommand(ctx, role).Err(); err != nil {
			return errors.New("could not update runtime database role")
		}
	}
	var result struct {
		Users []bson.M `bson:"users"`
	}
	if err := database.RunCommand(ctx, bson.D{{Key: "usersInfo", Value: username}}).Decode(&result); err != nil {
		return errors.New("could not inspect runtime database user")
	}
	if len(result.Users) > 0 {
		return nil
	}
	if err := database.RunCommand(ctx, bson.D{{Key: "createUser", Value: username}, {Key: "pwd", Value: password}, {Key: "roles", Value: bson.A{bson.M{"role": "mlmRuntime", "db": database.Name()}}}}).Err(); err != nil {
		return errors.New("could not create runtime database user")
	}
	return nil
}

// BootstrapAdmin never resets passwords, promotes an existing user, or creates a
// second administrator. A transaction guard also serializes concurrent installers.
func BootstrapAdmin(ctx context.Context, database *mongo.Database, username, password string) (bool, error) {
	session, err := database.Client().StartSession()
	if err != nil {
		return false, err
	}
	defer session.EndSession(ctx)
	created := false
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		created = false
		guard, err := database.Collection("admin_guard").UpdateByID(sc, "roles", bson.M{"$inc": bson.M{"revision": 1}})
		if err != nil {
			return nil, err
		}
		if guard.MatchedCount != 1 {
			return nil, errors.New("run database initialization before administrator bootstrap")
		}
		count, err := database.Collection("users").CountDocuments(sc, bson.M{"role": "admin"})
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, nil
		}
		service := authsvc.NewService(repository.NewUsersRepository(database), repository.NewSessionsRepository(database), time.Hour)
		if _, err := service.CreateAccount(sc, username, password, "", "admin"); err != nil {
			return nil, err
		}
		created = true
		return nil, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return created, err
}
