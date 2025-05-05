# Feedback Management API

This project is a Feedback Management API built using Go and the Fiber web framework. It provides endpoints for managing feedbacks, including CRUD operations and sentiment analysis.

## Features

- **CRUD Operations**: Create, Read, Update, and Delete feedbacks.
- **Sentiment Analysis**: Analyze the sentiment of feedbacks.
- **Batch Upload**: Upload multiple feedbacks at once.
- **Swagger Documentation**: API documentation available via Swagger UI.

## Prerequisites

- Go 1.23 or later
- MongoDB instance
- Docker (optional, for containerized deployment)

## The product

Go to the api : https://cicdm1.onrender.com

## Installation

1. Clone the repository:

   ```bash
   git clone <repository-url>
   cd CICD_M1_BACK
   ```
2. Install dependencies:

   ```bash
   go mod tidy
   ```
3. Set up environment variables:

   - Create a `.env` file in the root directory.
   - Add the following variables:
     ```env
     MONGO_URI=<your-mongodb-uri>
     DB_NAME=cicd
     COLLECTION_NAME=cheikh
     ```

## Running the Application

1. Start the application:

   ```bash
   go run main.go
   ```
2. Access the API:

   - Base URL: `http://localhost:8080`
   - Swagger UI: `http://localhost:8080/swagger/index.html`

## API Endpoints

### Root Endpoint

- **GET /**: Returns a welcome message.

### Feedbacks

- **GET /feedbacks**: Retrieve all feedbacks.
- **POST /feedbacks**: Add a new feedback.
- **PUT /feedbacks/{id}**: Update an existing feedback by ID.
- **DELETE /feedbacks/{id}**: Delete a feedback by ID.

### Batch Operations

- **POST /feedbacks/upload**: Upload multiple feedbacks.

### Sentiment Analysis

- **POST /feedbacks/{id}/analyze**: Analyze the sentiment of a single feedback by ID.
- **POST /feedbacks/analyze-all**: Perform sentiment analysis on all feedbacks.

## Docker Deployment

1. Build the Docker image:

   ```bash
   docker build -t feedback-api .
   ```
2. Run the container:

   ```bash
   docker run -p 8080:8080 --env-file .env feedback-api
   ```

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.

## License

This project is licensed under the MIT License.
