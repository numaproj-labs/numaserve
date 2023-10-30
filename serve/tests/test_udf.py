import logging
import os
import unittest
from datetime import datetime

import numpy as np
from orjson import orjson
from pynumaflow.mapper import Datum

from src.udf import InferenceUDF
from src._constants import BASE_DIR


logging.basicConfig(level=logging.DEBUG)


class TestInferenceUDF(unittest.TestCase):
    def setUp(self):
        self.udf = InferenceUDF(model_path=os.path.join(BASE_DIR, "model.onnx"), model_key="m_v1")

    def test_compute(self):
        y = self.udf.compute(np.random.randn(10, 5).astype(np.float32))
        self.assertTupleEqual((10, 1), y.shape)

    def test_exec(self):
        _datum_value = {
            "metadata": {
                "sender": "http:local:8443/callback",
                "requestID": "925857e6-032f-4154-a2ac-c093a6e2c2ec",
            },
            "payload": "eyJkYXRhIjpbWzEsMiwzLDQsNV1dfQ==",
        }

        datum = Datum(
            keys=["key"],
            value=orjson.dumps(_datum_value),
            event_time=datetime.now(),
            watermark=datetime.now(),
        )
        msgs = self.udf.exec(["key"], datum)

        output = orjson.loads(msgs[0].value)
        outpayload = self.udf.decode_payload(output["payload"])

        self.assertTupleEqual((1, 1), np.asarray(outpayload["data"]).shape)
        self.assertEqual("m_v1", outpayload["model_key"])
        self.assertEqual(1, len(msgs))


if __name__ == "__main__":
    unittest.main()
