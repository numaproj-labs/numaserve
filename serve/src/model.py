import numpy as np
import onnx
import onnxruntime as ort
import torch
from torch import nn, Tensor


class MyNN(nn.Module):
    def __init__(self, n_features):
        super().__init__()
        self.nn = nn.Sequential(nn.Linear(n_features, 1), nn.Sigmoid())

    def forward(self, x: Tensor) -> Tensor:
        return self.nn(x)


if __name__ == "__main__":
    model = MyNN(5)
    x = torch.randn(1, 5)

    torch.onnx.export(
        model,
        x,
        "/tmp/model.onnx",
        verbose=True,
        input_names=["input"],
        output_names=["output"],
        dynamic_axes={
            "input": {0: "batch_size"},
            "output": {0: "batch_size"},
        },
    )

    onnx_model = onnx.load("/tmp/model.onnx")
    onnx.checker.check_model(onnx_model)

    # Print a human readable representation of the graph
    print(onnx.helper.printable_graph(onnx_model.graph))

    ort_session = ort.InferenceSession("/tmp/model.onnx")
    input_ = np.random.randn(10, 5).astype(np.float32)
    print("input shape", input_.shape)
    outputs = ort_session.run(
        None,
        {"input": input_},
    )
    print(outputs[0].reshape(-1).tolist())
