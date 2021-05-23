package blog

import (
	"reflect"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/errors"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/app"

	"github.com/Aakanksha-jais/picshot-golang-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type blog struct {
}

func New() blog {
	return blog{}
}

// GetAll is used to retrieve all blogs that match the filter.
// BLogs can be filtered by account_id, blog_id and title.
func (b blog) GetAll(c *app.Context, filter *models.Blog) ([]*models.Blog, error) {
	collection := c.Mongo.DB().Collection("blogs")

	opts := options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}) // retrieve the blogs in reverse chronological order

	cursor, err := collection.Find(c, filter.GetFilter(), opts)
	if err != nil {
		return nil, errors.DBError{Err: err}
	}

	blogs := make([]*models.Blog, 0)

	for cursor.Next(c) {
		var blog models.Blog

		err := cursor.Decode(&blog)
		if err != nil {
			return nil, errors.DBError{Err: err}
		}

		blogs = append(blogs, &blog)
	}

	err = cursor.Close(c)
	if err != nil {
		return nil, errors.DBError{Err: err}
	}

	return blogs, nil
}

// GetByIDs retrieves all blogs whose IDs have been provided as parameter.
func (b blog) GetByIDs(c *app.Context, idList []string) ([]*models.Blog, error) {
	collection := c.Mongo.DB().Collection("blogs")

	cursor, err := collection.Find(c, bson.M{"_id": bson.M{"$in": idList}})
	if err != nil {
		return nil, errors.DBError{Err: err}
	}

	blogs := make([]*models.Blog, 0)

	for cursor.Next(c) {
		var blog models.Blog

		err := cursor.Decode(&blog)
		if err != nil {
			return nil, errors.DBError{Err: err}
		}

		blogs = append(blogs, &blog)
	}

	err = cursor.Close(c)
	if err != nil {
		return nil, errors.DBError{Err: err}
	}

	return blogs, nil
}

// Get is used to retrieve a SINGLE blog that matches the filter.
// A blog can be filtered by account_id, blog_id and title.
func (b blog) Get(c *app.Context, filter *models.Blog) (*models.Blog, error) {
	if filter == nil {
		return nil, nil // todo
	}

	var blog models.Blog

	collection := c.Mongo.DB().Collection("blogs")

	res := collection.FindOne(c, filter.GetFilter())

	err := res.Err()
	switch err {
	case mongo.ErrNoDocuments:
		return nil, errors.DBError{Err: err} //todo change in service layer
	case nil:
		err = res.Decode(&blog)
		if err != nil {
			return nil, errors.DBError{Err: err}
		}

		return &blog, nil
	default:
		return nil, errors.DBError{Err: err}
	}
}

// Create is used to create a new blog.
func (b blog) Create(c *app.Context, model *models.Blog) (*models.Blog, error) {
	if model == nil {
		return nil, nil //todo
	}

	collection := c.Mongo.DB().Collection("blogs")

	res, err := collection.InsertOne(c, model) // nil is returned if InsertOne operation is successful
	if err != nil {
		return nil, errors.DBError{Err: err}
	}

	id := res.InsertedID

	return b.Get(c, &models.Blog{BlogID: id.(string)})
}

// Update updates the blog by its ID.
func (b blog) Update(c *app.Context, model *models.Blog) (*models.Blog, error) {
	if model == nil {
		return nil, nil //todo
	}

	collection := c.Mongo.DB().Collection("blogs")

	res := collection.FindOneAndUpdate(c, bson.M{"_id": model.BlogID}, generateFilter(*model))

	if err := res.Err(); err != nil {
		return nil, errors.DBError{Err: err}
	}

	if !reflect.DeepEqual(model.Images, []string(nil)) {
		r := collection.FindOneAndUpdate(c, bson.M{"_id": model.BlogID}, bson.M{"$push": bson.M{"images": bson.M{"$each": model.Images}}})
		if err := r.Err(); err != nil {
			return nil, errors.DBError{Err: err}
		}
	}

	if !reflect.DeepEqual(model.Tags, []string(nil)) {
		r := collection.FindOneAndUpdate(c, bson.M{"_id": model.BlogID}, bson.M{"$push": bson.M{"tags": bson.M{"$each": model.Tags}}})
		if err := r.Err(); err != nil {
			return nil, errors.DBError{Err: err}
		}
	}

	return b.Get(c, &models.Blog{BlogID: model.BlogID})
}

func generateFilter(model models.Blog) bson.M {
	update := bson.M{}
	if model.Title != "" {
		update["title"] = model.Title
	}

	if model.Summary != "" {
		update["summary"] = model.Summary
	}

	if model.Content != "" {
		update["content"] = model.Content
	}

	return bson.M{"$set": update}
}

// Delete deletes a blog by its ID.
func (b blog) Delete(c *app.Context, blogID string) error {
	collection := c.Mongo.DB().Collection("blogs")

	res := collection.FindOneAndDelete(c, bson.D{bson.E{Key: "_id", Value: blogID}})

	if err := res.Err(); err != nil {
		return errors.DBError{Err: err}
	}

	return nil
}
