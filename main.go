package main

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "cicd_m1_back/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	swagger "github.com/arsmn/fiber-swagger/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/getsentry/sentry-go"
	sentryfiber "github.com/getsentry/sentry-go/fiber"
)

// fiber-swagger middleware

type MongoClient struct {
	Client     *mongo.Client
	Collection *mongo.Collection
}

type Data bson.M

type SentimentResponse struct {
	Score float64 `json:"score"`
}

func InitMongoDB(uri, dbName, collectionName string) (*MongoClient, error) {
	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}
	collection := client.Database(dbName).Collection(collectionName)
	return &MongoClient{Client: client, Collection: collection}, nil
}

func main() {
	// Initialiser Sentry
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              "https://db0f0c26144b686782c222fddcd2665f@o4509507319103488.ingest.de.sentry.io/4509513579626576",
		TracesSampleRate: 1.0, // Capture 100% des transactions pour le tracing
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
	}
	defer sentry.Flush(2 * time.Second)

	// Initialiser Fiber
	app := fiber.New()

	// Middleware Sentry avec condition
	sentryHandler := sentryfiber.New(sentryfiber.Options{
		Repanic:         false, // Désactiver le repanic pour éviter les erreurs critiques
		WaitForDelivery: true,
	})
	app.Use(func(c *fiber.Ctx) error {
		// Ne pas capturer les erreurs sauf pour certaines routes
		if c.Path() == "/foo" || c.Path() == "/generate-error" {
			return sentryHandler(c)
		}
		return c.Next()
	})

	// Middleware pour enrichir les événements Sentry
	enhanceSentryEvent := func(ctx *fiber.Ctx) error {
		if hub := sentryfiber.GetHubFromContext(ctx); hub != nil {
			hub.Scope().SetTag("someRandomTag", "maybeYouNeedIt")
		}
		return ctx.Next()
	}

	// Exemple de route avec enrichissement Sentry
	app.All("/foo", enhanceSentryEvent, func(c *fiber.Ctx) error {
		if hub := sentryfiber.GetHubFromContext(c); hub != nil {
			hub.CaptureMessage("Non-critical event captured on /foo")
		}
		return c.JSON(fiber.Map{"message": "Event captured on /foo"})
	})

	// Exemple de route avec capture de message Sentry
	app.All("/", func(ctx *fiber.Ctx) error {
		if hub := sentryfiber.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("unwantedQuery", "someQueryDataMaybe")
				hub.CaptureMessage("User provided unwanted query string, but we recovered just fine")
			})
		}
		return ctx.SendStatus(fiber.StatusOK)
	})

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Middleware CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	// Nouvelle route pour générer une erreur
	app.Get("/generate-error", func(c *fiber.Ctx) error {
		sentry.CaptureMessage("This is a test error captured by Sentry")
		return c.Status(500).JSON(fiber.Map{"error": "This is a test error"})
	})

	// Route pour gérer les erreurs 404
	app.All("/*", func(c *fiber.Ctx) error {
		log.Printf("404 Error: Path %s not found", c.Path())
		// Capture l'erreur avec Sentry
		sentry.CaptureMessage(fmt.Sprintf("404 Error: Path %s not found", c.Path()))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Route not found",
			"path":  c.Path(),
		})
	})

	// Démarrage du serveur Fiber
	if err := app.Listen(":8080"); err != nil {
		log.Fatal("Erreur démarrage serveur:", err)
	}
}
