import logging
from base64 import b64decode, b64encode

import numpy as np
import numpy.typing as npt
import onnx
from orjson import orjson
from pynumaflow.mapper import Datum, Messages, Message
import onnxruntime as ort


_LOGGER = logging.getLogger(__name__)


class InferenceUDF:
    __slots__ = ("model", "model_key", "ort_sess")

    def __init__(self, model_path: str, model_key: str | None = None):
        self.model = onnx.load(model_path)
        self.model_key = model_key or model_path
        onnx.checker.check_model(self.model)
        self.ort_sess = ort.InferenceSession(model_path)

    def __call__(self, keys: list[str], datum: Datum) -> Messages:
        return self.exec(keys, datum)

    def compute(self, input_: npt.NDArray[float]) -> npt.NDArray[float]:
        outputs = self.ort_sess.run(None, {"input": input_})
        return outputs[0]

    @staticmethod
    def decode_payload(encoded: bytes) -> dict[str, list]:
        return orjson.loads(b64decode(encoded))

    @staticmethod
    def encode_payload(payload: dict[str, list]) -> str:
        return b64encode(orjson.dumps(payload)).decode()

    def exec(self, keys: list[str], datum: Datum) -> Messages:
        mlserve_request = orjson.loads(datum.value)
        _LOGGER.debug("Input: %s", mlserve_request)

        try:
            payload = self.decode_payload(mlserve_request["payload"])
        except KeyError:
            raise RuntimeError("Payload not found in request!") from None

        x = np.asarray(payload["data"], dtype=np.float32)
        y = self.compute(x)

        mlserve_request["payload"] = self.encode_payload(
            {
                "data": y.tolist(),
                "model_key": self.model_key,
            }
        )

        _LOGGER.debug("Output: %s", mlserve_request)
        return Messages(Message(orjson.dumps(mlserve_request), keys=keys))
