from pathlib import Path

import pytest
from PIL import Image

from models.duplicates import ImageInput


@pytest.fixture
def test_images_dir(tmp_path: Path) -> Path:
    images_dir = tmp_path / "images"
    images_dir.mkdir(parents=True, exist_ok=True)

    test_specs = [
        ("red.png", (255, 0, 0), (10, 10)),
        ("green.png", (0, 255, 0), (20, 20)),
        ("blue.png", (0, 0, 255), (30, 30)),
    ]
    for filename, color, size in test_specs:
        image_path = images_dir / filename
        image = Image.new("RGB", size=size, color=color)
        image.save(image_path)

    return images_dir


@pytest.fixture
def sample_image_inputs(test_images_dir: Path) -> list[ImageInput]:
    images: list[ImageInput] = []
    for index, image_path in enumerate(sorted(test_images_dir.glob("*.png")), start=1):
        with Image.open(image_path) as image:
            width, height = image.size
        images.append(
            ImageInput(
                id=index,
                path=str(image_path),
                width=width,
                height=height,
                file_size=image_path.stat().st_size,
                format="png",
            )
        )
    return images
