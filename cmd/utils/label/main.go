"""
Implements utils/label-image buildpack.
The label-image buildpack adds any environment variables with the "GOOGLE_LABEL_" prefix as labels in the final application image.

Copyright 2025 Google LLC Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
"""

from fastapi import FastAPI
from pydantic import BaseModel
import asyncio

app = FastAPI()

class LabelConfig(BaseModel):
    environment_variables: dict[str, str]

@app.post("/detect")
async def detect() -> dict:
    """
    Detects if the label buildpack should be applied based on presence of GOOGLE_LABEL_ prefixed env vars.
    """
    # In a real implementation, this would check for GOOGLE_LABEL_ prefixed env vars
    return {"applies": True}

@app.post("/build")
async def build(config: LabelConfig) -> dict:
    """
    Adds labels from environment variables with the GOOGLE_LABEL_ prefix to the image.
    """
    labels = {
        key.replace("GOOGLE_LABEL_", "").lower(): value
        for key, value in config.environment_variables.items()
        if key.startswith("GOOGLE_LABEL_")
    }

    return {"labels": labels}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
