package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
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

func main() {
	app := fiber.New()

	uri := "mongodb+srv://cheikh:aless@cluster0.woq7hfj.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
	mongoClient, err := InitMongoDB(uri, "ci_cd", "test")
	if err != nil {
		log.Fatal("Erreur de connexion à MongoDB:", err)
	}
	defer mongoClient.Client.Disconnect(context.TODO())
	log.Println("Connexion à MongoDB établie !")

	// Welcome route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API de gestion des feedbacks")
	})

	// Lister tous les feedbacks
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

	// Ajouter un feedback
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

	// Mettre à jour un feedback
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

	// Supprimer un feedback
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

	// Upload multiple ou un seul document
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

	// Analyse de sentiment d’un feedback
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

	if err := app.Listen(":8080"); err != nil {
		log.Fatal("Erreur démarrage serveur:", err)
	}
}
