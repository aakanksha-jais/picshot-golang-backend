package app

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/datastore"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/test"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/constants"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/configs"
	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/log"
)

type App struct {
	server *server
	log.Logger
	configs.Config
	datastore.DataStore
}

func New() *App {
	app := &App{}

	// initialize configs
	app.readConfig()

	// initialize Logger
	app.Logger = log.NewLogger()

	app.initializeStores(app.Config)

	// For End-to-End Testing
	if strings.EqualFold(app.Get("ENV"), "development") {
		app.loadTestData()
	}

	// initialize server
	app.server = NewServer(app)

	return app
}

// GET adds a Handler for http GET method for a route pattern.
func (a *App) GET(pattern string, handler Handler) {
	a.add(http.MethodGet, pattern, handler)
}

// PUT adds a Handler for http PUT method for a route pattern.
func (a *App) PUT(pattern string, handler Handler) {
	a.add(http.MethodPut, pattern, handler)
}

// POST adds a Handler for http POST method for a route pattern.
func (a *App) POST(pattern string, handler Handler) {
	a.add(http.MethodPost, pattern, handler)
}

// DELETE adds a Handler for http DELETE method for a route pattern.
func (a *App) DELETE(pattern string, handler Handler) {
	a.add(http.MethodDelete, pattern, handler)
}

func (a *App) add(method, pattern string, h Handler) {
	a.server.Router.Add(method, pattern, h)
}

func (a App) Start() {
	a.server.Start(a.Logger)
}

// readConfig reads the configuration from the default location.
func (a *App) readConfig() {
	var configLocation string
	if _, err := os.Stat("./configs"); err == nil {
		configLocation = "./configs"
	}

	a.Config = configs.NewConfigLoader(configLocation)
}

func (a *App) initializeStores(config configs.Config) {
	var err error

	a.Mongo, err = datastore.GetNewMongoDB(a.Logger, config)
	if err != nil {
		go mongoRetry(config, a)
	}

	a.SQL, err = datastore.GetNewSQLClient(a.Logger, config)
	if err != nil {
		go sqlRetry(config, a)
	}

	a.S3, err = datastore.GetNewS3(a.Logger, config)
	if err != nil {
		go s3Retry(config, a)
	}
}

func mongoRetry(config configs.Config, app *App) {
	for i := 0; i < constants.MaxRetries; i++ {
		time.Sleep(time.Duration(constants.RetryDuration) * time.Second)

		app.Debug("retrying mongo connection")

		var err error

		app.Mongo, err = datastore.GetNewMongoDB(app, config)

		if err == nil {
			app.Info("mongo initialized successfully")

			break
		}
	}
}

func sqlRetry(config configs.Config, app *App) {
	for i := 0; i < constants.MaxRetries; i++ {
		time.Sleep(time.Duration(constants.RetryDuration) * time.Second)

		app.Debug("retrying sql connection")

		var err error

		app.SQL, err = datastore.GetNewSQLClient(app, config)

		if err == nil {
			app.Info("sql initialized successfully")

			break
		}
	}
}

func s3Retry(config configs.Config, app *App) {
	for i := 0; i < constants.MaxRetries; i++ {
		time.Sleep(time.Duration(constants.RetryDuration) * time.Second)

		app.Debug("retrying s3 session creation")

		var err error

		app.S3, err = datastore.GetNewS3(app, config)

		if err == nil {
			app.Info("s3 session initialized successfully")

			break
		}
	}
}

func (a *App) loadTestData() {
	test.AddTestData(a.Mongo, a.SQL, a.S3, a.Logger)

	a.Infof("test data has been loaded: all existing data across all databases is overwritten")
}
