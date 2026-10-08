import os
import uuid
from pathlib import Path

from dotenv import load_dotenv
from sentence_transformers import SentenceTransformer
from qdrant_client import QdrantClient, models

load_dotenv()

QDRANT_URL = os.getenv("QDRANT_URL")
COLLECTION = os.getenv("QDRANT_COLLECTION")
MODEL_NAME = os.getenv("EMBEDDING_MODEL")

model = SentenceTransformer(MODEL_NAME)
qdrant = QdrantClient(url=QDRANT_URL)

SOURCE = Path(__file__).parent / "fruits.txt"
DOCUMENT_ID = "fruits.txt"

chunks = [
    line.strip()
    for line in SOURCE.read_text(encoding="utf-8").splitlines()
    if line.strip()
]

if not chunks:
    raise ValueError("Source document is empty")

dimension = model.get_sentence_embedding_dimension()

if not qdrant.collection_exists(COLLECTION):
    qdrant.create_collection(
        collection_name=COLLECTION,
        vectors_config=models.VectorParams(
            size=dimension,
            distance=models.Distance.COSINE,
        ),
    )

# Delete old chunks for this document. Fine for a PoC, but not atomic.
qdrant.delete(
    collection_name=COLLECTION,
    points_selector=models.FilterSelector(
        filter=models.Filter(
            must=[
                models.FieldCondition(
                    key="document_id",
                    match=models.MatchValue(value=DOCUMENT_ID),
                )
            ]
        )
    ),
    wait=True,
)

embeddings = model.encode(chunks)

points = [
    models.PointStruct(
        id=str(uuid.uuid5(
            uuid.NAMESPACE_URL,
            f"{DOCUMENT_ID}:{i}",
        )),
        vector=vector.tolist(),
        payload={
            "document_id": DOCUMENT_ID,
            "chunk_id": i,
            "text": text,
            "embedding_model": MODEL_NAME,
        },
    )
    for i, (text, vector)
    in enumerate(zip(chunks, embeddings))
]

qdrant.upsert(collection_name=COLLECTION, points=points, wait=True)

print(f"Indexed {len(points)} chunks")