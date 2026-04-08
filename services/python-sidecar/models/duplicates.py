from pydantic import BaseModel


class ImageInput(BaseModel):
    id: int
    path: str
    width: int
    height: int
    file_size: int
    format: str
    sha256: str | None = None
    phash: str | None = None


class DetectRequest(BaseModel):
    threshold: int = 10
    images: list[ImageInput]


class TaskResponse(BaseModel):
    task_id: str
    status: str
    progress: float = 0.0
    message: str = ""


class GroupMember(BaseModel):
    image_id: int
    sha256: str | None
    phash: str | None
    distance: int
    is_recommended: bool
    recommendation_score: float | None
    recommendation_reasons: list[dict[str, object]] | None


class DuplicateGroup(BaseModel):
    group_id: int
    recommended_id: int
    members: list[GroupMember]


class DetectionResult(BaseModel):
    groups: list[DuplicateGroup]
    total_images: int
    total_groups: int
    skipped_images: list[dict[str, object]]
    computation_time_ms: int
