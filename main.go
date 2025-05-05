package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoClient encapsule le client MongoDB et la collection
type MongoClient struct {
	Client     *mongo.Client
	Collection *mongo.Collection
}

// Data représente une donnée générique
type Data bson.M

// InitMongoDB initialise la connexion à MongoDB
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
	// Initialiser Fiber
	app := fiber.New()

	// Initialiser MongoDB
	uri := "mongodb+srv://cheikh:aless@cluster0.woq7hfj.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
	mongoClient, err := InitMongoDB(uri, "ci_cd", "test")
	if err != nil {
		log.Fatal("Erreur de connexion à MongoDB:", err)
	}
	defer mongoClient.Client.Disconnect(context.TODO())
	log.Println("Connexion à MongoDB établie !")

	// Route de bienvenue
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to the Fiber RESTful API!")
	})

	// SELECT: Récupérer toutes les données
	app.Get("/data", func(c *fiber.Ctx) error {
		ctx := context.TODO()
		cursor, err := mongoClient.Collection.Find(ctx, bson.D{})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur lors de la récupération des données"})
		}
		defer cursor.Close(ctx)
		var data []Data
		if err := cursor.All(ctx, &data); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur lors du décodage des données"})
		}
		return c.JSON(data)
	})

	// ADD: Ajouter une nouvelle donnée
	app.Post("/data", func(c *fiber.Ctx) error {
		var data Data
		if err := c.BodyParser(&data); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Erreur de parsing des données"})
		}
		ctx := context.TODO()
		result, err := mongoClient.Collection.InsertOne(ctx, data)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur lors de l'ajout des données"})
		}
		return c.JSON(fiber.Map{"inserted_id": result.InsertedID})
	})

	// UPDATE: Mettre à jour une donnée par ID
	app.Put("/data/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID invalide"})
		}
		var data Data
		if err := c.BodyParser(&data); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Erreur de parsing des données"})
		}
		ctx := context.TODO()
		result, err := mongoClient.Collection.UpdateOne(
			ctx,
			bson.M{"_id": objID},
			bson.M{"$set": data},
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur lors de la mise à jour"})
		}
		if result.MatchedCount == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Donnée non trouvée"})
		}
		return c.JSON(fiber.Map{"modified_count": result.ModifiedCount})
	})

	// DELETE: Supprimer une donnée par ID
	app.Delete("/data/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID invalide"})
		}
		ctx := context.TODO()
		result, err := mongoClient.Collection.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Erreur lors de la suppression"})
		}
		if result.DeletedCount == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Donnée non trouvée"})
		}
		return c.JSON(fiber.Map{"deleted_count": result.DeletedCount})
	})

	app.Post("/upload", func(c *fiber.Ctx) error {
		// Try to parse as array first
		var arrayData []bson.M
		if err := c.BodyParser(&arrayData); err == nil && len(arrayData) > 0 {
			// Convert to interface slice for InsertMany
			documents := make([]interface{}, len(arrayData))
			for i, doc := range arrayData {
				documents[i] = doc
			}

			// Insert documents with options to ignore duplicates
			ctx := context.TODO()
			opts := options.InsertMany().SetOrdered(false)
			result, err := mongoClient.Collection.InsertMany(ctx, documents, opts)
			if err != nil {
				// Check for duplicate key errors
				if mongo.IsDuplicateKeyError(err) {
					return c.Status(fiber.StatusOK).JSON(fiber.Map{
						"message":        "Data uploaded with some duplicates ignored",
						"inserted_count": len(result.InsertedIDs),
						"inserted_ids":   result.InsertedIDs,
					})
				}
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to insert data into database",
				})
			}

			return c.JSON(fiber.Map{
				"message":        "Data uploaded and processed successfully",
				"inserted_count": len(result.InsertedIDs),
				"inserted_ids":   result.InsertedIDs,
			})
		}

		// If not an array, try as a single document
		var singleDoc bson.M
		if err := c.BodyParser(&singleDoc); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid JSON format",
			})
		}

		// Insert single document with option to ignore duplicates
		ctx := context.TODO()
		opts := options.InsertOne().SetBypassDocumentValidation(true)
		result, err := mongoClient.Collection.InsertOne(ctx, singleDoc, opts)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return c.JSON(fiber.Map{
					"message": "Document already exists, duplicate ignored",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to insert data into database",
			})
		}

		return c.JSON(fiber.Map{
			"message":     "Data uploaded and processed successfully",
			"inserted_id": result.InsertedID,
		})
	})
	if err := app.Listen(":8080"); err != nil {
		log.Fatal("Erreur lors du démarrage du serveur:", err)
	}
}
