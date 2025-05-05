from flask import Flask, request, jsonify
from textblob import TextBlob
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

@app.route("/analyze", methods=["POST"])
def analyze():
    data = request.get_json()
    commentaire = data.get("text", "")
    blob = TextBlob(commentaire)
    polarite = blob.sentiment.polarity
    return jsonify({"score": polarite})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
