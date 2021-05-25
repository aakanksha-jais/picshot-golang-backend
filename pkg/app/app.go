package app

import (
	"net/http"
	"os"
	"time"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/configs"
	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/log"
)

type App struct {
	server *server
	Logger log.Logger
	Config configs.Config
	DataStore
}

func New() *App {
	app := &App{}

	// initialize configs
	app.readConfig()

	// initialize Logger
	app.Logger = log.NewLogger()

	app.initializeStores(app.Config)

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

	a.Mongo, err = GetNewMongoDB(a.Logger, config)
	if err != nil {
		go mongoRetry(nil, a)
	}

	a.SQL, err = GetNewSQLClient(a.Logger, config)
	if err != nil {
		go sqlRetry(nil, a)
	}

	a.S3, err = GetNewS3(a.Logger, config)
	if err != nil {
		go s3Retry(nil, a)
	}
}

const maxRetries = 1
const retryDuration = 3

func mongoRetry(config configs.Config, app *App) {
	for i := 0; i < maxRetries; i++ {
		time.Sleep(time.Duration(retryDuration) * time.Second)

		app.Logger.Debug("retrying mongo connection")

		var err error

		app.Mongo, err = GetNewMongoDB(app.Logger, config)

		if err == nil {
			app.Logger.Info("mongo initialized successfully")

			break
		}
	}
}

func sqlRetry(config configs.Config, app *App) {
	for i := 0; i < maxRetries; i++ {
		time.Sleep(time.Duration(retryDuration) * time.Second)

		app.Logger.Debug("retrying sql connection")

		var err error

		app.SQL, err = GetNewSQLClient(app.Logger, config)

		if err == nil {
			app.Logger.Info("sql initialized successfully")

			break
		}
	}
}

func s3Retry(config configs.Config, app *App) {
	for i := 0; i < maxRetries; i++ {
		time.Sleep(time.Duration(retryDuration) * time.Second)

		app.Logger.Debug("retrying s3 session creation")

		var err error

		app.S3, err = GetNewS3(app.Logger, config)

		if err == nil {
			app.Logger.Info("s3 session initialized successfully")

			break
		}
	}
}