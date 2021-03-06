package blog

import (
	"strconv"
	"strings"

	"github.com/Aakanksha-jais/picshot-golang-backend/handlers"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/errors"

	"github.com/Aakanksha-jais/picshot-golang-backend/models"
	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/app"
	"github.com/Aakanksha-jais/picshot-golang-backend/services"
)

type blog struct {
	service services.Blog
}

func (b blog) Browse(ctx *app.Context) (interface{}, error) {
	panic("implement me")
}

func (b blog) Delete(ctx *app.Context) (interface{}, error) {
	id := ctx.Request.PathParam("blogid")

	return nil, b.service.Delete(ctx, id)
}

func New(service services.Blog) handlers.Blog {
	return blog{
		service: service,
	}
}

func (b blog) GetAll(ctx *app.Context) (interface{}, error) {
	limit, err := strconv.Atoi(ctx.Request.QueryParam("limit"))
	if err != nil || limit < 0 {
		return nil, errors.InvalidParam{Param: "limit"}
	}

	pageNo, err := strconv.Atoi(ctx.Request.QueryParam("pageno"))
	if err != nil || pageNo < 0 {
		return nil, errors.InvalidParam{Param: "pageno"}
	}

	return b.service.GetAll(ctx, nil, &models.Page{Limit: int64(limit), PageNo: int64(pageNo)})
}

func (b blog) GetAllByTag(ctx *app.Context) (interface{}, error) {
	tag := ctx.Request.PathParam("tag")

	limit, err := strconv.Atoi(ctx.Request.QueryParam("limit"))
	if err != nil || limit < 0 {
		return nil, errors.InvalidParam{Param: "limit"}
	}

	pageNo, err := strconv.Atoi(ctx.Request.QueryParam("pageno"))
	if err != nil || pageNo < 0 {
		return nil, errors.InvalidParam{Param: "pageno"}
	}

	return b.service.GetAllByTagName(ctx, tag, &models.Page{Limit: int64(limit), PageNo: int64(pageNo)})
}

func (b blog) GetBlogsByUser(ctx *app.Context) (interface{}, error) {
	accountID := ctx.Request.PathParam("accountid")

	if accountID == "" {
		return nil, errors.MissingParam{Param: "account ID"}
	}

	id, err := strconv.Atoi(accountID)
	if err != nil {
		return nil, errors.InvalidParam{Param: "account ID"}
	}

	return b.service.GetAll(ctx, &models.Blog{AccountID: int64(id)}, nil)
}

func (b blog) Get(ctx *app.Context) (interface{}, error) {
	blogID := ctx.Request.PathParam("blogid")

	return b.service.GetByID(ctx, blogID)
}

func (b blog) Create(ctx *app.Context) (interface{}, error) {
	fileHeaders := ctx.Request.ParseImages()

	blog := &models.Blog{
		Title:   ctx.Request.FormValue("title"),
		Summary: ctx.Request.FormValue("summary"),
		Content: ctx.Request.FormValue("content"),
		Tags:    strings.Split(ctx.Request.FormValue("tags"), ","),
	}

	return b.service.Create(ctx, blog, fileHeaders)
}

func (b blog) Update(ctx *app.Context) (interface{}, error) {
	fileHeaders := ctx.Request.ParseImages()

	tags := make([]string, 0)

	for _, tag := range strings.Split(ctx.Request.FormValue("tags"), ",") {
		tags = append(tags, strings.TrimSpace(tag))
	}

	blog := &models.Blog{
		BlogID:  ctx.Request.PathParam("blogid"),
		Title:   ctx.Request.FormValue("title"),
		Summary: ctx.Request.FormValue("summary"),
		Content: ctx.Request.FormValue("content"),
		Tags:    tags,
	}

	return b.service.Update(ctx, blog, fileHeaders)
}
