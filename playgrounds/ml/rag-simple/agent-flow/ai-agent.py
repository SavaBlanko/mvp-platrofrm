import os
import time
import uuid

import httpx
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient

load_dotenv()

QDRANT_URL = os.getenv("QDRANT_URL")
COLLECTION = os.getenv("QDRANT_COLLECTION")
MODEL_NAME = os.getenv("EMBEDDING_MODEL")
OLLAMA_URL = os.getenv("OLLAMA_URL")
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL")

app = FastAPI(title="RAG Agent")

embedding_model = SentenceTransformer(MODEL_NAME)
qdrant = QdrantClient(url=QDRANT_URL)


class Message(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    model: str
    messages: list[Message]
    stream: bool = False


@app.get("/health")
def health():
    return {"status": "ok"}


@app.get("/v1/models")
def models():
    return {
        "object": "list",
        "data": [{
            "id": "fruits-rag",
            "object": "model",
            "created": 0,
            "owned_by": "rag-agent",
        }],
    }


@app.post("/v1/chat/completions")
async def chat(request: ChatRequest):
    if request.model != "fruits-rag":
        raise HTTPException(400, "Unknown model")

    if request.stream:
        raise HTTPException(400, "Streaming not supported")

    questions = [
        m.content for m in request.messages
        if m.role == "user"
    ]

    if not questions:
        raise HTTPException(400, "No user question")

    question = questions[-1]

    # READ Qdrant
    vector = embedding_model.encode(question).tolist()

    results = qdrant.query_points(collection_name=COLLECTION, query=vector, limit=3, with_payload=True)

    context = "\n".join(
        point.payload.get("text", "")
        for point in results.points
    )

    prompt = f"""
Answer using only this context.
If the answer is not available, say you don't know.

Context:
{context}

Question:
{question}
"""

    async with httpx.AsyncClient(timeout=120) as client:
        response = await client.post(
            f"{OLLAMA_URL}/api/generate",
            json={
                "model": OLLAMA_MODEL,
                "prompt": prompt,
                "stream": False,
            },
        )
        response.raise_for_status()
        answer = response.json()["response"]

    return {
        "id": f"chatcmpl-{uuid.uuid4().hex}",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": request.model,
        "choices": [{
            "index": 0,
            "message": {
                "role": "assistant",
                "content": answer,
            },
            "finish_reason": "stop",
        }],
    }