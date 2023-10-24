package main

import (
	"benjitucker/bathrc-accounts/handlers"
	"flag"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/auth0-community/go-auth0"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	jose "gopkg.in/square/go-jose.v2"
)

var (
	audience string
	domain   string
	logger   log.Logger
)

func main() {
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.NewSyncLogger(logger)
	logger = log.With(logger,
		"service", "compliance-specifications-lambda",
		"time:", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)

	_ = level.Info(logger).Log("msg", "service started")

	flag.Parse()

	logLevel, exists := os.LookupEnv("LOG_LEVEL")
	if !exists {
		logLevel = "info"
	}

	switch logLevel {
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	default:
		logger = level.NewFilter(logger, level.AllowAll())
		_ = level.Error(logger).Log("msg", "bad logging level, defaulting to all")
	}

	setAuth0Variables()
	r := gin.Default()
	r.Use(CORSMiddleware())

	// This will ensure that the angular files are served correctly
	r.NoRoute(func(c *gin.Context) {
		dir, file := path.Split(c.Request.RequestURI)
		ext := filepath.Ext(file)
		if file == "" || ext == "" {
			c.File("./ui/dist/ui/index.html")
		} else {
			c.File("./ui/dist/ui/" + path.Join(dir, file))
		}
	})

	authorized := r.Group("/")
	authorized.Use(authRequired())
	authorized.GET("/todo", handlers.GetTodoListHandler)
	authorized.POST("/todo", handlers.AddTodoHandler)
	authorized.DELETE("/todo/:id", handlers.DeleteTodoHandler)
	authorized.PUT("/todo", handlers.CompleteTodoHandler)

	authorized.GET("/housing_location", handlers.GetHousingLocationListHandler)
	authorized.GET("/housing_location/:id", handlers.GetHousingLocationByIdHandler)

	err := r.Run(":13000")
	if err != nil {
		panic(err)
	}
}

func setAuth0Variables() {
	//audience = os.Getenv("AUTH0_API_IDENTIFIER")
	//audience = "https://terraform-express-resource-server"
	audience = "https://dev-otm8dxsmwyxedtdp.us.auth0.com/api/v2/"
	//domain = os.Getenv("AUTH0_DOMAIN")
	domain = "dev-otm8dxsmwyxedtdp.us.auth0.com" // or Tenant
}

// ValidateRequest will verify that a token received from an http request
// is valid and signy by authority
func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {

		var auth0Domain = "https://" + domain + "/"
		client := auth0.NewJWKClient(auth0.JWKClientOptions{URI: auth0Domain + ".well-known/jwks.json"}, nil)
		configuration := auth0.NewConfiguration(client, []string{audience}, auth0Domain, jose.RS256)
		validator := auth0.NewValidator(configuration, nil)

		_, err := validator.ValidateRequest(c.Request)

		if err != nil {
			_ = level.Error(logger).Log("err", err)
			terminateWithError(http.StatusUnauthorized, "token is not valid", c)
			return
		}
		c.Next()
	}
}

func terminateWithError(statusCode int, message string, c *gin.Context) {
	c.JSON(statusCode, gin.H{"error": message})
	c.Abort()
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE, GET, OPTIONS, POST, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

/* MONGO STUFF:

import (
  "context"
  "fmt"

  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/mongo"
  "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
  // Use the SetServerAPIOptions() method to set the Stable API version to 1
  serverAPI := options.ServerAPI(options.ServerAPIVersion1)
  opts := options.Client().ApplyURI("mongodb+srv://application-brc:TqUCImt2l31TBXil@atlascluster.b5bwuxg.mongodb.net/?retryWrites=true&w=majority").SetServerAPIOptions(serverAPI)

  // Create a new client and connect to the server
  client, err := mongo.Connect(context.TODO(), opts)
  if err != nil {
    panic(err)
  }

  defer func() {
    if err = client.Disconnect(context.TODO()); err != nil {
      panic(err)
    }
  }()

  // Send a ping to confirm a successful connection
  if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
    panic(err)
  }
  fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
}

*/
