import logging
import os

from pynumaflow.mapper import MultiProcMapper

from src import set_logger
from src.udf import InferenceUDF


LOGGER = logging.getLogger(__name__)
MODEL_PATH = os.getenv("OUTPUT_PATH", "model.onnx")
MODEL_KEY = os.getenv("MODEL_KEY", "model.onnx")


def serve():
    udf = InferenceUDF(model_path=MODEL_PATH, model_key=MODEL_KEY)
    server = MultiProcMapper(handler=udf)

    LOGGER.info("Running server with model path: %s", MODEL_PATH)

    server.start()


if __name__ == "__main__":
    set_logger()
    serve()

