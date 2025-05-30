package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	_ "cicd_m1_back/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	// Import the generated Swagger docs

	swagger "github.com/arsmn/fiber-swagger/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	app := fiber.New()
	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",                           // Autorise toutes les origines
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS", // Autorise ces méthodes HTTP
	}))

	uri := "mongodb+srv://cheikh:aless@cluster0.woq7hfj.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
	mongoClient, err := InitMongoDB(uri, "cicd", "cheikh")
	if err != nil {
		log.Fatal("Erreur de connexion à MongoDB:", err)
	}
	defer mongoClient.Client.Disconnect(context.TODO())
	log.Println("Connexion à MongoDB établie !")

	// @Summary Root Endpoint
	// @Description Returns a welcome message for the API.
	// @Tags root
	// @Accept json
	// @Produce json
	// @Success 200 {string} string "Welcome message"
	// @Router / [get]
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API de gestion des feedbacks")
	})

	// @Summary List Feedbacks
	// @Description Retrieve all feedbacks from the database.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Success 200 {array} map[string]interface{}
	// @Router /feedbacks [get]
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

	// @Summary Add Feedback
	// @Description Add a new feedback to the database.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Param feedback body map[string]interface{} true "Feedback data"
	// @Success 200 {object} map[string]interface{}
	// @Router /feedbacks [post]
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

	// @Summary Update Feedback
	// @Description Update an existing feedback by ID.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Param id path string true "Feedback ID"
	// @Param feedback body map[string]interface{} true "Updated feedback data"
	// @Success 200 {object} map[string]interface{}
	// @Router /feedbacks/{id} [put]
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

	// @Summary Delete Feedback
	// @Description Delete a feedback by ID.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Param id path string true "Feedback ID"
	// @Success 200 {object} map[string]interface{}
	// @Router /feedbacks/{id} [delete]
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

	// @Summary Upload Feedbacks
	// @Description Upload multiple or a single feedback.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Param feedbacks body []map[string]interface{} true "Array of feedbacks"
	// @Success 200 {object} map[string]interface{}
	// @Router /feedbacks/upload [post]
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

	// @Summary Analyze Feedback
	// @Description Analyze the sentiment of a single feedback by ID.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Param id path string true "Feedback ID"
	// @Success 200 {object} map[string]interface{}
	// @Router /feedbacks/{id}/analyze [post]
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
		resp, err := http.Post("http://localhost:5000/analyze", "application/json", bytes.NewBuffer(payload))
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

	// @Summary Analyze All Feedbacks
	// @Description Perform sentiment analysis on all feedbacks.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /feedbacks/analyze-all [post]
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

	// @Summary Average Score
	// @Description Calculate the average score of all feedbacks in the database.
	// @Tags feedbacks
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]float64 "Average score"
	// @Router /feedbacks/average-score [get]
	app.Get("/feedbacks/average-score", func(c *fiber.Ctx) error {
		ctx := context.TODO()

		// MongoDB aggregation pipeline to calculate the average score
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

	// Démarrage du serveur Fiber
	if err := app.Listen(":8080"); err != nil {
		log.Fatal("Erreur démarrage serveur:", err)
	}
}
