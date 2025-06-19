package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"cicd_m1_back/config"
	_ "cicd_m1_back/docs"

	"github.com/getsentry/sentry-go"
	sentryfiber "github.com/getsentry/sentry-go/fiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	// Import the generated Swagger docss

	swagger "github.com/arsmn/fiber-swagger/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func setupRoutes(app *fiber.App, mongoClient *MongoClient, api_url_sent string) {
	// Root Endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API de gestion des feedbacks")
	})

	// List Feedbacks
	app.Get("/feedbacks", func(c *fiber.Ctx) error {
		ctx := context.TODO()
		cursor, err := mongoClient.Collection.Find(ctx, bson.D{})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur de récupération"})
		}
		defer cursor.Close(ctx)

		var data []Data
		if err := cursor.All(ctx, &data); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur de décodage"})
		}
		return c.JSON(data)
	})

	// Add Feedback
	app.Post("/feedbacks", func(c *fiber.Ctx) error {
		var data Data
		if err := c.BodyParser(&data); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Erreur de parsing"})
		}
		ctx := context.TODO()
		result, err := mongoClient.Collection.InsertOne(ctx, data)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur d'insertion"})
		}
		return c.JSON(fiber.Map{"inserted_id": result.InsertedID})
	})

	// Update Feedback
	app.Put("/feedbacks/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
		}
		var data Data
		if err := c.BodyParser(&data); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Erreur de parsing"})
		}
		ctx := context.TODO()
		result, err := mongoClient.Collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": data})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur mise à jour"})
		}
		if result.MatchedCount == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Donnée non trouvée"})
		}
		return c.JSON(fiber.Map{"modified_count": result.ModifiedCount})
	})

	// Delete Feedback
	app.Delete("/feedbacks/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "ID invalide"})
		}
		ctx := context.TODO()
		result, err := mongoClient.Collection.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur suppression"})
		}
		if result.DeletedCount == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Non trouvé"})
		}
		return c.JSON(fiber.Map{"deleted_count": result.DeletedCount})
	})

	// Upload Feedbacks
	app.Post("/feedbacks/upload", func(c *fiber.Ctx) error {
		var arrayData []bson.M
		if err := c.BodyParser(&arrayData); err == nil && len(arrayData) > 0 {
			documents := make([]interface{}, len(arrayData))
			for i, doc := range arrayData {
				documents[i] = doc
			}
			ctx := context.TODO()
			opts := options.InsertMany().SetOrdered(false)
			result, err := mongoClient.Collection.InsertMany(ctx, documents, opts)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Erreur insertion multiple"})
			}
			return c.JSON(fiber.Map{"inserted_count": len(result.InsertedIDs)})
		}

		var singleDoc bson.M
		if err := c.BodyParser(&singleDoc); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Format JSON invalide"})
		}
		ctx := context.TODO()
		result, err := mongoClient.Collection.InsertOne(ctx, singleDoc)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur insertion simple"})
		}
		return c.JSON(fiber.Map{"inserted_id": result.InsertedID})
	})

	// Analyze Feedback
	app.Post("/feedbacks/:id/analyze", func(c *fiber.Ctx) error {
		id := c.Params("id")
		ctx := context.TODO()

		var doc bson.M
		err := mongoClient.Collection.FindOne(ctx, bson.M{"id": id}).Decode(&doc)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Feedback non trouvé"})
		}

		text, ok := doc["text"].(string)
		if !ok {
			return c.Status(400).JSON(fiber.Map{"error": "Champ texte invalide ou absent"})
		}

		payload, _ := json.Marshal(map[string]string{"text": text})
		resp, err := http.Post(api_url_sent, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur appel service sentiment"})
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		var sentiment SentimentResponse
		json.Unmarshal(body, &sentiment)

		_, err = mongoClient.Collection.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"score": sentiment.Score}})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur mise à jour score"})
		}

		return c.JSON(fiber.Map{"id": id, "score": sentiment.Score})
	})

	// Analyze All Feedbacks
	app.Post("/feedbacks/analyze-all", func(c *fiber.Ctx) error {
		ctx := context.TODO()

		cursor, err := mongoClient.Collection.Find(ctx, bson.D{})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur de récupération des feedbacks"})
		}
		defer cursor.Close(ctx)

		var updated int
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				continue
			}

			text, ok := doc["text"].(string)
			if !ok {
				continue
			}

			id, ok := doc["id"].(string)
			if !ok {
				continue
			}

			payload, _ := json.Marshal(map[string]string{"text": text})
			resp, err := http.Post("https://cicd-m1-back-sent.onrender.com/analyze", "application/json", bytes.NewBuffer(payload))
			if err != nil {
				continue
			}
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			var sentiment SentimentResponse
			json.Unmarshal(body, &sentiment)

			_, err = mongoClient.Collection.UpdateOne(ctx, bson.M{"id": id}, bson.M{"$set": bson.M{"score": sentiment.Score}})
			if err == nil {
				updated++
			}
		}

		return c.JSON(fiber.Map{"updated_count": updated})
	})

	// Average Score
	app.Get("/feedbacks/average-score", func(c *fiber.Ctx) error {
		ctx := context.TODO()

		pipeline := mongo.Pipeline{
			{
				{"$group", bson.D{{"_id", nil}, {"averageScore", bson.D{{"$avg", "$score"}}}}},
			},
		}

		cursor, err := mongoClient.Collection.Aggregate(ctx, pipeline)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur de calcul de la moyenne"})
		}
		defer cursor.Close(ctx)

		var result []bson.M
		if err := cursor.All(ctx, &result); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur de décodage des résultats"})
		}

		if len(result) == 0 {
			return c.JSON(fiber.Map{"average_score": 0})
		}

		averageScore, ok := result[0]["averageScore"].(float64)
		if !ok {
			return c.Status(500).JSON(fiber.Map{"error": "Erreur de conversion de la moyenne"})
		}

		return c.JSON(fiber.Map{"average_score": averageScore})
	})

	// Foo Error
	app.Get("/foo-error", func(c *fiber.Ctx) error {
		err := fmt.Errorf("Non-critical foo error occurred")
		log.Println(err)
		sentry.CaptureException(err)
		return c.JSON(fiber.Map{"message": "Non-critical foo error reported to Sentry"})
	})

	// Generate Error
	app.Get("/generate-error", func(c *fiber.Ctx) error {
		err := fmt.Errorf("This is a test error captured by Sentry")
		sentry.CaptureException(err)
		return c.Status(500).JSON(fiber.Map{"error": "This is a test error"})
	})
}

