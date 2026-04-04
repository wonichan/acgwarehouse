"""Scoring module tests."""

# pyright: reportMissingImports=false

from compute.scoring import compute_recommendation_score, select_recommended


def test_compute_recommendation_score_returns_score_and_reasons():
    score, reasons = compute_recommendation_score(1920, 1080, 2_000_000, "png")

    assert isinstance(score, float)
    assert isinstance(reasons, list)


def test_reasons_contains_three_expected_factors():
    _, reasons = compute_recommendation_score(1920, 1080, 2_000_000, "png")

    factors = [reason["factor"] for reason in reasons]
    assert factors == ["resolution", "file_size", "format"]


def test_each_reason_has_factor_value_score_weight_keys():
    _, reasons = compute_recommendation_score(1920, 1080, 2_000_000, "png")

    for reason in reasons:
        assert set(reason.keys()) == {"factor", "value", "score", "weight"}


def test_higher_resolution_gets_higher_resolution_score():
    _, low_reasons = compute_recommendation_score(640, 360, 2_000_000, "png")
    _, high_reasons = compute_recommendation_score(3840, 2160, 2_000_000, "png")

    assert high_reasons[0]["score"] > low_reasons[0]["score"]


def test_png_gets_higher_format_score_than_jpeg():
    _, png_reasons = compute_recommendation_score(1920, 1080, 2_000_000, "png")
    _, jpeg_reasons = compute_recommendation_score(1920, 1080, 2_000_000, "jpeg")

    assert png_reasons[2]["score"] > jpeg_reasons[2]["score"]


def test_composite_score_is_weighted_sum():
    score, reasons = compute_recommendation_score(1920, 1080, 2_000_000, "png")
    expected = round(sum(reason["score"] * reason["weight"] for reason in reasons), 1)

    assert score == expected


def test_select_recommended_picks_highest_scoring_member():
    members = [
        {
            "image_id": 1,
            "width": 640,
            "height": 360,
            "file_size": 200_000,
            "format": "jpeg",
        },
        {
            "image_id": 2,
            "width": 3840,
            "height": 2160,
            "file_size": 8_000_000,
            "format": "png",
        },
    ]

    recommended_index, score, reasons = select_recommended(members)

    assert recommended_index == 1
    assert score > 0
    assert len(reasons) == 3


def test_select_recommended_single_member_returns_itself():
    members = [
        {
            "image_id": 1,
            "width": 640,
            "height": 360,
            "file_size": 200_000,
            "format": "jpeg",
        },
    ]

    recommended_index, score, reasons = select_recommended(members)

    assert recommended_index == 0
    assert score >= 0
    assert len(reasons) == 3
