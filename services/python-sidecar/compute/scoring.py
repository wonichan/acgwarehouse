FORMAT_PRIORITY = {
    "png": 1.0,
    "webp": 0.9,
    "jpeg": 0.7,
    "jpg": 0.7,
    "gif": 0.5,
    "bmp": 0.6,
}


def compute_recommendation_score(
    width: int,
    height: int,
    file_size: int,
    format: str,
) -> tuple[float, list[dict]]:
    resolution_score = min(100.0, (width * height / 8_294_400) * 100)
    file_size_score = min(100.0, (file_size / 10_485_760) * 100)
    format_score = FORMAT_PRIORITY.get((format or "").lower(), 0.5) * 100

    reasons = [
        {
            "factor": "resolution",
            "value": f"{width}x{height}",
            "score": round(resolution_score, 1),
            "weight": 0.5,
        },
        {
            "factor": "file_size",
            "value": str(file_size),
            "score": round(file_size_score, 1),
            "weight": 0.3,
        },
        {
            "factor": "format",
            "value": format or "unknown",
            "score": round(format_score, 1),
            "weight": 0.2,
        },
    ]

    composite = round(sum(reason["score"] * reason["weight"] for reason in reasons), 1)
    return composite, reasons


def select_recommended(members: list[dict]) -> tuple[int, float, list[dict]]:
    if not members:
        raise ValueError("members must not be empty")

    best_index = 0
    best_score, best_reasons = compute_recommendation_score(
        width=members[0]["width"],
        height=members[0]["height"],
        file_size=members[0]["file_size"],
        format=members[0]["format"],
    )

    for index in range(1, len(members)):
        score, reasons = compute_recommendation_score(
            width=members[index]["width"],
            height=members[index]["height"],
            file_size=members[index]["file_size"],
            format=members[index]["format"],
        )
        if score > best_score:
            best_index = index
            best_score = score
            best_reasons = reasons

    return best_index, best_score, best_reasons