func setupMiddlewares(app *fiber.App) {
	// CORS Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:8080, ${API_URL_SENTIMENT}, ${API_URL_FRONT}",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type, Authorization",
	}))

	// Sentry Middleware
	app.Use(sentryfiber.New(sentryfiber.Options{}))

	// Middleware pour gérer les erreurs 404
	app.Use(func(c *fiber.Ctx) error {
		err := c.Next()
		if c.Response().StatusCode() == fiber.StatusNotFound {
			notFoundErr := fmt.Errorf("404 Error: Path %s not found", c.Path())
			log.Println(notFoundErr)
			sentry.CaptureException(notFoundErr)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Route not found",
				"path":  c.Path(),
			})
		}
		return err
	})
}

func main() {
	app := fiber.New()
	app.Get("/swagger/*", swagger.HandlerDefault)

	uri := "mongodb+srv://cheikh:aless@cluster0.woq7hfj.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
	mongoClient, err := InitMongoDB(uri, "cicd", "cheikh")
	if err != nil {
		log.Fatal("Erreur de connexion à MongoDB:", err)
	}
	defer mongoClient.Client.Disconnect(context.TODO())
	log.Println("Connexion à MongoDB établie !")

	// Initialize Sentry
	err = sentry.Init(sentry.ClientOptions{
		Dsn:              "https://db0f0c26144b686782c222fddcd2665f@o4509507319103488.ingest.de.sentry.io/4509513579626576",
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Flush(2 * time.Second)

	// Récupération de l'URL de l'API Python pour l'analyse de sentiment
	api_url_sent := config.Config("API_URL_SENTIMENT")

	// Setup middlewares
	setupMiddlewares(app)

	// Setup routes
	setupRoutes(app, mongoClient, api_url_sent)

	// Start the server
	if err := app.Listen(":8080"); err != nil {
		log.Fatal("Erreur démarrage serveur:", err)
	}
}
