from flask import Flask, request, jsonify
from textblob import TextBlob

app = Flask(__name__)

@app.route('/analyze', methods=['POST'])
def analyze():
    data = request.get_json()
    text = data.get("text", "")
    blob = TextBlob(text)
    score = blob.sentiment.polarity
    return jsonify({"score": score})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
