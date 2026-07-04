from flask import Flask, render_template, request, jsonify
import numpy as np
import pickle
import os
from dataclasses import dataclass
from typing import List
import math

app = Flask(__name__)

@dataclass
class Document:
    id: str
    title: str
    source: str
    section: str
    text: str
    embedding: List[float]
    
    def to_dict(self):
        return {
            'id': self.id,
            'title': self.title,
            'source': self.source,
            'section': self.section,
            'text': self.text[:500] + '...' if len(self.text) > 500 else self.text
        }

class RAGSearcher:
    def __init__(self, index_path):
        self.docs = []
        self.load_index(index_path)
    
    def load_index(self, path):
        if not os.path.exists(path):
            print(f'Index not found: {path}')
            return
        
        with open(path, 'rb') as f:
            import gob
            # Simple pickle fallback
            try:
                data = pickle.load(f)
                if isinstance(data, dict) and 'docs' in data:
                    for doc in data['docs']:
                        self.docs.append(Document(**doc))
                print(f'Loaded {len(self.docs)} documents')
            except Exception as e:
                print(f'Error loading index: {e}')
    
    def cosine_similarity(self, a, b):
        dot = sum(x*y for x,y in zip(a,b))
        norm_a = math.sqrt(sum(x*x for x in a))
        norm_b = math.sqrt(sum(x*x for x in b))
        if norm_a == 0 or norm_b == 0:
            return 0
        return dot / (norm_a * norm_b)
    
    def keyword_score(self, query, text):
        query_lower = query.lower()
        text_lower = text.lower()
        tokens = query_lower.split()
        matched = sum(1 for t in tokens if t in text_lower)
        score = matched / len(tokens) if tokens else 0
        if query_lower in text_lower:
            score += 0.2
        return min(score, 1.0)
    
    def search(self, query, query_embedding, top_k=5):
        results = []
        for doc in self.docs:
            vector_score = self.cosine_similarity(query_embedding, doc.embedding)
            keyword_score = self.keyword_score(query, doc.title + ' ' + doc.text)
            score = round(vector_score * 0.5 + keyword_score * 0.5, 6)
            results.append((score, doc))
        
        results.sort(reverse=True, key=lambda x: x[0])
        return [(score, doc.to_dict()) for score, doc in results[:top_k]]

# Global searcher
searcher = None

def init_searcher():
    global searcher
    index_path = os.path.join('data', 'index.bin')
    searcher = RAGSearcher(index_path)

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/api/search', methods=['POST'])
def search():
    data = request.json
    query = data.get('query', '').strip()
    top_k = min(int(data.get('top_k', 5)), 10)
    
    if not query:
        return jsonify({'error': 'Query is required'}), 400
    
    if searcher is None or len(searcher.docs) == 0:
        return jsonify({'error': 'Index not loaded'}), 503
    
    # TODO: Get embedding from embedding service
    # For now, use dummy embedding
    query_embedding = [0.0] * 512
    
    results = searcher.search(query, query_embedding, top_k)
    
    return jsonify({
        'query': query,
        'results': [{'score': score, 'doc': doc} for score, doc in results],
        'total': len(results)
    })

@app.route('/api/status')
def status():
    return jsonify({
        'loaded': searcher is not None and len(searcher.docs) > 0,
        'documents': len(searcher.docs) if searcher else 0
    })

if __name__ == '__main__':
    init_searcher()
    app.run(host='127.0.0.1', port=9093, debug=True)
